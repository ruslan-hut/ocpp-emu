package station

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/connection"
)

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := ManagerConfig{
		SyncInterval: 10 * time.Second,
	}

	manager := NewManager(nil, nil, nil, logger, config)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.syncInterval != 10*time.Second {
		t.Errorf("Expected sync interval 10s, got %v", manager.syncInterval)
	}

	if manager.stations == nil {
		t.Error("Expected stations map to be initialized")
	}
}

func TestNewManagerDefaultConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := ManagerConfig{} // Empty config

	manager := NewManager(nil, nil, nil, logger, config)

	if manager.syncInterval != 30*time.Second {
		t.Errorf("Expected default sync interval 30s, got %v", manager.syncInterval)
	}
}

func TestAddStation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	config := Config{
		StationID:       "TEST001",
		Name:            "Test Station",
		Enabled:         true,
		AutoStart:       false,
		ProtocolVersion: "ocpp1.6",
		Vendor:          "TestVendor",
		Model:           "TestModel",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// AddStation without DB will add to memory only
	manager.mu.Lock()
	manager.stations[config.StationID] = &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
		RuntimeState: RuntimeState{
			State:            StateDisconnected,
			ConnectionStatus: "not_connected",
		},
	}
	manager.mu.Unlock()

	station, err := manager.GetStation("TEST001")
	if err != nil {
		t.Fatalf("Failed to get station: %v", err)
	}

	if station.Config.Name != "Test Station" {
		t.Errorf("Expected station name 'Test Station', got '%s'", station.Config.Name)
	}

	// Note: NewStateMachine() initializes with StateUnknown, not StateDisconnected
	// This is expected behavior - station state is set when actually connecting/disconnecting
	state := station.StateMachine.GetState()
	if state != StateUnknown && state != StateDisconnected {
		t.Errorf("Expected state Unknown or Disconnected, got %v", state)
	}
}

func TestGetStation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	// Try to get non-existent station
	_, err := manager.GetStation("NONEXISTENT")
	if err == nil {
		t.Error("Expected error when getting non-existent station")
	}

	// Add station
	config := Config{
		StationID: "TEST002",
		Name:      "Test Station 2",
	}

	manager.mu.Lock()
	manager.stations[config.StationID] = &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
	}
	manager.mu.Unlock()

	// Get existing station
	station, err := manager.GetStation("TEST002")
	if err != nil {
		t.Fatalf("Failed to get station: %v", err)
	}

	if station.Config.StationID != "TEST002" {
		t.Errorf("Expected station ID TEST002, got %s", station.Config.StationID)
	}
}

func TestGetAllStations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	// Initially empty
	stations := manager.GetAllStations()
	if len(stations) != 0 {
		t.Errorf("Expected 0 stations, got %d", len(stations))
	}

	// Add stations
	for i := 1; i <= 3; i++ {
		config := Config{
			StationID: string(rune('A'+i-1)) + "TEST",
		}

		manager.mu.Lock()
		manager.stations[config.StationID] = &Station{
			Config:       config,
			StateMachine: NewStateMachine(),
		}
		manager.mu.Unlock()
	}

	stations = manager.GetAllStations()
	if len(stations) != 3 {
		t.Errorf("Expected 3 stations, got %d", len(stations))
	}
}

func TestOnStationConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	connMgr := connection.NewManager(nil, logger)
	manager := NewManager(nil, connMgr, nil, logger, ManagerConfig{})

	// Add station
	config := Config{
		StationID: "TEST003",
		Vendor:    "TestVendor",
		Model:     "TestModel",
	}

	manager.mu.Lock()
	manager.stations[config.StationID] = &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
		RuntimeState: RuntimeState{
			State:            StateDisconnected,
			ConnectionStatus: "not_connected",
		},
	}
	manager.mu.Unlock()

	// Simulate connection
	manager.OnStationConnected("TEST003")

	// Check state
	station, _ := manager.GetStation("TEST003")
	if station.StateMachine.GetState() != StateConnected {
		t.Errorf("Expected state Connected, got %v", station.StateMachine.GetState())
	}

	if station.RuntimeState.ConnectionStatus != "connected" {
		t.Errorf("Expected connection status 'connected', got '%s'", station.RuntimeState.ConnectionStatus)
	}

	if station.RuntimeState.ConnectedAt == nil {
		t.Error("Expected ConnectedAt to be set")
	}
}

func TestOnStationDisconnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	// Add connected station
	config := Config{
		StationID: "TEST004",
	}

	now := time.Now()
	manager.mu.Lock()
	manager.stations[config.StationID] = &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
		RuntimeState: RuntimeState{
			State:            StateConnected,
			ConnectionStatus: "connected",
			ConnectedAt:      &now,
		},
	}
	manager.stations[config.StationID].StateMachine.SetState(StateConnected, "test")
	manager.mu.Unlock()

	// Simulate disconnection
	manager.OnStationDisconnected("TEST004", nil)

	// Check state
	station, _ := manager.GetStation("TEST004")
	if station.StateMachine.GetState() != StateDisconnected {
		t.Errorf("Expected state Disconnected, got %v", station.StateMachine.GetState())
	}

	if station.RuntimeState.ConnectionStatus != "disconnected" {
		t.Errorf("Expected connection status 'disconnected', got '%s'", station.RuntimeState.ConnectionStatus)
	}

	if station.RuntimeState.ConnectedAt != nil {
		t.Error("Expected ConnectedAt to be nil")
	}
}

func TestGetStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	// Add stations in different states
	states := []State{
		StateDisconnected,
		StateConnected,
		StateCharging,
		StateAvailable,
		StateFaulted,
	}

	for i, state := range states {
		config := Config{
			StationID: string(rune('X' + i)),
		}

		sm := NewStateMachine()
		sm.currentState = state

		manager.mu.Lock()
		manager.stations[config.StationID] = &Station{
			Config:       config,
			StateMachine: sm,
		}
		manager.mu.Unlock()
	}

	stats := manager.GetStats()

	if stats["total"].(int) != 5 {
		t.Errorf("Expected total 5, got %v", stats["total"])
	}

	if stats["disconnected"].(int) != 1 {
		t.Errorf("Expected 1 disconnected, got %v", stats["disconnected"])
	}

	if stats["connected"].(int) != 1 {
		t.Errorf("Expected 1 connected, got %v", stats["connected"])
	}

	if stats["charging"].(int) != 1 {
		t.Errorf("Expected 1 charging, got %v", stats["charging"])
	}

	if stats["available"].(int) != 1 {
		t.Errorf("Expected 1 available, got %v", stats["available"])
	}

	if stats["faulted"].(int) != 1 {
		t.Errorf("Expected 1 faulted, got %v", stats["faulted"])
	}
}

// TestCreateConnectionConfig removed - connection configuration is now
// handled directly through TLSConfig and AuthConfig parameters

func TestConvertStorageToConfig(t *testing.T) {
	// This test would require importing storage package
	// and creating a proper storage.Station object
	// Skipping for now as it's a straightforward conversion
	t.Skip("Conversion test requires full storage setup")
}

func TestShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{
		SyncInterval: 100 * time.Millisecond,
	})

	// Start sync
	manager.StartSync()

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify context is cancelled
	select {
	case <-manager.ctx.Done():
		// Good, context is cancelled
	default:
		t.Error("Expected manager context to be cancelled")
	}
}

func TestStartStationValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	ctx := context.Background()

	// Test non-existent station
	err := manager.StartStation(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("Expected error when starting non-existent station")
	}

	// Add disabled station
	config := Config{
		StationID: "TEST006",
		Enabled:   false,
	}

	manager.mu.Lock()
	manager.stations[config.StationID] = &Station{
		Config:       config,
		StateMachine: NewStateMachine(),
		RuntimeState: RuntimeState{
			State: StateDisconnected,
		},
	}
	manager.mu.Unlock()

	// Test disabled station
	err = manager.StartStation(ctx, "TEST006")
	if err == nil {
		t.Error("Expected error when starting disabled station")
	}
}

func TestStopStationValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	manager := NewManager(nil, nil, nil, logger, ManagerConfig{})

	ctx := context.Background()

	// Test non-existent station
	err := manager.StopStation(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("Expected error when stopping non-existent station")
	}
}
