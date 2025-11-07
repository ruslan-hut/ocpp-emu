package connection

import (
	"time"
)

// ConnectionState represents the state of a WebSocket connection
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateReconnecting ConnectionState = "reconnecting"
	StateError        ConnectionState = "error"
	StateClosed       ConnectionState = "closed"
)

// ConnectionConfig holds configuration for a WebSocket connection
type ConnectionConfig struct {
	// Connection settings
	URL               string
	StationID         string
	ProtocolVersion   string // "1.6", "2.0.1", "2.1"
	Subprotocol       string // Derived from ProtocolVersion
	ConnectionTimeout time.Duration
	WriteTimeout      time.Duration
	ReadTimeout       time.Duration
	PingInterval      time.Duration
	PongTimeout       time.Duration

	// Reconnection settings
	MaxReconnectAttempts int
	ReconnectBackoff     time.Duration
	ReconnectMaxBackoff  time.Duration

	// TLS settings
	TLSEnabled    bool
	TLSCACert     string
	TLSClientCert string
	TLSClientKey  string
	TLSSkipVerify bool

	// Authentication
	BasicAuthUsername string
	BasicAuthPassword string
	BearerToken       string

	// Callbacks
	OnConnected    func()
	OnDisconnected func(error)
	OnMessage      func([]byte)
	OnError        func(error)
}

// ConnectionStats holds statistics about a connection
type ConnectionStats struct {
	StationID         string
	State             ConnectionState
	ConnectedAt       *time.Time
	DisconnectedAt    *time.Time
	LastMessageAt     *time.Time
	ReconnectAttempts int
	MessagesSent      int64
	MessagesReceived  int64
	BytesSent         int64
	BytesReceived     int64
	LastError         string
}

// MessageType represents the type of message to send
type MessageType int

const (
	TextMessage   MessageType = 1
	BinaryMessage MessageType = 2
	CloseMessage  MessageType = 8
	PingMessage   MessageType = 9
	PongMessage   MessageType = 10
)

// Message represents a message to be sent over WebSocket
type Message struct {
	Type      MessageType
	Data      []byte
	StationID string
}
