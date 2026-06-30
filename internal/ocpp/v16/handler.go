package v16

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
)

// Handler handles OCPP 1.6 protocol messages
type Handler struct {
	logger *slog.Logger

	// Callbacks for handling incoming requests from CSMS
	OnRemoteStartTransaction func(stationID string, req *RemoteStartTransactionRequest) (*RemoteStartTransactionResponse, error)
	OnRemoteStopTransaction  func(stationID string, req *RemoteStopTransactionRequest) (*RemoteStopTransactionResponse, error)
	OnReset                  func(stationID string, req *ResetRequest) (*ResetResponse, error)
	OnUnlockConnector        func(stationID string, req *UnlockConnectorRequest) (*UnlockConnectorResponse, error)
	OnChangeAvailability     func(stationID string, req *ChangeAvailabilityRequest) (*ChangeAvailabilityResponse, error)
	OnChangeConfiguration    func(stationID string, req *ChangeConfigurationRequest) (*ChangeConfigurationResponse, error)
	OnGetConfiguration       func(stationID string, req *GetConfigurationRequest) (*GetConfigurationResponse, error)
	OnClearCache             func(stationID string, req *ClearCacheRequest) (*ClearCacheResponse, error)
	OnDataTransfer           func(stationID string, req *DataTransferRequest) (*DataTransferResponse, error)

	// Callback for sending messages
	SendMessage func(stationID string, data []byte) error
}

// NewHandler creates a new OCPP 1.6 handler
func NewHandler(logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		logger: logger,
	}
}

// HandleCall processes incoming Call messages from CSMS
func (h *Handler) HandleCall(stationID string, call *ocpp.Call) (interface{}, error) {
	h.logger.Debug("Handling OCPP 1.6 Call", "stationId", stationID, "action", call.Action)

	switch Action(call.Action) {
	case ActionRemoteStartTransaction:
		return h.handleRemoteStartTransaction(stationID, call)
	case ActionRemoteStopTransaction:
		return h.handleRemoteStopTransaction(stationID, call)
	case ActionReset:
		return h.handleReset(stationID, call)
	case ActionUnlockConnector:
		return h.handleUnlockConnector(stationID, call)
	case ActionChangeAvailability:
		return h.handleChangeAvailability(stationID, call)
	case ActionChangeConfiguration:
		return h.handleChangeConfiguration(stationID, call)
	case ActionGetConfiguration:
		return h.handleGetConfiguration(stationID, call)
	case ActionClearCache:
		return h.handleClearCache(stationID, call)
	case ActionDataTransfer:
		return h.handleDataTransfer(stationID, call)
	default:
		return nil, fmt.Errorf("action not implemented: %s", call.Action)
	}
}

func (h *Handler) handleRemoteStartTransaction(stationID string, call *ocpp.Call) (*RemoteStartTransactionResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionRemoteStartTransaction), call, h.OnRemoteStartTransaction, &RemoteStartTransactionResponse{Status: "Rejected"})
}

func (h *Handler) handleRemoteStopTransaction(stationID string, call *ocpp.Call) (*RemoteStopTransactionResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionRemoteStopTransaction), call, h.OnRemoteStopTransaction, &RemoteStopTransactionResponse{Status: "Rejected"})
}

func (h *Handler) handleReset(stationID string, call *ocpp.Call) (*ResetResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionReset), call, h.OnReset, &ResetResponse{Status: "Rejected"})
}

func (h *Handler) handleUnlockConnector(stationID string, call *ocpp.Call) (*UnlockConnectorResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionUnlockConnector), call, h.OnUnlockConnector, &UnlockConnectorResponse{Status: "NotSupported"})
}

func (h *Handler) handleChangeAvailability(stationID string, call *ocpp.Call) (*ChangeAvailabilityResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionChangeAvailability), call, h.OnChangeAvailability, &ChangeAvailabilityResponse{Status: "Rejected"})
}

func (h *Handler) handleChangeConfiguration(stationID string, call *ocpp.Call) (*ChangeConfigurationResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionChangeConfiguration), call, h.OnChangeConfiguration, &ChangeConfigurationResponse{Status: "NotSupported"})
}

func (h *Handler) handleGetConfiguration(stationID string, call *ocpp.Call) (*GetConfigurationResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetConfiguration), call, h.OnGetConfiguration, &GetConfigurationResponse{})
}

func (h *Handler) handleClearCache(stationID string, call *ocpp.Call) (*ClearCacheResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionClearCache), call, h.OnClearCache, &ClearCacheResponse{Status: "Rejected"})
}

func (h *Handler) handleDataTransfer(stationID string, call *ocpp.Call) (*DataTransferResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionDataTransfer), call, h.OnDataTransfer, &DataTransferResponse{Status: "UnknownVendorId"})
}

// ==================== Outgoing Messages (Charge Point → CSMS) ====================

// SendBootNotification sends a BootNotification request
func (h *Handler) SendBootNotification(stationID string, req *BootNotificationRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionBootNotification), req)
}

// SendHeartbeat sends a Heartbeat request
func (h *Handler) SendHeartbeat(stationID string) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionHeartbeat), HeartbeatRequest{})
}

// SendStatusNotification sends a StatusNotification request
func (h *Handler) SendStatusNotification(stationID string, req *StatusNotificationRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == nil {
		now := DateTime{Time: time.Now()}
		req.Timestamp = &now
	}

	return ocpp.SendCall(h.SendMessage, stationID, string(ActionStatusNotification), req)
}

// SendAuthorize sends an Authorize request
func (h *Handler) SendAuthorize(stationID string, req *AuthorizeRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionAuthorize), req)
}

// SendStartTransaction sends a StartTransaction request
func (h *Handler) SendStartTransaction(stationID string, req *StartTransactionRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionStartTransaction), req)
}

// SendStopTransaction sends a StopTransaction request
func (h *Handler) SendStopTransaction(stationID string, req *StopTransactionRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionStopTransaction), req)
}

// SendMeterValues sends a MeterValues request
func (h *Handler) SendMeterValues(stationID string, req *MeterValuesRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionMeterValues), req)
}

// SendDataTransfer sends a DataTransfer request
func (h *Handler) SendDataTransfer(stationID string, req *DataTransferRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionDataTransfer), req)
}

// ==================== Response Handlers ====================

// HandleCallResult processes CallResult responses from CSMS
func (h *Handler) HandleCallResult(stationID string, result *ocpp.CallResult, originalAction Action) (interface{}, error) {
	h.logger.Debug("Handling OCPP 1.6 CallResult", "stationId", stationID, "action", originalAction)

	action := string(originalAction)
	switch originalAction {
	case ActionBootNotification:
		return ocpp.UnmarshalResult[BootNotificationResponse](result, action)
	case ActionHeartbeat:
		return ocpp.UnmarshalResult[HeartbeatResponse](result, action)
	case ActionStatusNotification:
		return ocpp.UnmarshalResult[StatusNotificationResponse](result, action)
	case ActionAuthorize:
		return ocpp.UnmarshalResult[AuthorizeResponse](result, action)
	case ActionStartTransaction:
		return ocpp.UnmarshalResult[StartTransactionResponse](result, action)
	case ActionStopTransaction:
		return ocpp.UnmarshalResult[StopTransactionResponse](result, action)
	case ActionMeterValues:
		return ocpp.UnmarshalResult[MeterValuesResponse](result, action)
	case ActionDataTransfer:
		return ocpp.UnmarshalResult[DataTransferResponse](result, action)
	default:
		return nil, fmt.Errorf("unknown action for CallResult: %s", originalAction)
	}
}
