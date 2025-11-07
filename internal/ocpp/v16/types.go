package v16

import (
	"time"
)

// Action represents OCPP 1.6 action names
type Action string

const (
	// Core Profile Actions
	ActionAuthorize              Action = "Authorize"
	ActionBootNotification       Action = "BootNotification"
	ActionChangeAvailability     Action = "ChangeAvailability"
	ActionChangeConfiguration    Action = "ChangeConfiguration"
	ActionClearCache             Action = "ClearCache"
	ActionDataTransfer           Action = "DataTransfer"
	ActionGetConfiguration       Action = "GetConfiguration"
	ActionHeartbeat              Action = "Heartbeat"
	ActionMeterValues            Action = "MeterValues"
	ActionRemoteStartTransaction Action = "RemoteStartTransaction"
	ActionRemoteStopTransaction  Action = "RemoteStopTransaction"
	ActionReset                  Action = "Reset"
	ActionStartTransaction       Action = "StartTransaction"
	ActionStatusNotification     Action = "StatusNotification"
	ActionStopTransaction        Action = "StopTransaction"
	ActionUnlockConnector        Action = "UnlockConnector"

	// Firmware Management Profile
	ActionGetDiagnostics                Action = "GetDiagnostics"
	ActionDiagnosticsStatusNotification Action = "DiagnosticsStatusNotification"
	ActionFirmwareStatusNotification    Action = "FirmwareStatusNotification"
	ActionUpdateFirmware                Action = "UpdateFirmware"

	// Smart Charging Profile
	ActionClearChargingProfile Action = "ClearChargingProfile"
	ActionGetCompositeSchedule Action = "GetCompositeSchedule"
	ActionSetChargingProfile   Action = "SetChargingProfile"

	// Remote Trigger Profile
	ActionTriggerMessage Action = "TriggerMessage"

	// Reservation Profile
	ActionReserveNow        Action = "ReserveNow"
	ActionCancelReservation Action = "CancelReservation"
)

// ChargePointStatus represents the status of a charge point connector
type ChargePointStatus string

const (
	ChargePointStatusAvailable     ChargePointStatus = "Available"
	ChargePointStatusPreparing     ChargePointStatus = "Preparing"
	ChargePointStatusCharging      ChargePointStatus = "Charging"
	ChargePointStatusSuspendedEVSE ChargePointStatus = "SuspendedEVSE"
	ChargePointStatusSuspendedEV   ChargePointStatus = "SuspendedEV"
	ChargePointStatusFinishing     ChargePointStatus = "Finishing"
	ChargePointStatusReserved      ChargePointStatus = "Reserved"
	ChargePointStatusUnavailable   ChargePointStatus = "Unavailable"
	ChargePointStatusFaulted       ChargePointStatus = "Faulted"
)

// ChargePointErrorCode represents error codes for charge point status
type ChargePointErrorCode string

const (
	ChargePointErrorNoError              ChargePointErrorCode = "NoError"
	ChargePointErrorConnectorLockFailure ChargePointErrorCode = "ConnectorLockFailure"
	ChargePointErrorEVCommunicationError ChargePointErrorCode = "EVCommunicationError"
	ChargePointErrorGroundFailure        ChargePointErrorCode = "GroundFailure"
	ChargePointErrorHighTemperature      ChargePointErrorCode = "HighTemperature"
	ChargePointErrorInternalError        ChargePointErrorCode = "InternalError"
	ChargePointErrorLocalListConflict    ChargePointErrorCode = "LocalListConflict"
	ChargePointErrorOtherError           ChargePointErrorCode = "OtherError"
	ChargePointErrorOverCurrentFailure   ChargePointErrorCode = "OverCurrentFailure"
	ChargePointErrorPowerMeterFailure    ChargePointErrorCode = "PowerMeterFailure"
	ChargePointErrorPowerSwitchFailure   ChargePointErrorCode = "PowerSwitchFailure"
	ChargePointErrorReaderFailure        ChargePointErrorCode = "ReaderFailure"
	ChargePointErrorResetFailure         ChargePointErrorCode = "ResetFailure"
	ChargePointErrorUnderVoltage         ChargePointErrorCode = "UnderVoltage"
	ChargePointErrorOverVoltage          ChargePointErrorCode = "OverVoltage"
	ChargePointErrorWeakSignal           ChargePointErrorCode = "WeakSignal"
)

// RegistrationStatus represents the registration status from CSMS
type RegistrationStatus string

const (
	RegistrationStatusAccepted RegistrationStatus = "Accepted"
	RegistrationStatusPending  RegistrationStatus = "Pending"
	RegistrationStatusRejected RegistrationStatus = "Rejected"
)

// AuthorizationStatus represents the authorization status
type AuthorizationStatus string

const (
	AuthorizationStatusAccepted     AuthorizationStatus = "Accepted"
	AuthorizationStatusBlocked      AuthorizationStatus = "Blocked"
	AuthorizationStatusExpired      AuthorizationStatus = "Expired"
	AuthorizationStatusInvalid      AuthorizationStatus = "Invalid"
	AuthorizationStatusConcurrentTx AuthorizationStatus = "ConcurrentTx"
)

// Measurand represents the type of value being measured
type Measurand string

const (
	MeasurandCurrentExport                Measurand = "Current.Export"
	MeasurandCurrentImport                Measurand = "Current.Import"
	MeasurandCurrentOffered               Measurand = "Current.Offered"
	MeasurandEnergyActiveExportRegister   Measurand = "Energy.Active.Export.Register"
	MeasurandEnergyActiveImportRegister   Measurand = "Energy.Active.Import.Register"
	MeasurandEnergyReactiveExportRegister Measurand = "Energy.Reactive.Export.Register"
	MeasurandEnergyReactiveImportRegister Measurand = "Energy.Reactive.Import.Register"
	MeasurandEnergyActiveExportInterval   Measurand = "Energy.Active.Export.Interval"
	MeasurandEnergyActiveImportInterval   Measurand = "Energy.Active.Import.Interval"
	MeasurandEnergyReactiveExportInterval Measurand = "Energy.Reactive.Export.Interval"
	MeasurandEnergyReactiveImportInterval Measurand = "Energy.Reactive.Import.Interval"
	MeasurandFrequency                    Measurand = "Frequency"
	MeasurandPowerActiveExport            Measurand = "Power.Active.Export"
	MeasurandPowerActiveImport            Measurand = "Power.Active.Import"
	MeasurandPowerFactor                  Measurand = "Power.Factor"
	MeasurandPowerOffered                 Measurand = "Power.Offered"
	MeasurandPowerReactiveExport          Measurand = "Power.Reactive.Export"
	MeasurandPowerReactiveImport          Measurand = "Power.Reactive.Import"
	MeasurandRPM                          Measurand = "RPM"
	MeasurandSoC                          Measurand = "SoC"
	MeasurandTemperature                  Measurand = "Temperature"
	MeasurandVoltage                      Measurand = "Voltage"
)

// ReadingContext represents the context of a meter value reading
type ReadingContext string

const (
	ReadingContextInterruptionBegin ReadingContext = "Interruption.Begin"
	ReadingContextInterruptionEnd   ReadingContext = "Interruption.End"
	ReadingContextOther             ReadingContext = "Other"
	ReadingContextSampleClock       ReadingContext = "Sample.Clock"
	ReadingContextSamplePeriodic    ReadingContext = "Sample.Periodic"
	ReadingContextTransactionBegin  ReadingContext = "Transaction.Begin"
	ReadingContextTransactionEnd    ReadingContext = "Transaction.End"
	ReadingContextTrigger           ReadingContext = "Trigger"
)

// Location represents the location of a measurement
type Location string

const (
	LocationBody   Location = "Body"
	LocationCable  Location = "Cable"
	LocationEV     Location = "EV"
	LocationInlet  Location = "Inlet"
	LocationOutlet Location = "Outlet"
)

// UnitOfMeasure represents the unit of measure
type UnitOfMeasure string

const (
	UnitOfMeasureWh         UnitOfMeasure = "Wh"
	UnitOfMeasureKWh        UnitOfMeasure = "kWh"
	UnitOfMeasureVarh       UnitOfMeasure = "varh"
	UnitOfMeasureKvarh      UnitOfMeasure = "kvarh"
	UnitOfMeasureW          UnitOfMeasure = "W"
	UnitOfMeasureKW         UnitOfMeasure = "kW"
	UnitOfMeasureVA         UnitOfMeasure = "VA"
	UnitOfMeasureKVA        UnitOfMeasure = "kVA"
	UnitOfMeasureVar        UnitOfMeasure = "var"
	UnitOfMeasureKvar       UnitOfMeasure = "kvar"
	UnitOfMeasureA          UnitOfMeasure = "A"
	UnitOfMeasureV          UnitOfMeasure = "V"
	UnitOfMeasureCelsius    UnitOfMeasure = "Celsius"
	UnitOfMeasureFahrenheit UnitOfMeasure = "Fahrenheit"
	UnitOfMeasureK          UnitOfMeasure = "K"
	UnitOfMeasurePercent    UnitOfMeasure = "Percent"
)

// Reason represents the reason for stopping a transaction
type Reason string

const (
	ReasonEmergencyStop  Reason = "EmergencyStop"
	ReasonEVDisconnected Reason = "EVDisconnected"
	ReasonHardReset      Reason = "HardReset"
	ReasonLocal          Reason = "Local"
	ReasonOther          Reason = "Other"
	ReasonPowerLoss      Reason = "PowerLoss"
	ReasonReboot         Reason = "Reboot"
	ReasonRemote         Reason = "Remote"
	ReasonSoftReset      Reason = "SoftReset"
	ReasonUnlockCommand  Reason = "UnlockCommand"
	ReasonDeAuthorized   Reason = "DeAuthorized"
)

// DateTime is a custom type for OCPP date-time format
type DateTime struct {
	time.Time
}

// MarshalJSON implements custom JSON marshaling for DateTime
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + dt.Time.Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON implements custom JSON unmarshaling for DateTime
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := string(data[1 : len(data)-1])

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	dt.Time = t
	return nil
}

// IdTagInfo contains information about an ID tag
type IdTagInfo struct {
	ExpiryDate  *DateTime           `json:"expiryDate,omitempty"`
	ParentIdTag string              `json:"parentIdTag,omitempty"`
	Status      AuthorizationStatus `json:"status"`
}

// SampledValue represents a single sampled value in a meter values reading
type SampledValue struct {
	Value     string         `json:"value"`
	Context   ReadingContext `json:"context,omitempty"`
	Format    string         `json:"format,omitempty"` // Raw or SignedData
	Measurand Measurand      `json:"measurand,omitempty"`
	Phase     string         `json:"phase,omitempty"` // L1, L2, L3, N, L1-N, L2-N, L3-N, L1-L2, L2-L3, L3-L1
	Location  Location       `json:"location,omitempty"`
	Unit      UnitOfMeasure  `json:"unit,omitempty"`
}

// MeterValue represents a collection of meter value samples
type MeterValue struct {
	Timestamp    DateTime       `json:"timestamp"`
	SampledValue []SampledValue `json:"sampledValue"`
}
