package station

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/connection"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
	v16 "github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
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
	v16Handler    *v16.Handler // OCPP 1.6 message handler
}

// Station represents a managed charging station instance
type Station struct {
	Config         Config
	StateMachine   *StateMachine
	RuntimeState   RuntimeState
	SessionManager *SessionManager // Enhanced session manager for charging
	mu             sync.RWMutex
	lastSync       time.Time

	// Heartbeat management
	heartbeatCancel context.CancelFunc
	heartbeatDone   chan struct{}

	// Pending requests tracking (message ID -> action)
	pendingRequests map[string]string
	pendingMu       sync.RWMutex

	// Pending StartTransaction tracking (message ID -> {connector ID, idTag})
	// Used to update the correct connector when CSMS responds with transaction ID
	pendingStartTx   map[string]int
	pendingStartTags map[string]string
	pendingStartMu   sync.RWMutex

	// Pending Authorize tracking (message ID -> response channel)
	// Used to wait for actual CSMS response
	pendingAuthResp   map[string]chan *v16.AuthorizeResponse
	pendingAuthRespMu sync.RWMutex

	// Failed authorizations tracking (idTag -> timestamp)
	// Used to reject transactions for recently rejected ID tags
	failedAuths   map[string]time.Time
	failedAuthsMu sync.RWMutex
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

	m := &Manager{
		stations:      make(map[string]*Station),
		db:            db,
		connManager:   connManager,
		messageLogger: messageLogger,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		syncInterval:  config.SyncInterval,
	}

	// Initialize OCPP 1.6 handler
	m.v16Handler = v16.NewHandler(logger)
	m.v16Handler.SendMessage = connManager.SendMessage
	m.setupV16HandlerCallbacks()

	return m
}

// setupV16HandlerCallbacks sets up callbacks for OCPP 1.6 handler
func (m *Manager) setupV16HandlerCallbacks() {
	// RemoteStartTransaction handler
	m.v16Handler.OnRemoteStartTransaction = func(stationID string, req *v16.RemoteStartTransactionRequest) (*v16.RemoteStartTransactionResponse, error) {
		m.logger.Info("Handling RemoteStartTransaction", "stationId", stationID, "idTag", req.IdTag)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v16.RemoteStartTransactionResponse{Status: "Rejected"}, nil
		}

		// Determine connector ID
		connectorID := 1
		if req.ConnectorId != nil {
			connectorID = *req.ConnectorId
		}

		// Start charging session
		_, err := station.SessionManager.StartCharging(connectorID, req.IdTag)
		if err != nil {
			m.logger.Error("Failed to start charging", "error", err)
			return &v16.RemoteStartTransactionResponse{Status: "Rejected"}, nil
		}

		return &v16.RemoteStartTransactionResponse{
			Status: "Accepted",
		}, nil
	}

	// RemoteStopTransaction handler
	m.v16Handler.OnRemoteStopTransaction = func(stationID string, req *v16.RemoteStopTransactionRequest) (*v16.RemoteStopTransactionResponse, error) {
		m.logger.Info("Handling RemoteStopTransaction", "stationId", stationID, "transactionId", req.TransactionId)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v16.RemoteStopTransactionResponse{Status: "Rejected"}, nil
		}

		// Find connector with this transaction
		connectors := station.SessionManager.GetAllConnectors()
		var targetConnectorID int
		found := false

		for _, connector := range connectors {
			tx := connector.GetTransaction()
			if tx != nil && tx.ID == req.TransactionId {
				targetConnectorID = connector.ID
				found = true
				break
			}
		}

		if !found {
			m.logger.Warn("Transaction not found", "transactionId", req.TransactionId)
			return &v16.RemoteStopTransactionResponse{Status: "Rejected"}, nil
		}

		// Stop charging session
		err := station.SessionManager.StopCharging(targetConnectorID, v16.ReasonRemote)
		if err != nil {
			m.logger.Error("Failed to stop charging", "error", err)
			return &v16.RemoteStopTransactionResponse{Status: "Rejected"}, nil
		}

		return &v16.RemoteStopTransactionResponse{
			Status: "Accepted",
		}, nil
	}

	// Reset handler
	m.v16Handler.OnReset = func(stationID string, req *v16.ResetRequest) (*v16.ResetResponse, error) {
		m.logger.Info("Handling Reset", "stationId", stationID, "type", req.Type)

		// TODO: Implement actual reset logic
		// For now, accept the request
		return &v16.ResetResponse{
			Status: "Accepted",
		}, nil
	}

	// UnlockConnector handler
	m.v16Handler.OnUnlockConnector = func(stationID string, req *v16.UnlockConnectorRequest) (*v16.UnlockConnectorResponse, error) {
		m.logger.Info("Handling UnlockConnector", "stationId", stationID, "connectorId", req.ConnectorId)

		// TODO: Implement actual unlock logic
		// For now, return not supported
		return &v16.UnlockConnectorResponse{
			Status: "NotSupported",
		}, nil
	}

	// ChangeAvailability handler
	m.v16Handler.OnChangeAvailability = func(stationID string, req *v16.ChangeAvailabilityRequest) (*v16.ChangeAvailabilityResponse, error) {
		m.logger.Info("Handling ChangeAvailability", "stationId", stationID, "connectorId", req.ConnectorId, "type", req.Type)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v16.ChangeAvailabilityResponse{Status: "Rejected"}, nil
		}

		// Change availability
		err := station.SessionManager.ChangeAvailability(req.ConnectorId, req.Type)
		if err != nil {
			// If changing while charging, schedule for later
			m.logger.Warn("Cannot change availability immediately", "error", err)
			return &v16.ChangeAvailabilityResponse{Status: "Scheduled"}, nil
		}

		return &v16.ChangeAvailabilityResponse{
			Status: "Accepted",
		}, nil
	}

	// ChangeConfiguration handler
	m.v16Handler.OnChangeConfiguration = func(stationID string, req *v16.ChangeConfigurationRequest) (*v16.ChangeConfigurationResponse, error) {
		m.logger.Info("Handling ChangeConfiguration", "stationId", stationID, "key", req.Key, "value", req.Value)

		// TODO: Implement actual configuration change logic
		// For now, return not supported
		return &v16.ChangeConfigurationResponse{
			Status: "NotSupported",
		}, nil
	}

	// GetConfiguration handler
	m.v16Handler.OnGetConfiguration = func(stationID string, req *v16.GetConfigurationRequest) (*v16.GetConfigurationResponse, error) {
		m.logger.Info("Handling GetConfiguration", "stationId", stationID, "keys", req.Key)

		// TODO: Implement actual configuration retrieval logic
		// For now, return empty response
		return &v16.GetConfigurationResponse{
			ConfigurationKey: []v16.KeyValue{},
			UnknownKey:       req.Key,
		}, nil
	}

	// ClearCache handler
	m.v16Handler.OnClearCache = func(stationID string, req *v16.ClearCacheRequest) (*v16.ClearCacheResponse, error) {
		m.logger.Info("Handling ClearCache", "stationId", stationID)

		// TODO: Implement actual cache clear logic
		// For now, accept the request
		return &v16.ClearCacheResponse{
			Status: "Accepted",
		}, nil
	}

	// DataTransfer handler
	m.v16Handler.OnDataTransfer = func(stationID string, req *v16.DataTransferRequest) (*v16.DataTransferResponse, error) {
		m.logger.Info("Handling DataTransfer", "stationId", stationID, "vendorId", req.VendorId, "messageId", req.MessageId)

		// TODO: Implement actual data transfer logic
		// For now, return unknown vendor
		return &v16.DataTransferResponse{
			Status: "UnknownVendorId",
		}, nil
	}
}

// setupSessionManagerCallbacks wires up SessionManager callbacks to OCPP message handlers
func (m *Manager) setupSessionManagerCallbacks(station *Station) {
	stationID := station.Config.StationID

	// SendStatusNotification - sends status notification to CSMS
	station.SessionManager.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		req := &v16.StatusNotificationRequest{
			ConnectorId:     connectorID,
			Status:          status,
			ErrorCode:       errorCode,
			Info:            info,
			VendorId:        station.Config.Vendor,
			VendorErrorCode: "",
		}

		call, err := m.v16Handler.SendStatusNotification(stationID, req)
		if err != nil {
			m.logger.Error("Failed to send StatusNotification",
				"stationId", stationID,
				"connectorId", connectorID,
				"error", err,
			)
			return err
		}

		// Store sent message
		go m.storeMessage(stationID, "sent", call)

		return nil
	}

	// SendMeterValues - sends meter values to CSMS
	station.SessionManager.SendMeterValues = func(connectorID int, transactionID *int, meterValues []v16.MeterValue) error {
		req := &v16.MeterValuesRequest{
			ConnectorId:   connectorID,
			TransactionId: transactionID,
			MeterValue:    meterValues,
		}

		call, err := m.v16Handler.SendMeterValues(stationID, req)
		if err != nil {
			m.logger.Error("Failed to send MeterValues",
				"stationId", stationID,
				"connectorId", connectorID,
				"error", err,
			)
			return err
		}

		// Store sent message
		go m.storeMessage(stationID, "sent", call)

		return nil
	}

	// SendAuthorize - sends authorization request to CSMS and waits for response
	station.SessionManager.SendAuthorize = func(idTag string) (*v16.AuthorizeResponse, error) {
		req := &v16.AuthorizeRequest{
			IdTag: idTag,
		}

		call, err := m.v16Handler.SendAuthorize(stationID, req)
		if err != nil {
			m.logger.Error("Failed to send Authorize",
				"stationId", stationID,
				"idTag", idTag,
				"error", err,
			)
			return nil, err
		}

		// Track pending request
		station.pendingMu.Lock()
		station.pendingRequests[call.UniqueID] = string(v16.ActionAuthorize)
		station.pendingMu.Unlock()

		// Create response channel and register it
		respChan := make(chan *v16.AuthorizeResponse, 1)
		station.pendingAuthRespMu.Lock()
		station.pendingAuthResp[call.UniqueID] = respChan
		station.pendingAuthRespMu.Unlock()

		// Store sent message
		go m.storeMessage(stationID, "sent", call)

		m.logger.Debug("Waiting for Authorize response from CSMS",
			"stationId", stationID,
			"idTag", idTag,
			"messageId", call.UniqueID,
		)

		// Wait for response with timeout
		select {
		case resp := <-respChan:
			m.logger.Info("Received real Authorize response",
				"stationId", stationID,
				"idTag", idTag,
				"status", resp.IdTagInfo.Status,
			)
			return resp, nil
		case <-time.After(10 * time.Second):
			// Cleanup on timeout
			station.pendingAuthRespMu.Lock()
			delete(station.pendingAuthResp, call.UniqueID)
			station.pendingAuthRespMu.Unlock()

			m.logger.Error("Timeout waiting for Authorize response",
				"stationId", stationID,
				"idTag", idTag,
			)
			return nil, fmt.Errorf("timeout waiting for authorization response")
		}
	}

	// SendStartTransaction - sends start transaction request to CSMS
	station.SessionManager.SendStartTransaction = func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error) {
		req := &v16.StartTransactionRequest{
			ConnectorId: connectorID,
			IdTag:       idTag,
			MeterStart:  meterStart,
			Timestamp:   v16.DateTime{Time: timestamp},
		}

		call, err := m.v16Handler.SendStartTransaction(stationID, req)
		if err != nil {
			m.logger.Error("Failed to send StartTransaction",
				"stationId", stationID,
				"connectorId", connectorID,
				"idTag", idTag,
				"error", err,
			)
			return nil, err
		}

		// Track pending request
		station.pendingMu.Lock()
		station.pendingRequests[call.UniqueID] = string(v16.ActionStartTransaction)
		station.pendingMu.Unlock()

		// Track which connector and idTag are starting this transaction
		station.pendingStartMu.Lock()
		station.pendingStartTx[call.UniqueID] = connectorID
		station.pendingStartTags[call.UniqueID] = idTag
		station.pendingStartMu.Unlock()

		m.logger.Info("Tracking StartTransaction for response",
			"stationId", stationID,
			"messageId", call.UniqueID,
			"connectorId", connectorID,
			"idTag", idTag,
		)

		// Store sent message
		go m.storeMessage(stationID, "sent", call)

		// For now, return a placeholder transaction ID
		// TODO: Implement async request/response tracking
		// The real transaction ID comes from CSMS response
		return &v16.StartTransactionResponse{
			TransactionId: 1,
			IdTagInfo: v16.IdTagInfo{
				Status: "Accepted",
			},
		}, nil
	}

	// SendStopTransaction - sends stop transaction request to CSMS
	station.SessionManager.SendStopTransaction = func(transactionID int, idTag string, meterStop int, timestamp time.Time, reason v16.Reason) (*v16.StopTransactionResponse, error) {
		req := &v16.StopTransactionRequest{
			TransactionId: transactionID,
			IdTag:         idTag,
			MeterStop:     meterStop,
			Timestamp:     v16.DateTime{Time: timestamp},
			Reason:        reason,
		}

		call, err := m.v16Handler.SendStopTransaction(stationID, req)
		if err != nil {
			m.logger.Error("Failed to send StopTransaction",
				"stationId", stationID,
				"transactionId", transactionID,
				"error", err,
			)
			return nil, err
		}

		// Track pending request
		station.pendingMu.Lock()
		station.pendingRequests[call.UniqueID] = string(v16.ActionStopTransaction)
		station.pendingMu.Unlock()

		// Store sent message
		go m.storeMessage(stationID, "sent", call)

		// For now, return accepted status
		// TODO: Implement async request/response tracking
		return &v16.StopTransactionResponse{
			IdTagInfo: &v16.IdTagInfo{
				Status: "Accepted",
			},
		}, nil
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

		// Create session manager for the station
		sessionManager := NewSessionManager(config.StationID, config.Connectors, m.logger)

		// Set up transaction repository
		transactionRepo := storage.NewTransactionRepository(m.db)
		sessionManager.SetTransactionRepository(transactionRepo)
		sessionManager.SetProtocolVersion(config.ProtocolVersion)

		// Create station instance
		station := &Station{
			Config:            config,
			StateMachine:      NewStateMachine(),
			SessionManager:    sessionManager,
			pendingRequests:   make(map[string]string),
			pendingStartTx:    make(map[string]int),
			pendingStartTags:  make(map[string]string),
			pendingAuthResp:   make(map[string]chan *v16.AuthorizeResponse),
			failedAuths:       make(map[string]time.Time),
			RuntimeState: RuntimeState{
				State:            StateDisconnected,
				ConnectionStatus: "not_connected",
			},
		}

		sessionManager.SetStationStateCallback(func(newState State, reason string) {
			station.mu.RLock()
			isConnected := station.RuntimeState.ConnectionStatus == "connected"
			station.mu.RUnlock()

			if !isConnected {
				return
			}

			station.StateMachine.SetState(newState, reason)

			station.mu.Lock()
			station.RuntimeState.State = newState
			station.mu.Unlock()
		})

		// Set up session manager callbacks
		m.setupSessionManagerCallbacks(station)

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

// ReconcileStationData reconciles connector states with active transactions from MongoDB
// This ensures data consistency after restarts
func (m *Manager) ReconcileStationData(ctx context.Context) error {
	m.logger.Info("Reconciling station data with MongoDB")

	m.mu.RLock()
	stationIDs := make([]string, 0, len(m.stations))
	for stationID := range m.stations {
		stationIDs = append(stationIDs, stationID)
	}
	m.mu.RUnlock()

	transactionRepo := storage.NewTransactionRepository(m.db)
	reconciledCount := 0
	resetCount := 0

	for _, stationID := range stationIDs {
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			continue
		}

		// Get all active transactions for this station from MongoDB
		activeTransactions, err := transactionRepo.GetActive(ctx, stationID)
		if err != nil {
			m.logger.Error("Failed to get active transactions for reconciliation",
				"stationId", stationID,
				"error", err,
			)
			continue
		}

		// Build a map of connector ID -> active transaction
		txByConnector := make(map[int]*storage.Transaction)
		for i := range activeTransactions {
			tx := &activeTransactions[i]
			txByConnector[tx.ConnectorID] = tx
		}

		// Check each connector
		for _, connector := range station.SessionManager.connectors {
			connectorID := connector.ID
			currentState := connector.GetState()

			// Check if connector is in a transaction-related state
			isTransactionState := currentState == ConnectorStateCharging ||
				currentState == ConnectorStatePreparing ||
				currentState == ConnectorStateSuspendedEV ||
				currentState == ConnectorStateSuspendedEVSE ||
				currentState == ConnectorStateFinishing

			dbTransaction, hasDBTransaction := txByConnector[connectorID]

			if isTransactionState && !hasDBTransaction {
				// Connector thinks it has a transaction but DB doesn't - reset it
				m.logger.Warn("Connector in transaction state but no active transaction in DB - resetting",
					"stationId", stationID,
					"connectorId", connectorID,
					"state", currentState,
				)

				err := connector.SetState(ConnectorStateAvailable, v16.ChargePointErrorNoError, "Reset after restart - no transaction found")
				if err != nil {
					m.logger.Error("Failed to reset connector state",
						"stationId", stationID,
						"connectorId", connectorID,
						"error", err,
					)
				} else {
					resetCount++
				}
			} else if !isTransactionState && hasDBTransaction {
				// DB has transaction but connector doesn't think it's charging - restore it
				m.logger.Warn("Active transaction in DB but connector not in transaction state - restoring",
					"stationId", stationID,
					"connectorId", connectorID,
					"transactionId", dbTransaction.TransactionID,
					"state", currentState,
				)

				// Restore transaction to connector
				connector.mu.Lock()
				connector.Transaction = &Transaction{
					ID:              dbTransaction.TransactionID,
					IDTag:           dbTransaction.IDTag,
					ConnectorID:     connectorID,
					StartTime:       dbTransaction.StartTimestamp,
					StartMeterValue: dbTransaction.MeterStart,
					CurrentMeter:    dbTransaction.MeterStart, // Will be updated by meter values
					MeterValues:     make([]MeterValueSample, 0),
				}
				connector.mu.Unlock()

				// Set connector state to Charging
				err := connector.SetState(ConnectorStateCharging, v16.ChargePointErrorNoError, "Restored from database")
				if err != nil {
					m.logger.Error("Failed to restore connector state",
						"stationId", stationID,
						"connectorId", connectorID,
						"error", err,
					)
				} else {
					// Resume meter value simulation
					if err := station.SessionManager.ResumeMeterValues(connectorID); err != nil {
						m.logger.Error("Failed to resume meter values",
							"stationId", stationID,
							"connectorId", connectorID,
							"error", err,
						)
					}
					reconciledCount++
				}
			} else if isTransactionState && hasDBTransaction {
				// Both agree there's a transaction - ensure transaction ID matches
				currentTx := connector.GetTransaction()
				if currentTx == nil {
					// Connector state says transaction but no transaction object - restore it
					connector.mu.Lock()
					connector.Transaction = &Transaction{
						ID:              dbTransaction.TransactionID,
						IDTag:           dbTransaction.IDTag,
						ConnectorID:     connectorID,
						StartTime:       dbTransaction.StartTimestamp,
						StartMeterValue: dbTransaction.MeterStart,
						CurrentMeter:    dbTransaction.MeterStart,
						MeterValues:     make([]MeterValueSample, 0),
					}
					connector.mu.Unlock()

					m.logger.Info("Restored transaction object to connector",
						"stationId", stationID,
						"connectorId", connectorID,
						"transactionId", dbTransaction.TransactionID,
					)

					// Resume meter value simulation
					if err := station.SessionManager.ResumeMeterValues(connectorID); err != nil {
						m.logger.Error("Failed to resume meter values",
							"stationId", stationID,
							"connectorId", connectorID,
							"error", err,
						)
					}
					reconciledCount++
				} else if currentTx.ID != dbTransaction.TransactionID {
					// Transaction IDs don't match - update to DB value
					m.logger.Warn("Transaction ID mismatch - updating to DB value",
						"stationId", stationID,
						"connectorId", connectorID,
						"currentId", currentTx.ID,
						"dbId", dbTransaction.TransactionID,
					)

					err := connector.UpdateTransactionID(dbTransaction.TransactionID)
					if err != nil {
						m.logger.Error("Failed to update transaction ID",
							"stationId", stationID,
							"connectorId", connectorID,
							"error", err,
						)
					} else {
						reconciledCount++
					}
				} else {
					// Transaction exists and IDs match - ensure meter values are running
					// Check if meter value ticker exists
					station.SessionManager.mu.RLock()
					_, hasTicker := station.SessionManager.meterValueTickers[connectorID]
					station.SessionManager.mu.RUnlock()

					if !hasTicker {
						m.logger.Info("Meter value simulation not running - resuming",
							"stationId", stationID,
							"connectorId", connectorID,
							"transactionId", currentTx.ID,
						)

						if err := station.SessionManager.ResumeMeterValues(connectorID); err != nil {
							m.logger.Error("Failed to resume meter values",
								"stationId", stationID,
								"connectorId", connectorID,
								"error", err,
							)
						}
					}
				}
			}
		}
	}

	m.logger.Info("Station data reconciliation completed",
		"reconciled", reconciledCount,
		"reset", resetCount,
	)
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

	// Check if already started
	if station.StateMachine.IsConnected() {
		station.mu.Unlock()
		return fmt.Errorf("station already connected: %s", stationID)
	}

	if !station.Config.Enabled {
		station.mu.Unlock()
		return fmt.Errorf("station is disabled: %s", stationID)
	}

	m.logger.Info("Starting station", "stationId", stationID)

	// Update state while holding lock
	station.StateMachine.SetState(StateConnecting, "manual start")
	station.RuntimeState.State = StateConnecting
	station.RuntimeState.ConnectionStatus = "connecting"
	station.RuntimeState.LastError = ""

	// Capture configuration needed for connection
	url := station.Config.CSMSURL
	protocol := station.Config.ProtocolVersion
	var authConfig *connection.AuthConfig
	if station.Config.CSMSAuth != nil {
		auth := *station.Config.CSMSAuth
		switch auth.Type {
		case "basic":
			authConfig = &connection.AuthConfig{
				Type:     "basic",
				Username: auth.Username,
				Password: auth.Password,
			}
		case "bearer":
			authConfig = &connection.AuthConfig{
				Type:  "bearer",
				Token: auth.Token,
			}
		}
	}

	station.mu.Unlock()

	// Initiate WebSocket connection
	err := m.connManager.ConnectStation(
		stationID,
		url,
		protocol,
		nil,
		authConfig,
	)
	if err != nil {
		station.mu.Lock()
		station.StateMachine.SetState(StateFaulted, "connection failed")
		station.RuntimeState.State = StateFaulted
		station.RuntimeState.LastError = err.Error()
		station.RuntimeState.ConnectionStatus = "error"
		station.mu.Unlock()
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
		Config:            config,
		StateMachine:      NewStateMachine(),
		pendingRequests:   make(map[string]string),
		pendingStartTx:    make(map[string]int),
		pendingStartTags:  make(map[string]string),
		pendingAuthResp:   make(map[string]chan *v16.AuthorizeResponse),
		failedAuths:       make(map[string]time.Time),
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
	m.logger.Info("Started state synchronization", "interval", m.syncInterval.String())
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
	// Copy station references while holding the manager lock so we can release it
	// before taking individual station locks. This prevents lock-order inversions
	// (manager -> station) that previously led to deadlocks.
	type syncTarget struct {
		id      string
		station *Station
	}

	m.mu.RLock()
	targets := make([]syncTarget, 0, len(m.stations))
	for stationID, station := range m.stations {
		targets = append(targets, syncTarget{
			id:      stationID,
			station: station,
		})
	}
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count := 0
	for _, target := range targets {
		station := target.station
		if station == nil {
			continue
		}

		station.mu.RLock()
		shouldSync := time.Since(station.lastSync) >= m.syncInterval/2
		station.mu.RUnlock()
		if !shouldSync {
			continue
		}

		if err := m.saveStationToDB(ctx, station); err != nil {
			m.logger.Error("Failed to sync station state",
				"stationId", target.id,
				"error", err,
			)
		} else {
			station.mu.Lock()
			station.lastSync = time.Now()
			station.mu.Unlock()
			count++
		}
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
	now := time.Now()
	station.StateMachine.SetState(StateConnected, "websocket connected")
	station.RuntimeState.State = StateConnected
	station.RuntimeState.ConnectionStatus = "connected"
	station.RuntimeState.ConnectedAt = &now
	station.RuntimeState.LastError = ""
	station.mu.Unlock()

	m.logger.Info("Station connected", "stationId", stationID)

	if station.SessionManager != nil {
		station.SessionManager.NotifyStationState("initial connector snapshot")
	}

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

	// Stop heartbeat
	m.stopHeartbeat(station)

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

	// Get station to determine protocol version
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		m.logger.Error("Station not found", "stationId", stationID)
		m.sendNotImplementedError(stationID, call.UniqueID, call.Action)
		return
	}

	station.mu.RLock()
	protocolVersion := station.Config.ProtocolVersion
	station.mu.RUnlock()

	// Route to appropriate handler based on protocol version
	var response interface{}
	var err error

	switch protocolVersion {
	case "ocpp1.6", "1.6":
		response, err = m.v16Handler.HandleCall(stationID, call)
	default:
		m.logger.Error("Unsupported protocol version", "stationId", stationID, "version", protocolVersion)
		m.sendNotImplementedError(stationID, call.UniqueID, call.Action)
		return
	}

	// Handle errors
	if err != nil {
		m.logger.Error("Failed to handle call", "stationId", stationID, "action", call.Action, "error", err)
		m.sendNotImplementedError(stationID, call.UniqueID, call.Action)
		return
	}

	// Send response
	if err := m.sendCallResult(stationID, call.UniqueID, response); err != nil {
		m.logger.Error("Failed to send response", "stationId", stationID, "error", err)
	}
}

// handleCallResult handles CallResult responses
func (m *Manager) handleCallResult(stationID string, result *ocpp.CallResult) {
	m.logger.Info("Received CallResult", "stationId", stationID, "uniqueId", result.UniqueID)

	// Store message in MongoDB
	go m.storeMessage(stationID, "received", result)

	// Get station
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		m.logger.Warn("Station not found for CallResult", "stationId", stationID)
		return
	}

	// Get and remove pending request
	station.pendingMu.Lock()
	action, hasPending := station.pendingRequests[result.UniqueID]
	if hasPending {
		delete(station.pendingRequests, result.UniqueID)
	}
	station.pendingMu.Unlock()

	if !hasPending {
		m.logger.Debug("No pending request for CallResult", "stationId", stationID, "uniqueId", result.UniqueID)
		return
	}

	// Handle response based on action
	switch v16.Action(action) {
	case v16.ActionBootNotification:
		m.handleBootNotificationResponse(stationID, station, result)
	case v16.ActionHeartbeat:
		m.handleHeartbeatResponse(stationID, station, result)
	case v16.ActionAuthorize:
		m.handleAuthorizeResponse(stationID, station, result)
	case v16.ActionStartTransaction:
		m.handleStartTransactionResponse(stationID, station, result)
	default:
		m.logger.Debug("CallResult for action", "stationId", stationID, "action", action)
	}
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

	// Track pending request
	station.pendingMu.Lock()
	station.pendingRequests[call.UniqueID] = string(v16.ActionBootNotification)
	station.pendingMu.Unlock()

	data, err := call.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal BootNotification", "stationId", stationID, "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send BootNotification", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("Sent BootNotification", "stationId", stationID, "uniqueId", call.UniqueID)

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

// sendCallResult sends a CallResult response
func (m *Manager) sendCallResult(stationID, uniqueID string, payload interface{}) error {
	callResult, err := ocpp.NewCallResult(uniqueID, payload)
	if err != nil {
		return fmt.Errorf("failed to create CallResult: %w", err)
	}

	data, err := callResult.ToBytes()
	if err != nil {
		return fmt.Errorf("failed to marshal CallResult: %w", err)
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		return fmt.Errorf("failed to send CallResult: %w", err)
	}

	// Store sent message
	go m.storeMessage(stationID, "sent", callResult)

	return nil
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
	if m.messageLogger != nil {
		if err := m.messageLogger.LogMessage(stationID, direction, message, protocolVersion); err != nil {
			m.logger.Error("Failed to log message",
				"stationId", stationID,
				"direction", direction,
				"error", err,
			)
		}
	}
}

// saveStationToDB persists station to MongoDB
func (m *Manager) saveStationToDB(ctx context.Context, station *Station) error {
	station.mu.RLock()
	dbStation := m.convertConfigToStorage(station.Config)

	// Add runtime state fields
	dbStation.ConnectionStatus = station.RuntimeState.ConnectionStatus
	dbStation.LastHeartbeat = station.RuntimeState.LastHeartbeat
	dbStation.LastError = station.RuntimeState.LastError
	station.mu.RUnlock()

	// Update connector states from SessionManager
	if station.SessionManager != nil {
		connectors := station.SessionManager.GetAllConnectors()
		for i, connector := range connectors {
			// Find matching connector in dbStation and update status
			for j := range dbStation.Connectors {
				if dbStation.Connectors[j].ID == connector.ID {
					dbStation.Connectors[j].Status = string(connector.GetState())

					// Update current transaction ID if active
					if connector.HasActiveTransaction() {
						tx := connector.GetTransaction()
						if tx != nil {
							txID := tx.ID
							dbStation.Connectors[j].CurrentTransactionID = &txID
						}
					} else {
						dbStation.Connectors[j].CurrentTransactionID = nil
					}
					break
				}
			}
			_ = i // unused
		}
	}

	// Don't include _id in the replacement document (it's immutable)
	// MongoDB will preserve the existing _id
	dbStation.ID = ""

	collection := m.db.StationsCollection

	opts := options.Replace().SetUpsert(true)
	_, err := collection.ReplaceOne(
		ctx,
		bson.M{"station_id": station.Config.StationID},
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

	var connected, disconnected, charging, available, faulted, unavailable int

	for _, station := range m.stations {
		station.mu.RLock()
		state := station.StateMachine.GetState()
		station.mu.RUnlock()

		if station.StateMachine.IsConnected() {
			connected++
		} else if state == StateDisconnected || state == StateUnknown {
			disconnected++
		}

		switch state {
		case StateCharging:
			charging++
		case StateAvailable:
			available++
		case StateFaulted:
			faulted++
		case StateUnavailable:
			unavailable++
		}
	}

	stats := map[string]interface{}{
		"total":        len(m.stations),
		"connected":    connected,
		"disconnected": disconnected,
		"charging":     charging,
		"available":    available,
		"faulted":      faulted,
		"unavailable":  unavailable,
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

// handleBootNotificationResponse processes BootNotification responses and starts heartbeat
func (m *Manager) handleBootNotificationResponse(stationID string, station *Station, result *ocpp.CallResult) {
	var resp v16.BootNotificationResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		m.logger.Error("Failed to unmarshal BootNotification response", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("BootNotification accepted",
		"stationId", stationID,
		"status", resp.Status,
		"interval", resp.Interval,
		"currentTime", resp.CurrentTime,
	)

	if resp.Status == "Accepted" {
		// Start heartbeat with interval from CSMS (or use configured default)
		interval := resp.Interval
		if interval <= 0 {
			station.mu.RLock()
			interval = station.Config.Simulation.HeartbeatInterval
			station.mu.RUnlock()
			if interval <= 0 {
				interval = 60 // Default to 60 seconds
			}
		}

		m.startHeartbeat(stationID, station, interval)

		// Send initial StatusNotification for all connectors
		go m.sendAllConnectorStatus(stationID, station)
	}
}

// handleHeartbeatResponse processes Heartbeat responses
func (m *Manager) handleHeartbeatResponse(stationID string, station *Station, result *ocpp.CallResult) {
	var resp v16.HeartbeatResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		m.logger.Error("Failed to unmarshal Heartbeat response", "stationId", stationID, "error", err)
		return
	}

	now := time.Now()
	station.mu.Lock()
	station.RuntimeState.LastHeartbeat = &now
	station.mu.Unlock()

	m.logger.Debug("Heartbeat acknowledged", "stationId", stationID, "serverTime", resp.CurrentTime)
}

// handleAuthorizeResponse processes Authorize responses
func (m *Manager) handleAuthorizeResponse(stationID string, station *Station, result *ocpp.CallResult) {
	var resp v16.AuthorizeResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		m.logger.Error("Failed to unmarshal Authorize response", "stationId", stationID, "error", err)
		return
	}

	m.logger.Debug("Processing Authorize response",
		"stationId", stationID,
		"messageId", result.UniqueID,
		"status", resp.IdTagInfo.Status,
	)

	// Find the response channel and send the response
	station.pendingAuthRespMu.Lock()
	respChan, found := station.pendingAuthResp[result.UniqueID]
	delete(station.pendingAuthResp, result.UniqueID)
	station.pendingAuthRespMu.Unlock()

	if found {
		// Send response to waiting goroutine (non-blocking)
		select {
		case respChan <- &resp:
			m.logger.Debug("Sent Authorize response to waiting goroutine",
				"stationId", stationID,
				"messageId", result.UniqueID,
			)
		default:
			m.logger.Warn("Failed to send Authorize response - channel full or closed",
				"stationId", stationID,
				"messageId", result.UniqueID,
			)
		}
	} else {
		m.logger.Warn("No waiting goroutine for Authorize response",
			"stationId", stationID,
			"messageId", result.UniqueID,
		)
	}
}

// handleStartTransactionResponse processes StartTransaction responses
func (m *Manager) handleStartTransactionResponse(stationID string, station *Station, result *ocpp.CallResult) {
	var resp v16.StartTransactionResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		m.logger.Error("Failed to unmarshal StartTransaction response", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("StartTransaction response received",
		"stationId", stationID,
		"messageId", result.UniqueID,
		"transactionId", resp.TransactionId,
		"idTagStatus", resp.IdTagInfo.Status,
	)

	// Check if transaction was accepted
	if resp.IdTagInfo.Status != v16.AuthorizationStatusAccepted {
		m.logger.Warn("Transaction rejected by CSMS",
			"stationId", stationID,
			"transactionId", resp.TransactionId,
			"status", resp.IdTagInfo.Status,
		)

		// Find which connector this response is for and stop the transaction
		station.pendingStartMu.Lock()
		connectorID, found := station.pendingStartTx[result.UniqueID]
		delete(station.pendingStartTx, result.UniqueID)
		station.pendingStartMu.Unlock()

		if found {
			// Get the connector
			connector, err := station.SessionManager.GetConnector(connectorID)
			if err == nil && connector.HasActiveTransaction() {
				m.logger.Info("Stopping transaction that was rejected by CSMS",
					"stationId", stationID,
					"connectorId", connectorID,
					"transactionId", resp.TransactionId,
				)

				// Stop the transaction
				if err := station.SessionManager.StopCharging(connectorID, v16.ReasonDeAuthorized); err != nil {
					m.logger.Error("Failed to stop rejected transaction",
						"stationId", stationID,
						"connectorId", connectorID,
						"error", err,
					)
				}
			}
		}

		return
	}

	// Find which connector and idTag this response is for
	station.pendingStartMu.Lock()
	m.logger.Debug("Looking up pending StartTransaction",
		"stationId", stationID,
		"messageId", result.UniqueID,
		"pendingCount", len(station.pendingStartTx),
	)
	connectorID, found := station.pendingStartTx[result.UniqueID]
	idTag := station.pendingStartTags[result.UniqueID]
	delete(station.pendingStartTx, result.UniqueID)
	delete(station.pendingStartTags, result.UniqueID)
	station.pendingStartMu.Unlock()

	if !found {
		m.logger.Warn("No pending StartTransaction found for message",
			"stationId", stationID,
			"messageId", result.UniqueID,
		)
		return
	}

	m.logger.Info("Found pending StartTransaction for connector",
		"stationId", stationID,
		"messageId", result.UniqueID,
		"connectorId", connectorID,
		"idTag", idTag,
	)

	// Get the specific connector
	connector, err := station.SessionManager.GetConnector(connectorID)
	if err != nil {
		m.logger.Error("Failed to get connector for StartTransaction response",
			"stationId", stationID,
			"connectorId", connectorID,
			"error", err,
		)
		return
	}

	// Wait for transaction to be created (handle race condition)
	// The response might arrive before the local transaction is fully created
	var oldID int
	transactionFound := false

	for i := 0; i < 10; i++ {
		if connector.HasActiveTransaction() {
			// Get old ID before updating (using a copy is OK here for reading)
			tx := connector.GetTransaction()
			if tx != nil {
				oldID = tx.ID
				transactionFound = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !transactionFound {
		m.logger.Warn("Transaction not found after waiting",
			"stationId", stationID,
			"connectorId", connectorID,
			"transactionId", resp.TransactionId,
		)
		return
	}

	// Update transaction ID with the one from CSMS (using the connector method to update the original)
	err = connector.UpdateTransactionID(resp.TransactionId)
	if err != nil {
		m.logger.Error("Failed to update transaction ID on connector",
			"stationId", stationID,
			"connectorId", connectorID,
			"error", err,
		)
		return
	}

	m.logger.Info("Updated transaction ID from CSMS",
		"stationId", stationID,
		"connectorId", connectorID,
		"oldTransactionId", oldID,
		"newTransactionId", resp.TransactionId,
	)

	// Update transaction in database
	if m.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		transactionRepo := storage.NewTransactionRepository(m.db)
		err := transactionRepo.UpdateTransactionID(ctx, stationID, oldID, resp.TransactionId)
		if err != nil {
			m.logger.Error("Failed to update transaction ID in database",
				"stationId", stationID,
				"oldTransactionId", oldID,
				"newTransactionId", resp.TransactionId,
				"error", err,
			)
		}
	}
}

// startHeartbeat starts periodic heartbeat for a station
func (m *Manager) startHeartbeat(stationID string, station *Station, intervalSeconds int) {
	// Stop any existing heartbeat
	m.stopHeartbeat(station)

	interval := time.Duration(intervalSeconds) * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	station.mu.Lock()
	station.heartbeatCancel = cancel
	station.heartbeatDone = done
	station.mu.Unlock()

	m.logger.Info("Starting heartbeat",
		"stationId", stationID,
		"intervalSeconds", intervalSeconds,
		"interval", interval.String(),
	)

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.logger.Debug("Heartbeat stopped", "stationId", stationID)
				return
			case <-ticker.C:
				m.sendHeartbeat(stationID, station)
			}
		}
	}()
}

// stopHeartbeat stops the heartbeat for a station
func (m *Manager) stopHeartbeat(station *Station) {
	station.mu.Lock()
	if station.heartbeatCancel != nil {
		station.heartbeatCancel()
		station.heartbeatCancel = nil
	}
	done := station.heartbeatDone
	station.mu.Unlock()

	// Wait for heartbeat goroutine to finish
	if done != nil {
		<-done
	}
}

// sendHeartbeat sends a heartbeat message
func (m *Manager) sendHeartbeat(stationID string, station *Station) {
	call, err := ocpp.NewCall(string(v16.ActionHeartbeat), v16.HeartbeatRequest{})
	if err != nil {
		m.logger.Error("Failed to create Heartbeat", "stationId", stationID, "error", err)
		return
	}

	// Track pending request
	station.pendingMu.Lock()
	station.pendingRequests[call.UniqueID] = string(v16.ActionHeartbeat)
	station.pendingMu.Unlock()

	data, err := call.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal Heartbeat", "stationId", stationID, "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send Heartbeat", "stationID", stationID, "error", err)
		return
	}

	m.logger.Debug("Sent Heartbeat", "stationId", stationID, "uniqueId", call.UniqueID)

	// Store sent message
	go m.storeMessage(stationID, "sent", call)
}

// sendAllConnectorStatus sends StatusNotification for all connectors
func (m *Manager) sendAllConnectorStatus(stationID string, station *Station) {
	if station.SessionManager == nil {
		return
	}

	connectors := station.SessionManager.GetAllConnectors()
	for _, connector := range connectors {
		state := connector.GetState()
		errorCode := connector.GetErrorCode()
		m.sendStatusNotification(stationID, connector.ID, v16.ChargePointStatus(state), errorCode, "")
	}
}

// sendStatusNotification sends a StatusNotification message
func (m *Manager) sendStatusNotification(stationID string, connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) {
	req := &v16.StatusNotificationRequest{
		ConnectorId: connectorID,
		ErrorCode:   errorCode,
		Status:      status,
		Info:        info,
	}

	now := v16.DateTime{Time: time.Now()}
	req.Timestamp = &now

	call, err := ocpp.NewCall(string(v16.ActionStatusNotification), req)
	if err != nil {
		m.logger.Error("Failed to create StatusNotification", "stationId", stationID, "error", err)
		return
	}

	data, err := call.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal StatusNotification", "stationId", stationID, "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send StatusNotification", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("Sent StatusNotification",
		"stationId", stationID,
		"connectorId", connectorID,
		"status", status,
		"uniqueId", call.UniqueID,
	)

	// Store sent message
	go m.storeMessage(stationID, "sent", call)
}

// GetConnectors returns the connectors for a station with their current state
func (m *Manager) GetConnectors(ctx context.Context, stationID string) ([]map[string]interface{}, error) {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("station not found: %s", stationID)
	}

	if station.SessionManager == nil {
		return nil, fmt.Errorf("session manager not initialized for station: %s", stationID)
	}

	connectors := station.SessionManager.GetAllConnectors()
	result := make([]map[string]interface{}, 0, len(connectors))

	for _, connector := range connectors {
		connectorData := map[string]interface{}{
			"id":        connector.ID,
			"type":      connector.Type,
			"maxPower":  connector.MaxPower,
			"state":     string(connector.GetState()),
			"errorCode": string(connector.GetErrorCode()),
		}

		// Add transaction info if active
		if connector.HasActiveTransaction() {
			tx := connector.GetTransaction()
			if tx != nil {
				tx.mu.RLock()
				connectorData["transaction"] = map[string]interface{}{
					"id":              tx.ID,
					"idTag":           tx.IDTag,
					"startTime":       tx.StartTime,
					"startMeterValue": tx.StartMeterValue,
					"currentMeter":    tx.CurrentMeter,
				}
				tx.mu.RUnlock()
			}
		}

		result = append(result, connectorData)
	}

	// Sort connectors by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		idI, okI := result[i]["id"].(int)
		idJ, okJ := result[j]["id"].(int)
		if okI && okJ {
			return idI < idJ
		}
		return false
	})

	return result, nil
}

// StartCharging initiates a charging session on a connector
func (m *Manager) StartCharging(ctx context.Context, stationID string, connectorID int, idTag string) error {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	if station.SessionManager == nil {
		return fmt.Errorf("session manager not initialized for station: %s", stationID)
	}

	station.mu.RLock()
	isConnected := station.RuntimeState.ConnectionStatus == "connected"
	station.mu.RUnlock()

	if !isConnected {
		return fmt.Errorf("station is not connected to CSMS")
	}

	m.logger.Info("Starting charging session",
		"stationId", stationID,
		"connectorId", connectorID,
		"idTag", idTag,
	)

	// Start charging via session manager
	transactionID, err := station.SessionManager.StartCharging(connectorID, idTag)
	if err != nil {
		m.logger.Error("Failed to start charging",
			"stationId", stationID,
			"connectorId", connectorID,
			"error", err,
		)
		return err
	}

	m.logger.Info("Charging session started successfully",
		"stationId", stationID,
		"connectorId", connectorID,
		"idTag", idTag,
		"transactionId", transactionID,
	)

	return nil
}

// StopCharging stops a charging session on a connector
func (m *Manager) StopCharging(ctx context.Context, stationID string, connectorID int, reason string) error {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	if station.SessionManager == nil {
		return fmt.Errorf("session manager not initialized for station: %s", stationID)
	}

	m.logger.Info("Stopping charging session",
		"stationId", stationID,
		"connectorId", connectorID,
		"reason", reason,
	)

	// Convert reason string to v16.Reason type
	var stopReason v16.Reason
	switch reason {
	case "Local":
		stopReason = v16.ReasonLocal
	case "Remote":
		stopReason = v16.ReasonRemote
	case "EVDisconnected":
		stopReason = v16.ReasonEVDisconnected
	case "HardReset":
		stopReason = v16.ReasonHardReset
	case "SoftReset":
		stopReason = v16.ReasonSoftReset
	case "PowerLoss":
		stopReason = v16.ReasonPowerLoss
	case "EmergencyStop":
		stopReason = v16.ReasonEmergencyStop
	case "DeAuthorized":
		stopReason = v16.ReasonDeAuthorized
	default:
		stopReason = v16.ReasonLocal
	}

	// Stop charging via session manager
	if err := station.SessionManager.StopCharging(connectorID, stopReason); err != nil {
		m.logger.Error("Failed to stop charging",
			"stationId", stationID,
			"connectorId", connectorID,
			"error", err,
		)
		return fmt.Errorf("failed to stop charging: %w", err)
	}

	m.logger.Info("Charging session stopped successfully",
		"stationId", stationID,
		"connectorId", connectorID,
		"reason", reason,
	)

	return nil
}

// SendCustomMessage sends a custom OCPP message to the CSMS
// This allows testing with arbitrary messages crafted by the user
func (m *Manager) SendCustomMessage(ctx context.Context, stationID string, messageJSON []byte) error {
	m.mu.RLock()
	station, exists := m.stations[stationID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("station not found: %s", stationID)
	}

	// Check if station is connected
	if station.RuntimeState.ConnectionStatus != "connected" {
		return fmt.Errorf("station is not connected")
	}

	// Validate that it's valid JSON
	var msgArray []interface{}
	if err := json.Unmarshal(messageJSON, &msgArray); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Basic OCPP message validation (should be array with 3 or 4 elements)
	if len(msgArray) < 3 || len(msgArray) > 4 {
		return fmt.Errorf("invalid OCPP message format: expected array with 3-4 elements")
	}

	// First element should be message type (2=Call, 3=CallResult, 4=CallError)
	msgType, ok := msgArray[0].(float64)
	if !ok {
		return fmt.Errorf("invalid message type: must be a number")
	}

	if msgType != 2 && msgType != 3 && msgType != 4 {
		return fmt.Errorf("invalid message type: must be 2 (Call), 3 (CallResult), or 4 (CallError)")
	}

	// Send the raw message
	if err := m.connManager.SendMessage(stationID, messageJSON); err != nil {
		m.logger.Error("Failed to send custom message",
			"stationId", stationID,
			"error", err,
		)
		return fmt.Errorf("failed to send custom message: %w", err)
	}

	m.logger.Info("Sent custom message",
		"stationId", stationID,
		"messageType", int(msgType),
	)

	// Store the message for logging
	var call *ocpp.Call
	if msgType == 2 && len(msgArray) >= 4 {
		// It's a Call message
		uniqueID, _ := msgArray[1].(string)
		action, _ := msgArray[2].(string)
		payload, _ := json.Marshal(msgArray[3])

		call = &ocpp.Call{
			MessageTypeID: ocpp.MessageTypeCall,
			UniqueID:      uniqueID,
			Action:        action,
			Payload:       payload,
		}
		go m.storeMessage(stationID, "sent", call)
	}

	return nil
}
