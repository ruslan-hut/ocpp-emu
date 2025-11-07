package station

import (
	"time"
)

// Config represents a station configuration
type Config struct {
	// Identity
	ID        string
	StationID string
	Name      string
	Enabled   bool
	AutoStart bool

	// Protocol
	ProtocolVersion string

	// Hardware Info
	Vendor          string
	Model           string
	SerialNumber    string
	FirmwareVersion string
	ICCID           string
	IMSI            string

	// Connectors
	Connectors []ConnectorConfig

	// OCPP Features
	SupportedProfiles []string

	// Meter Values
	MeterValuesConfig MeterValuesConfig

	// CSMS Connection
	CSMSURL  string
	CSMSAuth *CSMSAuthConfig

	// Simulation
	Simulation SimulationConfig

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	Tags      []string
}

// ConnectorConfig represents a connector configuration
type ConnectorConfig struct {
	ID                   int
	Type                 string
	MaxPower             int
	Status               string
	CurrentTransactionID *int
}

// MeterValuesConfig represents meter values configuration
type MeterValuesConfig struct {
	Interval            int
	Measurands          []string
	AlignedDataInterval int
}

// CSMSAuthConfig represents CSMS authentication configuration
type CSMSAuthConfig struct {
	Type     string
	Username string
	Password string
	Token    string
}

// SimulationConfig represents simulation behavior configuration
type SimulationConfig struct {
	BootDelay                  int
	HeartbeatInterval          int
	StatusNotificationOnChange bool
	DefaultIDTag               string
	EnergyDeliveryRate         int
	RandomizeMeterValues       bool
	MeterValueVariance         float64
}

// RuntimeState represents the runtime state of a station
type RuntimeState struct {
	State            State
	ConnectionStatus string
	LastHeartbeat    *time.Time
	LastError        string
	ConnectedAt      *time.Time
	TransactionID    *int
	CurrentSession   *SessionInfo
}

// SessionInfo represents current session information
type SessionInfo struct {
	TransactionID int
	IDTag         string
	StartTime     time.Time
	StartMeter    int
	CurrentMeter  int
}
