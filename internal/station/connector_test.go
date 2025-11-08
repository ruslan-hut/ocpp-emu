package station

import (
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
)

func TestNewConnector(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	if connector.ID != 1 {
		t.Errorf("Expected ID 1, got %d", connector.ID)
	}

	if connector.Type != "Type2" {
		t.Errorf("Expected Type 'Type2', got %s", connector.Type)
	}

	if connector.MaxPower != 22000 {
		t.Errorf("Expected MaxPower 22000, got %d", connector.MaxPower)
	}

	if connector.State != ConnectorStateAvailable {
		t.Errorf("Expected state Available, got %s", connector.State)
	}

	if connector.ErrorCode != v16.ChargePointErrorNoError {
		t.Errorf("Expected error code NoError, got %s", connector.ErrorCode)
	}
}

func TestConnector_StateTransitions(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	tests := []struct {
		name      string
		fromState ConnectorState
		toState   ConnectorState
		wantErr   bool
	}{
		{
			name:      "Available to Preparing",
			fromState: ConnectorStateAvailable,
			toState:   ConnectorStatePreparing,
			wantErr:   false,
		},
		{
			name:      "Preparing to Charging",
			fromState: ConnectorStatePreparing,
			toState:   ConnectorStateCharging,
			wantErr:   false,
		},
		{
			name:      "Charging to Finishing",
			fromState: ConnectorStateCharging,
			toState:   ConnectorStateFinishing,
			wantErr:   false,
		},
		{
			name:      "Finishing to Available",
			fromState: ConnectorStateFinishing,
			toState:   ConnectorStateAvailable,
			wantErr:   false,
		},
		{
			name:      "Available to Charging (invalid)",
			fromState: ConnectorStateAvailable,
			toState:   ConnectorStateCharging,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial state
			connector.State = tt.fromState

			// Attempt transition
			err := connector.SetState(tt.toState, v16.ChargePointErrorNoError, "")

			if (err != nil) != tt.wantErr {
				t.Errorf("SetState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && connector.GetState() != tt.toState {
				t.Errorf("Expected state %s, got %s", tt.toState, connector.GetState())
			}
		})
	}
}

func TestConnector_StartTransaction(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	err := connector.StartTransaction(123, "TAG123", 1000)
	if err != nil {
		t.Fatalf("StartTransaction failed: %v", err)
	}

	if !connector.HasActiveTransaction() {
		t.Error("Expected active transaction")
	}

	tx := connector.GetTransaction()
	if tx == nil {
		t.Fatal("Transaction is nil")
	}

	if tx.ID != 123 {
		t.Errorf("Expected transaction ID 123, got %d", tx.ID)
	}

	if tx.IDTag != "TAG123" {
		t.Errorf("Expected IDTag 'TAG123', got %s", tx.IDTag)
	}

	if tx.StartMeterValue != 1000 {
		t.Errorf("Expected start meter 1000, got %d", tx.StartMeterValue)
	}
}

func TestConnector_StartTransaction_AlreadyActive(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start first transaction
	err := connector.StartTransaction(123, "TAG123", 1000)
	if err != nil {
		t.Fatalf("First StartTransaction failed: %v", err)
	}

	// Try to start second transaction
	err = connector.StartTransaction(456, "TAG456", 2000)
	if err == nil {
		t.Error("Expected error when starting second transaction")
	}
}

func TestConnector_StopTransaction(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	err := connector.StartTransaction(123, "TAG123", 1000)
	if err != nil {
		t.Fatalf("StartTransaction failed: %v", err)
	}

	// Stop transaction
	err = connector.StopTransaction(5000, v16.ReasonLocal)
	if err != nil {
		t.Fatalf("StopTransaction failed: %v", err)
	}

	tx := connector.GetTransaction()
	if tx == nil {
		t.Fatal("Transaction is nil after stop")
	}

	if tx.StopTime == nil {
		t.Error("Stop time is nil")
	}

	if tx.StopMeterValue == nil {
		t.Error("Stop meter value is nil")
	} else if *tx.StopMeterValue != 5000 {
		t.Errorf("Expected stop meter 5000, got %d", *tx.StopMeterValue)
	}

	if tx.StopReason != v16.ReasonLocal {
		t.Errorf("Expected stop reason Local, got %s", tx.StopReason)
	}
}

func TestConnector_UpdateMeter(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	connector.StartTransaction(123, "TAG123", 1000)

	// Update meter
	err := connector.UpdateMeter(3500)
	if err != nil {
		t.Fatalf("UpdateMeter failed: %v", err)
	}

	tx := connector.GetTransaction()
	if tx.CurrentMeter != 3500 {
		t.Errorf("Expected current meter 3500, got %d", tx.CurrentMeter)
	}
}

func TestConnector_AddMeterValue(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	connector.StartTransaction(123, "TAG123", 1000)

	// Add meter value
	sample := MeterValueSample{
		Timestamp: time.Now(),
		Value:     2500,
		Measurand: "Energy.Active.Import.Register",
		Unit:      "Wh",
		Context:   "Sample.Periodic",
		Location:  "Outlet",
	}

	err := connector.AddMeterValue(sample)
	if err != nil {
		t.Fatalf("AddMeterValue failed: %v", err)
	}

	tx := connector.GetTransaction()
	if len(tx.MeterValues) != 1 {
		t.Errorf("Expected 1 meter value, got %d", len(tx.MeterValues))
	}

	if tx.CurrentMeter != 2500 {
		t.Errorf("Expected current meter 2500, got %d", tx.CurrentMeter)
	}
}

func TestConnector_Reserve(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	expiryDate := time.Now().Add(1 * time.Hour)
	err := connector.Reserve(1, "TAG123", expiryDate, "")
	if err != nil {
		t.Fatalf("Reserve failed: %v", err)
	}

	if !connector.IsReserved() {
		t.Error("Expected connector to be reserved")
	}

	if !connector.IsReservedFor("TAG123") {
		t.Error("Expected connector to be reserved for TAG123")
	}

	if connector.IsReservedFor("TAG456") {
		t.Error("Expected connector NOT to be reserved for TAG456")
	}
}

func TestConnector_ReservationExpiry(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	// Reserve with expired time
	expiryDate := time.Now().Add(-1 * time.Hour)
	connector.Reserve(1, "TAG123", expiryDate, "")

	// Should not be reserved (expired)
	if connector.IsReservedFor("TAG123") {
		t.Error("Expected reservation to be expired")
	}
}

func TestConnector_CancelReservation(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	// Reserve
	expiryDate := time.Now().Add(1 * time.Hour)
	connector.Reserve(1, "TAG123", expiryDate, "")

	// Cancel
	err := connector.CancelReservation()
	if err != nil {
		t.Fatalf("CancelReservation failed: %v", err)
	}

	if connector.IsReserved() {
		t.Error("Expected connector to not be reserved after cancellation")
	}
}

func TestConnector_GetTransactionDuration(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	connector.StartTransaction(123, "TAG123", 1000)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	duration := connector.GetTransactionDuration()
	if duration < 100*time.Millisecond {
		t.Errorf("Expected duration >= 100ms, got %v", duration)
	}
}

func TestConnector_GetEnergyDelivered(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	connector.StartTransaction(123, "TAG123", 1000)

	// Update meter
	connector.UpdateMeter(5000)

	energy := connector.GetEnergyDelivered()
	if energy != 4000 {
		t.Errorf("Expected energy delivered 4000 Wh, got %d", energy)
	}
}

func TestConnector_IsAvailable(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	// Available state
	connector.State = ConnectorStateAvailable
	if !connector.IsAvailable() {
		t.Error("Expected connector to be available")
	}

	// Preparing state
	connector.State = ConnectorStatePreparing
	if !connector.IsAvailable() {
		t.Error("Expected connector to be available in Preparing state")
	}

	// Charging state
	connector.State = ConnectorStateCharging
	if connector.IsAvailable() {
		t.Error("Expected connector to not be available in Charging state")
	}
}

func TestConnector_IsCharging(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	connector.State = ConnectorStateCharging
	if !connector.IsCharging() {
		t.Error("Expected connector to be charging")
	}

	connector.State = ConnectorStateAvailable
	if connector.IsCharging() {
		t.Error("Expected connector to not be charging")
	}
}

func TestConnector_IsFaulted(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	connector.State = ConnectorStateFaulted
	if !connector.IsFaulted() {
		t.Error("Expected connector to be faulted")
	}

	connector.State = ConnectorStateAvailable
	if connector.IsFaulted() {
		t.Error("Expected connector to not be faulted")
	}
}

func TestConnector_StateChangeCallback(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)

	var callbackCalled bool
	var oldState, newState ConnectorState

	connector.onStateChange = func(connectorID int, old, new ConnectorState) {
		callbackCalled = true
		oldState = old
		newState = new
	}

	connector.SetState(ConnectorStatePreparing, v16.ChargePointErrorNoError, "")

	// Give callback time to execute (it runs in goroutine)
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Expected state change callback to be called")
	}

	if oldState != ConnectorStateAvailable {
		t.Errorf("Expected old state Available, got %s", oldState)
	}

	if newState != ConnectorStatePreparing {
		t.Errorf("Expected new state Preparing, got %s", newState)
	}
}

func TestConnector_ClearTransaction(t *testing.T) {
	connector := NewConnector(1, "Type2", 22000)
	connector.State = ConnectorStatePreparing

	// Start transaction
	connector.StartTransaction(123, "TAG123", 1000)

	if !connector.HasActiveTransaction() {
		t.Error("Expected active transaction")
	}

	// Clear transaction
	connector.ClearTransaction()

	if connector.HasActiveTransaction() {
		t.Error("Expected no active transaction after clear")
	}

	if connector.GetTransaction() != nil {
		t.Error("Expected transaction to be nil after clear")
	}
}
