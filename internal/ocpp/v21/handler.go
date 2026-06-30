package v21

import (
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
	return ocpp.HandleRequest(stationID, string(ActionCostUpdated), call, h.OnCostUpdated, &CostUpdatedResponse{})
}

func (h *Handler) handleCustomerInformation(stationID string, call *ocpp.Call) (*CustomerInformationResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionCustomerInformation), call, h.OnCustomerInformation, &CustomerInformationResponse{Status: "Rejected"})
}

// ==================== Display Message Handlers ====================

func (h *Handler) handleSetDisplayMessage(stationID string, call *ocpp.Call) (*SetDisplayMessageResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionSetDisplayMessage), call, h.OnSetDisplayMessage, &SetDisplayMessageResponse{Status: DisplayMessageStatusAccepted})
}

func (h *Handler) handleGetDisplayMessages(stationID string, call *ocpp.Call) (*GetDisplayMessagesResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetDisplayMessages), call, h.OnGetDisplayMessages, &GetDisplayMessagesResponse{Status: "Unknown"})
}

func (h *Handler) handleClearDisplayMessage(stationID string, call *ocpp.Call) (*ClearDisplayMessageResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionClearDisplayMessage), call, h.OnClearDisplayMessage, &ClearDisplayMessageResponse{Status: ClearMessageStatusUnknown})
}

// ==================== Reservation Handlers ====================

func (h *Handler) handleReserveNow(stationID string, call *ocpp.Call) (*ReserveNowResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionReserveNow), call, h.OnReserveNow, &ReserveNowResponse{Status: ReservationStatusRejected})
}

func (h *Handler) handleCancelReservation(stationID string, call *ocpp.Call) (*CancelReservationResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionCancelReservation), call, h.OnCancelReservation, &CancelReservationResponse{Status: CancelReservationStatusRejected})
}

// ==================== Charging Profile Handlers ====================

func (h *Handler) handleSetChargingProfile(stationID string, call *ocpp.Call) (*SetChargingProfileResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionSetChargingProfile), call, h.OnSetChargingProfile, &SetChargingProfileResponse{Status: ChargingProfileStatusAccepted})
}

func (h *Handler) handleGetChargingProfiles(stationID string, call *ocpp.Call) (*GetChargingProfilesResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetChargingProfiles), call, h.OnGetChargingProfiles, &GetChargingProfilesResponse{Status: "NoProfiles"})
}

func (h *Handler) handleClearChargingProfile(stationID string, call *ocpp.Call) (*ClearChargingProfileResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionClearChargingProfile), call, h.OnClearChargingProfile, &ClearChargingProfileResponse{Status: ClearChargingProfileStatusUnknown})
}

func (h *Handler) handleGetCompositeSchedule(stationID string, call *ocpp.Call) (*GetCompositeScheduleResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetCompositeSchedule), call, h.OnGetCompositeSchedule, &GetCompositeScheduleResponse{Status: "Rejected"})
}

// ==================== Local List Handlers ====================

func (h *Handler) handleGetLocalListVersion(stationID string, call *ocpp.Call) (*GetLocalListVersionResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetLocalListVersion), call, h.OnGetLocalListVersion, &GetLocalListVersionResponse{VersionNumber: 0})
}

func (h *Handler) handleSendLocalList(stationID string, call *ocpp.Call) (*SendLocalListResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionSendLocalList), call, h.OnSendLocalList, &SendLocalListResponse{Status: "Accepted"})
}

// ==================== Firmware Handlers ====================

func (h *Handler) handleUpdateFirmware(stationID string, call *ocpp.Call) (*UpdateFirmwareResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionUpdateFirmware), call, h.OnUpdateFirmware, &UpdateFirmwareResponse{Status: "Accepted"})
}

func (h *Handler) handleSetNetworkProfile(stationID string, call *ocpp.Call) (*SetNetworkProfileResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionSetNetworkProfile), call, h.OnSetNetworkProfile, &SetNetworkProfileResponse{Status: "Rejected"})
}

func (h *Handler) handleGetLog(stationID string, call *ocpp.Call) (*GetLogResponse, error) {
	return ocpp.HandleRequest(stationID, string(ActionGetLog), call, h.OnGetLog, &GetLogResponse{Status: "Accepted"})
}

// ==================== Outgoing Message Methods (CS → CSMS) ====================

// SendNotifyCustomerInformation sends a NotifyCustomerInformation request
func (h *Handler) SendNotifyCustomerInformation(stationID string, req *NotifyCustomerInformationRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionNotifyCustomerInformation), req)
}

// SendNotifyEVChargingNeeds sends a NotifyEVChargingNeeds request
func (h *Handler) SendNotifyEVChargingNeeds(stationID string, req *NotifyEVChargingNeedsRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionNotifyEVChargingNeeds), req)
}

// SendNotifyDisplayMessages sends a NotifyDisplayMessages request
func (h *Handler) SendNotifyDisplayMessages(stationID string, req *NotifyDisplayMessagesRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionNotifyDisplayMessages), req)
}

// SendReportChargingProfiles sends a ReportChargingProfiles request
func (h *Handler) SendReportChargingProfiles(stationID string, req *ReportChargingProfilesRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionReportChargingProfiles), req)
}

// SendNotifyChargingLimit sends a NotifyChargingLimit request
func (h *Handler) SendNotifyChargingLimit(stationID string, req *NotifyChargingLimitRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionNotifyChargingLimit), req)
}

// SendClearedChargingLimit sends a ClearedChargingLimit request
func (h *Handler) SendClearedChargingLimit(stationID string, req *ClearedChargingLimitRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionClearedChargingLimit), req)
}

// SendLogStatusNotification sends a LogStatusNotification request
func (h *Handler) SendLogStatusNotification(stationID string, req *LogStatusNotificationRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionLogStatusNotification), req)
}

// SendFirmwareStatusNotification sends a FirmwareStatusNotification request
func (h *Handler) SendFirmwareStatusNotification(stationID string, req *FirmwareStatusNotificationRequest) (*ocpp.Call, error) {
	return ocpp.SendCall(h.Handler.SendMessage, stationID, string(ActionFirmwareStatusNotification), req)
}
