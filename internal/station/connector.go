package station

import (
	"fmt"
	"sync"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
)

// ConnectorState represents the state of a connector following OCPP 1.6 spec
type ConnectorState string

const (
	ConnectorStateAvailable     ConnectorState = "Available"
	ConnectorStatePreparing     ConnectorState = "Preparing"
	ConnectorStateCharging      ConnectorState = "Charging"
	ConnectorStateSuspendedEVSE ConnectorState = "SuspendedEVSE"
	ConnectorStateSuspendedEV   ConnectorState = "SuspendedEV"
	ConnectorStateFinishing     ConnectorState = "Finishing"
	ConnectorStateReserved      ConnectorState = "Reserved"
	ConnectorStateUnavailable   ConnectorState = "Unavailable"
	ConnectorStateFaulted       ConnectorState = "Faulted"
)

// Connector represents a charging connector with its state and transaction
type Connector struct {
	ID              int
	Type            string
	MaxPower        int
	State           ConnectorState
	ErrorCode       v16.ChargePointErrorCode
	Info            string
	VendorErrorCode string
	Transaction     *Transaction
	Reservation     *Reservation
	LastStateChange time.Time
	mu              sync.RWMutex
	onStateChange   func(connectorID int, oldState, newState ConnectorState)
}

// Transaction represents an active charging transaction
type Transaction struct {
	ID              int
	IDTag           string
	ConnectorID     int
	StartTime       time.Time
	StartMeterValue int
	CurrentMeter    int
	StopTime        *time.Time
	StopMeterValue  *int
	StopReason      v16.Reason
	MeterValues     []MeterValueSample
	mu              sync.RWMutex
}

// MeterValueSample represents a meter value reading
type MeterValueSample struct {
	Timestamp time.Time
	Value     int
	Measurand string
	Unit      string
	Context   string
	Location  string
}

// Reservation represents a connector reservation
type Reservation struct {
	ID          int
	IDTag       string
	ExpiryDate  time.Time
	ParentIDTag string
}

// NewConnector creates a new connector
func NewConnector(id int, connectorType string, maxPower int) *Connector {
	return &Connector{
		ID:              id,
		Type:            connectorType,
		MaxPower:        maxPower,
		State:           ConnectorStateAvailable,
		ErrorCode:       v16.ChargePointErrorNoError,
		LastStateChange: time.Now(),
	}
}

// GetState returns the current connector state (thread-safe)
func (c *Connector) GetState() ConnectorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

// GetErrorCode returns the current error code (thread-safe)
func (c *Connector) GetErrorCode() v16.ChargePointErrorCode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ErrorCode
}

// GetTransaction returns the current transaction (thread-safe copy)
func (c *Connector) GetTransaction() *Transaction {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Transaction == nil {
		return nil
	}

	// Return a copy
	txCopy := *c.Transaction
	return &txCopy
}

// SetState changes the connector state
func (c *Connector) SetState(newState ConnectorState, errorCode v16.ChargePointErrorCode, info string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldState := c.State

	// Validate state transition
	if !c.canTransitionTo(newState) {
		return fmt.Errorf("invalid state transition from %s to %s", oldState, newState)
	}

	c.State = newState
	c.ErrorCode = errorCode
	c.Info = info
	c.LastStateChange = time.Now()

	// Trigger callback if set
	if c.onStateChange != nil {
		go c.onStateChange(c.ID, oldState, newState)
	}

	return nil
}

// canTransitionTo validates if a state transition is allowed
func (c *Connector) canTransitionTo(newState ConnectorState) bool {
	// Define valid transitions according to OCPP 1.6 spec
	validTransitions := map[ConnectorState][]ConnectorState{
		ConnectorStateAvailable: {
			ConnectorStatePreparing,
			ConnectorStateReserved,
			ConnectorStateUnavailable,
			ConnectorStateFaulted,
		},
		ConnectorStatePreparing: {
			ConnectorStateCharging,
			ConnectorStateAvailable,
			ConnectorStateSuspendedEVSE,
			ConnectorStateSuspendedEV,
			ConnectorStateFaulted,
		},
		ConnectorStateCharging: {
			ConnectorStateSuspendedEVSE,
			ConnectorStateSuspendedEV,
			ConnectorStateFinishing,
			ConnectorStateFaulted,
		},
		ConnectorStateSuspendedEVSE: {
			ConnectorStateCharging,
			ConnectorStateFinishing,
			ConnectorStateFaulted,
		},
		ConnectorStateSuspendedEV: {
			ConnectorStateCharging,
			ConnectorStateFinishing,
			ConnectorStateFaulted,
		},
		ConnectorStateFinishing: {
			ConnectorStateAvailable,
			ConnectorStateFaulted,
		},
		ConnectorStateReserved: {
			ConnectorStateAvailable,
			ConnectorStatePreparing,
			ConnectorStateFaulted,
		},
		ConnectorStateUnavailable: {
			ConnectorStateAvailable,
			ConnectorStateFaulted,
		},
		ConnectorStateFaulted: {
			ConnectorStateAvailable,
			ConnectorStateUnavailable,
		},
	}

	allowedStates, exists := validTransitions[c.State]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}

	return false
}

// StartTransaction starts a new transaction on this connector
func (c *Connector) StartTransaction(transactionID int, idTag string, meterStart int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Transaction != nil {
		return fmt.Errorf("connector %d already has an active transaction", c.ID)
	}

	if c.State != ConnectorStatePreparing && c.State != ConnectorStateAvailable {
		return fmt.Errorf("connector %d is not ready to start transaction (state: %s)", c.ID, c.State)
	}

	c.Transaction = &Transaction{
		ID:              transactionID,
		IDTag:           idTag,
		ConnectorID:     c.ID,
		StartTime:       time.Now(),
		StartMeterValue: meterStart,
		CurrentMeter:    meterStart,
		MeterValues:     make([]MeterValueSample, 0),
	}

	return nil
}

// StopTransaction stops the active transaction
func (c *Connector) StopTransaction(meterStop int, reason v16.Reason) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Transaction == nil {
		return fmt.Errorf("connector %d has no active transaction", c.ID)
	}

	now := time.Now()
	c.Transaction.StopTime = &now
	c.Transaction.StopMeterValue = &meterStop
	c.Transaction.StopReason = reason

	return nil
}

// ClearTransaction removes the transaction from the connector
func (c *Connector) ClearTransaction() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Transaction = nil
}

// AddMeterValue adds a meter value sample to the current transaction
func (c *Connector) AddMeterValue(sample MeterValueSample) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Transaction == nil {
		return fmt.Errorf("connector %d has no active transaction", c.ID)
	}

	c.Transaction.mu.Lock()
	c.Transaction.MeterValues = append(c.Transaction.MeterValues, sample)
	c.Transaction.CurrentMeter = sample.Value
	c.Transaction.mu.Unlock()

	return nil
}

// UpdateMeter updates the current meter value
func (c *Connector) UpdateMeter(value int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Transaction == nil {
		return fmt.Errorf("connector %d has no active transaction", c.ID)
	}

	c.Transaction.mu.Lock()
	c.Transaction.CurrentMeter = value
	c.Transaction.mu.Unlock()

	return nil
}

// Reserve reserves the connector
func (c *Connector) Reserve(reservationID int, idTag string, expiryDate time.Time, parentIDTag string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.State != ConnectorStateAvailable {
		return fmt.Errorf("connector %d is not available for reservation (state: %s)", c.ID, c.State)
	}

	c.Reservation = &Reservation{
		ID:          reservationID,
		IDTag:       idTag,
		ExpiryDate:  expiryDate,
		ParentIDTag: parentIDTag,
	}

	return nil
}

// CancelReservation cancels the reservation
func (c *Connector) CancelReservation() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Reservation == nil {
		return fmt.Errorf("connector %d has no active reservation", c.ID)
	}

	c.Reservation = nil
	return nil
}

// IsReserved checks if connector is reserved
func (c *Connector) IsReserved() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Reservation != nil
}

// IsReservedFor checks if connector is reserved for a specific ID tag
func (c *Connector) IsReservedFor(idTag string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Reservation == nil {
		return false
	}

	// Check if reservation has expired
	if time.Now().After(c.Reservation.ExpiryDate) {
		return false
	}

	return c.Reservation.IDTag == idTag || c.Reservation.ParentIDTag == idTag
}

// HasActiveTransaction checks if connector has an active transaction
func (c *Connector) HasActiveTransaction() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Transaction != nil && c.Transaction.StopTime == nil
}

// IsAvailable checks if connector is available for charging
func (c *Connector) IsAvailable() bool {
	state := c.GetState()
	return state == ConnectorStateAvailable || state == ConnectorStatePreparing
}

// IsCharging checks if connector is actively charging
func (c *Connector) IsCharging() bool {
	state := c.GetState()
	return state == ConnectorStateCharging
}

// IsFaulted checks if connector is faulted
func (c *Connector) IsFaulted() bool {
	return c.GetState() == ConnectorStateFaulted
}

// GetTransactionDuration returns the duration of the current transaction
func (c *Connector) GetTransactionDuration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Transaction == nil {
		return 0
	}

	if c.Transaction.StopTime != nil {
		return c.Transaction.StopTime.Sub(c.Transaction.StartTime)
	}

	return time.Since(c.Transaction.StartTime)
}

// GetEnergyDelivered returns the energy delivered in Wh
func (c *Connector) GetEnergyDelivered() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Transaction == nil {
		return 0
	}

	return c.Transaction.CurrentMeter - c.Transaction.StartMeterValue
}
