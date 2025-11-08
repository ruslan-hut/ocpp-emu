package v16

import (
	"encoding/json"
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

// handleRemoteStartTransaction handles RemoteStartTransaction request
func (h *Handler) handleRemoteStartTransaction(stationID string, call *ocpp.Call) (*RemoteStartTransactionResponse, error) {
	var req RemoteStartTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RemoteStartTransaction request: %w", err)
	}

	if h.OnRemoteStartTransaction == nil {
		return &RemoteStartTransactionResponse{Status: "Rejected"}, nil
	}

	return h.OnRemoteStartTransaction(stationID, &req)
}

// handleRemoteStopTransaction handles RemoteStopTransaction request
func (h *Handler) handleRemoteStopTransaction(stationID string, call *ocpp.Call) (*RemoteStopTransactionResponse, error) {
	var req RemoteStopTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RemoteStopTransaction request: %w", err)
	}

	if h.OnRemoteStopTransaction == nil {
		return &RemoteStopTransactionResponse{Status: "Rejected"}, nil
	}

	return h.OnRemoteStopTransaction(stationID, &req)
}

// handleReset handles Reset request
func (h *Handler) handleReset(stationID string, call *ocpp.Call) (*ResetResponse, error) {
	var req ResetRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Reset request: %w", err)
	}

	if h.OnReset == nil {
		return &ResetResponse{Status: "Rejected"}, nil
	}

	return h.OnReset(stationID, &req)
}

// handleUnlockConnector handles UnlockConnector request
func (h *Handler) handleUnlockConnector(stationID string, call *ocpp.Call) (*UnlockConnectorResponse, error) {
	var req UnlockConnectorRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UnlockConnector request: %w", err)
	}

	if h.OnUnlockConnector == nil {
		return &UnlockConnectorResponse{Status: "NotSupported"}, nil
	}

	return h.OnUnlockConnector(stationID, &req)
}

// handleChangeAvailability handles ChangeAvailability request
func (h *Handler) handleChangeAvailability(stationID string, call *ocpp.Call) (*ChangeAvailabilityResponse, error) {
	var req ChangeAvailabilityRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ChangeAvailability request: %w", err)
	}

	if h.OnChangeAvailability == nil {
		return &ChangeAvailabilityResponse{Status: "Rejected"}, nil
	}

	return h.OnChangeAvailability(stationID, &req)
}

// handleChangeConfiguration handles ChangeConfiguration request
func (h *Handler) handleChangeConfiguration(stationID string, call *ocpp.Call) (*ChangeConfigurationResponse, error) {
	var req ChangeConfigurationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ChangeConfiguration request: %w", err)
	}

	if h.OnChangeConfiguration == nil {
		return &ChangeConfigurationResponse{Status: "NotSupported"}, nil
	}

	return h.OnChangeConfiguration(stationID, &req)
}

// handleGetConfiguration handles GetConfiguration request
func (h *Handler) handleGetConfiguration(stationID string, call *ocpp.Call) (*GetConfigurationResponse, error) {
	var req GetConfigurationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetConfiguration request: %w", err)
	}

	if h.OnGetConfiguration == nil {
		return &GetConfigurationResponse{}, nil
	}

	return h.OnGetConfiguration(stationID, &req)
}

// handleClearCache handles ClearCache request
func (h *Handler) handleClearCache(stationID string, call *ocpp.Call) (*ClearCacheResponse, error) {
	var req ClearCacheRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ClearCache request: %w", err)
	}

	if h.OnClearCache == nil {
		return &ClearCacheResponse{Status: "Rejected"}, nil
	}

	return h.OnClearCache(stationID, &req)
}

// handleDataTransfer handles DataTransfer request
func (h *Handler) handleDataTransfer(stationID string, call *ocpp.Call) (*DataTransferResponse, error) {
	var req DataTransferRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DataTransfer request: %w", err)
	}

	if h.OnDataTransfer == nil {
		return &DataTransferResponse{Status: "UnknownVendorId"}, nil
	}

	return h.OnDataTransfer(stationID, &req)
}

// ==================== Outgoing Messages (Charge Point → CSMS) ====================

// SendBootNotification sends a BootNotification request
func (h *Handler) SendBootNotification(stationID string, req *BootNotificationRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionBootNotification), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create BootNotification call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BootNotification: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send BootNotification: %w", err)
		}
	}

	return call, nil
}

// SendHeartbeat sends a Heartbeat request
func (h *Handler) SendHeartbeat(stationID string) (*ocpp.Call, error) {
	req := HeartbeatRequest{}
	call, err := ocpp.NewCall(string(ActionHeartbeat), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create Heartbeat call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Heartbeat: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send Heartbeat: %w", err)
		}
	}

	return call, nil
}

// SendStatusNotification sends a StatusNotification request
func (h *Handler) SendStatusNotification(stationID string, req *StatusNotificationRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == nil {
		now := DateTime{Time: time.Now()}
		req.Timestamp = &now
	}

	call, err := ocpp.NewCall(string(ActionStatusNotification), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create StatusNotification call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal StatusNotification: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send StatusNotification: %w", err)
		}
	}

	return call, nil
}

// SendAuthorize sends an Authorize request
func (h *Handler) SendAuthorize(stationID string, req *AuthorizeRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionAuthorize), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create Authorize call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Authorize: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send Authorize: %w", err)
		}
	}

	return call, nil
}

// SendStartTransaction sends a StartTransaction request
func (h *Handler) SendStartTransaction(stationID string, req *StartTransactionRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionStartTransaction), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create StartTransaction call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal StartTransaction: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send StartTransaction: %w", err)
		}
	}

	return call, nil
}

// SendStopTransaction sends a StopTransaction request
func (h *Handler) SendStopTransaction(stationID string, req *StopTransactionRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionStopTransaction), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create StopTransaction call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal StopTransaction: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send StopTransaction: %w", err)
		}
	}

	return call, nil
}

// SendMeterValues sends a MeterValues request
func (h *Handler) SendMeterValues(stationID string, req *MeterValuesRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionMeterValues), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create MeterValues call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MeterValues: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send MeterValues: %w", err)
		}
	}

	return call, nil
}

// SendDataTransfer sends a DataTransfer request
func (h *Handler) SendDataTransfer(stationID string, req *DataTransferRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionDataTransfer), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create DataTransfer call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DataTransfer: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send DataTransfer: %w", err)
		}
	}

	return call, nil
}

// ==================== Response Handlers ====================

// HandleCallResult processes CallResult responses from CSMS
func (h *Handler) HandleCallResult(stationID string, result *ocpp.CallResult, originalAction Action) (interface{}, error) {
	h.logger.Debug("Handling OCPP 1.6 CallResult", "stationId", stationID, "action", originalAction)

	switch originalAction {
	case ActionBootNotification:
		var resp BootNotificationResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal BootNotification response: %w", err)
		}
		return &resp, nil

	case ActionHeartbeat:
		var resp HeartbeatResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Heartbeat response: %w", err)
		}
		return &resp, nil

	case ActionStatusNotification:
		var resp StatusNotificationResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal StatusNotification response: %w", err)
		}
		return &resp, nil

	case ActionAuthorize:
		var resp AuthorizeResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Authorize response: %w", err)
		}
		return &resp, nil

	case ActionStartTransaction:
		var resp StartTransactionResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal StartTransaction response: %w", err)
		}
		return &resp, nil

	case ActionStopTransaction:
		var resp StopTransactionResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal StopTransaction response: %w", err)
		}
		return &resp, nil

	case ActionMeterValues:
		var resp MeterValuesResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MeterValues response: %w", err)
		}
		return &resp, nil

	case ActionDataTransfer:
		var resp DataTransferResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DataTransfer response: %w", err)
		}
		return &resp, nil

	default:
		return nil, fmt.Errorf("unknown action for CallResult: %s", originalAction)
	}
}
