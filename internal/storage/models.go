package storage

import (
	"time"
)

// Message represents an OCPP message in the database
type Message struct {
	ID               string                 `bson:"_id,omitempty"`
	StationID        string                 `bson:"station_id"`
	Direction        string                 `bson:"direction"`        // "sent" or "received"
	MessageType      string                 `bson:"message_type"`     // "Call", "CallResult", "CallError"
	Action           string                 `bson:"action"`           // e.g., "BootNotification", "Heartbeat"
	MessageID        string                 `bson:"message_id"`       // Unique message ID
	ProtocolVersion  string                 `bson:"protocol_version"` // "1.6", "2.0.1", "2.1"
	Payload          map[string]interface{} `bson:"payload"`          // Message payload
	Timestamp        time.Time              `bson:"timestamp"`        // Message timestamp
	CorrelationID    string                 `bson:"correlation_id"`   // Link request with response
	ErrorCode        string                 `bson:"error_code,omitempty"`
	ErrorDescription string                 `bson:"error_description,omitempty"`
	CreatedAt        time.Time              `bson:"created_at"`
}

// Transaction represents a charging transaction
type Transaction struct {
	ID              string    `bson:"_id,omitempty"`
	TransactionID   int       `bson:"transaction_id"`
	StationID       string    `bson:"station_id"`
	ConnectorID     int       `bson:"connector_id"`
	IDTag           string    `bson:"id_tag"`
	StartTimestamp  time.Time `bson:"start_timestamp"`
	StopTimestamp   time.Time `bson:"stop_timestamp,omitempty"`
	MeterStart      int       `bson:"meter_start"`      // Wh
	MeterStop       int       `bson:"meter_stop"`       // Wh
	EnergyConsumed  int       `bson:"energy_consumed"`  // Wh
	Reason          string    `bson:"reason,omitempty"` // Stop reason
	Status          string    `bson:"status"`           // "active", "completed", "failed"
	ProtocolVersion string    `bson:"protocol_version"`
	CreatedAt       time.Time `bson:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"`
}

// Station represents a charging station configuration
type Station struct {
	ID                string            `bson:"_id,omitempty"`
	StationID         string            `bson:"station_id"`
	Name              string            `bson:"name"`
	Enabled           bool              `bson:"enabled"`
	AutoStart         bool              `bson:"auto_start"`
	ProtocolVersion   string            `bson:"protocol_version"`
	Vendor            string            `bson:"vendor"`
	Model             string            `bson:"model"`
	SerialNumber      string            `bson:"serial_number"`
	FirmwareVersion   string            `bson:"firmware_version"`
	ICCID             string            `bson:"iccid,omitempty"`
	IMSI              string            `bson:"imsi,omitempty"`
	Connectors        []Connector       `bson:"connectors"`
	SupportedProfiles []string          `bson:"supported_profiles"`
	MeterValuesConfig MeterValuesConfig `bson:"meter_values_config"`
	CSMSURL           string            `bson:"csms_url"`
	CSMSAuth          CSMSAuth          `bson:"csms_auth,omitempty"`
	Simulation        SimulationConfig  `bson:"simulation"`
	ConnectionStatus  string            `bson:"connection_status"`
	LastHeartbeat     *time.Time        `bson:"last_heartbeat,omitempty"`
	LastError         string            `bson:"last_error,omitempty"`
	CreatedAt         time.Time         `bson:"created_at"`
	UpdatedAt         time.Time         `bson:"updated_at"`
	CreatedBy         string            `bson:"created_by,omitempty"`
	Tags              []string          `bson:"tags,omitempty"`
}

// Connector represents a charging connector
type Connector struct {
	ID                   int    `bson:"id"`
	Type                 string `bson:"type"`      // Type2, CCS, CHAdeMO, etc.
	MaxPower             int    `bson:"max_power"` // Watts
	Status               string `bson:"status"`    // Available, Occupied, Faulted, etc.
	CurrentTransactionID *int   `bson:"current_transaction_id,omitempty"`
}

// MeterValuesConfig holds meter values configuration
type MeterValuesConfig struct {
	Interval            int      `bson:"interval"`              // Seconds
	Measurands          []string `bson:"measurands"`            // List of measurand types
	AlignedDataInterval int      `bson:"aligned_data_interval"` // Seconds
}

// CSMSAuth holds authentication credentials for CSMS
type CSMSAuth struct {
	Type     string `bson:"type"` // basic, bearer, certificate
	Username string `bson:"username,omitempty"`
	Password string `bson:"password,omitempty"`
}

// SimulationConfig holds simulation behavior settings
type SimulationConfig struct {
	BootDelay                  int     `bson:"boot_delay"`         // Seconds
	HeartbeatInterval          int     `bson:"heartbeat_interval"` // Seconds
	StatusNotificationOnChange bool    `bson:"status_notification_on_change"`
	DefaultIDTag               string  `bson:"default_id_tag"`
	EnergyDeliveryRate         int     `bson:"energy_delivery_rate"` // Watts
	RandomizeMeterValues       bool    `bson:"randomize_meter_values"`
	MeterValueVariance         float64 `bson:"meter_value_variance"` // 0.0-1.0
}

// Session represents a WebSocket session
type Session struct {
	ID                string     `bson:"_id,omitempty"`
	StationID         string     `bson:"station_id"`
	CSMSURL           string     `bson:"csms_url"`
	ConnectedAt       time.Time  `bson:"connected_at"`
	DisconnectedAt    *time.Time `bson:"disconnected_at,omitempty"`
	Status            string     `bson:"status"` // active, disconnected
	ReconnectAttempts int        `bson:"reconnect_attempts"`
	LastMessageAt     *time.Time `bson:"last_message_at,omitempty"`
	ProtocolVersion   string     `bson:"protocol_version"`
	Subprotocol       string     `bson:"subprotocol"`
	CreatedAt         time.Time  `bson:"created_at"`
	UpdatedAt         time.Time  `bson:"updated_at"`
}

// MeterValue represents a meter value sample (time-series data)
type MeterValue struct {
	ID        string             `bson:"_id,omitempty"`
	Timestamp time.Time          `bson:"timestamp"`
	Metadata  MeterValueMetadata `bson:"metadata"`
	Value     float64            `bson:"value"`
	Unit      string             `bson:"unit"`
	Context   string             `bson:"context"`  // Sample.Periodic, Transaction.Begin, etc.
	Format    string             `bson:"format"`   // Raw, SignedData
	Location  string             `bson:"location"` // Outlet, Inlet, Body
}

// MeterValueMetadata holds metadata for meter values
type MeterValueMetadata struct {
	StationID     string `bson:"station_id"`
	ConnectorID   int    `bson:"connector_id"`
	TransactionID int    `bson:"transaction_id"`
	Measurand     string `bson:"measurand"` // Energy.Active.Import.Register, etc.
}
