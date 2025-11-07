package v16

// Core Profile Message Payloads

// =========== Authorize ===========

// AuthorizeRequest represents an Authorize request
type AuthorizeRequest struct {
	IdTag string `json:"idTag" validate:"required,max=20"`
}

// AuthorizeResponse represents an Authorize response
type AuthorizeResponse struct {
	IdTagInfo IdTagInfo `json:"idTagInfo"`
}

// =========== BootNotification ===========

// BootNotificationRequest represents a BootNotification request
type BootNotificationRequest struct {
	ChargePointVendor       string `json:"chargePointVendor" validate:"required,max=20"`
	ChargePointModel        string `json:"chargePointModel" validate:"required,max=20"`
	ChargePointSerialNumber string `json:"chargePointSerialNumber,omitempty" validate:"max=25"`
	ChargeBoxSerialNumber   string `json:"chargeBoxSerialNumber,omitempty" validate:"max=25"`
	FirmwareVersion         string `json:"firmwareVersion,omitempty" validate:"max=50"`
	Iccid                   string `json:"iccid,omitempty" validate:"max=20"`
	Imsi                    string `json:"imsi,omitempty" validate:"max=20"`
	MeterType               string `json:"meterType,omitempty" validate:"max=25"`
	MeterSerialNumber       string `json:"meterSerialNumber,omitempty" validate:"max=25"`
}

// BootNotificationResponse represents a BootNotification response
type BootNotificationResponse struct {
	Status      RegistrationStatus `json:"status"`
	CurrentTime DateTime           `json:"currentTime"`
	Interval    int                `json:"interval"` // Heartbeat interval in seconds
}

// =========== DataTransfer ===========

// DataTransferRequest represents a DataTransfer request
type DataTransferRequest struct {
	VendorId  string `json:"vendorId" validate:"required,max=255"`
	MessageId string `json:"messageId,omitempty" validate:"max=50"`
	Data      string `json:"data,omitempty"`
}

// DataTransferResponse represents a DataTransfer response
type DataTransferResponse struct {
	Status string `json:"status"` // Accepted, Rejected, UnknownMessageId, UnknownVendorId
	Data   string `json:"data,omitempty"`
}

// =========== Heartbeat ===========

// HeartbeatRequest represents a Heartbeat request
type HeartbeatRequest struct {
	// Empty payload
}

// HeartbeatResponse represents a Heartbeat response
type HeartbeatResponse struct {
	CurrentTime DateTime `json:"currentTime"`
}

// =========== MeterValues ===========

// MeterValuesRequest represents a MeterValues request
type MeterValuesRequest struct {
	ConnectorId   int          `json:"connectorId" validate:"required,gte=0"`
	TransactionId *int         `json:"transactionId,omitempty"`
	MeterValue    []MeterValue `json:"meterValue" validate:"required,min=1"`
}

// MeterValuesResponse represents a MeterValues response
type MeterValuesResponse struct {
	// Empty payload
}

// =========== StartTransaction ===========

// StartTransactionRequest represents a StartTransaction request
type StartTransactionRequest struct {
	ConnectorId   int      `json:"connectorId" validate:"required,gt=0"`
	IdTag         string   `json:"idTag" validate:"required,max=20"`
	MeterStart    int      `json:"meterStart" validate:"required"`
	Timestamp     DateTime `json:"timestamp" validate:"required"`
	ReservationId *int     `json:"reservationId,omitempty"`
}

// StartTransactionResponse represents a StartTransaction response
type StartTransactionResponse struct {
	IdTagInfo     IdTagInfo `json:"idTagInfo"`
	TransactionId int       `json:"transactionId"`
}

// =========== StatusNotification ===========

// StatusNotificationRequest represents a StatusNotification request
type StatusNotificationRequest struct {
	ConnectorId     int                  `json:"connectorId" validate:"required,gte=0"`
	ErrorCode       ChargePointErrorCode `json:"errorCode" validate:"required"`
	Info            string               `json:"info,omitempty" validate:"max=50"`
	Status          ChargePointStatus    `json:"status" validate:"required"`
	Timestamp       *DateTime            `json:"timestamp,omitempty"`
	VendorId        string               `json:"vendorId,omitempty" validate:"max=255"`
	VendorErrorCode string               `json:"vendorErrorCode,omitempty" validate:"max=50"`
}

// StatusNotificationResponse represents a StatusNotification response
type StatusNotificationResponse struct {
	// Empty payload
}

// =========== StopTransaction ===========

// StopTransactionRequest represents a StopTransaction request
type StopTransactionRequest struct {
	IdTag           string       `json:"idTag,omitempty" validate:"max=20"`
	MeterStop       int          `json:"meterStop" validate:"required"`
	Timestamp       DateTime     `json:"timestamp" validate:"required"`
	TransactionId   int          `json:"transactionId" validate:"required"`
	Reason          Reason       `json:"reason,omitempty"`
	TransactionData []MeterValue `json:"transactionData,omitempty"`
}

// StopTransactionResponse represents a StopTransaction response
type StopTransactionResponse struct {
	IdTagInfo *IdTagInfo `json:"idTagInfo,omitempty"`
}

// =========== Remote Start/Stop Transaction ===========

// RemoteStartTransactionRequest represents a RemoteStartTransaction request
type RemoteStartTransactionRequest struct {
	ConnectorId     *int        `json:"connectorId,omitempty" validate:"omitempty,gt=0"`
	IdTag           string      `json:"idTag" validate:"required,max=20"`
	ChargingProfile interface{} `json:"chargingProfile,omitempty"` // Complex type, simplified for now
}

// RemoteStartTransactionResponse represents a RemoteStartTransactionResponse
type RemoteStartTransactionResponse struct {
	Status string `json:"status"` // Accepted, Rejected
}

// RemoteStopTransactionRequest represents a RemoteStopTransaction request
type RemoteStopTransactionRequest struct {
	TransactionId int `json:"transactionId" validate:"required"`
}

// RemoteStopTransactionResponse represents a RemoteStopTransaction response
type RemoteStopTransactionResponse struct {
	Status string `json:"status"` // Accepted, Rejected
}

// =========== Reset ===========

// ResetRequest represents a Reset request
type ResetRequest struct {
	Type string `json:"type" validate:"required"` // Hard, Soft
}

// ResetResponse represents a Reset response
type ResetResponse struct {
	Status string `json:"status"` // Accepted, Rejected
}

// =========== UnlockConnector ===========

// UnlockConnectorRequest represents an UnlockConnector request
type UnlockConnectorRequest struct {
	ConnectorId int `json:"connectorId" validate:"required,gt=0"`
}

// UnlockConnectorResponse represents an UnlockConnector response
type UnlockConnectorResponse struct {
	Status string `json:"status"` // Unlocked, UnlockFailed, NotSupported
}

// =========== ChangeAvailability ===========

// ChangeAvailabilityRequest represents a ChangeAvailability request
type ChangeAvailabilityRequest struct {
	ConnectorId int    `json:"connectorId" validate:"required,gte=0"`
	Type        string `json:"type" validate:"required"` // Inoperative, Operative
}

// ChangeAvailabilityResponse represents a ChangeAvailability response
type ChangeAvailabilityResponse struct {
	Status string `json:"status"` // Accepted, Rejected, Scheduled
}

// =========== GetConfiguration ===========

// GetConfigurationRequest represents a GetConfiguration request
type GetConfigurationRequest struct {
	Key []string `json:"key,omitempty"` // List of configuration keys
}

// KeyValue represents a configuration key-value pair
type KeyValue struct {
	Key      string `json:"key" validate:"required,max=50"`
	Readonly bool   `json:"readonly"`
	Value    string `json:"value,omitempty" validate:"max=500"`
}

// GetConfigurationResponse represents a GetConfiguration response
type GetConfigurationResponse struct {
	ConfigurationKey []KeyValue `json:"configurationKey,omitempty"`
	UnknownKey       []string   `json:"unknownKey,omitempty"`
}

// =========== ChangeConfiguration ===========

// ChangeConfigurationRequest represents a ChangeConfiguration request
type ChangeConfigurationRequest struct {
	Key   string `json:"key" validate:"required,max=50"`
	Value string `json:"value" validate:"required,max=500"`
}

// ChangeConfigurationResponse represents a ChangeConfiguration response
type ChangeConfigurationResponse struct {
	Status string `json:"status"` // Accepted, Rejected, RebootRequired, NotSupported
}

// =========== ClearCache ===========

// ClearCacheRequest represents a ClearCache request
type ClearCacheRequest struct {
	// Empty payload
}

// ClearCacheResponse represents a ClearCache response
type ClearCacheResponse struct {
	Status string `json:"status"` // Accepted, Rejected
}
