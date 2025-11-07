package station

import (
	"sync"
	"time"
)

// State represents the state of a charging station
type State string

const (
	// StateUnknown is the initial unknown state
	StateUnknown State = "unknown"

	// StateDisconnected means station is not connected to CSMS
	StateDisconnected State = "disconnected"

	// StateConnecting means station is attempting to connect
	StateConnecting State = "connecting"

	// StateConnected means station is connected but not registered
	StateConnected State = "connected"

	// StateRegistered means station has completed BootNotification
	StateRegistered State = "registered"

	// StateAvailable means station is ready to charge
	StateAvailable State = "available"

	// StateCharging means station is actively charging
	StateCharging State = "charging"

	// StateFaulted means station has an error
	StateFaulted State = "faulted"

	// StateUnavailable means station is unavailable
	StateUnavailable State = "unavailable"

	// StateStopping means station is being stopped
	StateStopping State = "stopping"
)

// StateMachine manages state transitions for a station
type StateMachine struct {
	currentState  State
	previousState State
	stateHistory  []StateTransition
	mu            sync.RWMutex
}

// StateTransition represents a state change
type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState:  StateUnknown,
		previousState: StateUnknown,
		stateHistory:  make([]StateTransition, 0, 100),
	}
}

// GetState returns the current state
func (sm *StateMachine) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// GetPreviousState returns the previous state
func (sm *StateMachine) GetPreviousState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.previousState
}

// SetState transitions to a new state
func (sm *StateMachine) SetState(newState State, reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.currentState == newState {
		return
	}

	transition := StateTransition{
		From:      sm.currentState,
		To:        newState,
		Timestamp: time.Now(),
		Reason:    reason,
	}

	sm.previousState = sm.currentState
	sm.currentState = newState
	sm.stateHistory = append(sm.stateHistory, transition)

	// Keep only last 100 transitions
	if len(sm.stateHistory) > 100 {
		sm.stateHistory = sm.stateHistory[1:]
	}
}

// GetHistory returns the state history
func (sm *StateMachine) GetHistory() []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy
	history := make([]StateTransition, len(sm.stateHistory))
	copy(history, sm.stateHistory)
	return history
}

// CanTransition checks if a transition is valid
func (sm *StateMachine) CanTransition(to State) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Define valid transitions
	validTransitions := map[State][]State{
		StateUnknown:      {StateDisconnected, StateConnecting},
		StateDisconnected: {StateConnecting, StateFaulted},
		StateConnecting:   {StateConnected, StateDisconnected, StateFaulted},
		StateConnected:    {StateRegistered, StateDisconnected, StateFaulted},
		StateRegistered:   {StateAvailable, StateDisconnected, StateFaulted},
		StateAvailable:    {StateCharging, StateUnavailable, StateDisconnected, StateFaulted, StateStopping},
		StateCharging:     {StateAvailable, StateDisconnected, StateFaulted, StateStopping},
		StateFaulted:      {StateAvailable, StateDisconnected, StateUnavailable},
		StateUnavailable:  {StateAvailable, StateDisconnected},
		StateStopping:     {StateDisconnected},
	}

	allowedStates, exists := validTransitions[sm.currentState]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return true
		}
	}

	return false
}

// IsConnected returns true if station is in a connected state
func (sm *StateMachine) IsConnected() bool {
	state := sm.GetState()
	return state == StateConnected ||
		state == StateRegistered ||
		state == StateAvailable ||
		state == StateCharging
}

// IsOperational returns true if station can charge
func (sm *StateMachine) IsOperational() bool {
	state := sm.GetState()
	return state == StateAvailable || state == StateCharging
}
