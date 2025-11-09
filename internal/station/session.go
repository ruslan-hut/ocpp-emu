package station

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	v16 "github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
)

// SessionManager manages charging sessions for a station
type SessionManager struct {
	stationID            string
	connectors           map[int]*Connector
	nextTransactionID    int
	mu                   sync.RWMutex
	logger               *slog.Logger
	transactionRepo      *storage.TransactionRepository
	protocolVersion      string
	stationStateCallback func(State, string)

	// Callbacks for OCPP message sending
	SendAuthorize          func(idTag string) (*v16.AuthorizeResponse, error)
	SendStartTransaction   func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error)
	SendStopTransaction    func(transactionID int, idTag string, meterStop int, timestamp time.Time, reason v16.Reason) (*v16.StopTransactionResponse, error)
	SendStatusNotification func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error
	SendMeterValues        func(connectorID int, transactionID *int, meterValues []v16.MeterValue) error

	// Meter value simulation
	meterValueTickers map[int]*time.Ticker
	stopChans         map[int]chan struct{}
}

// NewSessionManager creates a new session manager
func NewSessionManager(stationID string, connectorConfigs []ConnectorConfig, logger *slog.Logger) *SessionManager {
	if logger == nil {
		logger = slog.Default()
	}

	sm := &SessionManager{
		stationID:         stationID,
		connectors:        make(map[int]*Connector),
		nextTransactionID: 1,
		logger:            logger,
		meterValueTickers: make(map[int]*time.Ticker),
		stopChans:         make(map[int]chan struct{}),
	}

	// Initialize connectors
	for _, config := range connectorConfigs {
		connector := NewConnector(config.ID, config.Type, config.MaxPower)

		// Set initial state from config
		if config.Status != "" {
			state := ConnectorState(config.Status)
			connector.State = state
		}

		// Set up state change callback
		connector.onStateChange = sm.onConnectorStateChange

		sm.connectors[config.ID] = connector
	}

	return sm
}

// GetConnector returns a connector by ID
func (sm *SessionManager) GetConnector(connectorID int) (*Connector, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	connector, exists := sm.connectors[connectorID]
	if !exists {
		return nil, fmt.Errorf("connector %d not found", connectorID)
	}

	return connector, nil
}

// GetAllConnectors returns all connectors
func (sm *SessionManager) GetAllConnectors() []*Connector {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	connectors := make([]*Connector, 0, len(sm.connectors))
	for _, c := range sm.connectors {
		connectors = append(connectors, c)
	}

	return connectors
}

// SetTransactionRepository sets the transaction repository for persistence
func (sm *SessionManager) SetTransactionRepository(repo *storage.TransactionRepository) {
	sm.transactionRepo = repo
}

// SetProtocolVersion sets the protocol version for transaction logging
func (sm *SessionManager) SetProtocolVersion(version string) {
	sm.protocolVersion = version
}

// SetStationStateCallback registers a callback for station-level state changes derived from connector states.
func (sm *SessionManager) SetStationStateCallback(callback func(State, string)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stationStateCallback = callback
}

// NotifyStationState triggers evaluation of the current connector states and invokes the registered callback.
func (sm *SessionManager) NotifyStationState(reason string) {
	sm.evaluateStationState(reason)
}

// Authorize authorizes an ID tag
func (sm *SessionManager) Authorize(idTag string) (*v16.IdTagInfo, error) {
	sm.logger.Info("Authorizing ID tag", "stationId", sm.stationID, "idTag", idTag)

	if sm.SendAuthorize == nil {
		// Offline authorization - accept by default
		return &v16.IdTagInfo{
			Status: v16.AuthorizationStatusAccepted,
		}, nil
	}

	resp, err := sm.SendAuthorize(idTag)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	return &resp.IdTagInfo, nil
}

// StartCharging initiates a charging session
func (sm *SessionManager) StartCharging(connectorID int, idTag string) (int, error) {
	sm.logger.Info("Starting charging session",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"idTag", idTag,
	)

	// Get connector
	connector, err := sm.GetConnector(connectorID)
	if err != nil {
		return 0, err
	}

	// Check if connector is available
	if !connector.IsAvailable() {
		return 0, fmt.Errorf("connector %d is not available (state: %s)", connectorID, connector.GetState())
	}

	// Check reservation
	if connector.IsReserved() && !connector.IsReservedFor(idTag) {
		return 0, fmt.Errorf("connector %d is reserved for another ID tag", connectorID)
	}

	// Authorize (optional - can be done separately)
	authInfo, err := sm.Authorize(idTag)
	if err != nil {
		return 0, fmt.Errorf("authorization failed: %w", err)
	}

	if authInfo.Status != v16.AuthorizationStatusAccepted {
		return 0, fmt.Errorf("authorization rejected: %s", authInfo.Status)
	}

	// Transition to Preparing
	if err := connector.SetState(ConnectorStatePreparing, v16.ChargePointErrorNoError, "Preparing to charge"); err != nil {
		return 0, fmt.Errorf("failed to set state to Preparing: %w", err)
	}

	// Send StatusNotification
	if sm.SendStatusNotification != nil {
		sm.SendStatusNotification(connectorID, v16.ChargePointStatusPreparing, v16.ChargePointErrorNoError, "Preparing to charge")
	}

	// Generate transaction ID
	sm.mu.Lock()
	transactionID := sm.nextTransactionID
	sm.nextTransactionID++
	sm.mu.Unlock()

	// Get meter start value (initial reading)
	meterStart := 0
	if tx := connector.GetTransaction(); tx != nil {
		meterStart = tx.CurrentMeter
	}

	// Send StartTransaction
	var startResp *v16.StartTransactionResponse
	if sm.SendStartTransaction != nil {
		startResp, err = sm.SendStartTransaction(connectorID, idTag, meterStart, time.Now())
		if err != nil {
			// Rollback state
			connector.SetState(ConnectorStateAvailable, v16.ChargePointErrorNoError, "")
			return 0, fmt.Errorf("failed to send StartTransaction: %w", err)
		}

		// Use transaction ID from CSMS
		transactionID = startResp.TransactionId

		// Check authorization status
		if startResp.IdTagInfo.Status != v16.AuthorizationStatusAccepted {
			connector.SetState(ConnectorStateAvailable, v16.ChargePointErrorNoError, "")
			return 0, fmt.Errorf("transaction rejected by CSMS: %s", startResp.IdTagInfo.Status)
		}
	}

	// Start local transaction
	if err := connector.StartTransaction(transactionID, idTag, meterStart); err != nil {
		connector.SetState(ConnectorStateAvailable, v16.ChargePointErrorNoError, "")
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Persist transaction to database
	if sm.transactionRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dbTransaction := storage.Transaction{
			TransactionID:   transactionID,
			StationID:       sm.stationID,
			ConnectorID:     connectorID,
			IDTag:           idTag,
			StartTimestamp:  time.Now(),
			MeterStart:      meterStart,
			MeterStop:       0,
			EnergyConsumed:  0,
			Status:          "active",
			ProtocolVersion: sm.protocolVersion,
		}

		if err := sm.transactionRepo.Create(ctx, dbTransaction); err != nil {
			sm.logger.Error("Failed to persist transaction to database",
				"transactionId", transactionID,
				"error", err,
			)
			// Continue anyway - local transaction is started
		}
	}

	// Transition to Charging
	if err := connector.SetState(ConnectorStateCharging, v16.ChargePointErrorNoError, "Charging"); err != nil {
		connector.StopTransaction(meterStart, v16.ReasonOther)
		connector.ClearTransaction()
		return 0, fmt.Errorf("failed to set state to Charging: %w", err)
	}

	// Send StatusNotification
	if sm.SendStatusNotification != nil {
		sm.SendStatusNotification(connectorID, v16.ChargePointStatusCharging, v16.ChargePointErrorNoError, "Charging")
	}

	// Start meter value simulation
	sm.startMeterValueSimulation(connector, transactionID)

	sm.logger.Info("Charging session started",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"transactionId", transactionID,
	)

	return transactionID, nil
}

// StopCharging stops a charging session
func (sm *SessionManager) StopCharging(connectorID int, reason v16.Reason) error {
	sm.logger.Info("Stopping charging session",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"reason", reason,
	)

	// Get connector
	connector, err := sm.GetConnector(connectorID)
	if err != nil {
		return err
	}

	// Check if there's an active transaction
	if !connector.HasActiveTransaction() {
		return fmt.Errorf("connector %d has no active transaction", connectorID)
	}

	// Stop meter value simulation
	sm.stopMeterValueSimulation(connectorID)

	// Get transaction details
	tx := connector.GetTransaction()
	if tx == nil {
		return fmt.Errorf("connector %d transaction is nil", connectorID)
	}

	// Transition to Finishing
	if err := connector.SetState(ConnectorStateFinishing, v16.ChargePointErrorNoError, "Finishing"); err != nil {
		sm.logger.Warn("Failed to set state to Finishing", "error", err)
	}

	// Send StatusNotification
	if sm.SendStatusNotification != nil {
		sm.SendStatusNotification(connectorID, v16.ChargePointStatusFinishing, v16.ChargePointErrorNoError, "Finishing")
	}

	// Get final meter value
	meterStop := tx.CurrentMeter

	// Stop local transaction
	if err := connector.StopTransaction(meterStop, reason); err != nil {
		sm.logger.Error("Failed to stop transaction locally", "error", err)
	}

	// Send StopTransaction
	if sm.SendStopTransaction != nil {
		_, err = sm.SendStopTransaction(tx.ID, tx.IDTag, meterStop, time.Now(), reason)
		if err != nil {
			sm.logger.Error("Failed to send StopTransaction", "error", err)
			// Continue anyway - local state is already stopped
		}
	}

	// Update transaction in database
	if sm.transactionRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sm.transactionRepo.Complete(ctx, tx.ID, sm.stationID, meterStop, string(reason)); err != nil {
			sm.logger.Error("Failed to update transaction in database",
				"transactionId", tx.ID,
				"error", err,
			)
			// Continue anyway - local state is updated
		}
	}

	// Clear transaction
	connector.ClearTransaction()

	// Transition back to Available
	if err := connector.SetState(ConnectorStateAvailable, v16.ChargePointErrorNoError, ""); err != nil {
		sm.logger.Warn("Failed to set state to Available", "error", err)
	}

	// Send StatusNotification
	if sm.SendStatusNotification != nil {
		sm.SendStatusNotification(connectorID, v16.ChargePointStatusAvailable, v16.ChargePointErrorNoError, "")
	}

	sm.logger.Info("Charging session stopped",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"transactionId", tx.ID,
		"energyDelivered", meterStop-tx.StartMeterValue,
	)

	return nil
}

// ChangeAvailability changes the availability of a connector
func (sm *SessionManager) ChangeAvailability(connectorID int, availabilityType string) error {
	sm.logger.Info("Changing connector availability",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"type", availabilityType,
	)

	// If connectorID is 0, change all connectors
	if connectorID == 0 {
		for id := range sm.connectors {
			if err := sm.changeConnectorAvailability(id, availabilityType); err != nil {
				sm.logger.Error("Failed to change connector availability", "connectorId", id, "error", err)
			}
		}
		return nil
	}

	return sm.changeConnectorAvailability(connectorID, availabilityType)
}

func (sm *SessionManager) changeConnectorAvailability(connectorID int, availabilityType string) error {
	connector, err := sm.GetConnector(connectorID)
	if err != nil {
		return err
	}

	var newState ConnectorState
	var status v16.ChargePointStatus

	if availabilityType == "Operative" {
		newState = ConnectorStateAvailable
		status = v16.ChargePointStatusAvailable
	} else {
		newState = ConnectorStateUnavailable
		status = v16.ChargePointStatusUnavailable
	}

	// Don't change if charging
	if connector.IsCharging() {
		sm.logger.Warn("Cannot change availability while charging", "connectorId", connectorID)
		return fmt.Errorf("connector %d is charging, schedule change for later", connectorID)
	}

	if err := connector.SetState(newState, v16.ChargePointErrorNoError, ""); err != nil {
		return err
	}

	// Send StatusNotification
	if sm.SendStatusNotification != nil {
		sm.SendStatusNotification(connectorID, status, v16.ChargePointErrorNoError, "")
	}

	return nil
}

// startMeterValueSimulation starts sending periodic meter values
func (sm *SessionManager) startMeterValueSimulation(connector *Connector, transactionID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	connectorID := connector.ID

	// Stop any existing ticker
	if ticker, exists := sm.meterValueTickers[connectorID]; exists {
		ticker.Stop()
	}
	if stopChan, exists := sm.stopChans[connectorID]; exists {
		close(stopChan)
	}

	// Create new ticker (every 60 seconds)
	ticker := time.NewTicker(60 * time.Second)
	stopChan := make(chan struct{})

	sm.meterValueTickers[connectorID] = ticker
	sm.stopChans[connectorID] = stopChan

	// Start goroutine to send meter values
	go func() {
		for {
			select {
			case <-ticker.C:
				sm.sendMeterValue(connector, transactionID)
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// stopMeterValueSimulation stops sending meter values
func (sm *SessionManager) stopMeterValueSimulation(connectorID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if ticker, exists := sm.meterValueTickers[connectorID]; exists {
		ticker.Stop()
		delete(sm.meterValueTickers, connectorID)
	}

	if stopChan, exists := sm.stopChans[connectorID]; exists {
		close(stopChan)
		delete(sm.stopChans, connectorID)
	}
}

// sendMeterValue sends a meter value sample
func (sm *SessionManager) sendMeterValue(connector *Connector, transactionID int) {
	if !connector.HasActiveTransaction() {
		return
	}

	tx := connector.GetTransaction()
	if tx == nil || tx.ID != transactionID {
		return
	}

	// Simulate power consumption (random between 5-7.5 kW)
	powerWatts := 5000 + rand.Intn(2500)

	// Energy increment (Wh) = Power (W) * time (h)
	// 60 seconds = 1/60 hour
	energyIncrement := powerWatts / 60

	// Update meter
	newMeter := tx.CurrentMeter + energyIncrement
	connector.UpdateMeter(newMeter)

	// Create meter value sample
	if sm.SendMeterValues != nil {
		meterValues := []v16.MeterValue{
			{
				Timestamp: v16.DateTime{Time: time.Now()},
				SampledValue: []v16.SampledValue{
					{
						Value:     fmt.Sprintf("%d", newMeter),
						Context:   v16.ReadingContextSamplePeriodic,
						Measurand: v16.MeasurandEnergyActiveImportRegister,
						Unit:      v16.UnitOfMeasureWh,
						Location:  v16.LocationOutlet,
					},
					{
						Value:     fmt.Sprintf("%d", powerWatts),
						Context:   v16.ReadingContextSamplePeriodic,
						Measurand: v16.MeasurandPowerActiveImport,
						Unit:      v16.UnitOfMeasureW,
						Location:  v16.LocationOutlet,
					},
				},
			},
		}

		sm.SendMeterValues(connector.ID, &transactionID, meterValues)
	}

	sm.logger.Debug("Meter value sent",
		"stationId", sm.stationID,
		"connectorId", connector.ID,
		"transactionId", transactionID,
		"meter", newMeter,
		"power", powerWatts,
	)
}

// onConnectorStateChange is called when a connector state changes
func (sm *SessionManager) onConnectorStateChange(connectorID int, oldState, newState ConnectorState) {
	sm.logger.Info("Connector state changed",
		"stationId", sm.stationID,
		"connectorId", connectorID,
		"oldState", oldState,
		"newState", newState,
	)

	sm.evaluateStationState(fmt.Sprintf("connector %d transitioned from %s to %s", connectorID, oldState, newState))
}

// Shutdown stops all ongoing sessions and cleanup
func (sm *SessionManager) Shutdown(ctx context.Context) error {
	sm.logger.Info("Shutting down session manager", "stationId", sm.stationID)

	// Stop all active transactions
	for _, connector := range sm.GetAllConnectors() {
		if connector.HasActiveTransaction() {
			if err := sm.StopCharging(connector.ID, v16.ReasonReboot); err != nil {
				sm.logger.Error("Failed to stop charging during shutdown",
					"connectorId", connector.ID,
					"error", err,
				)
			}
		}

		// Stop meter value simulation
		sm.stopMeterValueSimulation(connector.ID)
	}

	return nil
}

func (sm *SessionManager) evaluateStationState(reason string) {
	sm.mu.RLock()
	callback := sm.stationStateCallback
	connectors := make([]*Connector, 0, len(sm.connectors))
	for _, connector := range sm.connectors {
		connectors = append(connectors, connector)
	}
	sm.mu.RUnlock()

	if callback == nil {
		return
	}

	newState := sm.aggregateConnectorState(connectors)
	callback(newState, reason)
}

func (sm *SessionManager) aggregateConnectorState(connectors []*Connector) State {
	if len(connectors) == 0 {
		return StateUnknown
	}

	var (
		hasFaulted     bool
		hasCharging    bool
		hasUnavailable bool
		hasAvailable   bool
	)

	for _, connector := range connectors {
		state := connector.GetState()

		switch state {
		case ConnectorStateFaulted:
			hasFaulted = true
		case ConnectorStateCharging, ConnectorStateSuspendedEV, ConnectorStateSuspendedEVSE, ConnectorStatePreparing, ConnectorStateFinishing:
			hasCharging = true
		case ConnectorStateUnavailable, ConnectorStateReserved:
			hasUnavailable = true
		case ConnectorStateAvailable:
			hasAvailable = true
		default:
			hasAvailable = true
		}
	}

	switch {
	case hasFaulted:
		return StateFaulted
	case hasCharging:
		return StateCharging
	case hasUnavailable:
		return StateUnavailable
	case hasAvailable:
		return StateAvailable
	default:
		return StateUnknown
	}
}
