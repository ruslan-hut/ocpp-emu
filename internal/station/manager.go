package station

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/connection"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Manager manages all charging stations
type Manager struct {
	stations      map[string]*Station
	mu            sync.RWMutex
	db            *storage.MongoDBClient
	connManager   *connection.Manager
	messageLogger *logging.MessageLogger
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	syncInterval  time.Duration
	syncWg        sync.WaitGroup
}

// Station represents a managed charging station instance
type Station struct {
	Config       Config
	StateMachine *StateMachine
	RuntimeState RuntimeState
	mu           sync.RWMutex
	lastSync     time.Time
}

// GetData returns a thread-safe copy of the station's config and runtime state
func (s *Station) GetData() (Config, RuntimeState) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config, s.RuntimeState
}

// ManagerConfig represents the manager configuration
type ManagerConfig struct {
	SyncInterval time.Duration // How often to sync state to MongoDB
}

// NewManager creates a new station manager
func NewManager(
	db *storage.MongoDBClient,
	connManager *connection.Manager,
	messageLogger *logging.MessageLogger,
	logger *slog.Logger,
	config ManagerConfig,
) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	if config.SyncInterval == 0 {
		config.SyncInterval = 30 * time.Second
	}

	return &Manager{
		stations:      make(map[string]*Station),
		db:            db,
		connManager:   connManager,
		messageLogger: messageLogger,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		syncInterval:  config.SyncInterval,
	}
}

// LoadStations loads all stations from MongoDB
func (m *Manager) LoadStations(ctx context.Context) error {
	m.logger.Info("Loading stations from MongoDB")

	collection := m.db.StationsCollection
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to query stations: %w", err)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		var dbStation storage.Station
		if err := cursor.Decode(&dbStation); err != nil {
			m.logger.Error("Failed to decode station", "error", err)
			continue
		}

		// Convert storage.Station to station.Config
		config := m.convertStorageToConfig(dbStation)

		// Create station instance
		station := &Station{
			Config:       config,
			StateMachine: NewStateMachine(),
			RuntimeState: RuntimeState{
				State:            StateDisconnected,
				ConnectionStatus: "not_connected",
			},
		}

		m.mu.Lock()
		m.stations[config.StationID] = station
		m.mu.Unlock()

		count++
		m.logger.Info("Loaded station",
			"stationId", config.StationID,
			"name", config.Name,
			"enabled", config.Enabled,
			"autoStart", config.AutoStart,
		)
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	m.logger.Info("Successfully loaded stations", "count", count)
	return nil
}

// AutoStart starts all stations with AutoStart=true
func (m *Manager) AutoStart(ctx context.Context) error {
	m.logger.Info("Starting auto-start stations")

	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for stationID, station := range m.stations {
		station.mu.RLock()
		shouldStart := station.Config.AutoStart && station.Config.Enabled
		station.mu.RUnlock()

		if shouldStart {
			if err := m.startStation(ctx, stationID); err != nil {
				m.logger.Error("Failed to auto-start station",
					"stationId", stationID,
					"error", err,
				)
				continue
			}
			count++
		}
	}

	m.logger.Info("Auto-start completed", "started", count)
	return nil
}

// StartStation starts a specific station
func (m *Manager) StartStation(ctx context.Context, stationID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.startStation(ctx, stationID)
}

// startStation is the internal start implementation (caller must hold read lock)
func (m *Manager) startStation(ctx context.Context, stationID string) error {
	station, exists := m.stations[stationID]
	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.mu.Lock()
	defer station.mu.Unlock()

	// Check if already started
	if station.StateMachine.IsConnected() {
		return fmt.Errorf("station already connected: %s", stationID)
	}

	if !station.Config.Enabled {
		return fmt.Errorf("station is disabled: %s", stationID)
	}

	m.logger.Info("Starting station", "stationId", stationID)

	// Update state
	station.StateMachine.SetState(StateConnecting, "manual start")
	station.RuntimeState.State = StateConnecting
	station.RuntimeState.ConnectionStatus = "connecting"

	// Create TLS and Auth configs
	var tlsConfig *connection.TLSConfig
	var authConfig *connection.AuthConfig

	if station.Config.CSMSAuth != nil && station.Config.CSMSAuth.Type == "basic" {
		authConfig = &connection.AuthConfig{
			Type:     "basic",
			Username: station.Config.CSMSAuth.Username,
			Password: station.Config.CSMSAuth.Password,
		}
	}

	// Initiate WebSocket connection
	err := m.connManager.ConnectStation(
		stationID,
		station.Config.CSMSURL,
		station.Config.ProtocolVersion,
		tlsConfig,
		authConfig,
	)
	if err != nil {
		station.StateMachine.SetState(StateFaulted, "connection failed")
		station.RuntimeState.State = StateFaulted
		station.RuntimeState.LastError = err.Error()
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

// StopStation stops a specific station
func (m *Manager) StopStation(ctx context.Context, stationID string) error {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.mu.Lock()
	defer station.mu.Unlock()

	m.logger.Info("Stopping station", "stationId", stationID)

	// Update state
	station.StateMachine.SetState(StateStopping, "manual stop")
	station.RuntimeState.State = StateStopping

	// Disconnect WebSocket
	if err := m.connManager.DisconnectStation(stationID); err != nil {
		m.logger.Error("Failed to disconnect station", "stationId", stationID, "error", err)
	}

	// Update final state
	station.StateMachine.SetState(StateDisconnected, "stopped")
	station.RuntimeState.State = StateDisconnected
	station.RuntimeState.ConnectionStatus = "disconnected"
	station.RuntimeState.ConnectedAt = nil

	return nil
}

// GetStation returns a station by ID
func (m *Manager) GetStation(stationID string) (*Station, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	station, exists := m.stations[stationID]
	if !exists {
		return nil, fmt.Errorf("station not found: %s", stationID)
	}

	return station, nil
}

// GetAllStations returns all stations
func (m *Manager) GetAllStations() map[string]*Station {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	stations := make(map[string]*Station, len(m.stations))
	for id, station := range m.stations {
		stations[id] = station
	}

	return stations
}

// AddStation adds a new station
func (m *Manager) AddStation(ctx context.Context, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.stations[config.StationID]; exists {
		return fmt.Errorf("station already exists: %s", config.StationID)
	}

	// Create station instance
	station := &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
		RuntimeState: RuntimeState{
			State:            StateDisconnected,
			ConnectionStatus: "not_connected",
		},
	}

	m.stations[config.StationID] = station

	// Persist to MongoDB
	if err := m.saveStationToDB(ctx, station); err != nil {
		delete(m.stations, config.StationID)
		return fmt.Errorf("failed to save station to database: %w", err)
	}

	m.logger.Info("Added new station", "stationId", config.StationID)
	return nil
}

// RemoveStation removes a station
func (m *Manager) RemoveStation(ctx context.Context, stationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, exists := m.stations[stationID]
	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	// Stop station if running
	if station.StateMachine.IsConnected() {
		m.mu.Unlock()
		if err := m.StopStation(ctx, stationID); err != nil {
			m.mu.Lock()
			return fmt.Errorf("failed to stop station: %w", err)
		}
		m.mu.Lock()
	}

	// Remove from memory
	delete(m.stations, stationID)

	// Remove from MongoDB
	collection := m.db.StationsCollection
	_, err := collection.DeleteOne(ctx, bson.M{"stationId": stationID})
	if err != nil {
		return fmt.Errorf("failed to delete station from database: %w", err)
	}

	m.logger.Info("Removed station", "stationId", stationID)
	return nil
}

// UpdateStation updates an existing station configuration
func (m *Manager) UpdateStation(ctx context.Context, stationID string, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, exists := m.stations[stationID]
	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.mu.Lock()
	station.Config = config
	station.Config.UpdatedAt = time.Now()
	station.mu.Unlock()

	// Persist to MongoDB
	if err := m.saveStationToDB(ctx, station); err != nil {
		return fmt.Errorf("failed to update station in database: %w", err)
	}

	m.logger.Info("Updated station", "stationId", stationID)
	return nil
}

// StartSync starts the background state synchronization
func (m *Manager) StartSync() {
	m.syncWg.Add(1)
	go m.syncLoop()
	m.logger.Info("Started state synchronization", "interval", m.syncInterval)
}

// syncLoop periodically syncs runtime state to MongoDB
func (m *Manager) syncLoop() {
	defer m.syncWg.Done()

	ticker := time.NewTicker(m.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("Stopping state synchronization")
			return
		case <-ticker.C:
			m.syncState()
		}
	}
}

// syncState synchronizes runtime state to MongoDB
func (m *Manager) syncState() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count := 0
	for stationID, station := range m.stations {
		station.mu.Lock()

		// Only sync if changed since last sync
		if time.Since(station.lastSync) < m.syncInterval/2 {
			station.mu.Unlock()
			continue
		}

		if err := m.saveStationToDB(ctx, station); err != nil {
			m.logger.Error("Failed to sync station state",
				"stationId", stationID,
				"error", err,
			)
		} else {
			station.lastSync = time.Now()
			count++
		}

		station.mu.Unlock()
	}

	if count > 0 {
		m.logger.Debug("Synchronized station states", "count", count)
	}
}

// OnStationConnected handles station connection events
func (m *Manager) OnStationConnected(stationID string) {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		m.logger.Warn("Connected station not found in manager", "stationId", stationID)
		return
	}

	station.mu.Lock()
	defer station.mu.Unlock()

	now := time.Now()
	station.StateMachine.SetState(StateConnected, "websocket connected")
	station.RuntimeState.State = StateConnected
	station.RuntimeState.ConnectionStatus = "connected"
	station.RuntimeState.ConnectedAt = &now
	station.RuntimeState.LastError = ""

	m.logger.Info("Station connected", "stationId", stationID)

	// Send BootNotification
	go m.sendBootNotification(stationID)
}

// OnStationDisconnected handles station disconnection events
func (m *Manager) OnStationDisconnected(stationID string, err error) {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	station.mu.Lock()
	defer station.mu.Unlock()

	station.StateMachine.SetState(StateDisconnected, "websocket disconnected")
	station.RuntimeState.State = StateDisconnected
	station.RuntimeState.ConnectionStatus = "disconnected"
	station.RuntimeState.ConnectedAt = nil

	if err != nil {
		station.RuntimeState.LastError = err.Error()
		m.logger.Error("Station disconnected with error", "stationId", stationID, "error", err)
	} else {
		m.logger.Info("Station disconnected", "stationId", stationID)
	}
}

// OnMessageReceived handles incoming OCPP messages
func (m *Manager) OnMessageReceived(stationID string, message []byte) {
	m.logger.Debug("Received message from station", "stationId", stationID, "size", len(message))

	// Parse OCPP message
	msg, err := ocpp.ParseMessage(message)
	if err != nil {
		m.logger.Error("Failed to parse OCPP message", "stationId", stationID, "error", err)
		return
	}

	// Handle different message types
	switch typedMsg := msg.(type) {
	case *ocpp.Call:
		m.handleCall(stationID, typedMsg)
	case *ocpp.CallResult:
		m.handleCallResult(stationID, typedMsg)
	case *ocpp.CallError:
		m.handleCallError(stationID, typedMsg)
	default:
		m.logger.Warn("Unknown message type", "stationId", stationID)
	}
}

// handleCall handles incoming Call messages from CSMS
func (m *Manager) handleCall(stationID string, call *ocpp.Call) {
	m.logger.Info("Received Call", "stationId", stationID, "action", call.Action)

	// Store message in MongoDB
	go m.storeMessage(stationID, "received", call)

	// Handle different actions (to be implemented)
	switch call.Action {
	case string(v16.ActionRemoteStartTransaction):
		// TODO: Handle RemoteStartTransaction
	case string(v16.ActionRemoteStopTransaction):
		// TODO: Handle RemoteStopTransaction
	case string(v16.ActionReset):
		// TODO: Handle Reset
	case string(v16.ActionChangeConfiguration):
		// TODO: Handle ChangeConfiguration
	default:
		// Send NotImplemented error
		m.sendNotImplementedError(stationID, call.UniqueID, call.Action)
	}
}

// handleCallResult handles CallResult responses
func (m *Manager) handleCallResult(stationID string, result *ocpp.CallResult) {
	m.logger.Info("Received CallResult", "stationId", stationID, "uniqueId", result.UniqueID)

	// Store message in MongoDB
	go m.storeMessage(stationID, "received", result)

	// TODO: Match with pending requests and update state
}

// handleCallError handles CallError responses
func (m *Manager) handleCallError(stationID string, callError *ocpp.CallError) {
	m.logger.Error("Received CallError",
		"stationId", stationID,
		"uniqueId", callError.UniqueID,
		"errorCode", callError.ErrorCode,
		"errorDesc", callError.ErrorDesc,
	)

	// Store message in MongoDB
	go m.storeMessage(stationID, "received", callError)
}

// sendBootNotification sends a BootNotification request
func (m *Manager) sendBootNotification(stationID string) {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	station.mu.RLock()
	req := v16.BootNotificationRequest{
		ChargePointVendor:       station.Config.Vendor,
		ChargePointModel:        station.Config.Model,
		ChargePointSerialNumber: station.Config.SerialNumber,
		FirmwareVersion:         station.Config.FirmwareVersion,
		Iccid:                   station.Config.ICCID,
		Imsi:                    station.Config.IMSI,
	}
	station.mu.RUnlock()

	call, err := ocpp.NewCall(string(v16.ActionBootNotification), req)
	if err != nil {
		m.logger.Error("Failed to create BootNotification", "stationId", stationID, "error", err)
		return
	}

	data, err := call.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal BootNotification", "stationId", stationID, "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send BootNotification", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("Sent BootNotification", "stationId", stationID)

	// Store sent message
	go m.storeMessage(stationID, "sent", call)
}

// sendNotImplementedError sends a NotImplemented error response
func (m *Manager) sendNotImplementedError(stationID, uniqueID, action string) {
	callError, err := ocpp.NewCallError(
		uniqueID,
		ocpp.ErrorCodeNotImplemented,
		fmt.Sprintf("Action %s not implemented", action),
		nil,
	)
	if err != nil {
		m.logger.Error("Failed to create CallError", "error", err)
		return
	}

	data, err := callError.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal CallError", "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send CallError", "stationId", stationID, "error", err)
	}
}

// storeMessage stores a message using the message logger
func (m *Manager) storeMessage(stationID, direction string, message interface{}) {
	// Get protocol version from station
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	protocolVersion := "ocpp1.6" // default
	if exists {
		station.mu.RLock()
		protocolVersion = station.Config.ProtocolVersion
		station.mu.RUnlock()
	}

	// Log message using message logger
	if err := m.messageLogger.LogMessage(stationID, direction, message, protocolVersion); err != nil {
		m.logger.Error("Failed to log message",
			"stationId", stationID,
			"direction", direction,
			"error", err,
		)
	}
}

// saveStationToDB persists station to MongoDB
func (m *Manager) saveStationToDB(ctx context.Context, station *Station) error {
	dbStation := m.convertConfigToStorage(station.Config)

	collection := m.db.StationsCollection

	opts := options.Replace().SetUpsert(true)
	_, err := collection.ReplaceOne(
		ctx,
		bson.M{"stationId": station.Config.StationID},
		dbStation,
		opts,
	)

	return err
}

// Note: createConnectionConfig method removed as connection configuration
// is now handled directly through TLSConfig and AuthConfig parameters
// in the ConnectStation method call

// convertStorageToConfig converts storage.Station to station.Config
func (m *Manager) convertStorageToConfig(dbStation storage.Station) Config {
	connectors := make([]ConnectorConfig, len(dbStation.Connectors))
	for i, conn := range dbStation.Connectors {
		connectors[i] = ConnectorConfig{
			ID:                   conn.ID,
			Type:                 conn.Type,
			MaxPower:             conn.MaxPower,
			Status:               conn.Status,
			CurrentTransactionID: conn.CurrentTransactionID,
		}
	}

	var csmsAuth *CSMSAuthConfig
	if dbStation.CSMSAuth.Type != "" {
		csmsAuth = &CSMSAuthConfig{
			Type:     dbStation.CSMSAuth.Type,
			Username: dbStation.CSMSAuth.Username,
			Password: dbStation.CSMSAuth.Password,
		}
	}

	return Config{
		ID:                dbStation.ID,
		StationID:         dbStation.StationID,
		Name:              dbStation.Name,
		Enabled:           dbStation.Enabled,
		AutoStart:         dbStation.AutoStart,
		ProtocolVersion:   dbStation.ProtocolVersion,
		Vendor:            dbStation.Vendor,
		Model:             dbStation.Model,
		SerialNumber:      dbStation.SerialNumber,
		FirmwareVersion:   dbStation.FirmwareVersion,
		ICCID:             dbStation.ICCID,
		IMSI:              dbStation.IMSI,
		Connectors:        connectors,
		SupportedProfiles: dbStation.SupportedProfiles,
		MeterValuesConfig: MeterValuesConfig{
			Interval:            dbStation.MeterValuesConfig.Interval,
			Measurands:          dbStation.MeterValuesConfig.Measurands,
			AlignedDataInterval: dbStation.MeterValuesConfig.AlignedDataInterval,
		},
		CSMSURL:  dbStation.CSMSURL,
		CSMSAuth: csmsAuth,
		Simulation: SimulationConfig{
			BootDelay:                  dbStation.Simulation.BootDelay,
			HeartbeatInterval:          dbStation.Simulation.HeartbeatInterval,
			StatusNotificationOnChange: dbStation.Simulation.StatusNotificationOnChange,
			DefaultIDTag:               dbStation.Simulation.DefaultIDTag,
			EnergyDeliveryRate:         dbStation.Simulation.EnergyDeliveryRate,
			RandomizeMeterValues:       dbStation.Simulation.RandomizeMeterValues,
			MeterValueVariance:         dbStation.Simulation.MeterValueVariance,
		},
		CreatedAt: dbStation.CreatedAt,
		UpdatedAt: dbStation.UpdatedAt,
		Tags:      dbStation.Tags,
	}
}

// convertConfigToStorage converts station.Config to storage.Station
func (m *Manager) convertConfigToStorage(config Config) storage.Station {
	connectors := make([]storage.Connector, len(config.Connectors))
	for i, conn := range config.Connectors {
		connectors[i] = storage.Connector{
			ID:                   conn.ID,
			Type:                 conn.Type,
			MaxPower:             conn.MaxPower,
			Status:               conn.Status,
			CurrentTransactionID: conn.CurrentTransactionID,
		}
	}

	var csmsAuth storage.CSMSAuth
	if config.CSMSAuth != nil {
		csmsAuth = storage.CSMSAuth{
			Type:     config.CSMSAuth.Type,
			Username: config.CSMSAuth.Username,
			Password: config.CSMSAuth.Password,
		}
	}

	return storage.Station{
		ID:                config.ID,
		StationID:         config.StationID,
		Name:              config.Name,
		Enabled:           config.Enabled,
		AutoStart:         config.AutoStart,
		ProtocolVersion:   config.ProtocolVersion,
		Vendor:            config.Vendor,
		Model:             config.Model,
		SerialNumber:      config.SerialNumber,
		FirmwareVersion:   config.FirmwareVersion,
		ICCID:             config.ICCID,
		IMSI:              config.IMSI,
		Connectors:        connectors,
		SupportedProfiles: config.SupportedProfiles,
		MeterValuesConfig: storage.MeterValuesConfig{
			Interval:            config.MeterValuesConfig.Interval,
			Measurands:          config.MeterValuesConfig.Measurands,
			AlignedDataInterval: config.MeterValuesConfig.AlignedDataInterval,
		},
		CSMSURL:  config.CSMSURL,
		CSMSAuth: csmsAuth,
		Simulation: storage.SimulationConfig{
			BootDelay:                  config.Simulation.BootDelay,
			HeartbeatInterval:          config.Simulation.HeartbeatInterval,
			StatusNotificationOnChange: config.Simulation.StatusNotificationOnChange,
			DefaultIDTag:               config.Simulation.DefaultIDTag,
			EnergyDeliveryRate:         config.Simulation.EnergyDeliveryRate,
			RandomizeMeterValues:       config.Simulation.RandomizeMeterValues,
			MeterValueVariance:         config.Simulation.MeterValueVariance,
		},
		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
		Tags:      config.Tags,
	}
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down station manager")

	// Cancel context to stop sync loop
	m.cancel()

	// Wait for sync loop to finish
	m.syncWg.Wait()

	// Stop all stations
	m.mu.RLock()
	stationIDs := make([]string, 0, len(m.stations))
	for stationID := range m.stations {
		stationIDs = append(stationIDs, stationID)
	}
	m.mu.RUnlock()

	for _, stationID := range stationIDs {
		if err := m.StopStation(ctx, stationID); err != nil {
			m.logger.Error("Failed to stop station during shutdown", "stationId", stationID, "error", err)
		}
	}

	// Final state sync
	m.syncState()

	// Shutdown message logger
	if m.messageLogger != nil {
		if err := m.messageLogger.Shutdown(); err != nil {
			m.logger.Error("Failed to shutdown message logger", "error", err)
		}
	}

	m.logger.Info("Station manager shutdown complete")
	return nil
}

// GetStats returns manager statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connected, disconnected, charging, available, faulted int

	for _, station := range m.stations {
		station.mu.RLock()
		state := station.StateMachine.GetState()
		station.mu.RUnlock()

		switch state {
		case StateConnected, StateRegistered:
			connected++
		case StateDisconnected:
			disconnected++
		case StateCharging:
			charging++
		case StateAvailable:
			available++
		case StateFaulted:
			faulted++
		}
	}

	stats := map[string]interface{}{
		"total":        len(m.stations),
		"connected":    connected,
		"disconnected": disconnected,
		"charging":     charging,
		"available":    available,
		"faulted":      faulted,
		"syncInterval": m.syncInterval.String(),
	}

	// Add message logger stats if available
	if m.messageLogger != nil {
		loggerStats := m.messageLogger.GetStats()
		stats["messages"] = map[string]interface{}{
			"total":              loggerStats.TotalMessages,
			"sent":               loggerStats.SentMessages,
			"received":           loggerStats.ReceivedMessages,
			"buffered":           loggerStats.BufferedMessages,
			"dropped":            loggerStats.DroppedMessages,
			"callMessages":       loggerStats.CallMessages,
			"callResultMessages": loggerStats.CallResultMessages,
			"callErrorMessages":  loggerStats.CallErrorMessages,
			"lastFlush":          loggerStats.LastFlush,
			"flushCount":         loggerStats.FlushCount,
		}
	}

	return stats
}
