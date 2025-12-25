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
	v201 "github.com/ruslanhut/ocpp-emu/internal/ocpp/v201"
	v21 "github.com/ruslanhut/ocpp-emu/internal/ocpp/v21"
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
	v16Handler    *v16.Handler  // OCPP 1.6 message handler
	v201Handler   *v201.Handler // OCPP 2.0.1 message handler
	v21Handler    *v21.Handler  // OCPP 2.1 message handler
}

// Station represents a managed charging station instance
type Station struct {
	Config           Config
	StateMachine     *StateMachine
	RuntimeState     RuntimeState
	SessionManager   *SessionManager        // Enhanced session manager for charging
	DeviceModel      *v201.DeviceModel      // OCPP 2.0.1 device model
	CertificateStore *v201.CertificateStore // ISO 15118 certificate management
	mu               sync.RWMutex
	lastSync         time.Time

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

	// Initialize OCPP 2.0.1 handler
	m.v201Handler = v201.NewHandler(logger)
	m.v201Handler.SendMessage = connManager.SendMessage
	m.setupV201HandlerCallbacks()

	// Initialize OCPP 2.1 handler
	m.v21Handler = v21.NewHandler(logger)
	m.v21Handler.SendMessage = connManager.SendMessage
	m.setupV21HandlerCallbacks()

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

// setupV201HandlerCallbacks sets up callbacks for OCPP 2.0.1 handler
func (m *Manager) setupV201HandlerCallbacks() {
	// RequestStartTransaction handler (replaces RemoteStartTransaction in 2.0.1)
	m.v201Handler.OnRequestStartTransaction = func(stationID string, req *v201.RequestStartTransactionRequest) (*v201.RequestStartTransactionResponse, error) {
		m.logger.Info("Handling RequestStartTransaction (2.0.1)", "stationId", stationID, "idToken", req.IdToken.IdToken)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v201.RequestStartTransactionResponse{Status: "Rejected"}, nil
		}

		// Determine EVSE/connector ID
		connectorID := 1
		if req.EvseId != nil {
			connectorID = *req.EvseId
		}

		// Start charging session
		_, err := station.SessionManager.StartCharging(connectorID, req.IdToken.IdToken)
		if err != nil {
			m.logger.Error("Failed to start charging", "error", err)
			return &v201.RequestStartTransactionResponse{Status: "Rejected"}, nil
		}

		return &v201.RequestStartTransactionResponse{
			Status: "Accepted",
		}, nil
	}

	// RequestStopTransaction handler (replaces RemoteStopTransaction in 2.0.1)
	m.v201Handler.OnRequestStopTransaction = func(stationID string, req *v201.RequestStopTransactionRequest) (*v201.RequestStopTransactionResponse, error) {
		m.logger.Info("Handling RequestStopTransaction (2.0.1)", "stationId", stationID, "transactionId", req.TransactionId)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v201.RequestStopTransactionResponse{Status: "Rejected"}, nil
		}

		// Find connector with this transaction
		connectors := station.SessionManager.GetAllConnectors()
		var targetConnectorID int
		found := false

		for _, connector := range connectors {
			tx := connector.GetTransaction()
			if tx != nil && tx.StringID == req.TransactionId {
				targetConnectorID = connector.ID
				found = true
				break
			}
		}

		if !found {
			m.logger.Warn("Transaction not found", "transactionId", req.TransactionId)
			return &v201.RequestStopTransactionResponse{Status: "Rejected"}, nil
		}

		// Stop charging session
		err := station.SessionManager.StopCharging(targetConnectorID, v16.ReasonRemote)
		if err != nil {
			m.logger.Error("Failed to stop charging", "error", err)
			return &v201.RequestStopTransactionResponse{Status: "Rejected"}, nil
		}

		return &v201.RequestStopTransactionResponse{
			Status: "Accepted",
		}, nil
	}

	// Reset handler
	m.v201Handler.OnReset = func(stationID string, req *v201.ResetRequest) (*v201.ResetResponse, error) {
		m.logger.Info("Handling Reset (2.0.1)", "stationId", stationID, "type", req.Type)

		// TODO: Implement actual reset logic
		return &v201.ResetResponse{
			Status: v201.ResetStatusAccepted,
		}, nil
	}

	// GetVariables handler - uses device model
	m.v201Handler.OnGetVariables = func(stationID string, req *v201.GetVariablesRequest) (*v201.GetVariablesResponse, error) {
		m.logger.Info("Handling GetVariables (2.0.1)", "stationId", stationID, "count", len(req.GetVariableData))

		// Get station to access device model
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.DeviceModel == nil {
			// Return rejected for all if station or device model not found
			results := make([]v201.GetVariableResult, len(req.GetVariableData))
			for i, data := range req.GetVariableData {
				results[i] = v201.GetVariableResult{
					AttributeStatus: v201.GetVariableStatusRejected,
					Component:       data.Component,
					Variable:        data.Variable,
				}
			}
			return &v201.GetVariablesResponse{GetVariableResult: results}, nil
		}

		// Query device model for each requested variable
		results := make([]v201.GetVariableResult, len(req.GetVariableData))
		for i, data := range req.GetVariableData {
			// Determine attribute type (default to Actual)
			attrType := v201.AttributeActual
			if data.AttributeType != nil {
				attrType = *data.AttributeType
			}

			// Get variable from device model
			value, status := station.DeviceModel.GetVariable(
				data.Component.Name,
				data.Component.Instance,
				data.Variable.Name,
				data.Variable.Instance,
				attrType,
			)

			attrTypeCopy := attrType
			results[i] = v201.GetVariableResult{
				AttributeStatus: status,
				AttributeType:   &attrTypeCopy,
				AttributeValue:  value,
				Component:       data.Component,
				Variable:        data.Variable,
			}

			m.logger.Debug("GetVariable result",
				"component", data.Component.Name,
				"variable", data.Variable.Name,
				"status", status,
				"value", value,
			)
		}
		return &v201.GetVariablesResponse{GetVariableResult: results}, nil
	}

	// SetVariables handler - uses device model
	m.v201Handler.OnSetVariables = func(stationID string, req *v201.SetVariablesRequest) (*v201.SetVariablesResponse, error) {
		m.logger.Info("Handling SetVariables (2.0.1)", "stationId", stationID, "count", len(req.SetVariableData))

		// Get station to access device model
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.DeviceModel == nil {
			// Return rejected for all if station or device model not found
			results := make([]v201.SetVariableResult, len(req.SetVariableData))
			for i, data := range req.SetVariableData {
				results[i] = v201.SetVariableResult{
					AttributeStatus: v201.SetVariableStatusRejected,
					Component:       data.Component,
					Variable:        data.Variable,
				}
			}
			return &v201.SetVariablesResponse{SetVariableResult: results}, nil
		}

		// Set each variable in device model
		results := make([]v201.SetVariableResult, len(req.SetVariableData))
		for i, data := range req.SetVariableData {
			// Determine attribute type (default to Actual)
			attrType := v201.AttributeActual
			if data.AttributeType != nil {
				attrType = *data.AttributeType
			}

			// Set variable in device model
			status := station.DeviceModel.SetVariable(
				data.Component.Name,
				data.Component.Instance,
				data.Variable.Name,
				data.Variable.Instance,
				attrType,
				data.AttributeValue,
			)

			attrTypeCopy := attrType
			results[i] = v201.SetVariableResult{
				AttributeStatus: status,
				AttributeType:   &attrTypeCopy,
				Component:       data.Component,
				Variable:        data.Variable,
			}

			m.logger.Debug("SetVariable result",
				"component", data.Component.Name,
				"variable", data.Variable.Name,
				"status", status,
				"value", data.AttributeValue,
			)
		}
		return &v201.SetVariablesResponse{SetVariableResult: results}, nil
	}

	// ChangeAvailability handler
	m.v201Handler.OnChangeAvailability = func(stationID string, req *v201.ChangeAvailabilityRequest) (*v201.ChangeAvailabilityResponse, error) {
		m.logger.Info("Handling ChangeAvailability (2.0.1)", "stationId", stationID, "status", req.OperationalStatus)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v201.ChangeAvailabilityResponse{Status: "Rejected"}, nil
		}

		// Determine which connector to change (0 means all)
		connectorID := 0
		if req.EVSE != nil {
			connectorID = req.EVSE.ID
		}

		// Map OCPP 2.0.1 status to 1.6 availability type
		availType := "Operative"
		if req.OperationalStatus == "Inoperative" {
			availType = "Inoperative"
		}

		// Change availability
		err := station.SessionManager.ChangeAvailability(connectorID, availType)
		if err != nil {
			m.logger.Warn("Cannot change availability immediately", "error", err)
			return &v201.ChangeAvailabilityResponse{Status: "Scheduled"}, nil
		}

		return &v201.ChangeAvailabilityResponse{Status: "Accepted"}, nil
	}

	// UnlockConnector handler
	m.v201Handler.OnUnlockConnector = func(stationID string, req *v201.UnlockConnectorRequest) (*v201.UnlockConnectorResponse, error) {
		m.logger.Info("Handling UnlockConnector (2.0.1)", "stationId", stationID, "evseId", req.EvseId, "connectorId", req.ConnectorId)

		// TODO: Implement actual unlock logic
		return &v201.UnlockConnectorResponse{Status: "UnknownConnector"}, nil
	}

	// ClearCache handler
	m.v201Handler.OnClearCache = func(stationID string, req *v201.ClearCacheRequest) (*v201.ClearCacheResponse, error) {
		m.logger.Info("Handling ClearCache (2.0.1)", "stationId", stationID)

		// TODO: Implement actual cache clear logic
		return &v201.ClearCacheResponse{Status: "Accepted"}, nil
	}

	// DataTransfer handler
	m.v201Handler.OnDataTransfer = func(stationID string, req *v201.DataTransferRequest) (*v201.DataTransferResponse, error) {
		m.logger.Info("Handling DataTransfer (2.0.1)", "stationId", stationID, "vendorId", req.VendorId, "messageId", req.MessageId)

		// TODO: Implement actual data transfer logic
		return &v201.DataTransferResponse{Status: v201.DataTransferStatusUnknownVendorId}, nil
	}

	// TriggerMessage handler
	m.v201Handler.OnTriggerMessage = func(stationID string, req *v201.TriggerMessageRequest) (*v201.TriggerMessageResponse, error) {
		m.logger.Info("Handling TriggerMessage (2.0.1)", "stationId", stationID, "requestedMessage", req.RequestedMessage)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v201.TriggerMessageResponse{Status: "Rejected"}, nil
		}

		// Handle different trigger types
		switch req.RequestedMessage {
		case "BootNotification":
			go m.sendBootNotification(stationID)
			return &v201.TriggerMessageResponse{Status: "Accepted"}, nil
		case "Heartbeat":
			go m.sendHeartbeat(stationID, station)
			return &v201.TriggerMessageResponse{Status: "Accepted"}, nil
		case "StatusNotification":
			go m.sendAllConnectorStatus(stationID, station)
			return &v201.TriggerMessageResponse{Status: "Accepted"}, nil
		case "SignCertificate":
			// Generate and send SignCertificate request for charging station certificate
			go m.sendSignCertificateRequest(stationID, station, v201.CertificateUseChargingStationCertificate)
			return &v201.TriggerMessageResponse{Status: "Accepted"}, nil
		default:
			return &v201.TriggerMessageResponse{Status: "NotImplemented"}, nil
		}
	}

	// GetTransactionStatus handler
	m.v201Handler.OnGetTransactionStatus = func(stationID string, req *v201.GetTransactionStatusRequest) (*v201.GetTransactionStatusResponse, error) {
		m.logger.Info("Handling GetTransactionStatus (2.0.1)", "stationId", stationID, "transactionId", req.TransactionId)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v201.GetTransactionStatusResponse{MessagesInQueue: false}, nil
		}

		// Check if transaction is ongoing
		ongoing := false
		if req.TransactionId != "" {
			connectors := station.SessionManager.GetAllConnectors()
			for _, connector := range connectors {
				tx := connector.GetTransaction()
				if tx != nil && tx.StringID == req.TransactionId {
					ongoing = true
					break
				}
			}
		}

		return &v201.GetTransactionStatusResponse{
			OngoingIndicator: &ongoing,
			MessagesInQueue:  false,
		}, nil
	}

	// ==================== Certificate Management Handlers ====================

	// CertificateSigned handler - CSMS sends signed certificate after CSR
	m.v201Handler.OnCertificateSigned = func(stationID string, req *v201.CertificateSignedRequest) (*v201.CertificateSignedResponse, error) {
		m.logger.Info("Handling CertificateSigned (2.0.1)", "stationId", stationID, "certType", req.CertificateType)

		// Get station to access certificate store
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.CertificateStore == nil {
			return &v201.CertificateSignedResponse{Status: "Rejected"}, nil
		}

		// Determine certificate type
		certType := v201.CertificateUseChargingStationCertificate
		if req.CertificateType == "V2GCertificate" {
			certType = v201.CertificateUseV2GCertificate
		}

		// Check if we have a pending CSR for this certificate type
		if !station.CertificateStore.HasPendingCSR(certType) {
			m.logger.Warn("No pending CSR for certificate type", "certType", certType)
			return &v201.CertificateSignedResponse{Status: "Rejected"}, nil
		}

		// Install the signed certificate
		status, err := station.CertificateStore.InstallSignedCertificate(certType, req.CertificateChain)
		if err != nil {
			m.logger.Error("Failed to install signed certificate", "error", err)
			return &v201.CertificateSignedResponse{Status: status}, nil
		}

		m.logger.Info("Installed signed certificate", "stationId", stationID, "certType", certType)
		return &v201.CertificateSignedResponse{Status: status}, nil
	}

	// DeleteCertificate handler
	m.v201Handler.OnDeleteCertificate = func(stationID string, req *v201.DeleteCertificateRequest) (*v201.DeleteCertificateResponse, error) {
		m.logger.Info("Handling DeleteCertificate (2.0.1)", "stationId", stationID,
			"hashAlgorithm", req.CertificateHashData.HashAlgorithm,
			"serialNumber", req.CertificateHashData.SerialNumber)

		// Get station to access certificate store
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.CertificateStore == nil {
			return &v201.DeleteCertificateResponse{Status: "Failed"}, nil
		}

		// Delete the certificate
		status := station.CertificateStore.DeleteCertificate(req.CertificateHashData)
		m.logger.Info("Deleted certificate", "stationId", stationID, "status", status, "serialNumber", req.CertificateHashData.SerialNumber)

		return &v201.DeleteCertificateResponse{Status: string(status)}, nil
	}

	// GetInstalledCertificateIds handler
	m.v201Handler.OnGetInstalledCertificateIds = func(stationID string, req *v201.GetInstalledCertificateIdsRequest) (*v201.GetInstalledCertificateIdsResponse, error) {
		m.logger.Info("Handling GetInstalledCertificateIds (2.0.1)", "stationId", stationID, "types", req.CertificateType)

		// Get station to access certificate store
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.CertificateStore == nil {
			return &v201.GetInstalledCertificateIdsResponse{
				Status:                   "NotFound",
				CertificateHashDataChain: []v201.CertificateHashDataChainType{},
			}, nil
		}

		// Get installed certificate IDs
		status, certChain := station.CertificateStore.GetInstalledCertificateIds(req.CertificateType)
		m.logger.Info("Got installed certificates", "stationId", stationID, "status", status, "count", len(certChain))

		return &v201.GetInstalledCertificateIdsResponse{
			Status:                   string(status),
			CertificateHashDataChain: certChain,
		}, nil
	}

	// InstallCertificate handler - for root certificates
	m.v201Handler.OnInstallCertificate = func(stationID string, req *v201.InstallCertificateRequest) (*v201.InstallCertificateResponse, error) {
		m.logger.Info("Handling InstallCertificate (2.0.1)", "stationId", stationID, "certType", req.CertificateType)

		// Get station to access certificate store
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists || station.CertificateStore == nil {
			return &v201.InstallCertificateResponse{Status: "Rejected"}, nil
		}

		// Install the certificate
		status, err := station.CertificateStore.InstallCertificate(v201.CertificateUseType(req.CertificateType), req.Certificate)
		if err != nil {
			m.logger.Error("Failed to install certificate", "error", err)
			return &v201.InstallCertificateResponse{Status: string(status)}, nil
		}

		m.logger.Info("Installed certificate", "stationId", stationID, "certType", req.CertificateType)
		return &v201.InstallCertificateResponse{Status: string(status)}, nil
	}
}

// setupV21HandlerCallbacks sets up callbacks for OCPP 2.1 handler
func (m *Manager) setupV21HandlerCallbacks() {
	// Since v21.Handler embeds v201.Handler, most callbacks are inherited
	// We only need to add 2.1-specific callbacks here

	// CostUpdated handler - CSMS updates running transaction cost
	m.v21Handler.OnCostUpdated = func(stationID string, req *v21.CostUpdatedRequest) (*v21.CostUpdatedResponse, error) {
		m.logger.Info("Handling CostUpdated (2.1)", "stationId", stationID, "transactionId", req.TransactionId, "totalCost", req.TotalCost)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			m.logger.Warn("Station not found for CostUpdated", "stationId", stationID)
			return &v21.CostUpdatedResponse{}, nil
		}

		// Find the transaction and update cost (for display purposes)
		connectors := station.SessionManager.GetAllConnectors()
		for _, connector := range connectors {
			tx := connector.GetTransaction()
			if tx != nil && tx.StringID == req.TransactionId {
				// Store cost for potential display/logging
				m.logger.Info("Updated transaction cost",
					"stationId", stationID,
					"transactionId", req.TransactionId,
					"cost", req.TotalCost,
				)
				break
			}
		}

		return &v21.CostUpdatedResponse{}, nil
	}

	// CustomerInformation handler - request customer information report
	m.v21Handler.OnCustomerInformation = func(stationID string, req *v21.CustomerInformationRequest) (*v21.CustomerInformationResponse, error) {
		m.logger.Info("Handling CustomerInformation (2.1)", "stationId", stationID, "requestId", req.RequestId)

		// For emulator, we accept but report no customer data
		go func() {
			time.Sleep(100 * time.Millisecond) // Brief delay before sending notification
			m.sendNotifyCustomerInformation(stationID, req.RequestId, "", true)
		}()

		return &v21.CustomerInformationResponse{Status: "Accepted"}, nil
	}

	// SetDisplayMessage handler - display message on station screen
	m.v21Handler.OnSetDisplayMessage = func(stationID string, req *v21.SetDisplayMessageRequest) (*v21.SetDisplayMessageResponse, error) {
		m.logger.Info("Handling SetDisplayMessage (2.1)",
			"stationId", stationID,
			"priority", req.Message.Priority,
			"content", req.Message.Message.Content,
		)

		// For emulator, we accept all display messages
		return &v21.SetDisplayMessageResponse{Status: v21.DisplayMessageStatusAccepted}, nil
	}

	// GetDisplayMessages handler - retrieve stored display messages
	m.v21Handler.OnGetDisplayMessages = func(stationID string, req *v21.GetDisplayMessagesRequest) (*v21.GetDisplayMessagesResponse, error) {
		m.logger.Info("Handling GetDisplayMessages (2.1)", "stationId", stationID, "requestId", req.RequestId)

		// For emulator, return that we have no stored messages (async response via NotifyDisplayMessages)
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.sendNotifyDisplayMessages(stationID, req.RequestId, nil, true)
		}()

		return &v21.GetDisplayMessagesResponse{Status: "Accepted"}, nil
	}

	// ClearDisplayMessage handler - clear specific display message
	m.v21Handler.OnClearDisplayMessage = func(stationID string, req *v21.ClearDisplayMessageRequest) (*v21.ClearDisplayMessageResponse, error) {
		m.logger.Info("Handling ClearDisplayMessage (2.1)", "stationId", stationID, "messageId", req.Id)

		// For emulator, report success (message not found since we don't store them)
		return &v21.ClearDisplayMessageResponse{Status: v21.ClearMessageStatusAccepted}, nil
	}

	// ReserveNow handler - make a reservation
	m.v21Handler.OnReserveNow = func(stationID string, req *v21.ReserveNowRequest) (*v21.ReserveNowResponse, error) {
		m.logger.Info("Handling ReserveNow (2.1)",
			"stationId", stationID,
			"reservationId", req.Id,
			"evseId", req.EvseId,
			"expiryDateTime", req.ExpiryDateTime,
		)

		// Get station
		m.mu.RLock()
		station, exists := m.stations[stationID]
		m.mu.RUnlock()

		if !exists {
			return &v21.ReserveNowResponse{Status: v21.ReservationStatusRejected}, nil
		}

		// Check if EVSE is available
		connectorID := 0
		if req.EvseId != nil {
			connectorID = *req.EvseId
		}

		if connectorID > 0 {
			connector, err := station.SessionManager.GetConnector(connectorID)
			if err != nil || connector == nil {
				return &v21.ReserveNowResponse{Status: v21.ReservationStatusUnavailable}, nil
			}
			state := connector.GetState()
			if state != ConnectorStateAvailable {
				if state == ConnectorStatePreparing || state == ConnectorStateCharging {
					return &v21.ReserveNowResponse{Status: v21.ReservationStatusOccupied}, nil
				}
				if state == ConnectorStateFaulted {
					return &v21.ReserveNowResponse{Status: v21.ReservationStatusFaulted}, nil
				}
				return &v21.ReserveNowResponse{Status: v21.ReservationStatusUnavailable}, nil
			}
		}

		// For emulator, accept the reservation
		m.logger.Info("Reservation accepted", "stationId", stationID, "reservationId", req.Id)
		return &v21.ReserveNowResponse{Status: v21.ReservationStatusAccepted}, nil
	}

	// CancelReservation handler - cancel a reservation
	m.v21Handler.OnCancelReservation = func(stationID string, req *v21.CancelReservationRequest) (*v21.CancelReservationResponse, error) {
		m.logger.Info("Handling CancelReservation (2.1)", "stationId", stationID, "reservationId", req.ReservationId)

		// For emulator, accept the cancellation
		return &v21.CancelReservationResponse{Status: v21.CancelReservationStatusAccepted}, nil
	}

	// SetChargingProfile handler - set/update charging profile
	m.v21Handler.OnSetChargingProfile = func(stationID string, req *v21.SetChargingProfileRequest) (*v21.SetChargingProfileResponse, error) {
		m.logger.Info("Handling SetChargingProfile (2.1)",
			"stationId", stationID,
			"evseId", req.EvseId,
			"profileId", req.ChargingProfile.ID,
			"purpose", req.ChargingProfile.ChargingProfilePurpose,
		)

		// For emulator, accept all charging profiles
		return &v21.SetChargingProfileResponse{Status: v21.ChargingProfileStatusAccepted}, nil
	}

	// GetChargingProfiles handler - get installed charging profiles
	m.v21Handler.OnGetChargingProfiles = func(stationID string, req *v21.GetChargingProfilesRequest) (*v21.GetChargingProfilesResponse, error) {
		m.logger.Info("Handling GetChargingProfiles (2.1)", "stationId", stationID, "requestId", req.RequestId)

		// For emulator, report no profiles (async response via ReportChargingProfiles)
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.sendReportChargingProfiles(stationID, req.RequestId, nil, true)
		}()

		return &v21.GetChargingProfilesResponse{Status: "NoProfiles"}, nil
	}

	// ClearChargingProfile handler - clear charging profiles
	m.v21Handler.OnClearChargingProfile = func(stationID string, req *v21.ClearChargingProfileRequest) (*v21.ClearChargingProfileResponse, error) {
		m.logger.Info("Handling ClearChargingProfile (2.1)",
			"stationId", stationID,
			"chargingProfileId", req.ChargingProfileId,
		)

		// For emulator, report unknown (no profiles stored)
		return &v21.ClearChargingProfileResponse{Status: v21.ClearChargingProfileStatusUnknown}, nil
	}

	// GetCompositeSchedule handler - get composite charging schedule
	m.v21Handler.OnGetCompositeSchedule = func(stationID string, req *v21.GetCompositeScheduleRequest) (*v21.GetCompositeScheduleResponse, error) {
		m.logger.Info("Handling GetCompositeSchedule (2.1)",
			"stationId", stationID,
			"evseId", req.EvseId,
			"duration", req.Duration,
		)

		// For emulator, report accepted but empty schedule
		return &v21.GetCompositeScheduleResponse{Status: "Accepted"}, nil
	}

	// GetLocalListVersion handler
	m.v21Handler.OnGetLocalListVersion = func(stationID string, req *v21.GetLocalListVersionRequest) (*v21.GetLocalListVersionResponse, error) {
		m.logger.Info("Handling GetLocalListVersion (2.1)", "stationId", stationID)
		return &v21.GetLocalListVersionResponse{VersionNumber: 0}, nil
	}

	// SendLocalList handler
	m.v21Handler.OnSendLocalList = func(stationID string, req *v21.SendLocalListRequest) (*v21.SendLocalListResponse, error) {
		m.logger.Info("Handling SendLocalList (2.1)", "stationId", stationID, "version", req.VersionNumber)
		return &v21.SendLocalListResponse{Status: "Accepted"}, nil
	}

	// UpdateFirmware handler
	m.v21Handler.OnUpdateFirmware = func(stationID string, req *v21.UpdateFirmwareRequest) (*v21.UpdateFirmwareResponse, error) {
		m.logger.Info("Handling UpdateFirmware (2.1)", "stationId", stationID, "requestId", req.RequestId)

		// Send firmware status notifications
		go func() {
			m.sendFirmwareStatusNotification(stationID, req.RequestId, "Downloading")
			time.Sleep(2 * time.Second)
			m.sendFirmwareStatusNotification(stationID, req.RequestId, "Downloaded")
			time.Sleep(1 * time.Second)
			m.sendFirmwareStatusNotification(stationID, req.RequestId, "Installing")
			time.Sleep(2 * time.Second)
			m.sendFirmwareStatusNotification(stationID, req.RequestId, "Installed")
		}()

		return &v21.UpdateFirmwareResponse{Status: "Accepted"}, nil
	}

	// SetNetworkProfile handler
	m.v21Handler.OnSetNetworkProfile = func(stationID string, req *v21.SetNetworkProfileRequest) (*v21.SetNetworkProfileResponse, error) {
		m.logger.Info("Handling SetNetworkProfile (2.1)", "stationId", stationID, "slot", req.ConfigurationSlot)
		return &v21.SetNetworkProfileResponse{Status: "Accepted"}, nil
	}

	// GetLog handler
	m.v21Handler.OnGetLog = func(stationID string, req *v21.GetLogRequest) (*v21.GetLogResponse, error) {
		m.logger.Info("Handling GetLog (2.1)", "stationId", stationID, "logType", req.LogType)

		// Send log status notifications
		go func() {
			m.sendLogStatusNotification(stationID, "Accepted")
			time.Sleep(1 * time.Second)
			m.sendLogStatusNotification(stationID, "Uploading")
			time.Sleep(2 * time.Second)
			m.sendLogStatusNotification(stationID, "Uploaded")
		}()

		return &v21.GetLogResponse{Status: "Accepted"}, nil
	}
}

// Helper methods for OCPP 2.1 async notifications

func (m *Manager) sendNotifyCustomerInformation(stationID string, requestId int, data string, tbc bool) {
	req := &v21.NotifyCustomerInformationRequest{
		Data:      data,
		SeqNo:     0,
		RequestId: requestId,
		Tbc:       tbc,
	}
	_, err := m.v21Handler.SendNotifyCustomerInformation(stationID, req)
	if err != nil {
		m.logger.Error("Failed to send NotifyCustomerInformation", "error", err)
	}
}

func (m *Manager) sendNotifyDisplayMessages(stationID string, requestId int, messages []v21.DisplayMessageType, tbc bool) {
	req := &v21.NotifyDisplayMessagesRequest{
		RequestId:   requestId,
		Tbc:         tbc,
		MessageInfo: messages,
	}
	_, err := m.v21Handler.SendNotifyDisplayMessages(stationID, req)
	if err != nil {
		m.logger.Error("Failed to send NotifyDisplayMessages", "error", err)
	}
}

func (m *Manager) sendReportChargingProfiles(stationID string, requestId int, profiles []v21.ChargingProfileType, tbc bool) {
	req := &v21.ReportChargingProfilesRequest{
		RequestId:           requestId,
		ChargingLimitSource: "Other",
		EvseId:              0,
		Tbc:                 tbc,
		ChargingProfile:     profiles,
	}
	_, err := m.v21Handler.SendReportChargingProfiles(stationID, req)
	if err != nil {
		m.logger.Error("Failed to send ReportChargingProfiles", "error", err)
	}
}

func (m *Manager) sendFirmwareStatusNotification(stationID string, requestId int, status string) {
	req := &v21.FirmwareStatusNotificationRequest{
		Status:    status,
		RequestId: &requestId,
	}
	_, err := m.v21Handler.SendFirmwareStatusNotification(stationID, req)
	if err != nil {
		m.logger.Error("Failed to send FirmwareStatusNotification", "error", err)
	}
}

func (m *Manager) sendLogStatusNotification(stationID string, status string) {
	req := &v21.LogStatusNotificationRequest{
		Status: status,
	}
	_, err := m.v21Handler.SendLogStatusNotification(stationID, req)
	if err != nil {
		m.logger.Error("Failed to send LogStatusNotification", "error", err)
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

		// Create station instance with device model
		deviceModel := v201.NewDeviceModel()
		deviceModel.UpdateStationInfo(config.Vendor, config.Model, config.SerialNumber, config.FirmwareVersion)

		// Add EVSE and Connector components for each connector
		for _, conn := range config.Connectors {
			deviceModel.AddEVSEComponent(conn.ID)
			deviceModel.AddConnectorComponent(conn.ID, 1, conn.Type)
		}

		// Create certificate store for ISO 15118 support
		certStore := v201.NewCertificateStore(config.StationID, config.Vendor, "US")

		station := &Station{
			Config:           config,
			StateMachine:     NewStateMachine(),
			SessionManager:   sessionManager,
			DeviceModel:      deviceModel,
			CertificateStore: certStore,
			pendingRequests:  make(map[string]string),
			pendingStartTx:   make(map[string]int),
			pendingStartTags: make(map[string]string),
			pendingAuthResp:  make(map[string]chan *v16.AuthorizeResponse),
			failedAuths:      make(map[string]time.Time),
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
		Config:           config,
		StateMachine:     NewStateMachine(),
		DeviceModel:      v201.NewDeviceModel(),
		CertificateStore: v201.NewCertificateStore(config.StationID, config.Vendor, "US"),
		pendingRequests:  make(map[string]string),
		pendingStartTx:   make(map[string]int),
		pendingStartTags: make(map[string]string),
		pendingAuthResp:  make(map[string]chan *v16.AuthorizeResponse),
		failedAuths:      make(map[string]time.Time),
		RuntimeState: RuntimeState{
			State:            StateDisconnected,
			ConnectionStatus: "not_connected",
		},
	}

	// Initialize device model with station info
	station.DeviceModel.UpdateStationInfo(config.Vendor, config.Model, config.SerialNumber, config.FirmwareVersion)

	// Add EVSE and Connector components for each connector
	for _, conn := range config.Connectors {
		station.DeviceModel.AddEVSEComponent(conn.ID)
		station.DeviceModel.AddConnectorComponent(conn.ID, 1, conn.Type)
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
	case "ocpp2.0.1", "2.0.1", "ocpp201":
		response, err = m.v201Handler.HandleCall(stationID, call)
	case "ocpp2.1", "2.1", "ocpp21":
		response, err = m.v21Handler.HandleCall(stationID, call)
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

	// Get protocol version
	station.mu.RLock()
	protocolVersion := station.Config.ProtocolVersion
	station.mu.RUnlock()

	// Handle response based on action and protocol version
	// Note: action names are the same in v16 and v201 for common messages
	switch action {
	case "BootNotification":
		m.handleBootNotificationResponse(stationID, station, result, protocolVersion)
	case "Heartbeat":
		m.handleHeartbeatResponse(stationID, station, result)
	case "Authorize":
		m.handleAuthorizeResponse(stationID, station, result)
	case "StartTransaction":
		m.handleStartTransactionResponse(stationID, station, result)
	case "TransactionEvent":
		m.handleTransactionEventResponse(stationID, station, result)
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
	protocolVersion := station.Config.ProtocolVersion
	vendor := station.Config.Vendor
	model := station.Config.Model
	serialNumber := station.Config.SerialNumber
	firmwareVersion := station.Config.FirmwareVersion
	iccid := station.Config.ICCID
	imsi := station.Config.IMSI
	station.mu.RUnlock()

	var call *ocpp.Call
	var err error

	switch protocolVersion {
	case "ocpp2.0.1", "2.0.1", "ocpp201":
		// OCPP 2.0.1 BootNotification
		req := v201.BootNotificationRequest{
			ChargingStation: v201.ChargingStation{
				Model:           model,
				VendorName:      vendor,
				SerialNumber:    serialNumber,
				FirmwareVersion: firmwareVersion,
			},
			Reason: v201.BootReasonPowerUp,
		}
		// Add modem info if available
		if iccid != "" || imsi != "" {
			req.ChargingStation.Modem = &v201.Modem{
				ICCID: iccid,
				IMSI:  imsi,
			}
		}
		call, err = ocpp.NewCall(string(v201.ActionBootNotification), req)
	default:
		// Default to OCPP 1.6
		req := v16.BootNotificationRequest{
			ChargePointVendor:       vendor,
			ChargePointModel:        model,
			ChargePointSerialNumber: serialNumber,
			FirmwareVersion:         firmwareVersion,
			Iccid:                   iccid,
			Imsi:                    imsi,
		}
		call, err = ocpp.NewCall(string(v16.ActionBootNotification), req)
	}

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

	m.logger.Info("Sent BootNotification", "stationId", stationID, "uniqueId", call.UniqueID, "protocol", protocolVersion)

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
func (m *Manager) handleBootNotificationResponse(stationID string, station *Station, result *ocpp.CallResult, protocolVersion string) {
	var status string
	var interval int

	switch protocolVersion {
	case "ocpp2.0.1", "2.0.1", "ocpp201":
		var resp v201.BootNotificationResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			m.logger.Error("Failed to unmarshal BootNotification response (2.0.1)", "stationId", stationID, "error", err)
			return
		}
		status = string(resp.Status)
		interval = resp.Interval
		m.logger.Info("BootNotification response (2.0.1)",
			"stationId", stationID,
			"status", resp.Status,
			"interval", resp.Interval,
			"currentTime", resp.CurrentTime,
		)
	default:
		var resp v16.BootNotificationResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			m.logger.Error("Failed to unmarshal BootNotification response", "stationId", stationID, "error", err)
			return
		}
		status = string(resp.Status)
		interval = resp.Interval
		m.logger.Info("BootNotification response",
			"stationId", stationID,
			"status", resp.Status,
			"interval", resp.Interval,
			"currentTime", resp.CurrentTime,
		)
	}

	if status == "Accepted" {
		// Start heartbeat with interval from CSMS (or use configured default)
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

// handleTransactionEventResponse processes TransactionEvent responses (OCPP 2.0.1)
func (m *Manager) handleTransactionEventResponse(stationID string, station *Station, result *ocpp.CallResult) {
	var resp v201.TransactionEventResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		m.logger.Error("Failed to unmarshal TransactionEvent response", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("TransactionEvent response received",
		"stationId", stationID,
		"messageId", result.UniqueID,
		"totalCost", resp.TotalCost,
		"chargingPriority", resp.ChargingPriority,
	)

	// Check if authorization info was returned
	if resp.IdTokenInfo != nil {
		if resp.IdTokenInfo.Status != v201.AuthorizationStatusAccepted {
			m.logger.Warn("Transaction authorization rejected",
				"stationId", stationID,
				"status", resp.IdTokenInfo.Status,
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

// sendSignCertificateRequest generates a CSR and sends SignCertificate request to CSMS
func (m *Manager) sendSignCertificateRequest(stationID string, station *Station, certType v201.CertificateUseType) {
	if station.CertificateStore == nil {
		m.logger.Error("Certificate store not initialized", "stationId", stationID)
		return
	}

	// Generate CSR
	csrPEM, err := station.CertificateStore.GenerateCSR(certType)
	if err != nil {
		m.logger.Error("Failed to generate CSR", "stationId", stationID, "error", err)
		return
	}

	// Determine certificate type string for OCPP message
	certTypeStr := "ChargingStationCertificate"
	if certType == v201.CertificateUseV2GCertificate {
		certTypeStr = "V2GCertificate"
	}

	// Create SignCertificate request
	req := &v201.SignCertificateRequest{
		Csr:             csrPEM,
		CertificateType: certTypeStr,
	}

	call, err := ocpp.NewCall(string(v201.ActionSignCertificate), req)
	if err != nil {
		m.logger.Error("Failed to create SignCertificate request", "stationId", stationID, "error", err)
		return
	}

	// Track pending request
	station.pendingMu.Lock()
	station.pendingRequests[call.UniqueID] = string(v201.ActionSignCertificate)
	station.pendingMu.Unlock()

	data, err := call.ToBytes()
	if err != nil {
		m.logger.Error("Failed to marshal SignCertificate", "stationId", stationID, "error", err)
		return
	}

	if err := m.connManager.SendMessage(stationID, data); err != nil {
		m.logger.Error("Failed to send SignCertificate", "stationId", stationID, "error", err)
		return
	}

	m.logger.Info("Sent SignCertificate request",
		"stationId", stationID,
		"certType", certTypeStr,
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
