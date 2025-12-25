package v21

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v201"
)

// Handler handles OCPP 2.1 messages, extending 2.0.1 functionality
type Handler struct {
	*v201.Handler // Embed 2.0.1 handler for inherited functionality

	// Cost and Tariff callbacks (CSMS → CS)
	OnCostUpdated         func(stationID string, req *CostUpdatedRequest) (*CostUpdatedResponse, error)
	OnCustomerInformation func(stationID string, req *CustomerInformationRequest) (*CustomerInformationResponse, error)

	// Display Message callbacks (CSMS → CS)
	OnSetDisplayMessage   func(stationID string, req *SetDisplayMessageRequest) (*SetDisplayMessageResponse, error)
	OnGetDisplayMessages  func(stationID string, req *GetDisplayMessagesRequest) (*GetDisplayMessagesResponse, error)
	OnClearDisplayMessage func(stationID string, req *ClearDisplayMessageRequest) (*ClearDisplayMessageResponse, error)

	// Reservation callbacks (CSMS → CS)
	OnReserveNow        func(stationID string, req *ReserveNowRequest) (*ReserveNowResponse, error)
	OnCancelReservation func(stationID string, req *CancelReservationRequest) (*CancelReservationResponse, error)

	// Charging Profile callbacks (CSMS → CS)
	OnSetChargingProfile   func(stationID string, req *SetChargingProfileRequest) (*SetChargingProfileResponse, error)
	OnGetChargingProfiles  func(stationID string, req *GetChargingProfilesRequest) (*GetChargingProfilesResponse, error)
	OnClearChargingProfile func(stationID string, req *ClearChargingProfileRequest) (*ClearChargingProfileResponse, error)
	OnGetCompositeSchedule func(stationID string, req *GetCompositeScheduleRequest) (*GetCompositeScheduleResponse, error)

	// Local List callbacks (CSMS → CS)
	OnGetLocalListVersion func(stationID string, req *GetLocalListVersionRequest) (*GetLocalListVersionResponse, error)
	OnSendLocalList       func(stationID string, req *SendLocalListRequest) (*SendLocalListResponse, error)

	// Firmware callbacks (CSMS → CS)
	OnUpdateFirmware    func(stationID string, req *UpdateFirmwareRequest) (*UpdateFirmwareResponse, error)
	OnSetNetworkProfile func(stationID string, req *SetNetworkProfileRequest) (*SetNetworkProfileResponse, error)
	OnGetLog            func(stationID string, req *GetLogRequest) (*GetLogResponse, error)
}

// NewHandler creates a new OCPP 2.1 handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		Handler: v201.NewHandler(logger),
	}
}

// HandleCall handles incoming OCPP 2.1 Call messages
func (h *Handler) HandleCall(stationID string, call *ocpp.Call) (interface{}, error) {
	action := Action(call.Action)

	// First check for 2.1-specific actions
	switch action {
	// Cost and Tariff
	case ActionCostUpdated:
		return h.handleCostUpdated(stationID, call)
	case ActionCustomerInformation:
		return h.handleCustomerInformation(stationID, call)

	// Display Messages
	case ActionSetDisplayMessage:
		return h.handleSetDisplayMessage(stationID, call)
	case ActionGetDisplayMessages:
		return h.handleGetDisplayMessages(stationID, call)
	case ActionClearDisplayMessage:
		return h.handleClearDisplayMessage(stationID, call)

	// Reservations
	case ActionReserveNow:
		return h.handleReserveNow(stationID, call)
	case ActionCancelReservation:
		return h.handleCancelReservation(stationID, call)

	// Charging Profiles
	case ActionSetChargingProfile:
		return h.handleSetChargingProfile(stationID, call)
	case ActionGetChargingProfiles:
		return h.handleGetChargingProfiles(stationID, call)
	case ActionClearChargingProfile:
		return h.handleClearChargingProfile(stationID, call)
	case ActionGetCompositeSchedule:
		return h.handleGetCompositeSchedule(stationID, call)

	// Local List
	case ActionGetLocalListVersion:
		return h.handleGetLocalListVersion(stationID, call)
	case ActionSendLocalList:
		return h.handleSendLocalList(stationID, call)

	// Firmware
	case ActionUpdateFirmware:
		return h.handleUpdateFirmware(stationID, call)
	case ActionSetNetworkProfile:
		return h.handleSetNetworkProfile(stationID, call)
	case ActionGetLog:
		return h.handleGetLog(stationID, call)

	default:
		// Fall back to 2.0.1 handler for inherited actions
		return h.Handler.HandleCall(stationID, call)
	}
}

// ==================== Cost and Tariff Handlers ====================

func (h *Handler) handleCostUpdated(stationID string, call *ocpp.Call) (*CostUpdatedResponse, error) {
	var req CostUpdatedRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CostUpdated request: %w", err)
	}

	if h.OnCostUpdated == nil {
		return &CostUpdatedResponse{}, nil
	}

	return h.OnCostUpdated(stationID, &req)
}

func (h *Handler) handleCustomerInformation(stationID string, call *ocpp.Call) (*CustomerInformationResponse, error) {
	var req CustomerInformationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CustomerInformation request: %w", err)
	}

	if h.OnCustomerInformation == nil {
		return &CustomerInformationResponse{Status: "Rejected"}, nil
	}

	return h.OnCustomerInformation(stationID, &req)
}

// ==================== Display Message Handlers ====================

func (h *Handler) handleSetDisplayMessage(stationID string, call *ocpp.Call) (*SetDisplayMessageResponse, error) {
	var req SetDisplayMessageRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SetDisplayMessage request: %w", err)
	}

	if h.OnSetDisplayMessage == nil {
		return &SetDisplayMessageResponse{Status: DisplayMessageStatusAccepted}, nil
	}

	return h.OnSetDisplayMessage(stationID, &req)
}

func (h *Handler) handleGetDisplayMessages(stationID string, call *ocpp.Call) (*GetDisplayMessagesResponse, error) {
	var req GetDisplayMessagesRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetDisplayMessages request: %w", err)
	}

	if h.OnGetDisplayMessages == nil {
		return &GetDisplayMessagesResponse{Status: "Unknown"}, nil
	}

	return h.OnGetDisplayMessages(stationID, &req)
}

func (h *Handler) handleClearDisplayMessage(stationID string, call *ocpp.Call) (*ClearDisplayMessageResponse, error) {
	var req ClearDisplayMessageRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ClearDisplayMessage request: %w", err)
	}

	if h.OnClearDisplayMessage == nil {
		return &ClearDisplayMessageResponse{Status: ClearMessageStatusUnknown}, nil
	}

	return h.OnClearDisplayMessage(stationID, &req)
}

// ==================== Reservation Handlers ====================

func (h *Handler) handleReserveNow(stationID string, call *ocpp.Call) (*ReserveNowResponse, error) {
	var req ReserveNowRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ReserveNow request: %w", err)
	}

	if h.OnReserveNow == nil {
		return &ReserveNowResponse{Status: ReservationStatusRejected}, nil
	}

	return h.OnReserveNow(stationID, &req)
}

func (h *Handler) handleCancelReservation(stationID string, call *ocpp.Call) (*CancelReservationResponse, error) {
	var req CancelReservationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CancelReservation request: %w", err)
	}

	if h.OnCancelReservation == nil {
		return &CancelReservationResponse{Status: CancelReservationStatusRejected}, nil
	}

	return h.OnCancelReservation(stationID, &req)
}

// ==================== Charging Profile Handlers ====================

func (h *Handler) handleSetChargingProfile(stationID string, call *ocpp.Call) (*SetChargingProfileResponse, error) {
	var req SetChargingProfileRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SetChargingProfile request: %w", err)
	}

	if h.OnSetChargingProfile == nil {
		return &SetChargingProfileResponse{Status: ChargingProfileStatusAccepted}, nil
	}

	return h.OnSetChargingProfile(stationID, &req)
}

func (h *Handler) handleGetChargingProfiles(stationID string, call *ocpp.Call) (*GetChargingProfilesResponse, error) {
	var req GetChargingProfilesRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetChargingProfiles request: %w", err)
	}

	if h.OnGetChargingProfiles == nil {
		return &GetChargingProfilesResponse{Status: "NoProfiles"}, nil
	}

	return h.OnGetChargingProfiles(stationID, &req)
}

func (h *Handler) handleClearChargingProfile(stationID string, call *ocpp.Call) (*ClearChargingProfileResponse, error) {
	var req ClearChargingProfileRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ClearChargingProfile request: %w", err)
	}

	if h.OnClearChargingProfile == nil {
		return &ClearChargingProfileResponse{Status: ClearChargingProfileStatusUnknown}, nil
	}

	return h.OnClearChargingProfile(stationID, &req)
}

func (h *Handler) handleGetCompositeSchedule(stationID string, call *ocpp.Call) (*GetCompositeScheduleResponse, error) {
	var req GetCompositeScheduleRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetCompositeSchedule request: %w", err)
	}

	if h.OnGetCompositeSchedule == nil {
		return &GetCompositeScheduleResponse{Status: "Rejected"}, nil
	}

	return h.OnGetCompositeSchedule(stationID, &req)
}

// ==================== Local List Handlers ====================

func (h *Handler) handleGetLocalListVersion(stationID string, call *ocpp.Call) (*GetLocalListVersionResponse, error) {
	var req GetLocalListVersionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetLocalListVersion request: %w", err)
	}

	if h.OnGetLocalListVersion == nil {
		return &GetLocalListVersionResponse{VersionNumber: 0}, nil
	}

	return h.OnGetLocalListVersion(stationID, &req)
}

func (h *Handler) handleSendLocalList(stationID string, call *ocpp.Call) (*SendLocalListResponse, error) {
	var req SendLocalListRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SendLocalList request: %w", err)
	}

	if h.OnSendLocalList == nil {
		return &SendLocalListResponse{Status: "Accepted"}, nil
	}

	return h.OnSendLocalList(stationID, &req)
}

// ==================== Firmware Handlers ====================

func (h *Handler) handleUpdateFirmware(stationID string, call *ocpp.Call) (*UpdateFirmwareResponse, error) {
	var req UpdateFirmwareRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UpdateFirmware request: %w", err)
	}

	if h.OnUpdateFirmware == nil {
		return &UpdateFirmwareResponse{Status: "Accepted"}, nil
	}

	return h.OnUpdateFirmware(stationID, &req)
}

func (h *Handler) handleSetNetworkProfile(stationID string, call *ocpp.Call) (*SetNetworkProfileResponse, error) {
	var req SetNetworkProfileRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SetNetworkProfile request: %w", err)
	}

	if h.OnSetNetworkProfile == nil {
		return &SetNetworkProfileResponse{Status: "Rejected"}, nil
	}

	return h.OnSetNetworkProfile(stationID, &req)
}

func (h *Handler) handleGetLog(stationID string, call *ocpp.Call) (*GetLogResponse, error) {
	var req GetLogRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetLog request: %w", err)
	}

	if h.OnGetLog == nil {
		return &GetLogResponse{Status: "Accepted"}, nil
	}

	return h.OnGetLog(stationID, &req)
}

// ==================== Outgoing Message Methods (CS → CSMS) ====================

// SendNotifyCustomerInformation sends a NotifyCustomerInformation request
func (h *Handler) SendNotifyCustomerInformation(stationID string, req *NotifyCustomerInformationRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionNotifyCustomerInformation), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotifyCustomerInformation call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NotifyCustomerInformation: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send NotifyCustomerInformation: %w", err)
		}
	}

	return call, nil
}

// SendNotifyEVChargingNeeds sends a NotifyEVChargingNeeds request
func (h *Handler) SendNotifyEVChargingNeeds(stationID string, req *NotifyEVChargingNeedsRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionNotifyEVChargingNeeds), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotifyEVChargingNeeds call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NotifyEVChargingNeeds: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send NotifyEVChargingNeeds: %w", err)
		}
	}

	return call, nil
}

// SendNotifyDisplayMessages sends a NotifyDisplayMessages request
func (h *Handler) SendNotifyDisplayMessages(stationID string, req *NotifyDisplayMessagesRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionNotifyDisplayMessages), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotifyDisplayMessages call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NotifyDisplayMessages: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send NotifyDisplayMessages: %w", err)
		}
	}

	return call, nil
}

// SendReportChargingProfiles sends a ReportChargingProfiles request
func (h *Handler) SendReportChargingProfiles(stationID string, req *ReportChargingProfilesRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionReportChargingProfiles), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReportChargingProfiles call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ReportChargingProfiles: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send ReportChargingProfiles: %w", err)
		}
	}

	return call, nil
}

// SendNotifyChargingLimit sends a NotifyChargingLimit request
func (h *Handler) SendNotifyChargingLimit(stationID string, req *NotifyChargingLimitRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionNotifyChargingLimit), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotifyChargingLimit call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NotifyChargingLimit: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send NotifyChargingLimit: %w", err)
		}
	}

	return call, nil
}

// SendClearedChargingLimit sends a ClearedChargingLimit request
func (h *Handler) SendClearedChargingLimit(stationID string, req *ClearedChargingLimitRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionClearedChargingLimit), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClearedChargingLimit call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ClearedChargingLimit: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send ClearedChargingLimit: %w", err)
		}
	}

	return call, nil
}

// SendLogStatusNotification sends a LogStatusNotification request
func (h *Handler) SendLogStatusNotification(stationID string, req *LogStatusNotificationRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionLogStatusNotification), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create LogStatusNotification call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LogStatusNotification: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send LogStatusNotification: %w", err)
		}
	}

	return call, nil
}

// SendFirmwareStatusNotification sends a FirmwareStatusNotification request
func (h *Handler) SendFirmwareStatusNotification(stationID string, req *FirmwareStatusNotificationRequest) (*ocpp.Call, error) {
	call, err := ocpp.NewCall(string(ActionFirmwareStatusNotification), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create FirmwareStatusNotification call: %w", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FirmwareStatusNotification: %w", err)
	}

	if h.Handler.SendMessage != nil {
		if err := h.Handler.SendMessage(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send FirmwareStatusNotification: %w", err)
		}
	}

	return call, nil
}
