// Package v201 implements OCPP 2.0.1 protocol messages and handling
package v201

import (
	"time"
)

// Action represents OCPP 2.0.1 action names
type Action string

const (
	// Provisioning
	ActionBootNotification   Action = "BootNotification"
	ActionHeartbeat          Action = "Heartbeat"
	ActionStatusNotification Action = "StatusNotification"
	ActionNotifyReport       Action = "NotifyReport"
	ActionGetBaseReport      Action = "GetBaseReport"
	ActionSetVariables       Action = "SetVariables"
	ActionGetVariables       Action = "GetVariables"
	ActionNotifyEvent        Action = "NotifyEvent"
	ActionReset              Action = "Reset"
	ActionSetNetworkProfile  Action = "SetNetworkProfile"
	ActionChangeAvailability Action = "ChangeAvailability"
	ActionClearCache         Action = "ClearCache"

	// Authorization
	ActionAuthorize              Action = "Authorize"
	ActionClearAuthorizationData Action = "ClearAuthorizationData"
	ActionGetLocalListVersion    Action = "GetLocalListVersion"
	ActionSendLocalList          Action = "SendLocalList"

	// Transaction
	ActionTransactionEvent        Action = "TransactionEvent"
	ActionRequestStartTransaction Action = "RequestStartTransaction"
	ActionRequestStopTransaction  Action = "RequestStopTransaction"
	ActionGetTransactionStatus    Action = "GetTransactionStatus"

	// Metering
	ActionMeterValues Action = "MeterValues"

	// Smart Charging
	ActionClearChargingProfile   Action = "ClearChargingProfile"
	ActionGetCompositeSchedule   Action = "GetCompositeSchedule"
	ActionSetChargingProfile     Action = "SetChargingProfile"
	ActionNotifyChargingLimit    Action = "NotifyChargingLimit"
	ActionReportChargingProfiles Action = "ReportChargingProfiles"

	// Remote Control
	ActionUnlockConnector Action = "UnlockConnector"
	ActionTriggerMessage  Action = "TriggerMessage"

	// Firmware
	ActionFirmwareStatusNotification Action = "FirmwareStatusNotification"
	ActionUpdateFirmware             Action = "UpdateFirmware"

	// Diagnostics
	ActionLogStatusNotification   Action = "LogStatusNotification"
	ActionGetLog                  Action = "GetLog"
	ActionClearVariableMonitoring Action = "ClearVariableMonitoring"
	ActionSetVariableMonitoring   Action = "SetVariableMonitoring"
	ActionSetMonitoringBase       Action = "SetMonitoringBase"
	ActionSetMonitoringLevel      Action = "SetMonitoringLevel"
	ActionNotifyMonitoringReport  Action = "NotifyMonitoringReport"

	// Security
	ActionSecurityEventNotification  Action = "SecurityEventNotification"
	ActionSignCertificate            Action = "SignCertificate"
	ActionCertificateSigned          Action = "CertificateSigned"
	ActionGet15118EVCertificate      Action = "Get15118EVCertificate"
	ActionGetCertificateStatus       Action = "GetCertificateStatus"
	ActionDeleteCertificate          Action = "DeleteCertificate"
	ActionGetInstalledCertificateIds Action = "GetInstalledCertificateIds"
	ActionInstallCertificate         Action = "InstallCertificate"

	// Data Transfer
	ActionDataTransfer Action = "DataTransfer"

	// Display Messages
	ActionNotifyDisplayMessages Action = "NotifyDisplayMessages"
	ActionSetDisplayMessage     Action = "SetDisplayMessage"
	ActionGetDisplayMessages    Action = "GetDisplayMessages"
	ActionClearDisplayMessage   Action = "ClearDisplayMessage"

	// Reservation
	ActionReserveNow        Action = "ReserveNow"
	ActionCancelReservation Action = "CancelReservation"
)

// ========== Enums ==========

// RegistrationStatusType represents registration status
type RegistrationStatusType string

const (
	RegistrationStatusAccepted RegistrationStatusType = "Accepted"
	RegistrationStatusPending  RegistrationStatusType = "Pending"
	RegistrationStatusRejected RegistrationStatusType = "Rejected"
)

// BootReasonType represents the reason for sending a BootNotification
type BootReasonType string

const (
	BootReasonApplicationReset BootReasonType = "ApplicationReset"
	BootReasonFirmwareUpdate   BootReasonType = "FirmwareUpdate"
	BootReasonLocalReset       BootReasonType = "LocalReset"
	BootReasonPowerUp          BootReasonType = "PowerUp"
	BootReasonRemoteReset      BootReasonType = "RemoteReset"
	BootReasonScheduledReset   BootReasonType = "ScheduledReset"
	BootReasonTriggered        BootReasonType = "Triggered"
	BootReasonUnknown          BootReasonType = "Unknown"
	BootReasonWatchdog         BootReasonType = "Watchdog"
)

// ConnectorStatusType represents the status of a connector
type ConnectorStatusType string

const (
	ConnectorStatusAvailable   ConnectorStatusType = "Available"
	ConnectorStatusOccupied    ConnectorStatusType = "Occupied"
	ConnectorStatusReserved    ConnectorStatusType = "Reserved"
	ConnectorStatusUnavailable ConnectorStatusType = "Unavailable"
	ConnectorStatusFaulted     ConnectorStatusType = "Faulted"
)

// AuthorizationStatusType represents authorization status
type AuthorizationStatusType string

const (
	AuthorizationStatusAccepted           AuthorizationStatusType = "Accepted"
	AuthorizationStatusBlocked            AuthorizationStatusType = "Blocked"
	AuthorizationStatusConcurrentTx       AuthorizationStatusType = "ConcurrentTx"
	AuthorizationStatusExpired            AuthorizationStatusType = "Expired"
	AuthorizationStatusInvalid            AuthorizationStatusType = "Invalid"
	AuthorizationStatusNoCredit           AuthorizationStatusType = "NoCredit"
	AuthorizationStatusNotAllowedTypeEVSE AuthorizationStatusType = "NotAllowedTypeEVSE"
	AuthorizationStatusNotAtThisLocation  AuthorizationStatusType = "NotAtThisLocation"
	AuthorizationStatusNotAtThisTime      AuthorizationStatusType = "NotAtThisTime"
	AuthorizationStatusUnknown            AuthorizationStatusType = "Unknown"
)

// IdTokenType represents the type of identifier token
type IdTokenType string

const (
	IdTokenTypeCentral         IdTokenType = "Central"
	IdTokenTypeEMAID           IdTokenType = "eMAID"
	IdTokenTypeISO14443        IdTokenType = "ISO14443"
	IdTokenTypeISO15693        IdTokenType = "ISO15693"
	IdTokenTypeKeyCode         IdTokenType = "KeyCode"
	IdTokenTypeLocal           IdTokenType = "Local"
	IdTokenTypeMacAddress      IdTokenType = "MacAddress"
	IdTokenTypeNoAuthorization IdTokenType = "NoAuthorization"
)

// TransactionEventType represents the type of transaction event
type TransactionEventType string

const (
	TransactionEventEnded   TransactionEventType = "Ended"
	TransactionEventStarted TransactionEventType = "Started"
	TransactionEventUpdated TransactionEventType = "Updated"
)

// TriggerReasonType represents the reason a transaction event was triggered
type TriggerReasonType string

const (
	TriggerReasonAuthorized           TriggerReasonType = "Authorized"
	TriggerReasonCablePluggedIn       TriggerReasonType = "CablePluggedIn"
	TriggerReasonChargingRateChanged  TriggerReasonType = "ChargingRateChanged"
	TriggerReasonChargingStateChanged TriggerReasonType = "ChargingStateChanged"
	TriggerReasonDeauthorized         TriggerReasonType = "Deauthorized"
	TriggerReasonEnergyLimitReached   TriggerReasonType = "EnergyLimitReached"
	TriggerReasonEVCommunicationLost  TriggerReasonType = "EVCommunicationLost"
	TriggerReasonEVConnectTimeout     TriggerReasonType = "EVConnectTimeout"
	TriggerReasonEVDeparted           TriggerReasonType = "EVDeparted"
	TriggerReasonEVDetected           TriggerReasonType = "EVDetected"
	TriggerReasonMeterValueClock      TriggerReasonType = "MeterValueClock"
	TriggerReasonMeterValuePeriodic   TriggerReasonType = "MeterValuePeriodic"
	TriggerReasonRemoteStart          TriggerReasonType = "RemoteStart"
	TriggerReasonRemoteStop           TriggerReasonType = "RemoteStop"
	TriggerReasonResetCommand         TriggerReasonType = "ResetCommand"
	TriggerReasonSignedDataReceived   TriggerReasonType = "SignedDataReceived"
	TriggerReasonStopAuthorized       TriggerReasonType = "StopAuthorized"
	TriggerReasonTimeLimitReached     TriggerReasonType = "TimeLimitReached"
	TriggerReasonTrigger              TriggerReasonType = "Trigger"
	TriggerReasonUnlockCommand        TriggerReasonType = "UnlockCommand"
	TriggerReasonAbnormalCondition    TriggerReasonType = "AbnormalCondition"
)

// ChargingStateType represents the state of the charging process
type ChargingStateType string

const (
	ChargingStateCharging      ChargingStateType = "Charging"
	ChargingStateEVConnected   ChargingStateType = "EVConnected"
	ChargingStateIdle          ChargingStateType = "Idle"
	ChargingStateSuspendedEV   ChargingStateType = "SuspendedEV"
	ChargingStateSuspendedEVSE ChargingStateType = "SuspendedEVSE"
)

// ReasonType represents the reason for stopping a transaction
type ReasonType string

const (
	ReasonDeAuthorized       ReasonType = "DeAuthorized"
	ReasonEmergencyStop      ReasonType = "EmergencyStop"
	ReasonEnergyLimitReached ReasonType = "EnergyLimitReached"
	ReasonEVDisconnected     ReasonType = "EVDisconnected"
	ReasonGroundFault        ReasonType = "GroundFault"
	ReasonImmediateReset     ReasonType = "ImmediateReset"
	ReasonLocal              ReasonType = "Local"
	ReasonLocalOutOfCredit   ReasonType = "LocalOutOfCredit"
	ReasonMasterPass         ReasonType = "MasterPass"
	ReasonOther              ReasonType = "Other"
	ReasonOvercurrentFault   ReasonType = "OvercurrentFault"
	ReasonPowerLoss          ReasonType = "PowerLoss"
	ReasonPowerQuality       ReasonType = "PowerQuality"
	ReasonReboot             ReasonType = "Reboot"
	ReasonRemote             ReasonType = "Remote"
	ReasonSOCLimitReached    ReasonType = "SOCLimitReached"
	ReasonStoppedByEV        ReasonType = "StoppedByEV"
	ReasonTimeLimitReached   ReasonType = "TimeLimitReached"
	ReasonTimeout            ReasonType = "Timeout"
)

// MeasurandType represents the type of measured value
type MeasurandType string

const (
	MeasurandCurrentExport                MeasurandType = "Current.Export"
	MeasurandCurrentImport                MeasurandType = "Current.Import"
	MeasurandCurrentOffered               MeasurandType = "Current.Offered"
	MeasurandEnergyActiveExportRegister   MeasurandType = "Energy.Active.Export.Register"
	MeasurandEnergyActiveImportRegister   MeasurandType = "Energy.Active.Import.Register"
	MeasurandEnergyReactiveExportRegister MeasurandType = "Energy.Reactive.Export.Register"
	MeasurandEnergyReactiveImportRegister MeasurandType = "Energy.Reactive.Import.Register"
	MeasurandEnergyActiveExportInterval   MeasurandType = "Energy.Active.Export.Interval"
	MeasurandEnergyActiveImportInterval   MeasurandType = "Energy.Active.Import.Interval"
	MeasurandEnergyReactiveExportInterval MeasurandType = "Energy.Reactive.Export.Interval"
	MeasurandEnergyReactiveImportInterval MeasurandType = "Energy.Reactive.Import.Interval"
	MeasurandEnergyActiveNet              MeasurandType = "Energy.Active.Net"
	MeasurandEnergyReactiveNet            MeasurandType = "Energy.Reactive.Net"
	MeasurandEnergyApparentNet            MeasurandType = "Energy.Apparent.Net"
	MeasurandEnergyApparentImport         MeasurandType = "Energy.Apparent.Import"
	MeasurandEnergyApparentExport         MeasurandType = "Energy.Apparent.Export"
	MeasurandFrequency                    MeasurandType = "Frequency"
	MeasurandPowerActiveExport            MeasurandType = "Power.Active.Export"
	MeasurandPowerActiveImport            MeasurandType = "Power.Active.Import"
	MeasurandPowerFactor                  MeasurandType = "Power.Factor"
	MeasurandPowerOffered                 MeasurandType = "Power.Offered"
	MeasurandPowerReactiveExport          MeasurandType = "Power.Reactive.Export"
	MeasurandPowerReactiveImport          MeasurandType = "Power.Reactive.Import"
	MeasurandSoC                          MeasurandType = "SoC"
	MeasurandVoltage                      MeasurandType = "Voltage"
)

// ReadingContextType represents the context of a meter reading
type ReadingContextType string

const (
	ReadingContextInterruptionBegin ReadingContextType = "Interruption.Begin"
	ReadingContextInterruptionEnd   ReadingContextType = "Interruption.End"
	ReadingContextOther             ReadingContextType = "Other"
	ReadingContextSampleClock       ReadingContextType = "Sample.Clock"
	ReadingContextSamplePeriodic    ReadingContextType = "Sample.Periodic"
	ReadingContextTransactionBegin  ReadingContextType = "Transaction.Begin"
	ReadingContextTransactionEnd    ReadingContextType = "Transaction.End"
	ReadingContextTrigger           ReadingContextType = "Trigger"
)

// LocationType represents the location of measurement
type LocationType string

const (
	LocationBody   LocationType = "Body"
	LocationCable  LocationType = "Cable"
	LocationEV     LocationType = "EV"
	LocationInlet  LocationType = "Inlet"
	LocationOutlet LocationType = "Outlet"
)

// PhaseType represents the phase(s) of measurement
type PhaseType string

const (
	PhaseL1   PhaseType = "L1"
	PhaseL2   PhaseType = "L2"
	PhaseL3   PhaseType = "L3"
	PhaseN    PhaseType = "N"
	PhaseL1N  PhaseType = "L1-N"
	PhaseL2N  PhaseType = "L2-N"
	PhaseL3N  PhaseType = "L3-N"
	PhaseL1L2 PhaseType = "L1-L2"
	PhaseL2L3 PhaseType = "L2-L3"
	PhaseL3L1 PhaseType = "L3-L1"
)

// ResetType represents the type of reset
type ResetType string

const (
	ResetImmediate ResetType = "Immediate"
	ResetOnIdle    ResetType = "OnIdle"
)

// ResetStatusType represents the result of a reset request
type ResetStatusType string

const (
	ResetStatusAccepted  ResetStatusType = "Accepted"
	ResetStatusRejected  ResetStatusType = "Rejected"
	ResetStatusScheduled ResetStatusType = "Scheduled"
)

// AttributeType represents the type of variable attribute
type AttributeType string

const (
	AttributeActual AttributeType = "Actual"
	AttributeTarget AttributeType = "Target"
	AttributeMinSet AttributeType = "MinSet"
	AttributeMaxSet AttributeType = "MaxSet"
)

// SetVariableStatusType represents the result of a SetVariables request
type SetVariableStatusType string

const (
	SetVariableStatusAccepted                  SetVariableStatusType = "Accepted"
	SetVariableStatusRejected                  SetVariableStatusType = "Rejected"
	SetVariableStatusUnknownComponent          SetVariableStatusType = "UnknownComponent"
	SetVariableStatusUnknownVariable           SetVariableStatusType = "UnknownVariable"
	SetVariableStatusNotSupportedAttributeType SetVariableStatusType = "NotSupportedAttributeType"
	SetVariableStatusRebootRequired            SetVariableStatusType = "RebootRequired"
)

// GetVariableStatusType represents the result of a GetVariables request
type GetVariableStatusType string

const (
	GetVariableStatusAccepted                  GetVariableStatusType = "Accepted"
	GetVariableStatusRejected                  GetVariableStatusType = "Rejected"
	GetVariableStatusUnknownComponent          GetVariableStatusType = "UnknownComponent"
	GetVariableStatusUnknownVariable           GetVariableStatusType = "UnknownVariable"
	GetVariableStatusNotSupportedAttributeType GetVariableStatusType = "NotSupportedAttributeType"
)

// DataTransferStatusType represents the status of a data transfer
type DataTransferStatusType string

const (
	DataTransferStatusAccepted         DataTransferStatusType = "Accepted"
	DataTransferStatusRejected         DataTransferStatusType = "Rejected"
	DataTransferStatusUnknownMessageId DataTransferStatusType = "UnknownMessageId"
	DataTransferStatusUnknownVendorId  DataTransferStatusType = "UnknownVendorId"
)

// ========== Data Types ==========

// DateTime wraps time.Time for OCPP date-time format
type DateTime struct {
	time.Time
}

// MarshalJSON implements custom JSON marshaling
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + dt.Time.Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON implements custom JSON unmarshaling
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	str := string(data[1 : len(data)-1])
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}
	dt.Time = t
	return nil
}

// StatusInfo provides additional status information
type StatusInfo struct {
	ReasonCode     string `json:"reasonCode"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
}

// EVSE represents an Electric Vehicle Supply Equipment
type EVSE struct {
	ID          int  `json:"id"`
	ConnectorId *int `json:"connectorId,omitempty"`
}

// ChargingStation represents the charging station identification
type ChargingStation struct {
	SerialNumber    string `json:"serialNumber,omitempty"`
	Model           string `json:"model"`
	VendorName      string `json:"vendorName"`
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
	Modem           *Modem `json:"modem,omitempty"`
}

// Modem represents modem information
type Modem struct {
	ICCID string `json:"iccid,omitempty"`
	IMSI  string `json:"imsi,omitempty"`
}

// IdToken represents an identifier token
type IdToken struct {
	IdToken        string           `json:"idToken"`
	Type           IdTokenType      `json:"type"`
	AdditionalInfo []AdditionalInfo `json:"additionalInfo,omitempty"`
}

// AdditionalInfo represents additional information for an IdToken
type AdditionalInfo struct {
	AdditionalIdToken string `json:"additionalIdToken"`
	Type              string `json:"type"`
}

// IdTokenInfo represents information about an IdToken authorization
type IdTokenInfo struct {
	Status              AuthorizationStatusType `json:"status"`
	CacheExpiryDateTime *DateTime               `json:"cacheExpiryDateTime,omitempty"`
	ChargingPriority    *int                    `json:"chargingPriority,omitempty"`
	Language1           string                  `json:"language1,omitempty"`
	Language2           string                  `json:"language2,omitempty"`
	GroupIdToken        *IdToken                `json:"groupIdToken,omitempty"`
	PersonalMessage     *MessageContent         `json:"personalMessage,omitempty"`
	EvseId              []int                   `json:"evseId,omitempty"`
}

// MessageContent represents a display message
type MessageContent struct {
	Format   string `json:"format"` // ASCII, HTML, URI, UTF8
	Language string `json:"language,omitempty"`
	Content  string `json:"content"`
}

// Transaction represents a transaction
type Transaction struct {
	TransactionId     string             `json:"transactionId"`
	ChargingState     *ChargingStateType `json:"chargingState,omitempty"`
	TimeSpentCharging *int               `json:"timeSpentCharging,omitempty"`
	StoppedReason     *ReasonType        `json:"stoppedReason,omitempty"`
	RemoteStartId     *int               `json:"remoteStartId,omitempty"`
}

// Component represents a device model component
type Component struct {
	Name     string `json:"name"`
	Instance string `json:"instance,omitempty"`
	EVSE     *EVSE  `json:"evse,omitempty"`
}

// Variable represents a device model variable
type Variable struct {
	Name     string `json:"name"`
	Instance string `json:"instance,omitempty"`
}

// SetVariableData represents data for setting a variable
type SetVariableData struct {
	AttributeType  *AttributeType `json:"attributeType,omitempty"`
	AttributeValue string         `json:"attributeValue"`
	Component      Component      `json:"component"`
	Variable       Variable       `json:"variable"`
}

// SetVariableResult represents the result of setting a variable
type SetVariableResult struct {
	AttributeType   *AttributeType        `json:"attributeType,omitempty"`
	AttributeStatus SetVariableStatusType `json:"attributeStatus"`
	Component       Component             `json:"component"`
	Variable        Variable              `json:"variable"`
	StatusInfo      *StatusInfo           `json:"statusInfo,omitempty"`
}

// GetVariableData represents data for getting a variable
type GetVariableData struct {
	AttributeType *AttributeType `json:"attributeType,omitempty"`
	Component     Component      `json:"component"`
	Variable      Variable       `json:"variable"`
}

// GetVariableResult represents the result of getting a variable
type GetVariableResult struct {
	AttributeType   *AttributeType        `json:"attributeType,omitempty"`
	AttributeStatus GetVariableStatusType `json:"attributeStatus"`
	AttributeValue  string                `json:"attributeValue,omitempty"`
	Component       Component             `json:"component"`
	Variable        Variable              `json:"variable"`
	StatusInfo      *StatusInfo           `json:"statusInfo,omitempty"`
}

// MeterValue represents a meter value sample
type MeterValue struct {
	Timestamp    DateTime       `json:"timestamp"`
	SampledValue []SampledValue `json:"sampledValue"`
}

// SampledValue represents a single sampled measurement value
type SampledValue struct {
	Value            float64             `json:"value"`
	Context          *ReadingContextType `json:"context,omitempty"`
	Measurand        *MeasurandType      `json:"measurand,omitempty"`
	Phase            *PhaseType          `json:"phase,omitempty"`
	Location         *LocationType       `json:"location,omitempty"`
	SignedMeterValue *SignedMeterValue   `json:"signedMeterValue,omitempty"`
	UnitOfMeasure    *UnitOfMeasure      `json:"unitOfMeasure,omitempty"`
}

// SignedMeterValue represents a signed meter value
type SignedMeterValue struct {
	SignedMeterData string `json:"signedMeterData"`
	SigningMethod   string `json:"signingMethod"`
	EncodingMethod  string `json:"encodingMethod"`
	PublicKey       string `json:"publicKey"`
}

// UnitOfMeasure represents a unit of measure
type UnitOfMeasure struct {
	Unit       string `json:"unit,omitempty"` // Wh, kWh, varh, kvarh, W, kW, VA, kVA, var, kvar, A, V, K, Celsius, Fahrenheit, Percent
	Multiplier *int   `json:"multiplier,omitempty"`
}
