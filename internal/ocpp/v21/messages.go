package v21

import (
	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v201"
)

// =========== Cost and Tariff Messages ===========

// CostUpdatedRequest represents a CostUpdated request (CSMS → CS)
type CostUpdatedRequest struct {
	TotalCost     float64 `json:"totalCost"`
	TransactionId string  `json:"transactionId"`
}

// CostUpdatedResponse represents a CostUpdated response (CS → CSMS)
type CostUpdatedResponse struct {
}

// NotifyCustomerInformationRequest represents a NotifyCustomerInformation request (CS → CSMS)
type NotifyCustomerInformationRequest struct {
	Data        string `json:"data"`
	Tbc         bool   `json:"tbc"` // To be continued
	SeqNo       int    `json:"seqNo"`
	GeneratedAt string `json:"generatedAt"`
	RequestId   int    `json:"requestId"`
}

// NotifyCustomerInformationResponse represents a NotifyCustomerInformation response (CSMS → CS)
type NotifyCustomerInformationResponse struct {
}

// CustomerInformationRequest represents a CustomerInformation request (CSMS → CS)
type CustomerInformationRequest struct {
	RequestId           int                           `json:"requestId"`
	Report              bool                          `json:"report"`
	Clear               bool                          `json:"clear"`
	CustomerIdentifier  *string                       `json:"customerIdentifier,omitempty"`
	IdToken             *IdToken                      `json:"idToken,omitempty"`
	CustomerCertificate *v201.CertificateHashDataType `json:"customerCertificate,omitempty"`
}

// CustomerInformationResponse represents a CustomerInformation response (CS → CSMS)
type CustomerInformationResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Invalid
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// NotifyEVChargingNeedsRequest represents a NotifyEVChargingNeeds request (CS → CSMS)
type NotifyEVChargingNeedsRequest struct {
	MaxScheduleTuples *int              `json:"maxScheduleTuples,omitempty"`
	EvseId            int               `json:"evseId"`
	ChargingNeeds     ChargingNeedsType `json:"chargingNeeds"`
}

// NotifyEVChargingNeedsResponse represents a NotifyEVChargingNeeds response (CSMS → CS)
type NotifyEVChargingNeedsResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Processing
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// NotifyEVChargingScheduleRequest represents a NotifyEVChargingSchedule request (CS → CSMS)
type NotifyEVChargingScheduleRequest struct {
	TimeBase         string               `json:"timeBase"`
	EvseId           int                  `json:"evseId"`
	ChargingSchedule ChargingScheduleType `json:"chargingSchedule"`
}

// NotifyEVChargingScheduleResponse represents a NotifyEVChargingSchedule response (CSMS → CS)
type NotifyEVChargingScheduleResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== Display Messages ===========

// SetDisplayMessageRequest represents a SetDisplayMessage request (CSMS → CS)
type SetDisplayMessageRequest struct {
	Message DisplayMessageType `json:"message"`
}

// SetDisplayMessageResponse represents a SetDisplayMessage response (CS → CSMS)
type SetDisplayMessageResponse struct {
	Status     DisplayMessageStatusType `json:"status"`
	StatusInfo *StatusInfo              `json:"statusInfo,omitempty"`
}

// GetDisplayMessagesRequest represents a GetDisplayMessages request (CSMS → CS)
type GetDisplayMessagesRequest struct {
	RequestId int                  `json:"requestId"`
	Id        []int                `json:"id,omitempty"`
	Priority  *MessagePriorityType `json:"priority,omitempty"`
	State     *MessageStateType    `json:"state,omitempty"`
}

// GetDisplayMessagesResponse represents a GetDisplayMessages response (CS → CSMS)
type GetDisplayMessagesResponse struct {
	Status     string      `json:"status"` // Accepted, Unknown
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ClearDisplayMessageRequest represents a ClearDisplayMessage request (CSMS → CS)
type ClearDisplayMessageRequest struct {
	Id int `json:"id"`
}

// ClearDisplayMessageResponse represents a ClearDisplayMessage response (CS → CSMS)
type ClearDisplayMessageResponse struct {
	Status     ClearMessageStatusType `json:"status"`
	StatusInfo *StatusInfo            `json:"statusInfo,omitempty"`
}

// NotifyDisplayMessagesRequest represents a NotifyDisplayMessages request (CS → CSMS)
type NotifyDisplayMessagesRequest struct {
	RequestId   int                  `json:"requestId"`
	Tbc         bool                 `json:"tbc"` // To be continued
	MessageInfo []DisplayMessageType `json:"messageInfo,omitempty"`
}

// NotifyDisplayMessagesResponse represents a NotifyDisplayMessages response (CSMS → CS)
type NotifyDisplayMessagesResponse struct {
}

// =========== Reservation Messages ===========

// ReserveNowRequest represents a ReserveNow request (CSMS → CS)
type ReserveNowRequest struct {
	Id             int            `json:"id"`
	ExpiryDateTime string         `json:"expiryDateTime"`
	ConnectorType  *ConnectorType `json:"connectorType,omitempty"`
	EvseId         *int           `json:"evseId,omitempty"`
	IdToken        IdToken        `json:"idToken"`
	GroupIdToken   *IdToken       `json:"groupIdToken,omitempty"`
}

// ReserveNowResponse represents a ReserveNow response (CS → CSMS)
type ReserveNowResponse struct {
	Status     ReservationStatusType `json:"status"`
	StatusInfo *StatusInfo           `json:"statusInfo,omitempty"`
}

// CancelReservationRequest represents a CancelReservation request (CSMS → CS)
type CancelReservationRequest struct {
	ReservationId int `json:"reservationId"`
}

// CancelReservationResponse represents a CancelReservation response (CS → CSMS)
type CancelReservationResponse struct {
	Status     CancelReservationStatusType `json:"status"`
	StatusInfo *StatusInfo                 `json:"statusInfo,omitempty"`
}

// =========== Charging Profile Messages ===========

// SetChargingProfileRequest represents a SetChargingProfile request (CSMS → CS)
type SetChargingProfileRequest struct {
	EvseId          int                 `json:"evseId"`
	ChargingProfile ChargingProfileType `json:"chargingProfile"`
}

// SetChargingProfileResponse represents a SetChargingProfile response (CS → CSMS)
type SetChargingProfileResponse struct {
	Status     ChargingProfileStatusType `json:"status"`
	StatusInfo *StatusInfo               `json:"statusInfo,omitempty"`
}

// GetChargingProfilesRequest represents a GetChargingProfiles request (CSMS → CS)
type GetChargingProfilesRequest struct {
	RequestId       int                           `json:"requestId"`
	EvseId          *int                          `json:"evseId,omitempty"`
	ChargingProfile *ChargingProfileCriterionType `json:"chargingProfile,omitempty"`
}

// ChargingProfileCriterionType represents criteria for selecting charging profiles
type ChargingProfileCriterionType struct {
	ChargingProfilePurpose *ChargingProfilePurposeType `json:"chargingProfilePurpose,omitempty"`
	StackLevel             *int                        `json:"stackLevel,omitempty"`
	ChargingProfileId      []int                       `json:"chargingProfileId,omitempty"`
	ChargingLimitSource    []string                    `json:"chargingLimitSource,omitempty"`
}

// GetChargingProfilesResponse represents a GetChargingProfiles response (CS → CSMS)
type GetChargingProfilesResponse struct {
	Status     string      `json:"status"` // Accepted, NoProfiles
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ReportChargingProfilesRequest represents a ReportChargingProfiles request (CS → CSMS)
type ReportChargingProfilesRequest struct {
	RequestId           int                   `json:"requestId"`
	ChargingLimitSource string                `json:"chargingLimitSource"`
	EvseId              int                   `json:"evseId"`
	ChargingProfile     []ChargingProfileType `json:"chargingProfile"`
	Tbc                 bool                  `json:"tbc"` // To be continued
}

// ReportChargingProfilesResponse represents a ReportChargingProfiles response (CSMS → CS)
type ReportChargingProfilesResponse struct {
}

// ClearChargingProfileRequest represents a ClearChargingProfile request (CSMS → CS)
type ClearChargingProfileRequest struct {
	ChargingProfileId       *int                      `json:"chargingProfileId,omitempty"`
	ChargingProfileCriteria *ClearChargingProfileType `json:"chargingProfileCriteria,omitempty"`
}

// ClearChargingProfileType represents criteria for clearing charging profiles
type ClearChargingProfileType struct {
	EvseId                 *int                        `json:"evseId,omitempty"`
	ChargingProfilePurpose *ChargingProfilePurposeType `json:"chargingProfilePurpose,omitempty"`
	StackLevel             *int                        `json:"stackLevel,omitempty"`
}

// ClearChargingProfileResponse represents a ClearChargingProfile response (CS → CSMS)
type ClearChargingProfileResponse struct {
	Status     ClearChargingProfileStatusType `json:"status"`
	StatusInfo *StatusInfo                    `json:"statusInfo,omitempty"`
}

// NotifyChargingLimitRequest represents a NotifyChargingLimit request (CS → CSMS)
type NotifyChargingLimitRequest struct {
	EvseId           *int                   `json:"evseId,omitempty"`
	ChargingLimit    ChargingLimitType      `json:"chargingLimit"`
	ChargingSchedule []ChargingScheduleType `json:"chargingSchedule,omitempty"`
}

// ChargingLimitType represents charging limit information
type ChargingLimitType struct {
	ChargingLimitSource string `json:"chargingLimitSource"` // EMS, Other, SO, CSO
	IsGridCritical      *bool  `json:"isGridCritical,omitempty"`
	IsLocalGeneration   *bool  `json:"isLocalGeneration,omitempty"`
}

// NotifyChargingLimitResponse represents a NotifyChargingLimit response (CSMS → CS)
type NotifyChargingLimitResponse struct {
}

// GetCompositeScheduleRequest represents a GetCompositeSchedule request (CSMS → CS)
type GetCompositeScheduleRequest struct {
	Duration         int     `json:"duration"`
	ChargingRateUnit *string `json:"chargingRateUnit,omitempty"` // W or A
	EvseId           int     `json:"evseId"`
}

// GetCompositeScheduleResponse represents a GetCompositeSchedule response (CS → CSMS)
type GetCompositeScheduleResponse struct {
	Status     string                 `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo            `json:"statusInfo,omitempty"`
	Schedule   *CompositeScheduleType `json:"schedule,omitempty"`
}

// CompositeScheduleType represents a composite charging schedule
type CompositeScheduleType struct {
	EvseId                 int                          `json:"evseId"`
	Duration               int                          `json:"duration"`
	ScheduleStart          string                       `json:"scheduleStart"`
	ChargingRateUnit       string                       `json:"chargingRateUnit"`
	ChargingSchedulePeriod []ChargingSchedulePeriodType `json:"chargingSchedulePeriod"`
}

// ClearedChargingLimitRequest represents a ClearedChargingLimit request (CS → CSMS)
type ClearedChargingLimitRequest struct {
	ChargingLimitSource string `json:"chargingLimitSource"`
	EvseId              *int   `json:"evseId,omitempty"`
}

// ClearedChargingLimitResponse represents a ClearedChargingLimit response (CSMS → CS)
type ClearedChargingLimitResponse struct {
}

// =========== Local Authorization List Messages ===========

// GetLocalListVersionRequest represents a GetLocalListVersion request (CSMS → CS)
type GetLocalListVersionRequest struct {
}

// GetLocalListVersionResponse represents a GetLocalListVersion response (CS → CSMS)
type GetLocalListVersionResponse struct {
	VersionNumber int `json:"versionNumber"`
}

// SendLocalListRequest represents a SendLocalList request (CSMS → CS)
type SendLocalListRequest struct {
	VersionNumber          int                     `json:"versionNumber"`
	UpdateType             string                  `json:"updateType"` // Differential, Full
	LocalAuthorizationList []AuthorizationDataType `json:"localAuthorizationList,omitempty"`
}

// AuthorizationDataType represents authorization data for local list
type AuthorizationDataType struct {
	IdToken     IdToken           `json:"idToken"`
	IdTokenInfo *v201.IdTokenInfo `json:"idTokenInfo,omitempty"`
}

// SendLocalListResponse represents a SendLocalList response (CS → CSMS)
type SendLocalListResponse struct {
	Status     string      `json:"status"` // Accepted, Failed, VersionMismatch
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== Firmware Messages ===========

// UpdateFirmwareRequest represents an UpdateFirmware request (CSMS → CS)
type UpdateFirmwareRequest struct {
	RequestId     int          `json:"requestId"`
	Firmware      FirmwareType `json:"firmware"`
	Retries       *int         `json:"retries,omitempty"`
	RetryInterval *int         `json:"retryInterval,omitempty"`
}

// FirmwareType represents firmware information
type FirmwareType struct {
	Location           string  `json:"location"`
	RetrieveDateTime   string  `json:"retrieveDateTime"`
	InstallDateTime    *string `json:"installDateTime,omitempty"`
	SigningCertificate *string `json:"signingCertificate,omitempty"`
	Signature          *string `json:"signature,omitempty"`
}

// UpdateFirmwareResponse represents an UpdateFirmware response (CS → CSMS)
type UpdateFirmwareResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, AcceptedCanceled, InvalidCertificate, RevokedCertificate
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== Network Profile Messages ===========

// SetNetworkProfileRequest represents a SetNetworkProfile request (CSMS → CS)
type SetNetworkProfileRequest struct {
	ConfigurationSlot int                          `json:"configurationSlot"`
	ConnectionData    NetworkConnectionProfileType `json:"connectionData"`
}

// NetworkConnectionProfileType represents network connection profile
type NetworkConnectionProfileType struct {
	OcppVersion     string   `json:"ocppVersion"`   // OCPP12, OCPP15, OCPP16, OCPP20
	OcppTransport   string   `json:"ocppTransport"` // JSON, SOAP
	OcppCsmsUrl     string   `json:"ocppCsmsUrl"`
	MessageTimeout  int      `json:"messageTimeout"`
	SecurityProfile int      `json:"securityProfile"`
	OcppInterface   string   `json:"ocppInterface"` // Wired0, Wired1, Wireless0, etc.
	VPN             *VPNType `json:"vpn,omitempty"`
	APN             *APNType `json:"apn,omitempty"`
}

// VPNType represents VPN configuration
type VPNType struct {
	Server   string  `json:"server"`
	User     *string `json:"user,omitempty"`
	Group    *string `json:"group,omitempty"`
	Password *string `json:"password,omitempty"`
	Key      *string `json:"key,omitempty"`
	Type     string  `json:"type"` // IKEv2, IPSec, L2TP, PPTP
}

// APNType represents APN configuration
type APNType struct {
	APN                     string  `json:"apn"`
	APNUserName             *string `json:"apnUserName,omitempty"`
	APNPassword             *string `json:"apnPassword,omitempty"`
	SimPin                  *int    `json:"simPin,omitempty"`
	PreferredNetwork        *string `json:"preferredNetwork,omitempty"`
	UseOnlyPreferredNetwork bool    `json:"useOnlyPreferredNetwork"`
	APNAuthentication       string  `json:"apnAuthentication"` // CHAP, NONE, PAP, AUTO
}

// SetNetworkProfileResponse represents a SetNetworkProfile response (CS → CSMS)
type SetNetworkProfileResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Failed
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== Log Messages ===========

// GetLogRequest represents a GetLog request (CSMS → CS)
type GetLogRequest struct {
	Log           LogParametersType `json:"log"`
	LogType       string            `json:"logType"` // DiagnosticsLog, SecurityLog
	RequestId     int               `json:"requestId"`
	Retries       *int              `json:"retries,omitempty"`
	RetryInterval *int              `json:"retryInterval,omitempty"`
}

// LogParametersType represents log upload parameters
type LogParametersType struct {
	RemoteLocation  string  `json:"remoteLocation"`
	OldestTimestamp *string `json:"oldestTimestamp,omitempty"`
	LatestTimestamp *string `json:"latestTimestamp,omitempty"`
}

// GetLogResponse represents a GetLog response (CS → CSMS)
type GetLogResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, AcceptedCanceled
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
	Filename   *string     `json:"filename,omitempty"`
}

// LogStatusNotificationRequest represents a LogStatusNotification request (CS → CSMS)
type LogStatusNotificationRequest struct {
	Status    string `json:"status"` // BadMessage, Idle, NotSupportedOperation, PermissionDenied, Uploaded, UploadFailure, Uploading
	RequestId *int   `json:"requestId,omitempty"`
}

// LogStatusNotificationResponse represents a LogStatusNotification response (CSMS → CS)
type LogStatusNotificationResponse struct {
}

// FirmwareStatusNotificationRequest represents a FirmwareStatusNotification request (CS → CSMS)
type FirmwareStatusNotificationRequest struct {
	Status    string `json:"status"`
	RequestId *int   `json:"requestId,omitempty"`
}

// FirmwareStatusNotificationResponse represents a FirmwareStatusNotification response (CSMS → CS)
type FirmwareStatusNotificationResponse struct {
}
