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

func (h *Handler) handleRequestStartTransaction(stationID string, call *ocpp.Call) (*RequestStartTransactionResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionRequestStartTransaction), call, h.OnRequestStartTransaction, &RequestStartTransactionResponse{Status: "Rejected"})
}

func (h *Handler) handleRequestStopTransaction(stationID string, call *ocpp.Call) (*RequestStopTransactionResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionRequestStopTransaction), call, h.OnRequestStopTransaction, &RequestStopTransactionResponse{Status: "Rejected"})
}

func (h *Handler) handleReset(stationID string, call *ocpp.Call) (*ResetResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionReset), call, h.OnReset, &ResetResponse{Status: ResetStatusRejected})
}

// handleGetVariables handles GetVariables request. Its default reply is derived
// from the request, so it does not use the shared ocpp.HandleRequest helper.
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

// handleSetVariables handles SetVariables request. Its default reply is derived
// from the request, so it does not use the shared ocpp.HandleRequest helper.
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

func (h *Handler) handleChangeAvailability(stationID string, call *ocpp.Call) (*ChangeAvailabilityResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionChangeAvailability), call, h.OnChangeAvailability, &ChangeAvailabilityResponse{Status: "Rejected"})
}

func (h *Handler) handleUnlockConnector(stationID string, call *ocpp.Call) (*UnlockConnectorResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionUnlockConnector), call, h.OnUnlockConnector, &UnlockConnectorResponse{Status: "UnknownConnector"})
}

func (h *Handler) handleClearCache(stationID string, call *ocpp.Call) (*ClearCacheResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionClearCache), call, h.OnClearCache, &ClearCacheResponse{Status: "Rejected"})
}

func (h *Handler) handleDataTransfer(stationID string, call *ocpp.Call) (*DataTransferResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionDataTransfer), call, h.OnDataTransfer, &DataTransferResponse{Status: DataTransferStatusUnknownVendorId})
}

func (h *Handler) handleTriggerMessage(stationID string, call *ocpp.Call) (*TriggerMessageResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionTriggerMessage), call, h.OnTriggerMessage, &TriggerMessageResponse{Status: "NotImplemented"})
}

func (h *Handler) handleGetTransactionStatus(stationID string, call *ocpp.Call) (*GetTransactionStatusResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetTransactionStatus), call, h.OnGetTransactionStatus, &GetTransactionStatusResponse{MessagesInQueue: false})
}

// ==================== Certificate Management Handlers (CSMS → CS) ====================

func (h *Handler) handleCertificateSigned(stationID string, call *ocpp.Call) (*CertificateSignedResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionCertificateSigned), call, h.OnCertificateSigned, &CertificateSignedResponse{Status: "Rejected"})
}

func (h *Handler) handleDeleteCertificate(stationID string, call *ocpp.Call) (*DeleteCertificateResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionDeleteCertificate), call, h.OnDeleteCertificate, &DeleteCertificateResponse{Status: "NotFound"})
}

func (h *Handler) handleGetInstalledCertificateIds(stationID string, call *ocpp.Call) (*GetInstalledCertificateIdsResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetInstalledCertificateIds), call, h.OnGetInstalledCertificateIds, &GetInstalledCertificateIdsResponse{Status: "NotFound"})
}

func (h *Handler) handleInstallCertificate(stationID string, call *ocpp.Call) (*InstallCertificateResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionInstallCertificate), call, h.OnInstallCertificate, &InstallCertificateResponse{Status: "Rejected"})
}

// ==================== Outgoing Messages (Charging Station → CSMS) ====================

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
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
	}

	return ocpp.SendCall(h.SendMessage, stationID, string(ActionStatusNotification), req)
}

// SendAuthorize sends an Authorize request
func (h *Handler) SendAuthorize(stationID string, req *AuthorizeRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionAuthorize), req)
}

// SendTransactionEvent sends a TransactionEvent request (replaces Start/StopTransaction from OCPP 1.6)
func (h *Handler) SendTransactionEvent(stationID string, req *TransactionEventRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
	}

	return ocpp.SendCall(h.SendMessage, stationID, string(ActionTransactionEvent), req)
}

// SendMeterValues sends a MeterValues request
func (h *Handler) SendMeterValues(stationID string, req *MeterValuesRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionMeterValues), req)
}

// SendSecurityEventNotification sends a SecurityEventNotification request
func (h *Handler) SendSecurityEventNotification(stationID string, req *SecurityEventNotificationRequest) (*ocpp.Call, error) {
	// Set timestamp if not provided
	if req.Timestamp == (DateTime{}) {
		req.Timestamp = DateTime{Time: time.Now()}
	}

	return ocpp.SendCall(h.SendMessage, stationID, string(ActionSecurityEventNotification), req)
}

// SendNotifyEvent sends a NotifyEvent request
func (h *Handler) SendNotifyEvent(stationID string, req *NotifyEventRequest) (*ocpp.Call, error) {
	// Set generatedAt if not provided
	if req.GeneratedAt == (DateTime{}) {
		req.GeneratedAt = DateTime{Time: time.Now()}
	}

	return ocpp.SendCall(h.SendMessage, stationID, string(ActionNotifyEvent), req)
}

// SendDataTransfer sends a DataTransfer request
func (h *Handler) SendDataTransfer(stationID string, req *DataTransferRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionDataTransfer), req)
}

// ==================== Certificate Management Outgoing (CS → CSMS) ====================

// SendSignCertificate sends a SignCertificate request
func (h *Handler) SendSignCertificate(stationID string, req *SignCertificateRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionSignCertificate), req)
}

// SendGet15118EVCertificate sends a Get15118EVCertificate request
func (h *Handler) SendGet15118EVCertificate(stationID string, req *Get15118EVCertificateRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionGet15118EVCertificate), req)
}

// SendGetCertificateStatus sends a GetCertificateStatus request
func (h *Handler) SendGetCertificateStatus(stationID string, req *GetCertificateStatusRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.SendMessage, stationID, string(ActionGetCertificateStatus), req)
}

// ==================== Response Handlers ====================

// HandleCallResult processes CallResult responses from CSMS
func (h *Handler) HandleCallResult(stationID string, result *ocpp.CallResult, originalAction Action) (interface{}, error) {
	h.logger.Debug("Handling OCPP 2.0.1 CallResult", "stationId", stationID, "action", originalAction)

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
	case ActionTransactionEvent:
		return ocpp.UnmarshalResult[TransactionEventResponse](result, action)
	case ActionMeterValues:
		return ocpp.UnmarshalResult[MeterValuesResponse](result, action)
	case ActionSecurityEventNotification:
		return ocpp.UnmarshalResult[SecurityEventNotificationResponse](result, action)
	case ActionNotifyEvent:
		return ocpp.UnmarshalResult[NotifyEventResponse](result, action)
	case ActionDataTransfer:
		return ocpp.UnmarshalResult[DataTransferResponse](result, action)
	// Certificate management responses
	case ActionSignCertificate:
		return ocpp.UnmarshalResult[SignCertificateResponse](result, action)
	case ActionGet15118EVCertificate:
		return ocpp.UnmarshalResult[Get15118EVCertificateResponse](result, action)
	case ActionGetCertificateStatus:
		return ocpp.UnmarshalResult[GetCertificateStatusResponse](result, action)
	default:
		return nil, fmt.Errorf("unknown action for CallResult: %s", originalAction)
	}
}
