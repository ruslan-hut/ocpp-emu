package v201

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
)

// Handler handles OCPP 2.0.1 protocol messages
type Handler struct {
	logger *slog.Logger

	// Callbacks for handling incoming requests from CSMS
	OnRequestStartTransaction func(stationID string, req *RequestStartTransactionRequest) (*RequestStartTransactionResponse, error)
	OnRequestStopTransaction  func(stationID string, req *RequestStopTransactionRequest) (*RequestStopTransactionResponse, error)
	OnReset                   func(stationID string, req *ResetRequest) (*ResetResponse, error)
	OnGetVariables            func(stationID string, req *GetVariablesRequest) (*GetVariablesResponse, error)
	OnSetVariables            func(stationID string, req *SetVariablesRequest) (*SetVariablesResponse, error)
	OnChangeAvailability      func(stationID string, req *ChangeAvailabilityRequest) (*ChangeAvailabilityResponse, error)
	OnUnlockConnector         func(stationID string, req *UnlockConnectorRequest) (*UnlockConnectorResponse, error)
	OnClearCache              func(stationID string, req *ClearCacheRequest) (*ClearCacheResponse, error)
	OnDataTransfer            func(stationID string, req *DataTransferRequest) (*DataTransferResponse, error)
	OnTriggerMessage          func(stationID string, req *TriggerMessageRequest) (*TriggerMessageResponse, error)
	OnGetTransactionStatus    func(stationID string, req *GetTransactionStatusRequest) (*GetTransactionStatusResponse, error)

	// Certificate management callbacks (CSMS → CS)
	OnCertificateSigned          func(stationID string, req *CertificateSignedRequest) (*CertificateSignedResponse, error)
	OnDeleteCertificate          func(stationID string, req *DeleteCertificateRequest) (*DeleteCertificateResponse, error)
	OnGetInstalledCertificateIds func(stationID string, req *GetInstalledCertificateIdsRequest) (*GetInstalledCertificateIdsResponse, error)
	OnInstallCertificate         func(stationID string, req *InstallCertificateRequest) (*InstallCertificateResponse, error)

	// Callback for sending messages
	SendMessage func(stationID string, data []byte) error
}

// NewHandler creates a new OCPP 2.0.1 handler
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
	h.logger.Debug("Handling OCPP 2.0.1 Call", "stationId", stationID, "action", call.Action)

	switch Action(call.Action) {
	case ActionRequestStartTransaction:
		return h.handleRequestStartTransaction(stationID, call)
	case ActionRequestStopTransaction:
		return h.handleRequestStopTransaction(stationID, call)
	case ActionReset:
		return h.handleReset(stationID, call)
	case ActionGetVariables:
		return h.handleGetVariables(stationID, call)
	case ActionSetVariables:
		return h.handleSetVariables(stationID, call)
	case ActionChangeAvailability:
		return h.handleChangeAvailability(stationID, call)
	case ActionUnlockConnector:
		return h.handleUnlockConnector(stationID, call)
	case ActionClearCache:
		return h.handleClearCache(stationID, call)
	case ActionDataTransfer:
		return h.handleDataTransfer(stationID, call)
	case ActionTriggerMessage:
		return h.handleTriggerMessage(stationID, call)
	case ActionGetTransactionStatus:
		return h.handleGetTransactionStatus(stationID, call)
	// Certificate management
	case ActionCertificateSigned:
		return h.handleCertificateSigned(stationID, call)
	case ActionDeleteCertificate:
		return h.handleDeleteCertificate(stationID, call)
	case ActionGetInstalledCertificateIds:
		return h.handleGetInstalledCertificateIds(stationID, call)
	case ActionInstallCertificate:
		return h.handleInstallCertificate(stationID, call)
	default:
		return nil, fmt.Errorf("action not implemented: %s", call.Action)
	}
}

// ==================== CSMS → CS Request Handlers ====================

// handleRequestStartTransaction handles RequestStartTransaction request
func (h *Handler) handleRequestStartTransaction(stationID string, call *ocpp.Call) (*RequestStartTransactionResponse, error) {
	var req RequestStartTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RequestStartTransaction request: %w", err)
	}

	if h.OnRequestStartTransaction == nil {
		return &RequestStartTransactionResponse{Status: "Rejected"}, nil
	}

	return h.OnRequestStartTransaction(stationID, &req)
}

// handleRequestStopTransaction handles RequestStopTransaction request
func (h *Handler) handleRequestStopTransaction(stationID string, call *ocpp.Call) (*RequestStopTransactionResponse, error) {
	var req RequestStopTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RequestStopTransaction request: %w", err)
	}

	if h.OnRequestStopTransaction == nil {
		return &RequestStopTransactionResponse{Status: "Rejected"}, nil
	}

	return h.OnRequestStopTransaction(stationID, &req)
}

// handleReset handles Reset request
func (h *Handler) handleReset(stationID string, call *ocpp.Call) (*ResetResponse, error) {
	var req ResetRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Reset request: %w", err)
	}

	if h.OnReset == nil {
		return &ResetResponse{Status: ResetStatusRejected}, nil
	}

	return h.OnReset(stationID, &req)
}

// handleGetVariables handles GetVariables request
func (h *Handler) handleGetVariables(stationID string, call *ocpp.Call) (*GetVariablesResponse, error) {
	var req GetVariablesRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetVariables request: %w", err)
	}

	if h.OnGetVariables == nil {
		// Return empty results for each requested variable
		results := make([]GetVariableResult, len(req.GetVariableData))
		for i, data := range req.GetVariableData {
			results[i] = GetVariableResult{
				AttributeStatus: GetVariableStatusRejected,
				Component:       data.Component,
				Variable:        data.Variable,
			}
		}
		return &GetVariablesResponse{GetVariableResult: results}, nil
	}

	return h.OnGetVariables(stationID, &req)
}

// handleSetVariables handles SetVariables request
func (h *Handler) handleSetVariables(stationID string, call *ocpp.Call) (*SetVariablesResponse, error) {
	var req SetVariablesRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SetVariables request: %w", err)
	}

	if h.OnSetVariables == nil {
		// Return rejected for each requested variable
		results := make([]SetVariableResult, len(req.SetVariableData))
		for i, data := range req.SetVariableData {
			results[i] = SetVariableResult{
				AttributeStatus: SetVariableStatusRejected,
				Component:       data.Component,
				Variable:        data.Variable,
			}
		}
		return &SetVariablesResponse{SetVariableResult: results}, nil
	}

	return h.OnSetVariables(stationID, &req)
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

// handleUnlockConnector handles UnlockConnector request
func (h *Handler) handleUnlockConnector(stationID string, call *ocpp.Call) (*UnlockConnectorResponse, error) {
	var req UnlockConnectorRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UnlockConnector request: %w", err)
	}

	if h.OnUnlockConnector == nil {
		return &UnlockConnectorResponse{Status: "UnknownConnector"}, nil
	}

	return h.OnUnlockConnector(stationID, &req)
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
		return &DataTransferResponse{Status: DataTransferStatusUnknownVendorId}, nil
	}

	return h.OnDataTransfer(stationID, &req)
}

// handleTriggerMessage handles TriggerMessage request
func (h *Handler) handleTriggerMessage(stationID string, call *ocpp.Call) (*TriggerMessageResponse, error) {
	var req TriggerMessageRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TriggerMessage request: %w", err)
	}

	if h.OnTriggerMessage == nil {
		return &TriggerMessageResponse{Status: "NotImplemented"}, nil
	}

	return h.OnTriggerMessage(stationID, &req)
}

// handleGetTransactionStatus handles GetTransactionStatus request
func (h *Handler) handleGetTransactionStatus(stationID string, call *ocpp.Call) (*GetTransactionStatusResponse, error) {
	var req GetTransactionStatusRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetTransactionStatus request: %w", err)
	}

	if h.OnGetTransactionStatus == nil {
		return &GetTransactionStatusResponse{MessagesInQueue: false}, nil
	}

	return h.OnGetTransactionStatus(stationID, &req)
}

// ==================== Certificate Management Handlers (CSMS → CS) ====================

// handleCertificateSigned handles CertificateSigned request
func (h *Handler) handleCertificateSigned(stationID string, call *ocpp.Call) (*CertificateSignedResponse, error) {
	var req CertificateSignedRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CertificateSigned request: %w", err)
	}

	if h.OnCertificateSigned == nil {
		return &CertificateSignedResponse{Status: "Rejected"}, nil
	}

	return h.OnCertificateSigned(stationID, &req)
}

// handleDeleteCertificate handles DeleteCertificate request
func (h *Handler) handleDeleteCertificate(stationID string, call *ocpp.Call) (*DeleteCertificateResponse, error) {
	var req DeleteCertificateRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DeleteCertificate request: %w", err)
	}

	if h.OnDeleteCertificate == nil {
		return &DeleteCertificateResponse{Status: "NotFound"}, nil
	}

	return h.OnDeleteCertificate(stationID, &req)
}

// handleGetInstalledCertificateIds handles GetInstalledCertificateIds request
func (h *Handler) handleGetInstalledCertificateIds(stationID string, call *ocpp.Call) (*GetInstalledCertificateIdsResponse, error) {
	var req GetInstalledCertificateIdsRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetInstalledCertificateIds request: %w", err)
	}

	if h.OnGetInstalledCertificateIds == nil {
		return &GetInstalledCertificateIdsResponse{Status: "NotFound"}, nil
	}

	return h.OnGetInstalledCertificateIds(stationID, &req)
}

// handleInstallCertificate handles InstallCertificate request
func (h *Handler) handleInstallCertificate(stationID string, call *ocpp.Call) (*InstallCertificateResponse, error) {
	var req InstallCertificateRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal InstallCertificate request: %w", err)
	}

	if h.OnInstallCertificate == nil {
		return &InstallCertificateResponse{Status: "Rejected"}, nil
	}

	return h.OnInstallCertificate(stationID, &req)
}

// ==================== Outgoing Messages (Charging Station → CSMS) ====================

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
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
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

// SendTransactionEvent sends a TransactionEvent request (replaces Start/StopTransaction from OCPP 1.6)
func (h *Handler) SendTransactionEvent(stationID string, req *TransactionEventRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
	}

	call, err := ocpp.NewCall(string(ActionTransactionEvent), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create TransactionEvent call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TransactionEvent: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send TransactionEvent: %w", err)
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

// SendSecurityEventNotification sends a SecurityEventNotification request
func (h *Handler) SendSecurityEventNotification(stationID string, req *SecurityEventNotificationRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
	}

	call, err := ocpp.NewCall(string(ActionSecurityEventNotification), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create SecurityEventNotification call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SecurityEventNotification: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send SecurityEventNotification: %w", err)
		}
	}

	return call, nil
}

// SendNotifyEvent sends a NotifyEvent request
func (h *Handler) SendNotifyEvent(stationID string, req *NotifyEventRequest) (*ocpp.Call, error) {
	// Set generatedAt if not provided
	if req.GeneratedAt == (DateTime{}) {
		req.GeneratedAt = DateTime{Time: time.Now()}
	}

	call, err := ocpp.NewCall(string(ActionNotifyEvent), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotifyEvent call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NotifyEvent: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send NotifyEvent: %w", err)
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

// ==================== Certificate Management Outgoing (CS → CSMS) ====================

// SendSignCertificate sends a SignCertificate request
func (h *Handler) SendSignCertificate(stationID string, req *SignCertificateRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionSignCertificate), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create SignCertificate call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SignCertificate: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send SignCertificate: %w", err)
		}
	}

	return call, nil
}

// SendGet15118EVCertificate sends a Get15118EVCertificate request
func (h *Handler) SendGet15118EVCertificate(stationID string, req *Get15118EVCertificateRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionGet15118EVCertificate), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create Get15118EVCertificate call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Get15118EVCertificate: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send Get15118EVCertificate: %w", err)
		}
	}

	return call, nil
}

// SendGetCertificateStatus sends a GetCertificateStatus request
func (h *Handler) SendGetCertificateStatus(stationID string, req *GetCertificateStatusRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionGetCertificateStatus), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create GetCertificateStatus call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GetCertificateStatus: %w", err)
	}

	if h.SendMessage != nil {
		if err := h.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send GetCertificateStatus: %w", err)
		}
	}

	return call, nil
}

// ==================== Response Handlers ====================

// HandleCallResult processes CallResult responses from CSMS
func (h *Handler) HandleCallResult(stationID string, result *ocpp.CallResult, originalAction Action) (interface{}, error) {
	h.logger.Debug("Handling OCPP 2.0.1 CallResult", "stationId", stationID, "action", originalAction)

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

	case ActionTransactionEvent:
		var resp TransactionEventResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TransactionEvent response: %w", err)
		}
		return &resp, nil

	case ActionMeterValues:
		var resp MeterValuesResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MeterValues response: %w", err)
		}
		return &resp, nil

	case ActionSecurityEventNotification:
		var resp SecurityEventNotificationResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SecurityEventNotification response: %w", err)
		}
		return &resp, nil

	case ActionNotifyEvent:
		var resp NotifyEventResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal NotifyEvent response: %w", err)
		}
		return &resp, nil

	case ActionDataTransfer:
		var resp DataTransferResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DataTransfer response: %w", err)
		}
		return &resp, nil

	// Certificate management responses
	case ActionSignCertificate:
		var resp SignCertificateResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SignCertificate response: %w", err)
		}
		return &resp, nil

	case ActionGet15118EVCertificate:
		var resp Get15118EVCertificateResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Get15118EVCertificate response: %w", err)
		}
		return &resp, nil

	case ActionGetCertificateStatus:
		var resp GetCertificateStatusResponse
		if err := json.Unmarshal(result.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal GetCertificateStatus response: %w", err)
		}
		return &resp, nil

	default:
		return nil, fmt.Errorf("unknown action for CallResult: %s", originalAction)
	}
}
