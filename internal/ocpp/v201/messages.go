package v201

// =========== BootNotification ===========

// BootNotificationRequest represents a BootNotification request (CS → CSMS)
type BootNotificationRequest struct {
	ChargingStation ChargingStation `json:"chargingStation"`
	Reason          BootReasonType  `json:"reason"`
}

// BootNotificationResponse represents a BootNotification response (CSMS → CS)
type BootNotificationResponse struct {
	CurrentTime DateTime               `json:"currentTime"`
	Interval    int                    `json:"interval"` // Heartbeat interval in seconds
	Status      RegistrationStatusType `json:"status"`
	StatusInfo  *StatusInfo            `json:"statusInfo,omitempty"`
}

// =========== Heartbeat ===========

// HeartbeatRequest represents a Heartbeat request (CS → CSMS)
type HeartbeatRequest struct {
	// Empty payload
}

// HeartbeatResponse represents a Heartbeat response (CSMS → CS)
type HeartbeatResponse struct {
	CurrentTime DateTime `json:"currentTime"`
}

// =========== StatusNotification ===========

// StatusNotificationRequest represents a StatusNotification request (CS → CSMS)
type StatusNotificationRequest struct {
	Timestamp       DateTime            `json:"timestamp"`
	ConnectorStatus ConnectorStatusType `json:"connectorStatus"`
	EvseId          int                 `json:"evseId"`
	ConnectorId     int                 `json:"connectorId"`
}

// StatusNotificationResponse represents a StatusNotification response (CSMS → CS)
type StatusNotificationResponse struct {
	// Empty payload
}

// =========== Authorize ===========

// AuthorizeRequest represents an Authorize request (CS → CSMS)
type AuthorizeRequest struct {
	IdToken                     IdToken           `json:"idToken"`
	Certificate                 string            `json:"certificate,omitempty"`
	Iso15118CertificateHashData []OCSPRequestData `json:"iso15118CertificateHashData,omitempty"`
}

// OCSPRequestData represents OCSP request data for certificate validation
type OCSPRequestData struct {
	HashAlgorithm  string `json:"hashAlgorithm"` // SHA256, SHA384, SHA512
	IssuerNameHash string `json:"issuerNameHash"`
	IssuerKeyHash  string `json:"issuerKeyHash"`
	SerialNumber   string `json:"serialNumber"`
	ResponderURL   string `json:"responderURL"`
}

// AuthorizeResponse represents an Authorize response (CSMS → CS)
type AuthorizeResponse struct {
	IdTokenInfo       IdTokenInfo `json:"idTokenInfo"`
	CertificateStatus *string     `json:"certificateStatus,omitempty"` // Accepted, SignatureError, CertificateExpired, etc.
}

// =========== TransactionEvent ===========

// TransactionEventRequest represents a TransactionEvent request (CS → CSMS)
type TransactionEventRequest struct {
	EventType          TransactionEventType `json:"eventType"`
	Timestamp          DateTime             `json:"timestamp"`
	TriggerReason      TriggerReasonType    `json:"triggerReason"`
	SeqNo              int                  `json:"seqNo"`
	TransactionInfo    Transaction          `json:"transactionInfo"`
	Offline            *bool                `json:"offline,omitempty"`
	NumberOfPhasesUsed *int                 `json:"numberOfPhasesUsed,omitempty"`
	CableMaxCurrent    *float64             `json:"cableMaxCurrent,omitempty"`
	ReservationId      *int                 `json:"reservationId,omitempty"`
	EVSE               *EVSE                `json:"evse,omitempty"`
	IdToken            *IdToken             `json:"idToken,omitempty"`
	MeterValue         []MeterValue         `json:"meterValue,omitempty"`
}

// TransactionEventResponse represents a TransactionEvent response (CSMS → CS)
type TransactionEventResponse struct {
	TotalCost              *float64        `json:"totalCost,omitempty"`
	ChargingPriority       *int            `json:"chargingPriority,omitempty"`
	IdTokenInfo            *IdTokenInfo    `json:"idTokenInfo,omitempty"`
	UpdatedPersonalMessage *MessageContent `json:"updatedPersonalMessage,omitempty"`
}

// =========== MeterValues ===========

// MeterValuesRequest represents a MeterValues request (CS → CSMS)
type MeterValuesRequest struct {
	EvseId     int          `json:"evseId"`
	MeterValue []MeterValue `json:"meterValue"`
}

// MeterValuesResponse represents a MeterValues response (CSMS → CS)
type MeterValuesResponse struct {
	// Empty payload
}

// =========== RequestStartTransaction ===========

// RequestStartTransactionRequest represents a RequestStartTransaction request (CSMS → CS)
type RequestStartTransactionRequest struct {
	IdToken         IdToken          `json:"idToken"`
	RemoteStartId   int              `json:"remoteStartId"`
	EvseId          *int             `json:"evseId,omitempty"`
	GroupIdToken    *IdToken         `json:"groupIdToken,omitempty"`
	ChargingProfile *ChargingProfile `json:"chargingProfile,omitempty"`
}

// ChargingProfile represents a charging profile
type ChargingProfile struct {
	Id                     int                `json:"id"`
	StackLevel             int                `json:"stackLevel"`
	ChargingProfilePurpose string             `json:"chargingProfilePurpose"`   // ChargePointMaxProfile, TxDefaultProfile, TxProfile
	ChargingProfileKind    string             `json:"chargingProfileKind"`      // Absolute, Recurring, Relative
	RecurrencyKind         string             `json:"recurrencyKind,omitempty"` // Daily, Weekly
	ValidFrom              *DateTime          `json:"validFrom,omitempty"`
	ValidTo                *DateTime          `json:"validTo,omitempty"`
	ChargingSchedule       []ChargingSchedule `json:"chargingSchedule"`
	TransactionId          string             `json:"transactionId,omitempty"`
}

// ChargingSchedule represents a charging schedule
type ChargingSchedule struct {
	Id                     int                      `json:"id"`
	StartSchedule          *DateTime                `json:"startSchedule,omitempty"`
	Duration               *int                     `json:"duration,omitempty"`
	ChargingRateUnit       string                   `json:"chargingRateUnit"` // W, A
	MinChargingRate        *float64                 `json:"minChargingRate,omitempty"`
	ChargingSchedulePeriod []ChargingSchedulePeriod `json:"chargingSchedulePeriod"`
	SalesTariff            *SalesTariff             `json:"salesTariff,omitempty"`
}

// ChargingSchedulePeriod represents a period in a charging schedule
type ChargingSchedulePeriod struct {
	StartPeriod  int     `json:"startPeriod"`
	Limit        float64 `json:"limit"`
	NumberPhases *int    `json:"numberPhases,omitempty"`
	PhaseToUse   *int    `json:"phaseToUse,omitempty"`
}

// SalesTariff represents tariff information
type SalesTariff struct {
	Id                     int                `json:"id"`
	SalesTariffDescription string             `json:"salesTariffDescription,omitempty"`
	NumEPriceLevels        *int               `json:"numEPriceLevels,omitempty"`
	SalesTariffEntry       []SalesTariffEntry `json:"salesTariffEntry"`
}

// SalesTariffEntry represents a tariff entry
type SalesTariffEntry struct {
	EPriceLevel          *int                 `json:"ePriceLevel,omitempty"`
	RelativeTimeInterval RelativeTimeInterval `json:"relativeTimeInterval"`
	ConsumptionCost      []ConsumptionCost    `json:"consumptionCost,omitempty"`
}

// RelativeTimeInterval represents a time interval
type RelativeTimeInterval struct {
	Start    int  `json:"start"`
	Duration *int `json:"duration,omitempty"`
}

// ConsumptionCost represents consumption cost
type ConsumptionCost struct {
	StartValue float64 `json:"startValue"`
	Cost       []Cost  `json:"cost"`
}

// Cost represents a cost element
type Cost struct {
	CostKind         string `json:"costKind"` // CarbonDioxideEmission, RelativePricePercentage, RenewableGenerationPercentage
	Amount           int    `json:"amount"`
	AmountMultiplier *int   `json:"amountMultiplier,omitempty"`
}

// RequestStartTransactionResponse represents a RequestStartTransaction response (CS → CSMS)
type RequestStartTransactionResponse struct {
	Status        string      `json:"status"` // Accepted, Rejected
	TransactionId string      `json:"transactionId,omitempty"`
	StatusInfo    *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== RequestStopTransaction ===========

// RequestStopTransactionRequest represents a RequestStopTransaction request (CSMS → CS)
type RequestStopTransactionRequest struct {
	TransactionId string `json:"transactionId"`
}

// RequestStopTransactionResponse represents a RequestStopTransaction response (CS → CSMS)
type RequestStopTransactionResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== GetVariables ===========

// GetVariablesRequest represents a GetVariables request (CSMS → CS)
type GetVariablesRequest struct {
	GetVariableData []GetVariableData `json:"getVariableData"`
}

// GetVariablesResponse represents a GetVariables response (CS → CSMS)
type GetVariablesResponse struct {
	GetVariableResult []GetVariableResult `json:"getVariableResult"`
}

// =========== SetVariables ===========

// SetVariablesRequest represents a SetVariables request (CSMS → CS)
type SetVariablesRequest struct {
	SetVariableData []SetVariableData `json:"setVariableData"`
}

// SetVariablesResponse represents a SetVariables response (CS → CSMS)
type SetVariablesResponse struct {
	SetVariableResult []SetVariableResult `json:"setVariableResult"`
}

// =========== Reset ===========

// ResetRequest represents a Reset request (CSMS → CS)
type ResetRequest struct {
	Type   ResetType `json:"type"`
	EvseId *int      `json:"evseId,omitempty"`
}

// ResetResponse represents a Reset response (CS → CSMS)
type ResetResponse struct {
	Status     ResetStatusType `json:"status"`
	StatusInfo *StatusInfo     `json:"statusInfo,omitempty"`
}

// =========== ChangeAvailability ===========

// ChangeAvailabilityRequest represents a ChangeAvailability request (CSMS → CS)
type ChangeAvailabilityRequest struct {
	OperationalStatus string `json:"operationalStatus"` // Inoperative, Operative
	EVSE              *EVSE  `json:"evse,omitempty"`
}

// ChangeAvailabilityResponse represents a ChangeAvailability response (CS → CSMS)
type ChangeAvailabilityResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Scheduled
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== UnlockConnector ===========

// UnlockConnectorRequest represents an UnlockConnector request (CSMS → CS)
type UnlockConnectorRequest struct {
	EvseId      int `json:"evseId"`
	ConnectorId int `json:"connectorId"`
}

// UnlockConnectorResponse represents an UnlockConnector response (CS → CSMS)
type UnlockConnectorResponse struct {
	Status     string      `json:"status"` // Unlocked, UnlockFailed, OngoingAuthorizedTransaction, UnknownConnector
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== ClearCache ===========

// ClearCacheRequest represents a ClearCache request (CSMS → CS)
type ClearCacheRequest struct {
	// Empty payload
}

// ClearCacheResponse represents a ClearCache response (CS → CSMS)
type ClearCacheResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== DataTransfer ===========

// DataTransferRequest represents a DataTransfer request (bidirectional)
type DataTransferRequest struct {
	VendorId  string      `json:"vendorId"`
	MessageId string      `json:"messageId,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// DataTransferResponse represents a DataTransfer response (bidirectional)
type DataTransferResponse struct {
	Status     DataTransferStatusType `json:"status"`
	Data       interface{}            `json:"data,omitempty"`
	StatusInfo *StatusInfo            `json:"statusInfo,omitempty"`
}

// =========== TriggerMessage ===========

// TriggerMessageRequest represents a TriggerMessage request (CSMS → CS)
type TriggerMessageRequest struct {
	RequestedMessage string `json:"requestedMessage"` // BootNotification, Heartbeat, MeterValues, StatusNotification, etc.
	EVSE             *EVSE  `json:"evse,omitempty"`
}

// TriggerMessageResponse represents a TriggerMessage response (CS → CSMS)
type TriggerMessageResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, NotImplemented
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== SecurityEventNotification ===========

// SecurityEventNotificationRequest represents a SecurityEventNotification request (CS → CSMS)
type SecurityEventNotificationRequest struct {
	Type      string   `json:"type"` // FirmwareUpdated, FailedToAuthenticateAtCsms, CentralSystemFailedToAuthenticate, etc.
	Timestamp DateTime `json:"timestamp"`
	TechInfo  string   `json:"techInfo,omitempty"`
}

// SecurityEventNotificationResponse represents a SecurityEventNotification response (CSMS → CS)
type SecurityEventNotificationResponse struct {
	// Empty payload
}

// =========== NotifyEvent ===========

// NotifyEventRequest represents a NotifyEvent request (CS → CSMS)
type NotifyEventRequest struct {
	GeneratedAt DateTime    `json:"generatedAt"`
	SeqNo       int         `json:"seqNo"`
	Tbc         *bool       `json:"tbc,omitempty"` // To Be Continued
	EventData   []EventData `json:"eventData"`
}

// EventData represents event data
type EventData struct {
	EventId               int       `json:"eventId"`
	Timestamp             DateTime  `json:"timestamp"`
	Trigger               string    `json:"trigger"` // Alerting, Delta, Periodic
	Cause                 *int      `json:"cause,omitempty"`
	ActualValue           string    `json:"actualValue"`
	TechCode              string    `json:"techCode,omitempty"`
	TechInfo              string    `json:"techInfo,omitempty"`
	Cleared               *bool     `json:"cleared,omitempty"`
	TransactionId         string    `json:"transactionId,omitempty"`
	Component             Component `json:"component"`
	Variable              Variable  `json:"variable"`
	EventNotificationType string    `json:"eventNotificationType"` // HardWiredNotification, HardWiredMonitor, PreconfiguredMonitor, CustomMonitor
}

// NotifyEventResponse represents a NotifyEvent response (CSMS → CS)
type NotifyEventResponse struct {
	// Empty payload
}

// =========== GetTransactionStatus ===========

// GetTransactionStatusRequest represents a GetTransactionStatus request (CSMS → CS)
type GetTransactionStatusRequest struct {
	TransactionId string `json:"transactionId,omitempty"`
}

// GetTransactionStatusResponse represents a GetTransactionStatus response (CS → CSMS)
type GetTransactionStatusResponse struct {
	OngoingIndicator *bool `json:"ongoingIndicator,omitempty"`
	MessagesInQueue  bool  `json:"messagesInQueue"`
}

// =========== Certificate Management ===========

// CertificateHashDataType contains hash data for certificate identification
type CertificateHashDataType struct {
	HashAlgorithm  string `json:"hashAlgorithm"`  // SHA256, SHA384, SHA512
	IssuerNameHash string `json:"issuerNameHash"` // Base64 encoded hash of issuer name
	IssuerKeyHash  string `json:"issuerKeyHash"`  // Base64 encoded hash of issuer public key
	SerialNumber   string `json:"serialNumber"`   // Serial number of the certificate
}

// CertificateHashDataChainType contains certificate hash data with optional child certificates
type CertificateHashDataChainType struct {
	CertificateType          string                    `json:"certificateType"` // V2GRootCertificate, MORootCertificate, CSMSRootCertificate, etc.
	CertificateHashData      CertificateHashDataType   `json:"certificateHashData"`
	ChildCertificateHashData []CertificateHashDataType `json:"childCertificateHashData,omitempty"`
}

// =========== SignCertificate ===========

// SignCertificateRequest represents a SignCertificate request (CS → CSMS)
type SignCertificateRequest struct {
	Csr             string `json:"csr"`                       // Base64 encoded PKCS#10 certificate signing request
	CertificateType string `json:"certificateType,omitempty"` // ChargingStationCertificate, V2GCertificate
}

// SignCertificateResponse represents a SignCertificate response (CSMS → CS)
type SignCertificateResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== CertificateSigned ===========

// CertificateSignedRequest represents a CertificateSigned request (CSMS → CS)
type CertificateSignedRequest struct {
	CertificateChain string `json:"certificateChain"`          // PEM encoded X.509 certificate chain
	CertificateType  string `json:"certificateType,omitempty"` // ChargingStationCertificate, V2GCertificate
}

// CertificateSignedResponse represents a CertificateSigned response (CS → CSMS)
type CertificateSignedResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== Get15118EVCertificate ===========

// Get15118EVCertificateRequest represents a Get15118EVCertificate request (CS → CSMS)
type Get15118EVCertificateRequest struct {
	Iso15118SchemaVersion string `json:"iso15118SchemaVersion"` // Schema version used for the 15118 session
	Action                string `json:"action"`                // Install, Update
	ExiRequest            string `json:"exiRequest"`            // Base64 encoded EXI stream
}

// Get15118EVCertificateResponse represents a Get15118EVCertificate response (CSMS → CS)
type Get15118EVCertificateResponse struct {
	Status      string      `json:"status"` // Accepted, Failed
	ExiResponse string      `json:"exiResponse"`
	StatusInfo  *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== GetCertificateStatus ===========

// GetCertificateStatusRequest represents a GetCertificateStatus request (CS → CSMS)
type GetCertificateStatusRequest struct {
	OcspRequestData OCSPRequestData `json:"ocspRequestData"`
}

// GetCertificateStatusResponse represents a GetCertificateStatus response (CSMS → CS)
type GetCertificateStatusResponse struct {
	Status     string      `json:"status"`               // Accepted, Failed
	OcspResult string      `json:"ocspResult,omitempty"` // Base64 encoded OCSP response
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== DeleteCertificate ===========

// DeleteCertificateRequest represents a DeleteCertificate request (CSMS → CS)
type DeleteCertificateRequest struct {
	CertificateHashData CertificateHashDataType `json:"certificateHashData"`
}

// DeleteCertificateResponse represents a DeleteCertificate response (CS → CSMS)
type DeleteCertificateResponse struct {
	Status     string      `json:"status"` // Accepted, Failed, NotFound
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// =========== GetInstalledCertificateIds ===========

// GetInstalledCertificateIdsRequest represents a GetInstalledCertificateIds request (CSMS → CS)
type GetInstalledCertificateIdsRequest struct {
	CertificateType []string `json:"certificateType,omitempty"` // V2GRootCertificate, MORootCertificate, CSMSRootCertificate, etc.
}

// GetInstalledCertificateIdsResponse represents a GetInstalledCertificateIds response (CS → CSMS)
type GetInstalledCertificateIdsResponse struct {
	Status                   string                         `json:"status"` // Accepted, NotFound
	CertificateHashDataChain []CertificateHashDataChainType `json:"certificateHashDataChain,omitempty"`
	StatusInfo               *StatusInfo                    `json:"statusInfo,omitempty"`
}

// =========== InstallCertificate ===========

// InstallCertificateRequest represents an InstallCertificate request (CSMS → CS)
type InstallCertificateRequest struct {
	CertificateType string `json:"certificateType"` // V2GRootCertificate, MORootCertificate, CSMSRootCertificate, ManufacturerRootCertificate
	Certificate     string `json:"certificate"`     // PEM encoded X.509 certificate
}

// InstallCertificateResponse represents an InstallCertificate response (CS → CSMS)
type InstallCertificateResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Failed
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}
