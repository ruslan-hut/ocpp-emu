# Task 2.5: Enhanced State Machine for Charging Sessions

## Overview

This document describes the implementation of the enhanced state machine for managing charging sessions in the OCPP emulator. The state machine follows the OCPP 1.6 specification for connector states and charging session lifecycle.

## Architecture

The charging session state machine consists of three main components:

### 1. Connector (`internal/station/connector.go`)

The `Connector` struct represents a physical charging connector with its state, transactions, and reservations.

**Key Features:**
- Thread-safe operations using `sync.RWMutex`
- OCPP 1.6 compliant state transitions
- Transaction tracking with meter values
- Reservation support
- State change callbacks for reactive updates

**Connector States:**
```go
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
```

**State Transition Rules:**

Valid transitions are enforced by the `canTransitionTo()` method according to OCPP 1.6 spec:

- **Available** → Preparing, Reserved, Unavailable, Faulted
- **Preparing** → Charging, Available, SuspendedEVSE, SuspendedEV, Faulted
- **Charging** → SuspendedEVSE, SuspendedEV, Finishing, Faulted
- **SuspendedEVSE** → Charging, Finishing, Faulted
- **SuspendedEV** → Charging, Finishing, Faulted
- **Finishing** → Available, Faulted
- **Reserved** → Available, Preparing, Faulted
- **Unavailable** → Available, Faulted
- **Faulted** → Available, Unavailable

### 2. SessionManager (`internal/station/session.go`)

The `SessionManager` orchestrates multiple connectors and manages the complete charging session lifecycle.

**Key Features:**
- Manages multiple connectors per station
- Handles authorization workflow
- Coordinates transaction start/stop
- Automatic meter value simulation
- OCPP message sending via callbacks
- Graceful shutdown with active transaction cleanup

**Charging Session Workflow:**

1. **Authorization** (optional):
   - Call `Authorize(idTag)` to validate user ID
   - Falls back to offline authorization if callback not set

2. **Start Charging**:
   - Call `StartCharging(connectorID, idTag)`
   - Validates connector availability
   - Checks reservations
   - Transitions: Available → Preparing → Charging
   - Sends StatusNotification and StartTransaction to CSMS
   - Starts automatic meter value simulation

3. **During Charging**:
   - Meter values sent every 60 seconds
   - Simulates energy consumption (5-7.5 kW)
   - Can update connector state (e.g., suspend)

4. **Stop Charging**:
   - Call `StopCharging(connectorID, reason)`
   - Transitions: Charging → Finishing → Available
   - Sends StopTransaction to CSMS
   - Stops meter value simulation

**Callback Interface:**

The SessionManager uses callbacks to send OCPP messages:

```go
type SessionManager struct {
    // Callbacks for OCPP message sending
    SendAuthorize           func(idTag string) (*v16.AuthorizeResponse, error)
    SendStartTransaction    func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error)
    SendStopTransaction     func(transactionID int, idTag string, meterStop int, timestamp time.Time, reason v16.Reason) (*v16.StopTransactionResponse, error)
    SendStatusNotification  func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error
    SendMeterValues         func(connectorID int, transactionID *int, meterValues []v16.MeterValue) error
}
```

### 3. Integration with Station Manager (`internal/station/manager.go`)

The Station Manager integrates SessionManager with OCPP v16 handlers:

**Integration Points:**

1. **Station Creation** (line 283-297):
   - Each station gets a SessionManager instance
   - Callbacks are wired to OCPP message handlers

2. **OCPP Message Handlers** (line 92-212):
   - `RemoteStartTransaction` → calls `SessionManager.StartCharging()`
   - `RemoteStopTransaction` → calls `SessionManager.StopCharging()`
   - `ChangeAvailability` → calls `SessionManager.ChangeAvailability()`

3. **Callback Wiring** (line 260-316):
   - `setupSessionManagerCallbacks()` wires SessionManager callbacks to v16 handler
   - Currently implements:
     - `SendStatusNotification` - sends status changes to CSMS
     - `SendMeterValues` - sends meter readings to CSMS
   - Offline mode for Authorize/Start/Stop (no async response handling yet)

## Transaction Tracking

### Transaction Structure

```go
type Transaction struct {
    ID              int              // Transaction ID (from CSMS or local)
    IDTag           string           // User identification tag
    ConnectorID     int              // Connector this transaction is on
    StartTime       time.Time        // When transaction started
    StartMeterValue int              // Initial meter reading (Wh)
    CurrentMeter    int              // Current meter reading (Wh)
    StopTime        *time.Time       // When transaction stopped (nil if active)
    StopMeterValue  *int             // Final meter reading (nil if active)
    StopReason      v16.Reason       // Reason for stopping
    MeterValues     []MeterValueSample // Historical meter readings
}
```

### Meter Value Simulation

- Simulates power consumption between 5-7.5 kW
- Meter values sent every 60 seconds
- Includes both energy (Wh) and power (W) readings
- Automatic cleanup when transaction stops

## Reservation Support

Connectors can be reserved for specific ID tags:

```go
type Reservation struct {
    ID          int        // Reservation ID
    IDTag       string     // ID tag this reservation is for
    ExpiryDate  time.Time  // When reservation expires
    ParentIDTag string     // Optional parent ID tag
}
```

**Reservation Workflow:**
1. `Reserve(reservationID, idTag, expiryDate, parentIDTag)` - creates reservation
2. `IsReserved()` - checks if reserved
3. `IsReservedFor(idTag)` - checks if reserved for specific tag (respects expiry)
4. `CancelReservation()` - removes reservation

Reservations prevent unauthorized users from starting charging sessions.

## Testing

Comprehensive test coverage includes:

### Connector Tests (`connector_test.go` - 17 tests)

- State transition validation
- Transaction lifecycle
- Reservation management
- Meter value updates
- State change callbacks
- Thread safety

### SessionManager Tests (`session_test.go` - 12 tests)

- Multi-connector management
- Authorization workflow
- Start/stop charging
- Availability changes
- Shutdown with active transactions
- Transaction ID management

All tests pass with 100% success rate.

## Usage Examples

### Basic Charging Session

```go
// Create session manager
sm := NewSessionManager("CP001", connectorConfigs, logger)

// Set up callbacks
sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, ...) error {
    // Send status to CSMS
    return nil
}

// Start charging
transactionID, err := sm.StartCharging(1, "TAG123")
if err != nil {
    log.Fatal(err)
}

// ... charging happens ...

// Stop charging
err = sm.StopCharging(1, v16.ReasonLocal)
if err != nil {
    log.Fatal(err)
}
```

### Change Availability

```go
// Make connector unavailable
err := sm.ChangeAvailability(1, "Inoperative")
if err != nil {
    // Returns error if charging (should schedule for later)
}

// Make connector available again
err = sm.ChangeAvailability(1, "Operative")
```

### Graceful Shutdown

```go
// Shutdown stops all active transactions
ctx := context.Background()
err := sm.Shutdown(ctx)
// All transactions stopped with ReasonReboot
```

## Implementation Notes

### Thread Safety

- All public methods use appropriate locking
- `sync.RWMutex` allows concurrent reads
- State changes are atomic
- Transaction operations are protected

### Offline vs Online Mode

**Offline Mode** (current implementation):
- Authorization always accepts
- Transaction IDs generated locally
- No communication with CSMS for start/stop

**Online Mode** (future):
- Requires async request/response handling
- Transaction IDs from CSMS
- Authorization status from CSMS
- Needs request tracking mechanism

### State Change Notifications

State changes trigger callbacks asynchronously:

```go
connector.onStateChange = func(connectorID int, oldState, newState ConnectorState) {
    // React to state changes
    // Runs in separate goroutine
}
```

This allows reactive programming patterns and decoupling.

## Future Enhancements

1. **Async Request/Response Handling**
   - Implement request tracking for Call/CallResult matching
   - Enable online mode for Authorize, StartTransaction, StopTransaction
   - Add timeout handling for CSMS responses

2. **Persistent Transactions**
   - Save active transactions to MongoDB
   - Restore transactions on restart
   - Handle crash recovery

3. **Advanced Features**
   - Smart charging profiles
   - Local authorization list
   - Charging schedules
   - Energy limit enforcement

4. **Metrics and Monitoring**
   - Transaction statistics
   - State transition metrics
   - Error rate tracking
   - Performance monitoring

## References

- OCPP 1.6 Specification - Section 4.2 (Charge Point Status)
- OCPP 1.6 Specification - Section 5.8 (Start Transaction)
- OCPP 1.6 Specification - Section 5.15 (Stop Transaction)
- Project Plan: `PLAN.md` - Task 2.5
