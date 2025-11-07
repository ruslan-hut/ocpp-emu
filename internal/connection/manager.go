package connection

import (
	"fmt"
	"log/slog"

	"github.com/ruslanhut/ocpp-emu/internal/config"
)

// Manager manages WebSocket connections for all stations
type Manager struct {
	pool   *ConnectionPool
	config *config.CSMSConfig
	logger *slog.Logger

	// Callbacks for message handling
	OnMessageReceived     func(stationID string, message []byte)
	OnStationConnected    func(stationID string)
	OnStationDisconnected func(stationID string, err error)
	OnStationError        func(stationID string, err error)
}

// NewManager creates a new connection manager
func NewManager(cfg *config.CSMSConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		pool:   NewConnectionPool(logger),
		config: cfg,
		logger: logger,
	}
}

// ConnectStation establishes a connection for a station
func (m *Manager) ConnectStation(stationID, url, protocolVersion string, tlsConfig *TLSConfig, auth *AuthConfig) error {
	// Check if already connected
	if m.pool.Has(stationID) {
		return fmt.Errorf("station %s is already connected", stationID)
	}

	// Create connection configuration
	connConfig := ConnectionConfig{
		URL:                  url,
		StationID:            stationID,
		ProtocolVersion:      protocolVersion,
		ConnectionTimeout:    m.config.ConnectionTimeout,
		MaxReconnectAttempts: m.config.MaxReconnectAttempts,
		ReconnectBackoff:     m.config.ReconnectBackoff,
		ReconnectMaxBackoff:  60 * m.config.ReconnectBackoff, // Max 60x backoff
	}

	// Apply TLS configuration
	if tlsConfig != nil {
		connConfig.TLSEnabled = tlsConfig.Enabled
		connConfig.TLSCACert = tlsConfig.CACert
		connConfig.TLSClientCert = tlsConfig.ClientCert
		connConfig.TLSClientKey = tlsConfig.ClientKey
		connConfig.TLSSkipVerify = tlsConfig.InsecureSkipVerify
	} else if m.config.TLS.Enabled {
		// Use default TLS config from CSMS settings
		connConfig.TLSEnabled = m.config.TLS.Enabled
		connConfig.TLSCACert = m.config.TLS.CACert
		connConfig.TLSClientCert = m.config.TLS.ClientCert
		connConfig.TLSClientKey = m.config.TLS.ClientKey
		connConfig.TLSSkipVerify = m.config.TLS.InsecureSkipVerify
	}

	// Apply authentication
	if auth != nil {
		if auth.Type == "basic" {
			connConfig.BasicAuthUsername = auth.Username
			connConfig.BasicAuthPassword = auth.Password
		} else if auth.Type == "bearer" {
			connConfig.BearerToken = auth.Token
		}
	}

	// Set up callbacks
	connConfig.OnConnected = func() {
		m.logger.Info("Station connected", "station_id", stationID)
		if m.OnStationConnected != nil {
			m.OnStationConnected(stationID)
		}
	}

	connConfig.OnDisconnected = func(err error) {
		m.logger.Info("Station disconnected", "station_id", stationID, "error", err)
		if m.OnStationDisconnected != nil {
			m.OnStationDisconnected(stationID, err)
		}
	}

	connConfig.OnMessage = func(message []byte) {
		m.logger.Debug("Received message", "station_id", stationID, "size", len(message))
		if m.OnMessageReceived != nil {
			m.OnMessageReceived(stationID, message)
		}
	}

	connConfig.OnError = func(err error) {
		m.logger.Error("Station error", "station_id", stationID, "error", err)
		if m.OnStationError != nil {
			m.OnStationError(stationID, err)
		}
	}

	// Create WebSocket client
	client := NewWebSocketClient(connConfig, m.logger)

	// Add to pool
	if err := m.pool.Add(stationID, client); err != nil {
		return fmt.Errorf("failed to add connection to pool: %w", err)
	}

	// Connect
	if err := client.Connect(); err != nil {
		m.pool.Remove(stationID)
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

// DisconnectStation disconnects a station
func (m *Manager) DisconnectStation(stationID string) error {
	return m.pool.Remove(stationID)
}

// SendMessage sends a message to a specific station
func (m *Manager) SendMessage(stationID string, message []byte) error {
	return m.pool.Send(stationID, message)
}

// BroadcastMessage sends a message to all connected stations
func (m *Manager) BroadcastMessage(message []byte) error {
	return m.pool.Broadcast(message)
}

// GetConnectionStats returns connection statistics for a station
func (m *Manager) GetConnectionStats(stationID string) (ConnectionStats, error) {
	client, err := m.pool.Get(stationID)
	if err != nil {
		return ConnectionStats{}, err
	}

	return client.GetStats(), nil
}

// GetAllConnectionStats returns statistics for all connections
func (m *Manager) GetAllConnectionStats() map[string]ConnectionStats {
	return m.pool.GetStats()
}

// GetConnectedStations returns a list of connected station IDs
func (m *Manager) GetConnectedStations() []string {
	allConnections := m.pool.GetAll()
	var connected []string

	for stationID, client := range allConnections {
		if client.GetState() == StateConnected {
			connected = append(connected, stationID)
		}
	}

	return connected
}

// GetConnectionState returns the connection state for a station
func (m *Manager) GetConnectionState(stationID string) (ConnectionState, error) {
	client, err := m.pool.Get(stationID)
	if err != nil {
		return StateDisconnected, err
	}

	return client.GetState(), nil
}

// IsConnected checks if a station is connected
func (m *Manager) IsConnected(stationID string) bool {
	client, err := m.pool.Get(stationID)
	if err != nil {
		return false
	}

	return client.GetState() == StateConnected
}

// GetConnectedCount returns the number of connected stations
func (m *Manager) GetConnectedCount() int {
	return m.pool.GetConnectedCount()
}

// GetTotalCount returns the total number of stations (connected or not)
func (m *Manager) GetTotalCount() int {
	return m.pool.Size()
}

// DisconnectAll disconnects all stations
func (m *Manager) DisconnectAll() error {
	m.logger.Info("Disconnecting all stations")
	return m.pool.DisconnectAll()
}

// Shutdown gracefully shuts down the connection manager
func (m *Manager) Shutdown() error {
	m.logger.Info("Shutting down connection manager")
	return m.DisconnectAll()
}

// TLSConfig holds TLS configuration for a connection
type TLSConfig struct {
	Enabled            bool
	CACert             string
	ClientCert         string
	ClientKey          string
	InsecureSkipVerify bool
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type     string // "basic", "bearer", "certificate"
	Username string
	Password string
	Token    string
}
