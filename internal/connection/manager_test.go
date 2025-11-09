package connection

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/config"
)

// TestNewManager tests creating a new connection manager
func TestNewManager(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.CSMSConfig{
		ConnectionTimeout:    30 * time.Second,
		MaxReconnectAttempts: 5,
		ReconnectBackoff:     5 * time.Second,
	}

	manager := NewManager(cfg, logger)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.pool == nil {
		t.Fatal("Expected connection pool to be initialized")
	}

	if manager.GetTotalCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", manager.GetTotalCount())
	}
}

// TestConnectionPool tests the connection pool functionality
func TestConnectionPool(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	pool := NewConnectionPool(logger)

	if pool.Size() != 0 {
		t.Errorf("Expected empty pool, got size %d", pool.Size())
	}

	// Test HasConnection
	if pool.Has("test-station") {
		t.Error("Expected station to not exist")
	}

	// Test GetConnectedCount
	if count := pool.GetConnectedCount(); count != 0 {
		t.Errorf("Expected 0 connected stations, got %d", count)
	}
}

// TestGetSubprotocol tests subprotocol determination
func TestGetSubprotocol(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"1.6", "ocpp1.6"},
		{"2.0.1", "ocpp2.0.1"},
		{"2.1", "ocpp2.1"},
		{"unknown", "ocpp1.6"}, // defaults to 1.6
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := getSubprotocol(tt.version)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestConnectionStates tests connection state management
func TestConnectionStates(t *testing.T) {
	states := []ConnectionState{
		StateDisconnected,
		StateConnecting,
		StateConnected,
		StateReconnecting,
		StateError,
		StateClosed,
	}

	expectedStrings := []string{
		"disconnected",
		"connecting",
		"connected",
		"reconnecting",
		"error",
		"closed",
	}

	for i, state := range states {
		if string(state) != expectedStrings[i] {
			t.Errorf("Expected state %s, got %s", expectedStrings[i], string(state))
		}
	}
}

// TestManagerCallbacks tests that callbacks are properly set up
func TestManagerCallbacks(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.CSMSConfig{
		ConnectionTimeout:    30 * time.Second,
		MaxReconnectAttempts: 5,
		ReconnectBackoff:     5 * time.Second,
	}

	manager := NewManager(cfg, logger)

	// Test setting callbacks
	called := false

	manager.OnStationConnected = func(stationID string) {
		called = true
	}

	if manager.OnStationConnected == nil {
		t.Error("Expected callback to be set")
	}

	// Simulate callback
	manager.OnStationConnected("test-station")

	if !called {
		t.Error("Expected callback to be called")
	}
}

// TestBase64Encoding tests basic auth encoding
func TestBase64Encoding(t *testing.T) {
	result := base64Encode("test:password")
	if result == "" {
		t.Error("Expected non-empty base64 string")
	}

	// Test that it creates valid base64
	if len(result) < 4 {
		t.Error("Expected valid base64 encoding")
	}
}

// TestBasicAuth tests basic auth header creation
func TestBasicAuth(t *testing.T) {
	result := basicAuth("user", "pass")
	if result[:6] != "Basic " {
		t.Errorf("Expected 'Basic ' prefix, got %s", result[:6])
	}
}

// TestExplicitDisconnectNoReconnect tests that explicit disconnect doesn't trigger reconnection
func TestExplicitDisconnectNoReconnect(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	config := ConnectionConfig{
		URL:                  "ws://localhost:9999/ocpp",
		StationID:            "TEST001",
		ProtocolVersion:      "1.6",
		ConnectionTimeout:    5 * time.Second,
		MaxReconnectAttempts: 3,
		ReconnectBackoff:     1 * time.Second,
	}

	client := NewWebSocketClient(config, logger)

	// Verify initial state
	if client.GetState() != StateDisconnected {
		t.Errorf("Expected initial state to be disconnected, got %s", client.GetState())
	}

	// Call Disconnect() explicitly (simulating user stopping station)
	err := client.Disconnect()
	if err != nil {
		t.Errorf("Expected no error on disconnect, got %v", err)
	}

	// Verify that context was cancelled
	select {
	case <-client.ctx.Done():
		// Context should be cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled after Disconnect()")
	}

	// Verify final state is closed
	if client.GetState() != StateClosed {
		t.Errorf("Expected state to be closed after disconnect, got %s", client.GetState())
	}

	// Simulate handleDisconnect being called (as would happen when readPump exits)
	client.handleDisconnect(nil)

	// Give a bit of time for any potential reconnection goroutine
	time.Sleep(200 * time.Millisecond)

	// Verify state is still closed (not reconnecting)
	if client.GetState() != StateClosed {
		t.Errorf("Expected state to remain closed, got %s", client.GetState())
	}

	// Verify reconnect count hasn't increased
	stats := client.GetStats()
	if stats.ReconnectAttempts != 0 {
		t.Errorf("Expected 0 reconnect attempts after explicit disconnect, got %d", stats.ReconnectAttempts)
	}
}
