// Package v21 implements OCPP 2.1 protocol types and handling
// OCPP 2.1 extends OCPP 2.0.1 with additional features for cost/tariff,
// display messages, enhanced security, and improved reservations.
package v21

import (
	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v201"
)

// Action represents OCPP 2.1 message actions (extends 2.0.1)
type Action string

// OCPP 2.1 Actions - reuse 2.0.1 actions and add new ones
const (
	// Inherited from 2.0.1 (reexport for convenience)
	ActionHeartbeat                  Action = Action(v201.ActionHeartbeat)
	ActionBootNotification           Action = Action(v201.ActionBootNotification)
	ActionStatusNotification         Action = Action(v201.ActionStatusNotification)
	ActionAuthorize                  Action = Action(v201.ActionAuthorize)
	ActionTransactionEvent           Action = Action(v201.ActionTransactionEvent)
	ActionMeterValues                Action = Action(v201.ActionMeterValues)
	ActionGetVariables               Action = Action(v201.ActionGetVariables)
	ActionSetVariables               Action = Action(v201.ActionSetVariables)
	ActionReset                      Action = Action(v201.ActionReset)
	ActionRequestStartTransaction    Action = Action(v201.ActionRequestStartTransaction)
	ActionRequestStopTransaction     Action = Action(v201.ActionRequestStopTransaction)
	ActionChangeAvailability         Action = Action(v201.ActionChangeAvailability)
	ActionUnlockConnector            Action = Action(v201.ActionUnlockConnector)
	ActionClearCache                 Action = Action(v201.ActionClearCache)
	ActionDataTransfer               Action = Action(v201.ActionDataTransfer)
	ActionTriggerMessage             Action = Action(v201.ActionTriggerMessage)
	ActionGetTransactionStatus       Action = Action(v201.ActionGetTransactionStatus)
	ActionSignCertificate            Action = Action(v201.ActionSignCertificate)
	ActionCertificateSigned          Action = Action(v201.ActionCertificateSigned)
	ActionGet15118EVCertificate      Action = Action(v201.ActionGet15118EVCertificate)
	ActionGetCertificateStatus       Action = Action(v201.ActionGetCertificateStatus)
	ActionDeleteCertificate          Action = Action(v201.ActionDeleteCertificate)
	ActionGetInstalledCertificateIds Action = Action(v201.ActionGetInstalledCertificateIds)
	ActionInstallCertificate         Action = Action(v201.ActionInstallCertificate)
	ActionSecurityEventNotification  Action = Action(v201.ActionSecurityEventNotification)
	ActionNotifyReport               Action = Action(v201.ActionNotifyReport)
	ActionGetBaseReport              Action = Action(v201.ActionGetBaseReport)

	// OCPP 2.1 New Actions - Cost and Tariff
	ActionCostUpdated               Action = "CostUpdated"
	ActionNotifyCustomerInformation Action = "NotifyCustomerInformation"
	ActionCustomerInformation       Action = "CustomerInformation"
	ActionNotifyEVChargingNeeds     Action = "NotifyEVChargingNeeds"
	ActionNotifyEVChargingSchedule  Action = "NotifyEVChargingSchedule"

	// OCPP 2.1 New Actions - Display Messages
	ActionSetDisplayMessage     Action = "SetDisplayMessage"
	ActionGetDisplayMessages    Action = "GetDisplayMessages"
	ActionClearDisplayMessage   Action = "ClearDisplayMessage"
	ActionNotifyDisplayMessages Action = "NotifyDisplayMessages"

	// OCPP 2.1 New Actions - Reservations
	ActionReserveNow        Action = "ReserveNow"
	ActionCancelReservation Action = "CancelReservation"

	// OCPP 2.1 New Actions - Charging Profiles
	ActionSetChargingProfile     Action = "SetChargingProfile"
	ActionGetChargingProfiles    Action = "GetChargingProfiles"
	ActionClearChargingProfile   Action = "ClearChargingProfile"
	ActionReportChargingProfiles Action = "ReportChargingProfiles"
	ActionNotifyChargingLimit    Action = "NotifyChargingLimit"
	ActionGetCompositeSchedule   Action = "GetCompositeSchedule"
	ActionSetNetworkProfile      Action = "SetNetworkProfile"
	ActionGetLocalListVersion    Action = "GetLocalListVersion"
	ActionSendLocalList          Action = "SendLocalList"
	ActionClearedChargingLimit   Action = "ClearedChargingLimit"

	// OCPP 2.1 Additional Actions
	ActionUpdateFirmware             Action = "UpdateFirmware"
	ActionPublishFirmware            Action = "PublishFirmware"
	ActionUnpublishFirmware          Action = "UnpublishFirmware"
	ActionPublishFirmwareStatusNotif Action = "PublishFirmwareStatusNotification"
	ActionGetLog                     Action = "GetLog"
	ActionLogStatusNotification      Action = "LogStatusNotification"
	ActionFirmwareStatusNotification Action = "FirmwareStatusNotification"
)

// ============ Cost and Tariff Types ============

// CostKindType represents the kind of cost
type CostKindType string

const (
	CostKindCarbonDioxideEmission         CostKindType = "CarbonDioxideEmission"
	CostKindRelativePricePercentage       CostKindType = "RelativePricePercentage"
	CostKindRenewableGenerationPercentage CostKindType = "RenewableGenerationPercentage"
)

// CostType represents a cost element
type CostType struct {
	CostKind         CostKindType `json:"costKind"`
	Amount           int          `json:"amount"`
	AmountMultiplier *int         `json:"amountMultiplier,omitempty"`
}

// ConsumptionCostType represents consumption-based costs
type ConsumptionCostType struct {
	StartValue float64    `json:"startValue"`
	Cost       []CostType `json:"cost"`
}

// SalesTariffEntryType represents a sales tariff entry
type SalesTariffEntryType struct {
	RelativeTimeInterval RelativeTimeIntervalType `json:"relativeTimeInterval"`
	EPriceLevel          *int                     `json:"ePriceLevel,omitempty"`
	ConsumptionCost      []ConsumptionCostType    `json:"consumptionCost,omitempty"`
}

// RelativeTimeIntervalType represents a time interval
type RelativeTimeIntervalType struct {
	Start    int  `json:"start"`
	Duration *int `json:"duration,omitempty"`
}

// SalesTariffType represents a complete sales tariff
type SalesTariffType struct {
	ID                     int                    `json:"id"`
	SalesTariffDescription *string                `json:"salesTariffDescription,omitempty"`
	NumEPriceLevels        *int                   `json:"numEPriceLevels,omitempty"`
	SalesTariffEntry       []SalesTariffEntryType `json:"salesTariffEntry"`
}

// ChargingScheduleType represents a charging schedule (enhanced for 2.1)
type ChargingScheduleType struct {
	ID                     int                          `json:"id"`
	StartSchedule          *string                      `json:"startSchedule,omitempty"`
	Duration               *int                         `json:"duration,omitempty"`
	ChargingRateUnit       string                       `json:"chargingRateUnit"` // W or A
	MinChargingRate        *float64                     `json:"minChargingRate,omitempty"`
	ChargingSchedulePeriod []ChargingSchedulePeriodType `json:"chargingSchedulePeriod"`
	SalesTariff            *SalesTariffType             `json:"salesTariff,omitempty"`
}

// ChargingSchedulePeriodType represents a period in a charging schedule
type ChargingSchedulePeriodType struct {
	StartPeriod  int     `json:"startPeriod"`
	Limit        float64 `json:"limit"`
	NumberPhases *int    `json:"numberPhases,omitempty"`
	PhaseToUse   *int    `json:"phaseToUse,omitempty"`
}

// ============ Display Message Types ============

// MessagePriorityType represents message priority
type MessagePriorityType string

const (
	MessagePriorityAlwaysFront MessagePriorityType = "AlwaysFront"
	MessagePriorityInFront     MessagePriorityType = "InFront"
	MessagePriorityNormalCycle MessagePriorityType = "NormalCycle"
)

// MessageStateType represents message state
type MessageStateType string

const (
	MessageStateCharging    MessageStateType = "Charging"
	MessageStateFaulted     MessageStateType = "Faulted"
	MessageStateIdle        MessageStateType = "Idle"
	MessageStateUnavailable MessageStateType = "Unavailable"
)

// MessageFormatType represents message format
type MessageFormatType string

const (
	MessageFormatASCII MessageFormatType = "ASCII"
	MessageFormatHTML  MessageFormatType = "HTML"
	MessageFormatURI   MessageFormatType = "URI"
	MessageFormatUTF8  MessageFormatType = "UTF8"
)

// MessageContentType represents message content
type MessageContentType struct {
	Format   MessageFormatType `json:"format"`
	Language *string           `json:"language,omitempty"`
	Content  string            `json:"content"`
}

// ComponentType represents a component (reuse from 2.0.1)
type ComponentType = v201.Component

// DisplayMessageType represents a display message
type DisplayMessageType struct {
	ID            *int                `json:"id,omitempty"`
	Priority      MessagePriorityType `json:"priority"`
	State         *MessageStateType   `json:"state,omitempty"`
	StartDateTime *string             `json:"startDateTime,omitempty"`
	EndDateTime   *string             `json:"endDateTime,omitempty"`
	TransactionId *string             `json:"transactionId,omitempty"`
	Message       MessageContentType  `json:"message"`
}

// DisplayMessageStatusType represents the status of display message operations
type DisplayMessageStatusType string

const (
	DisplayMessageStatusAccepted             DisplayMessageStatusType = "Accepted"
	DisplayMessageStatusNotSupportedMsgFmt   DisplayMessageStatusType = "NotSupportedMessageFormat"
	DisplayMessageStatusRejected             DisplayMessageStatusType = "Rejected"
	DisplayMessageStatusNotSupportedPriority DisplayMessageStatusType = "NotSupportedPriority"
	DisplayMessageStatusNotSupportedState    DisplayMessageStatusType = "NotSupportedState"
	DisplayMessageStatusUnknownTransaction   DisplayMessageStatusType = "UnknownTransaction"
)

// ClearMessageStatusType represents the status of clear message operations
type ClearMessageStatusType string

const (
	ClearMessageStatusAccepted ClearMessageStatusType = "Accepted"
	ClearMessageStatusUnknown  ClearMessageStatusType = "Unknown"
)

// ============ Reservation Types ============

// ReservationStatusType represents reservation status
type ReservationStatusType string

const (
	ReservationStatusAccepted    ReservationStatusType = "Accepted"
	ReservationStatusFaulted     ReservationStatusType = "Faulted"
	ReservationStatusOccupied    ReservationStatusType = "Occupied"
	ReservationStatusRejected    ReservationStatusType = "Rejected"
	ReservationStatusUnavailable ReservationStatusType = "Unavailable"
)

// CancelReservationStatusType represents cancel reservation status
type CancelReservationStatusType string

const (
	CancelReservationStatusAccepted CancelReservationStatusType = "Accepted"
	CancelReservationStatusRejected CancelReservationStatusType = "Rejected"
)

// ConnectorType represents EV connector types
type ConnectorType string

const (
	ConnectorTypeCCS1        ConnectorType = "cCCS1"
	ConnectorTypeCCS2        ConnectorType = "cCCS2"
	ConnectorTypeCHAdeMO     ConnectorType = "cChaoJi"
	ConnectorTypeG105        ConnectorType = "cG105"
	ConnectorTypeGBT         ConnectorType = "cGBT"
	ConnectorTypeTesla       ConnectorType = "cTesla"
	ConnectorTypeType1       ConnectorType = "cType1"
	ConnectorTypeType2       ConnectorType = "cType2"
	ConnectorTypeS309_1P_16A ConnectorType = "s309-1P-16A"
	ConnectorTypeS309_1P_32A ConnectorType = "s309-1P-32A"
	ConnectorTypeS309_3P_16A ConnectorType = "s309-3P-16A"
	ConnectorTypeS309_3P_32A ConnectorType = "s309-3P-32A"
	ConnectorTypeSBS1P_16A   ConnectorType = "sBS1361"
	ConnectorTypeSCEE_7_7    ConnectorType = "sCEE-7-7"
	ConnectorTypeSOther1Ph   ConnectorType = "sType2"
	ConnectorTypeSOther3Ph   ConnectorType = "sType3"
	ConnectorTypeOther       ConnectorType = "Other"
)

// ============ Charging Profile Types ============

// ChargingProfilePurposeType represents the purpose of a charging profile
type ChargingProfilePurposeType string

const (
	ChargingProfilePurposeChargingStationExternalConstraints ChargingProfilePurposeType = "ChargingStationExternalConstraints"
	ChargingProfilePurposeChargingStationMaxProfile          ChargingProfilePurposeType = "ChargingStationMaxProfile"
	ChargingProfilePurposeTxDefaultProfile                   ChargingProfilePurposeType = "TxDefaultProfile"
	ChargingProfilePurposeTxProfile                          ChargingProfilePurposeType = "TxProfile"
)

// ChargingProfileKindType represents the kind of charging profile
type ChargingProfileKindType string

const (
	ChargingProfileKindAbsolute  ChargingProfileKindType = "Absolute"
	ChargingProfileKindRecurring ChargingProfileKindType = "Recurring"
	ChargingProfileKindRelative  ChargingProfileKindType = "Relative"
)

// RecurrencyKindType represents the recurrency kind
type RecurrencyKindType string

const (
	RecurrencyKindDaily  RecurrencyKindType = "Daily"
	RecurrencyKindWeekly RecurrencyKindType = "Weekly"
)

// ChargingProfileType represents a complete charging profile
type ChargingProfileType struct {
	ID                     int                        `json:"id"`
	StackLevel             int                        `json:"stackLevel"`
	ChargingProfilePurpose ChargingProfilePurposeType `json:"chargingProfilePurpose"`
	ChargingProfileKind    ChargingProfileKindType    `json:"chargingProfileKind"`
	RecurrencyKind         *RecurrencyKindType        `json:"recurrencyKind,omitempty"`
	ValidFrom              *string                    `json:"validFrom,omitempty"`
	ValidTo                *string                    `json:"validTo,omitempty"`
	TransactionId          *string                    `json:"transactionId,omitempty"`
	ChargingSchedule       []ChargingScheduleType     `json:"chargingSchedule"`
}

// ChargingProfileStatusType represents the status of charging profile operations
type ChargingProfileStatusType string

const (
	ChargingProfileStatusAccepted ChargingProfileStatusType = "Accepted"
	ChargingProfileStatusRejected ChargingProfileStatusType = "Rejected"
)

// ClearChargingProfileStatusType represents the status of clear charging profile
type ClearChargingProfileStatusType string

const (
	ClearChargingProfileStatusAccepted ClearChargingProfileStatusType = "Accepted"
	ClearChargingProfileStatusUnknown  ClearChargingProfileStatusType = "Unknown"
)

// ============ EV Charging Needs Types ============

// ACChargingParametersType represents AC charging parameters from EV
type ACChargingParametersType struct {
	EnergyAmount int `json:"energyAmount"`
	EVMinCurrent int `json:"evMinCurrent"`
	EVMaxCurrent int `json:"evMaxCurrent"`
	EVMaxVoltage int `json:"evMaxVoltage"`
}

// DCChargingParametersType represents DC charging parameters from EV
type DCChargingParametersType struct {
	EVMaxCurrent     int  `json:"evMaxCurrent"`
	EVMaxVoltage     int  `json:"evMaxVoltage"`
	EnergyAmount     *int `json:"energyAmount,omitempty"`
	EVMaxPower       *int `json:"evMaxPower,omitempty"`
	StateOfCharge    *int `json:"stateOfCharge,omitempty"`
	EVEnergyCapacity *int `json:"evEnergyCapacity,omitempty"`
	FullSoC          *int `json:"fullSoC,omitempty"`
	BulkSoC          *int `json:"bulkSoC,omitempty"`
}

// ChargingNeedsType represents the EV's charging needs
type ChargingNeedsType struct {
	RequestedEnergyTransfer string                    `json:"requestedEnergyTransfer"` // AC_single_phase, AC_two_phase, AC_three_phase, DC
	DepartureTime           *string                   `json:"departureTime,omitempty"`
	ACChargingParameters    *ACChargingParametersType `json:"acChargingParameters,omitempty"`
	DCChargingParameters    *DCChargingParametersType `json:"dcChargingParameters,omitempty"`
}

// ============ Reexport common types from v201 ============

// Reexport IdToken from v201
type IdToken = v201.IdToken

// Reexport EVSE from v201
type EVSE = v201.EVSE

// Reexport StatusInfo from v201
type StatusInfo = v201.StatusInfo
