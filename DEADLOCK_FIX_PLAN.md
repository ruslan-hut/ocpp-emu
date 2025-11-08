# Permanent Fix Plan: Station Manager Deadlock

## Executive Summary

**Issue**: Backend becomes unresponsive after 60-70 seconds due to lock ordering deadlock
**Root Cause**: `internal/station/manager.go:667` - `syncState()` holds manager lock while acquiring station locks
**Status**: Root cause identified, temporary workaround applied
**Priority**: HIGH - Must be fixed before production deployment

---

## Problem Analysis

### Current Problematic Code

```go
// File: internal/station/manager.go
// Line: 667
func (m *Manager) syncState() {
    m.mu.RLock()          // 1. Acquire manager READ lock
    defer m.mu.RUnlock()  // Hold for entire function duration

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    count := 0
    for stationID, station := range m.stations {
        station.mu.Lock()  // 2. Acquire station WRITE lock
                          // ⚠️ DEADLOCK RISK - holding both locks!

        // Only sync if changed since last sync
        if time.Since(station.lastSync) < m.syncInterval/2 {
            station.mu.Unlock()
            continue
        }

        if err := m.saveStationToDB(ctx, station); err != nil {
            m.logger.Error("Failed to sync station state",
                "stationId", stationID,
                "error", err,
            )
        } else {
            station.lastSync = time.Now()
            count++
        }

        station.mu.Unlock()
    }

    if count > 0 {
        m.logger.Debug("Synchronized station states", "count", count)
    }
}
```

### Why This Causes Deadlock

**Lock Acquisition Order**: Manager (READ) → Station (WRITE)

**Scenario**:
1. **Goroutine A** (sync loop): Acquires `manager.mu.RLock()`, then tries `station.mu.Lock()`
2. **Goroutine B** (HTTP handler): Has `station.mu.Lock()`, then tries `manager.mu.RLock()`
3. **Result**: Classic deadlock - each goroutine holds a lock the other needs

**Timeline**:
- t=0s: Backend starts
- t=30s: First sync (usually succeeds - no conflicting HTTP requests yet)
- t=60s: Second sync + HTTP requests → **DEADLOCK**
- t=70s: All goroutines blocked, backend unresponsive

---

## Permanent Fix

### Solution 1: Release Manager Lock Early (RECOMMENDED)

Copy station references while holding the manager lock, then release it before acquiring individual station locks.

```go
// File: internal/station/manager.go
// Line: 667
func (m *Manager) syncState() {
    // Step 1: Safely copy station references under manager lock
    m.mu.RLock()
    stationsToSync := make([]*Station, 0, len(m.stations))
    for _, station := range m.stations {
        stationsToSync = append(stationsToSync, station)
    }
    m.mu.RUnlock()  // ✅ Release manager lock BEFORE acquiring station locks

    // Step 2: Create context for MongoDB operations
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Step 3: Sync each station (now safe from deadlock)
    count := 0
    for _, station := range stationsToSync {
        station.mu.Lock()  // Now safe - no manager lock held

        // Only sync if changed since last sync
        if time.Since(station.lastSync) < m.syncInterval/2 {
            station.mu.Unlock()
            continue
        }

        // Save to database
        if err := m.saveStationToDB(ctx, station); err != nil {
            m.logger.Error("Failed to sync station state",
                "stationId", station.ID,
                "error", err,
            )
        } else {
            station.lastSync = time.Now()
            count++
        }

        station.mu.Unlock()
    }

    if count > 0 {
        m.logger.Debug("Synchronized station states", "count", count)
    }
}
```

**Benefits**:
- ✅ Eliminates deadlock by avoiding holding both locks simultaneously
- ✅ Minimal code changes
- ✅ Maintains existing functionality
- ✅ No performance impact

**Trade-offs**:
- Station map could change between copying references and syncing (acceptable - next sync will catch changes)
- Stations could be deleted from map while sync is in progress (safe - we still have reference)

### Solution 2: Use Read-Only Data (ALTERNATIVE)

Extract only the data needed for sync while holding locks, then perform MongoDB operations without any locks.

```go
func (m *Manager) syncState() {
    // Extract sync data under locks
    type syncData struct {
        station   *Station
        stationID string
        needsSync bool
    }

    m.mu.RLock()
    dataToSync := make([]syncData, 0, len(m.stations))
    for stationID, station := range m.stations {
        station.mu.RLock()
        needsSync := time.Since(station.lastSync) >= m.syncInterval/2
        dataToSync = append(dataToSync, syncData{
            station:   station,
            stationID: stationID,
            needsSync: needsSync,
        })
        station.mu.RUnlock()
    }
    m.mu.RUnlock()

    // Perform sync without holding any locks
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    count := 0
    for _, data := range dataToSync {
        if !data.needsSync {
            continue
        }

        if err := m.saveStationToDB(ctx, data.station); err != nil {
            m.logger.Error("Failed to sync station state",
                "stationId", data.stationID,
                "error", err,
            )
        } else {
            // Update lastSync timestamp under lock
            data.station.mu.Lock()
            data.station.lastSync = time.Now()
            data.station.mu.Unlock()
            count++
        }
    }

    if count > 0 {
        m.logger.Debug("Synchronized station states", "count", count)
    }
}
```

**Benefits**:
- ✅ Even safer - minimal lock holding time
- ✅ Better performance - no locks held during MongoDB operations

**Trade-offs**:
- More complex code
- Requires additional lock acquisition to update lastSync

---

## Implementation Plan

### Phase 1: Fix the Deadlock ✅ PRIORITY

**Files to Modify**:
1. `internal/station/manager.go` - Fix `syncState()` method

**Implementation**:
```bash
# 1. Apply Solution 1 (recommended)
# 2. Add comments explaining the lock ordering
# 3. Consider adding lock ordering documentation to the file header
```

**Testing**:
- Start backend with sync enabled
- Run load test for 5+ minutes
- Verify backend remains responsive
- Check container health status remains "healthy"

### Phase 2: Re-enable Message Logger

**Files to Modify**:
1. `cmd/server/main.go` - Uncomment message logger initialization (lines 76-92)
2. `cmd/server/main.go` - Uncomment message logger shutdown (lines 479-482)
3. Remove nil checks from API endpoints (if desired, or keep for safety)

**Changes**:
```go
// cmd/server/main.go

// BEFORE (current workaround):
var messageLogger *logging.MessageLogger = nil
logger.Info("Message logger DISABLED for testing")

// AFTER (re-enabled):
messageLogger := logging.NewMessageLogger(
    mongoClient,
    logger,
    logging.LoggerConfig{
        BufferSize:    1000,
        BatchSize:     100,
        FlushInterval: 5 * time.Second,
        LogLevel:      "info",
    },
)
messageLogger.Start()
logger.Info("Message logger initialized and started")
```

**Testing**:
- Verify messages are logged to MongoDB
- Check `/api/messages` endpoint returns data
- Monitor for any buffer overflow or blocking issues

### Phase 3: Re-enable Message Broadcaster

**Files to Modify**:
1. `cmd/server/main.go` - Uncomment broadcaster initialization (lines 70-74)
2. `cmd/server/main.go` - Connect to message logger (line 88)
3. `cmd/server/main.go` - Uncomment WebSocket handler (lines 343-345)
4. `cmd/server/main.go` - Uncomment WebSocket endpoints (lines 384-387)
5. `cmd/server/main.go` - Uncomment broadcaster shutdown (lines 484-487)

**Changes**:
```go
// cmd/server/main.go

// Initialize Message Broadcaster
messageBroadcaster := api.NewMessageBroadcaster(logger)
messageBroadcaster.Start()
logger.Info("Message broadcaster initialized and started")

// Initialize Message Logger
messageLogger := logging.NewMessageLogger(
    mongoClient,
    logger,
    logging.LoggerConfig{
        BufferSize:    1000,
        BatchSize:     100,
        FlushInterval: 5 * time.Second,
        LogLevel:      "info",
    },
)
messageLogger.SetBroadcaster(messageBroadcaster)  // Re-enable
messageLogger.Start()
logger.Info("Message logger initialized and started")

// Initialize WebSocket Handler
wsHandler := api.NewWebSocketHandler(messageBroadcaster, logger)
logger.Info("WebSocket handler initialized")

// Register WebSocket endpoints
mux.HandleFunc("/api/ws/messages", wsHandler.HandleMessages)
mux.HandleFunc("/api/ws/stats", wsHandler.HandleBroadcasterStats)
logger.Info("WebSocket endpoints registered")
```

**Testing**:
- Connect WebSocket client from frontend
- Verify real-time messages are received
- Test with multiple concurrent WebSocket connections
- Monitor for broadcaster deadlock issues (should be fine, but verify)

### Phase 4: Re-enable Station Manager Sync

**Files to Modify**:
1. `cmd/server/main.go` - Uncomment sync start (lines 133-137)

**Changes**:
```go
// cmd/server/main.go

// Start background state synchronization
stationManager.StartSync()
logger.Info("Station state synchronization started")
```

**Testing**:
- Run extended test (5+ minutes, multiple sync cycles)
- Verify backend remains responsive
- Check MongoDB for synced station states
- Monitor goroutine count and memory usage

---

## Testing Strategy

### Unit Tests

Add test for lock ordering:

```go
// File: internal/station/manager_test.go

func TestSyncStateNoDeadlock(t *testing.T) {
    // Create manager with test data
    manager := setupTestManager(t)

    // Simulate concurrent HTTP requests while sync is running
    done := make(chan bool)

    // Goroutine 1: Continuous sync
    go func() {
        for i := 0; i < 10; i++ {
            manager.syncState()
            time.Sleep(100 * time.Millisecond)
        }
        done <- true
    }()

    // Goroutine 2: Concurrent station access (simulating HTTP handlers)
    go func() {
        for i := 0; i < 100; i++ {
            manager.GetStation(context.Background(), "test-station")
            time.Sleep(10 * time.Millisecond)
        }
        done <- true
    }()

    // Wait for both with timeout
    timeout := time.After(30 * time.Second)
    for i := 0; i < 2; i++ {
        select {
        case <-done:
            // Success
        case <-timeout:
            t.Fatal("Deadlock detected - test timed out")
        }
    }
}
```

### Integration Tests

```bash
# Test script: test-deadlock-fix.sh

#!/bin/bash

echo "Starting backend..."
docker-compose up -d

echo "Waiting for backend to start..."
sleep 15

echo "Running extended load test (5 minutes)..."
for i in {1..60}; do
    echo "Iteration $i (t=$((i*5))s)"

    # Test health endpoint
    curl -s --max-time 5 http://localhost:3000/api/health > /dev/null
    if [ $? -ne 0 ]; then
        echo "❌ FAILED at t=$((i*5))s - Backend unresponsive"
        exit 1
    fi

    # Test stations endpoint
    curl -s --max-time 5 http://localhost:3000/api/stations > /dev/null
    if [ $? -ne 0 ]; then
        echo "❌ FAILED at t=$((i*5))s - Stations endpoint unresponsive"
        exit 1
    fi

    sleep 5
done

echo "✅ SUCCESS - Backend remained responsive for 5 minutes"
echo "Container health:"
docker-compose ps
```

### Performance Monitoring

```bash
# Monitor goroutine count over time
watch -n 5 'curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | grep "goroutine profile" || echo "pprof not enabled"'

# Monitor MongoDB connection pool
watch -n 5 'curl -s http://localhost:3000/api/health | jq .database'

# Monitor memory usage
watch -n 5 'docker stats ocpp-emu-backend --no-stream'
```

---

## Implementation Checklist

### Critical Path (Must Complete Before Production)

- [ ] **Phase 1: Fix Deadlock**
  - [ ] Implement Solution 1 in `syncState()`
  - [ ] Add code comments explaining lock ordering
  - [ ] Test with sync enabled for 5+ minutes
  - [ ] Verify container remains healthy
  - [ ] Run concurrent load test

### Post-Fix (Enable Full Functionality)

- [ ] **Phase 2: Re-enable Message Logger**
  - [ ] Uncomment initialization in main.go
  - [ ] Uncomment shutdown in main.go
  - [ ] Test message persistence to MongoDB
  - [ ] Verify API endpoints work

- [ ] **Phase 3: Re-enable Message Broadcaster**
  - [ ] Uncomment broadcaster initialization
  - [ ] Uncomment WebSocket handler
  - [ ] Uncomment WebSocket endpoints
  - [ ] Test WebSocket connections
  - [ ] Verify real-time message streaming

- [ ] **Phase 4: Re-enable Station Manager Sync**
  - [ ] Uncomment StartSync() call
  - [ ] Run extended stability test (30+ minutes)
  - [ ] Verify station state syncs to MongoDB

### Optional Improvements

- [ ] Add unit tests for lock ordering
- [ ] Add pprof debugging endpoints for production
- [ ] Document lock ordering rules in code comments
- [ ] Add monitoring/alerting for deadlock detection
- [ ] Consider adding lock timeout detection

---

## Rollback Plan

If issues occur after deploying the fix:

1. **Immediate**: Disable station sync again
   ```go
   // stationManager.StartSync()
   logger.Info("Station state synchronization DISABLED")
   ```

2. **Short-term**: Reduce sync interval to avoid frequent conflicts
   ```go
   station.ManagerConfig{
       SyncInterval: 5 * time.Minute,  // Increased from 30 seconds
   }
   ```

3. **Alternative**: Sync only on explicit triggers (shutdown, manual save)
   ```go
   // Remove StartSync() entirely
   // Add manual sync endpoint: POST /api/admin/sync
   ```

---

## Success Criteria

✅ **Fix is successful when**:
- Backend remains responsive for 30+ minutes with sync enabled
- Container health stays "healthy" continuously
- All API endpoints respond within < 1 second
- WebSocket connections remain stable
- No goroutine leaks detected
- Station state successfully syncs to MongoDB every 30 seconds

---

## Notes

### Why Solution 1 is Recommended

- **Simplicity**: Minimal code changes, easy to review
- **Safety**: Eliminates deadlock completely
- **Performance**: No measurable impact
- **Maintainability**: Clear and understandable

### Lock Ordering Best Practices

Going forward, establish this rule in the codebase:

**Lock Ordering Rule**: Always acquire locks in this order:
1. Manager locks (if needed)
2. Station locks (if needed)

**Never hold both simultaneously unless absolutely necessary.**

When you must hold both:
- Document why in code comments
- Consider refactoring to avoid it
- Add tests to prevent deadlock

### Future Considerations

1. **Lock-free alternatives**: Consider using channels for sync coordination
2. **Metrics**: Add Prometheus metrics for lock contention
3. **Timeouts**: Add lock acquisition timeouts with panic/recovery
4. **Documentation**: Create concurrency documentation for the codebase

---

**Document Version**: 1.0
**Created**: 2025-11-08
**Author**: AI Assistant (Claude)
**Status**: Ready for Implementation
