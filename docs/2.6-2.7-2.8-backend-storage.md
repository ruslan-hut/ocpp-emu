# Tasks 2.6, 2.7, 2.8: Backend Storage Features

## Overview

This document describes the backend storage features implemented for the OCPP emulator, including message persistence, transaction tracking, and runtime state synchronization with MongoDB.

## Completed Tasks

- **Task 2.6**: ✅ Persist messages to MongoDB (messages collection)
- **Task 2.7**: ✅ Persist transactions to MongoDB (transactions collection)
- **Task 2.8**: ✅ Implement station runtime state sync to MongoDB

## Architecture

### Storage Layer Structure

```
internal/storage/
├── mongodb.go          # MongoDB client and connection management
├── models.go           # Data models for MongoDB collections
└── transactions.go     # Transaction repository with CRUD operations

internal/logging/
└── message_logger.go   # Message logging with buffering and persistence
```

### MongoDB Collections

1. **Messages Collection** (`ocpp_messages`)
   - Stores all OCPP messages (Call, CallResult, CallError)
   - Buffered writes for performance
   - Indexed for efficient querying

2. **Transactions Collection** (`transactions`)
   - Tracks charging transactions from start to completion
   - Includes energy consumption and billing data
   - Indexed by transaction ID, station ID, and status

3. **Stations Collection** (`stations`)
   - Stores station configurations and runtime state
   - Synchronized periodically
   - Includes connector states and active transactions

4. **Sessions Collection** (`sessions`)
   - WebSocket session tracking
   - Connection history

5. **Meter Values Collection** (`meter_values`)
   - Time-series data for power/energy readings
   - Optimized for analytical queries

## Task 2.6: Message Persistence

### Implementation

The `MessageLogger` (`internal/logging/message_logger.go`) provides comprehensive message logging with:

**Features:**
- **Buffered Writes**: Messages buffered in memory channel
- **Batch Inserts**: MongoDB bulk inserts for efficiency
- **Configurable Flushing**: Periodic or size-based flushing
- **Statistics Tracking**: Message counts, dropped messages, etc.
- **Filtering & Search**: Query messages by station, type, action, date range

**Message Flow:**
```
OCPP Message → LogMessage() → Buffer → Batch → MongoDB
                                  ↓
                            Periodic Flush (5s)
                                  ↓
                            Batch Size (100)
```

**Configuration:**
```go
config := LoggerConfig{
    BufferSize:    1000,              // Channel buffer size
    BatchSize:     100,               // Messages per batch insert
    FlushInterval: 5 * time.Second,   // Periodic flush interval
}
```

**Usage Example:**
```go
// Create message logger
messageLogger := logging.NewMessageLogger(db, logger, config)
messageLogger.Start()

// Log OCPP message
err := messageLogger.LogMessage(
    "CP001",        // Station ID
    "sent",         // Direction
    callMessage,    // OCPP message
    "ocpp1.6",      // Protocol version
)

// Query messages
messages, err := messageLogger.GetMessages(ctx, MessageFilter{
    StationID:   "CP001",
    Direction:   "received",
    MessageType: "Call",
    Limit:       100,
})

// Cleanup old messages
deleted, err := messageLogger.DeleteOldMessages(ctx, 30*24*time.Hour)
```

**Message Model:**
```go
type Message struct {
    ID               string                 // MongoDB _id
    StationID        string                 // Charging station ID
    Direction        string                 // "sent" or "received"
    MessageType      string                 // "Call", "CallResult", "CallError"
    Action           string                 // OCPP action (e.g., "BootNotification")
    MessageID        string                 // Unique message ID
    ProtocolVersion  string                 // "1.6", "2.0.1", "2.1"
    Payload          map[string]interface{} // Message payload
    Timestamp        time.Time              // When message was sent/received
    CorrelationID    string                 // Links request with response
    ErrorCode        string                 // For CallError messages
    ErrorDescription string                 // For CallError messages
    CreatedAt        time.Time              // When record was created
}
```

**Performance:**
- Buffering prevents blocking main message flow
- Batch inserts reduce database round-trips
- Non-blocking buffer (messages dropped if full)
- Background processing via goroutine

**Statistics:**
```go
stats := messageLogger.GetStats()
// Returns: TotalMessages, SentMessages, ReceivedMessages,
//          CallMessages, CallResultMessages, CallErrorMessages,
//          BufferedMessages, DroppedMessages, LastFlush, FlushCount
```

## Task 2.7: Transaction Persistence

### Implementation

The `TransactionRepository` (`internal/storage/transactions.go`) provides complete transaction lifecycle management.

**Features:**
- **Automatic Persistence**: Transactions saved on start, updated on stop
- **Energy Tracking**: Calculates energy consumed (meterStop - meterStart)
- **Status Management**: Active, completed, failed states
- **Comprehensive Queries**: By station, ID tag, date range
- **Statistics**: Total energy, average consumption, counts

**Transaction Lifecycle:**
```
StartCharging()
    ↓
Create Transaction (status: "active")
    ↓
[Charging... meter values updating]
    ↓
StopCharging()
    ↓
Complete Transaction (status: "completed")
    ↓
Calculate energy consumed
```

**Transaction Model:**
```go
type Transaction struct {
    ID              string    // MongoDB _id
    TransactionID   int       // OCPP transaction ID
    StationID       string    // Charging station ID
    ConnectorID     int       // Connector number
    IDTag           string    // User authorization tag
    StartTimestamp  time.Time // When charging started
    StopTimestamp   time.Time // When charging stopped
    MeterStart      int       // Initial meter reading (Wh)
    MeterStop       int       // Final meter reading (Wh)
    EnergyConsumed  int       // Total energy delivered (Wh)
    Reason          string    // Stop reason (Local, Remote, etc.)
    Status          string    // "active", "completed", "failed"
    ProtocolVersion string    // OCPP version
    CreatedAt       time.Time // Record creation time
    UpdatedAt       time.Time // Last update time
}
```

**Integration with SessionManager:**

The SessionManager now automatically persists transactions:

```go
// session.go - StartCharging method
if sm.transactionRepo != nil {
    dbTransaction := storage.Transaction{
        TransactionID:   transactionID,
        StationID:       sm.stationID,
        ConnectorID:     connectorID,
        IDTag:           idTag,
        StartTimestamp:  time.Now(),
        MeterStart:      meterStart,
        Status:          "active",
        ProtocolVersion: sm.protocolVersion,
    }
    sm.transactionRepo.Create(ctx, dbTransaction)
}

// session.go - StopCharging method
if sm.transactionRepo != nil {
    sm.transactionRepo.Complete(ctx, tx.ID, sm.stationID, meterStop, string(reason))
}
```

**Repository Methods:**

```go
// Create a new transaction
Create(ctx, transaction) error

// Update transaction
Update(ctx, transactionID, stationID, updates) error

// Mark transaction as completed
Complete(ctx, transactionID, stationID, meterStop, reason) error

// Mark transaction as failed
MarkAsFailed(ctx, transactionID, stationID, reason) error

// Retrieve transactions
GetByID(ctx, transactionID, stationID) (*Transaction, error)
GetActive(ctx, stationID) ([]Transaction, error)
GetActiveByConnector(ctx, stationID, connectorID) (*Transaction, error)
GetByStation(ctx, stationID, limit, skip) ([]Transaction, error)
GetByIDTag(ctx, idTag, limit, skip) ([]Transaction, error)
GetByDateRange(ctx, stationID, startDate, endDate, limit, skip) ([]Transaction, error)

// Statistics
GetStats(ctx, stationID) (map[string]interface{}, error)
Count(ctx, filter) (int64, error)

// Cleanup
Delete(ctx, transactionID, stationID) error
DeleteOld(ctx, olderThan time.Duration) (int64, error)
```

**Usage Examples:**

```go
// Create repository
repo := storage.NewTransactionRepository(db)

// Get active transactions for a station
active, err := repo.GetActive(ctx, "CP001")

// Get transaction statistics
stats, err := repo.GetStats(ctx, "CP001")
// Returns: total, active, completed, failed, total_energy_wh, average_energy_wh

// Query by date range
transactions, err := repo.GetByDateRange(
    ctx,
    "CP001",
    startDate,
    endDate,
    100, // limit
    0,   // skip
)

// Cleanup old transactions (older than 90 days)
deleted, err := repo.DeleteOld(ctx, 90*24*time.Hour)
```

**Wire-up in Manager:**

```go
// manager.go - LoadStations
transactionRepo := storage.NewTransactionRepository(m.db)
sessionManager.SetTransactionRepository(transactionRepo)
sessionManager.SetProtocolVersion(config.ProtocolVersion)
```

## Task 2.8: Runtime State Synchronization

### Implementation

Enhanced the existing sync mechanism to include runtime state fields:

**Synchronized Fields:**

1. **Connection State:**
   - `ConnectionStatus`: "connected", "disconnected", "connecting", etc.
   - `LastHeartbeat`: Timestamp of last heartbeat received
   - `LastError`: Most recent error message

2. **Connector States:**
   - `Status`: Current state (Available, Charging, etc.)
   - `CurrentTransactionID`: Active transaction ID (if any)

**Sync Mechanism:**

```
Station Manager
    ↓
Periodic Sync Loop (every 30s)
    ↓
For each station:
    ├─ Copy RuntimeState fields
    ├─ Get connector states from SessionManager
    ├─ Get active transaction IDs
    └─ Update MongoDB with ReplaceOne (upsert)
```

**Enhanced saveStationToDB:**

```go
func (m *Manager) saveStationToDB(ctx context.Context, station *Station) error {
    station.mu.RLock()
    dbStation := m.convertConfigToStorage(station.Config)

    // Add runtime state fields
    dbStation.ConnectionStatus = station.RuntimeState.ConnectionStatus
    dbStation.LastHeartbeat = station.RuntimeState.LastHeartbeat
    dbStation.LastError = station.RuntimeState.LastError
    station.mu.RUnlock()

    // Update connector states from SessionManager
    if station.SessionManager != nil {
        connectors := station.SessionManager.GetAllConnectors()
        for _, connector := range connectors {
            // Update status and current transaction ID
            dbStation.Connectors[i].Status = string(connector.GetState())

            if connector.HasActiveTransaction() {
                tx := connector.GetTransaction()
                txID := tx.ID
                dbStation.Connectors[i].CurrentTransactionID = &txID
            } else {
                dbStation.Connectors[i].CurrentTransactionID = nil
            }
        }
    }

    // Save to MongoDB
    collection.ReplaceOne(ctx, filter, dbStation, opts)
}
```

**Sync Configuration:**

```go
// Manager initialization
config := ManagerConfig{
    SyncInterval: 30 * time.Second, // How often to sync
}

manager := NewManager(db, connManager, messageLogger, logger, config)
manager.StartSync() // Starts background sync loop
```

**State Updates:**

Runtime state is automatically updated on events:

```go
// OnStationConnected
station.RuntimeState.ConnectionStatus = "connected"
station.RuntimeState.ConnectedAt = &now
station.RuntimeState.LastError = ""

// OnStationDisconnected
station.RuntimeState.ConnectionStatus = "disconnected"
station.RuntimeState.ConnectedAt = nil
station.RuntimeState.LastError = err.Error()

// OnHeartbeat (future enhancement)
station.RuntimeState.LastHeartbeat = &now
```

## Database Indexes

The storage layer creates indexes for optimal query performance:

**Messages Collection:**
- `(station_id, timestamp)` - Station message history
- `(message_id)` - Unique message lookup
- `(correlation_id)` - Request/response matching
- `(action, timestamp)` - Action-based queries
- `(timestamp)` - Time-based queries

**Transactions Collection:**
- `(transaction_id)` - Unique, primary lookup
- `(station_id, start_timestamp)` - Station transaction history
- `(status)` - Query by active/completed/failed
- `(id_tag)` - User transaction history

**Stations Collection:**
- `(station_id)` - Unique, primary lookup
- `(connection_status)` - Find connected/disconnected stations
- `(enabled, auto_start)` - Query enabled stations
- `(tags)` - Tag-based filtering
- `(protocol_version)` - Version-based queries

## Performance Considerations

### Message Logging
- **Buffered writes**: Non-blocking message logging
- **Batch inserts**: Up to 100 messages per insert
- **Configurable flushing**: Balance between latency and throughput
- **Dropped message tracking**: Monitor buffer overflow

### Transaction Persistence
- **Async writes**: Non-blocking 5s timeout
- **Indexed queries**: Fast lookup by ID, station, date
- **Efficient updates**: Single document updates
- **Aggregation pipeline**: Statistics calculation

### State Synchronization
- **Periodic sync**: Reduces write frequency
- **Dirty checking**: Only sync if changed (optional)
- **Bulk operations**: Single ReplaceOne per station
- **Thread safety**: Mutex-protected state access

## Error Handling

All storage operations include proper error handling:

```go
// Non-critical errors are logged but don't block operation
if err := sm.transactionRepo.Create(ctx, dbTransaction); err != nil {
    sm.logger.Error("Failed to persist transaction", "error", err)
    // Continue anyway - local transaction is started
}

// Critical errors are propagated
transaction, err := repo.GetByID(ctx, txID, stationID)
if err != nil {
    return nil, fmt.Errorf("failed to retrieve transaction: %w", err)
}
```

## Testing

All storage features include comprehensive tests:

- ✅ Message logging tests (buffering, flushing, querying)
- ✅ Transaction CRUD tests
- ✅ SessionManager integration tests
- ✅ State synchronization tests

## Future Enhancements

1. **Message Retention Policies**
   - Automatic cleanup of old messages
   - Configurable retention periods
   - Archive to cold storage

2. **Transaction Analytics**
   - Real-time dashboards
   - Energy consumption trends
   - Peak usage analysis
   - Revenue reporting

3. **State Snapshots**
   - Point-in-time recovery
   - Historical state queries
   - Audit trail

4. **Performance Optimizations**
   - Connection pooling tuning
   - Bulk write optimization
   - Read replica support
   - Caching layer

## References

- MongoDB Driver: https://pkg.go.dev/go.mongodb.org/mongo-driver
- OCPP 1.6 Specification - Transaction handling
- Project Plan: `PLAN.md` - Tasks 2.6, 2.7, 2.8
