package connection

import (
	"fmt"
	"log/slog"
	"sync"
)

// ConnectionPool manages multiple WebSocket connections for different stations
type ConnectionPool struct {
	connections map[string]*WebSocketClient
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(logger *slog.Logger) *ConnectionPool {
	if logger == nil {
		logger = slog.Default()
	}

	return &ConnectionPool{
		connections: make(map[string]*WebSocketClient),
		logger:      logger,
	}
}

// Add adds a new connection to the pool
func (p *ConnectionPool) Add(stationID string, client *WebSocketClient) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.connections[stationID]; exists {
		return fmt.Errorf("connection for station %s already exists", stationID)
	}

	p.connections[stationID] = client
	p.logger.Info("Added connection to pool", "station_id", stationID)

	return nil
}

// Remove removes a connection from the pool
func (p *ConnectionPool) Remove(stationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.connections[stationID]
	if !exists {
		return fmt.Errorf("connection for station %s not found", stationID)
	}

	// Disconnect and remove
	if err := client.Disconnect(); err != nil {
		p.logger.Warn("Error disconnecting client",
			"station_id", stationID,
			"error", err,
		)
	}

	delete(p.connections, stationID)
	p.logger.Info("Removed connection from pool", "station_id", stationID)

	return nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(stationID string) (*WebSocketClient, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, exists := p.connections[stationID]
	if !exists {
		return nil, fmt.Errorf("connection for station %s not found", stationID)
	}

	return client, nil
}

// GetAll returns all connections in the pool
func (p *ConnectionPool) GetAll() map[string]*WebSocketClient {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to prevent external modifications
	copy := make(map[string]*WebSocketClient, len(p.connections))
	for k, v := range p.connections {
		copy[k] = v
	}

	return copy
}

// Send sends a message to a specific station
func (p *ConnectionPool) Send(stationID string, data []byte) error {
	client, err := p.Get(stationID)
	if err != nil {
		return err
	}

	return client.Send(data)
}

// Broadcast sends a message to all connected stations
func (p *ConnectionPool) Broadcast(data []byte) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var errors []error

	for stationID, client := range p.connections {
		if client.GetState() == StateConnected {
			if err := client.Send(data); err != nil {
				errors = append(errors, fmt.Errorf("failed to send to %s: %w", stationID, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed for %d stations", len(errors))
	}

	return nil
}

// GetStats returns statistics for all connections
func (p *ConnectionPool) GetStats() map[string]ConnectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]ConnectionStats, len(p.connections))
	for stationID, client := range p.connections {
		stats[stationID] = client.GetStats()
	}

	return stats
}

// GetConnectedCount returns the number of connected stations
func (p *ConnectionPool) GetConnectedCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, client := range p.connections {
		if client.GetState() == StateConnected {
			count++
		}
	}

	return count
}

// DisconnectAll disconnects all connections in the pool
func (p *ConnectionPool) DisconnectAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error

	for stationID, client := range p.connections {
		if err := client.Disconnect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect %s: %w", stationID, err))
		}
	}

	// Clear the pool
	p.connections = make(map[string]*WebSocketClient)

	if len(errors) > 0 {
		return fmt.Errorf("failed to disconnect %d stations", len(errors))
	}

	p.logger.Info("Disconnected all connections")
	return nil
}

// Size returns the number of connections in the pool
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// Has checks if a connection exists for a station
func (p *ConnectionPool) Has(stationID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.connections[stationID]
	return exists
}
