# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OCPP Charging Station Emulator - A web-based EV charging station emulator supporting OCPP 1.6, 2.0.1, and 2.1 protocols. Designed to test and diagnose OCPP-compliant remote servers (CSMS) by simulating 1-10 charging stations with comprehensive message logging and custom message crafting capabilities.

**Tech Stack:**
- **Backend:** Go 1.23+ with standard library HTTP
- **Frontend:** React 18 with Vite
- **Database:** MongoDB 7.0+
- **Logging:** Standard library `slog` (structured JSON logging)
- **WebSocket:** gorilla/websocket for OCPP and real-time message streaming

## Common Commands

### Development
```bash
# Run the backend server locally
go run cmd/server/main.go

# Run backend with hot-reload (requires air)
make dev

# Run frontend dev server
cd web && npm run dev

# Run tests (backend)
make test
go test -v ./internal/...

# Run specific test
go test -v -run TestName ./internal/package

# Format code
make fmt
```

### Docker
```bash
# Start all services (MongoDB, backend, frontend)
docker-compose up --build

# Start only MongoDB for local development
docker-compose up -d mongodb

# View logs
docker-compose logs -f

# Access MongoDB shell
docker-compose exec mongodb mongosh ocpp_emu

# Rebuild specific service
docker-compose build backend
```

### Build & Testing
```bash
# Build binary
make build

# Run all checks and build
make all

# Test with coverage
make test-coverage
```

## Architecture

### Core Components Flow

1. **Station Manager** (`internal/station/manager.go`)
   - Central orchestrator for all charging stations
   - Manages station lifecycle (create, start, stop, delete)
   - Handles OCPP message routing to appropriate protocol handlers
   - Maintains runtime state synchronization with MongoDB
   - Auto-starts stations on startup based on configuration

2. **Connection Manager** (`internal/connection/manager.go`)
   - Manages WebSocket connections to CSMS servers
   - One connection per station (station ID-based routing)
   - Handles reconnection logic with exponential backoff
   - Routes incoming messages to Station Manager
   - Thread-safe connection state tracking

3. **Message Logger** (`internal/logging/message_logger.go`)
   - Buffers and batch-inserts messages to MongoDB
   - Broadcasts messages in real-time via WebSocket to frontend
   - Configurable buffer size (1000) and flush interval (5s)
   - Categorizes messages by type (Call/CallResult/CallError)

4. **OCPP Protocol Handlers** (`internal/ocpp/`)
   - Protocol-specific message encoding/decoding
   - `v16/`: OCPP 1.6 implementation (primary focus)
   - `v201/` and `v21/`: Placeholders for future versions
   - Message validation and handler dispatch

5. **Session Manager** (`internal/station/session.go`)
   - Tracks charging sessions per connector
   - Manages transaction lifecycle (authorize, start, stop)
   - Calculates meter values and energy consumption
   - Persists sessions to MongoDB

### Key Data Flow

**Incoming Message (CSMS → Station):**
```
CSMS → Connection Manager → Station Manager → Protocol Handler → Session/Connector Logic
```

**Outgoing Message (Station → CSMS):**
```
Station Logic → Protocol Handler → Connection Manager → CSMS
                      ↓
              Message Logger → MongoDB + WebSocket Broadcast
```

**Station State Sync:**
```
Station Manager (periodic) → MongoDB (stations collection)
```

## Important Patterns

### Thread Safety
- All shared state uses `sync.RWMutex` locks
- Station Manager's `stations` map is protected
- Connection Manager's connection map is protected
- Always lock → read/write → unlock pattern
- **Critical:** Recent deadlock fixes in `manager.go` - be careful with nested locks

### Context Usage
- All long-running operations accept `context.Context`
- Graceful shutdown uses 30-second timeout context
- MongoDB operations use request context for proper cancellation

### Error Handling
- Structured logging with `slog` for all errors
- Errors logged with context (station_id, message_type, etc.)
- Non-fatal errors logged but don't crash the service

### Message Structure (OCPP)
```json
// Call
[2, "unique-id", "Action", {"payload": "data"}]

// CallResult
[3, "unique-id", {"result": "data"}]

// CallError
[4, "unique-id", "ErrorCode", "Description", {}]
```

## Configuration

### Application Config (`configs/config.yaml`)
- Server settings (port, host)
- MongoDB connection
- CSMS defaults (URL, timeouts, reconnection)
- Logging configuration

### Station Config (MongoDB `stations` collection)
- Stored as documents, editable via API or Web UI
- Contains: ID, protocol version, connectors, CSMS URL, features
- Auto-loaded on startup and synchronized periodically

### Environment Variables
- MongoDB URI can be overridden via `MONGODB_URI`
- Check `.env.example` for Docker setup

## Testing Strategy

- Unit tests for core logic (connectors, sessions, encoding)
- Test files follow `*_test.go` convention
- Use table-driven tests where applicable
- Mock external dependencies (MongoDB, WebSocket connections)
- **Important:** Tests check for race conditions with `-race` flag

## MongoDB Collections

- **stations**: Station configurations
- **messages**: OCPP message history (indexed by station_id, timestamp)
- **sessions**: Charging sessions
- **transactions**: Transaction records
- **meter_values**: Time-series data (optimized with MongoDB time-series collections)

## WebSocket API

### Real-time Message Stream
```
ws://localhost:8080/api/ws/messages?stationId=STATION_001&direction=sent
```
Query params: `stationId`, `direction` (sent/received), `messageType`

### Frontend Connection Pattern
Frontend connects to WebSocket on mount, receives JSON messages, displays in real-time inspector.

## Common Gotchas

1. **Station ID Format:** Must match OCPP specification (alphanumeric, can include dots/hyphens)
2. **Connector Numbering:** Connectors are 1-indexed (connector 0 is reserved in OCPP 1.6)
3. **Message IDs:** Must be unique per message (use UUID or counter)
4. **Deadlocks:** Recent fixes - avoid holding station manager lock when calling methods that acquire connection manager locks
5. **Context Cancellation:** Always check for context cancellation in long-running loops
6. **MongoDB Indexes:** Critical for message query performance - maintained by storage layer

## Current Development Focus

Project is in active development (v0.1.0). Recent work includes:
- Deadlock fixes in station manager
- Enhanced message statistics and normalization
- Docker multi-stage builds and production deployment
- Frontend message inspector improvements

See `PLAN.md` for detailed roadmap and `PHASE*.md` files for implementation documentation.
