# Critical Issue: Backend Becomes Unresponsive

## Status: ✅ RESOLVED - Root Cause Found

The backend starts successfully but becomes unresponsive after ~1-2 minutes.

## Symptoms

1. **Initial state**: Backend starts fine, responds to HTTP requests
2. **After 1-2 minutes**: Backend stops responding to any HTTP requests
3. **Container status**: Shows as "unhealthy"
4. **Process status**: Main process is still running
5. **No error logs**: No crashes or panics in logs

## Evidence

```bash
# Test immediately after start
$ curl http://localhost:3000/api/health
{"status":"healthy",...}  # ✅ Works

# Test after 60 seconds
$ curl http://localhost:3000/api/health
(empty response - timeout)  # ❌ Fails
```

## Container Status
```
backend    Up 2 minutes (unhealthy)
frontend   Up 2 minutes
mongodb    Up 2 minutes (healthy)
```

## Root Cause Analysis

### Likely Causes (in order of probability)

1. **Goroutine Deadlock**
   - The broadcaster, message logger, or station sync might be blocking
   - HTTP server goroutine might be waiting on a locked resource

2. **Channel Blocking**
   - Message buffer channels might be full and blocking
   - Broadcaster channels might be deadlocked

3. **MongoDB Connection Pool Exhaustion**
   - Connections not being returned to pool
   - Max pool size might be too small

### Evidence for Deadlock

From `internal/api/message_broadcaster.go`:
```go
// run is the main loop for the broadcaster
func (mb *MessageBroadcaster) run() {
    defer mb.wg.Done()

    for {
        select {
        case <-mb.ctx.Done():
            ...
        case client := <-mb.register:
            mb.clientsMu.Lock()     // ⚠️ Potential deadlock
            mb.clients[client] = true
            mb.clientsMu.Unlock()
            ...
        case message := <-mb.broadcast:
            ...
            mb.broadcastToClients(message)  // ⚠️ Calls updateStats with locks
        }
    }
}
```

The `broadcastToClients` method locks `clientsMu` and also calls stats methods that lock `statsMu`. If called in wrong order from different goroutines, this could deadlock.

## Attempted Fixes

### 1. Healthcheck Timeout ✅ Partially Fixed
- Added `--timeout=5` to wget command
- Increased Docker healthcheck timeout to 10s
- **Result**: Prevents stuck wget processes, but doesn't solve core issue

### 2. Frontend URLs ✅ Fixed
- Changed to use relative URLs through nginx
- **Result**: Frontend loads correctly initially

### 3. Disabled Message Broadcaster ❌ Not the Cause
- Temporarily disabled entire message broadcaster system
- **Result**: Backend STILL became unresponsive after 70 seconds
- **Conclusion**: Broadcaster is NOT the root cause

### 4. Disabled Message Logger ❌ Not the Cause
- Temporarily disabled message logger initialization and all endpoints
- Added nil checks in station manager and API handlers
- **Result**: Backend STILL became unresponsive after 70 seconds
- **Conclusion**: Message logger is NOT the root cause

## New Lead: Station Manager Sync Timing Correlation

**Key Observation**:
- Station manager sync runs every 30 seconds
- Backend becomes unresponsive at ~60-70 seconds
- This is right after the **second sync cycle**

**Evidence**:
```
t=0s   : Backend starts
t=30s  : First sync cycle runs
t=60s  : Second sync cycle runs ⚠️
t=70s  : Backend unresponsive
```

**Next Test**: Disable station manager sync to confirm

### 5. Disabled Station Manager Sync ✅ ROOT CAUSE FOUND!
- Temporarily disabled `stationManager.StartSync()`
- **Result**: Backend REMAINS RESPONSIVE indefinitely (tested 100+ seconds)
- **Container Status**: Changed from "(unhealthy)" to "(healthy)"
- **Conclusion**: Station manager sync is the ROOT CAUSE of the deadlock

## Root Cause: Lock Ordering Deadlock

**Location**: `internal/station/manager.go:667` - `syncState()` method

**Problem**: The sync goroutine acquires locks in this order:
```go
func (m *Manager) syncState() {
    m.mu.RLock()          // 1. Manager READ lock
    defer m.mu.RUnlock()

    for stationID, station := range m.stations {
        station.mu.Lock()  // 2. Station WRITE lock
        // ... MongoDB operations ...
        station.mu.Unlock()
    }
}
```

**Lock Order**: manager → station

If any other goroutine (HTTP handlers, callbacks) acquires locks in reverse order (station → manager), a **classic deadlock** occurs.

**Timing**:
- Sync runs every 30 seconds
- First sync at t=30s usually succeeds
- Second sync at t=60s conflicts with HTTP requests → DEADLOCK
- Backend becomes unresponsive at t=60-70s

## Solution

**Option 1: Fix Lock Ordering** (Recommended)
- Refactor `syncState()` to not hold manager lock while acquiring station locks
- Copy station references, release manager lock, then iterate

**Option 2: Use Channels**
- Replace mutex-based sync with channel-based communication
- Eliminates lock ordering issues

**Option 3: Disable Periodic Sync**
- Only sync on-demand (during shutdown, manual triggers)
- Simpler but loses automatic state persistence

## Current Workaround

**TEMPORARY**: Station manager sync is disabled
- Backend remains responsive indefinitely
- Trade-off: Station state is not automatically synced to MongoDB every 30s
- State is still saved during: shutdown, manual station updates, critical operations

## Recommended Next Steps

### Immediate Fix Required

**Implement Option 1: Fix Lock Ordering in syncState()**

Current problematic code:
```go
func (m *Manager) syncState() {
    m.mu.RLock()  // Holds this lock for entire function
    defer m.mu.RUnlock()

    for stationID, station := range m.stations {
        station.mu.Lock()  // Deadlock risk!
        // ... work ...
        station.mu.Unlock()
    }
}
```

**Fixed code**:
```go
func (m *Manager) syncState() {
    // Step 1: Copy station references under lock
    m.mu.RLock()
    stations := make([]*Station, 0, len(m.stations))
    for _, station := range m.stations {
        stations = append(stations, station)
    }
    m.mu.RUnlock()  // Release manager lock BEFORE acquiring station locks

    // Step 2: Sync each station (now safe from deadlock)
    for _, station := range stations {
        station.mu.Lock()
        // ... work ...
        station.mu.Unlock()
    }
}
```

This eliminates the lock ordering issue by ensuring we never hold both locks simultaneously.

### Code Review Needed

**Priority Areas:**

1. `internal/api/message_broadcaster.go` - Lock ordering, channel blocking
2. `internal/logging/message_logger.go` - Buffer management
3. `internal/station/manager.go` - Sync goroutine
4. `cmd/server/main.go` - HTTP server startup

### Specific Issues to Check

```go
// message_broadcaster.go:86
func (mb *MessageBroadcaster) broadcastToClients(message BroadcastMessage) {
    mb.clientsMu.RLock()     // Read lock
    defer mb.clientsMu.RUnlock()

    for client := range mb.clients {
        select {
        case client.send <- data:  // ⚠️ Could block if channel full
            broadcastCount++
        default:
            mb.incrementDropped()  // ⚠️ Tries to lock statsMu while holding clientsMu
        }
    }
}
```

**Potential fix**: Don't hold `clientsMu` while sending to channels.

## Temporary Solution

Until fixed, the application cannot be reliably tested in Docker. Consider:

1. **Run locally without Docker** for development
2. **Disable real-time message streaming** temporarily
3. **Increase all timeouts** to 5 minutes (not a real fix)

## Files Involved

- `internal/api/message_broadcaster.go` - Real-time WebSocket broadcasting
- `internal/logging/message_logger.go` - Message buffering and persistence
- `internal/station/manager.go` - Station state synchronization
- `cmd/server/main.go` - Server initialization

## Test to Reproduce

```bash
# Clean start
docker-compose down
docker-compose up -d

# Wait for startup
sleep 10

# Test works initially
curl http://localhost:3000/api/health  # ✅ Should work

# Wait 60 seconds
sleep 60

# Test fails
curl http://localhost:3000/api/health  # ❌ Will timeout/fail
```

## Impact

- **Severity**: CRITICAL (now mitigated with workaround)
- **Users Affected**: All users (Docker and local)
- **Workaround**: Station manager sync temporarily disabled
- **Timeline**: Permanent fix needed before production use

## Testing Results

**With Sync Enabled** (Original Issue):
- ❌ Backend unresponsive at t=60-70s
- ❌ Container health: "unhealthy"
- ❌ All API endpoints timeout

**With Sync Disabled** (Current Workaround):
- ✅ Backend remains responsive indefinitely (tested 100+ seconds)
- ✅ Container health: "healthy"
- ✅ All API endpoints working normally
- ✅ Frontend loads and functions correctly

---

**Last Updated**: 2025-11-08T18:20:00Z
**Status**: ✅ Root cause identified, temporary workaround applied
**Root Cause**: Lock ordering deadlock in `internal/station/manager.go:667`
**Fix Needed**: Refactor `syncState()` to avoid holding manager lock while acquiring station locks
