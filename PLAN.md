# OCPP Charging Station Emulator - Project Plan

## Overview
A web-based EV charging station emulator supporting OCPP 1.6, 2.0.1, and 2.1 protocols. Designed to test and diagnose OCPP-compliant remote servers (CSMS - Charging Station Management Systems) by simulating 1-10 charging stations with comprehensive message logging and custom message crafting capabilities.

## Goals
- Emulate realistic charging station behavior across multiple OCPP versions
- Provide detailed message-level debugging and logging
- Enable custom message crafting for edge case testing
- Help identify server-side issues and protocol compliance problems
- Support testing of authentication, authorization, and transaction flows

## Architecture

### High-Level Components
```
┌─────────────────────────────────────────────────────────┐
│                     Web UI (Frontend)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Station    │  │   Message    │  │   Scenario   │  │
│  │   Manager    │  │   Inspector  │  │   Runner     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                          ▲ WebSocket/HTTP
                          │
┌─────────────────────────────────────────────────────────┐
│                  Backend Server (Go)                     │
│  ┌──────────────────────────────────────────────────┐  │
│  │              WebSocket API Layer                  │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Station    │  │   Message    │  │   Session    │  │
│  │   Manager    │  │   Logger     │  │   Storage    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  OCPP 1.6    │  │  OCPP 2.0.1  │  │  OCPP 2.1    │  │
│  │   Engine     │  │   Engine     │  │   Engine     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                          ▲ OCPP WebSocket/SOAP
                          │
┌─────────────────────────────────────────────────────────┐
│              Remote CSMS (Server Under Test)             │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Backend (Go)

#### Station Manager
- Load station configurations from MongoDB on startup
- Initialize state machines for each enabled station
- Auto-connect stations marked with `auto_start: true`
- Each station maintains its own state machine in memory
- Sync runtime state back to MongoDB (connection status, last heartbeat)
- React to configuration changes from Web UI in real-time
- Connection lifecycle management (connect, disconnect, heartbeat)
- Hot-reload: Add/remove/modify stations without restart

#### OCPP Protocol Engines
**OCPP 1.6 Engine**
- WebSocket (JSON) and SOAP (XML) transport support
- Core profile: Boot Notification, Heartbeat, Status Notification, Authorize
- Firmware Management profile: Diagnostics, Firmware Update
- Smart Charging profile: Charging profiles, composite schedules
- Remote Control profile: Remote Start/Stop, Reset, Unlock

**OCPP 2.0.1 & 2.1 Engines**
- WebSocket (JSON) transport (SOAP deprecated)
- Enhanced security: ISO 15118, certificate management
- Device model: Variables, components, characteristics
- Advanced transaction handling
- Display messages and cost calculations
- Data transfer and diagnostics improvements

#### Message Handler
- Parse incoming OCPP messages (Call, CallResult, CallError)
- Route messages to appropriate protocol engine
- Validate message format and payload
- Handle message sequencing and correlation (unique message IDs)
- Support for custom/malformed message injection

#### Message Logger
- Log all incoming and outgoing messages with timestamps
- Store message metadata (direction, type, station ID, protocol version)
- Filter and search capabilities
- Export logs (JSON, CSV, text formats)
- Real-time streaming to Web UI

#### Session Storage
- In-memory storage for active sessions (fast access)
- MongoDB persistence for:
  - Message history and logs
  - Transaction records
  - Authorization records
  - Meter values
  - Station configurations
  - Session history
- Station state persistence across restarts
- Time-series data optimization for meter values

#### WebSocket Manager
- Manage WebSocket connections to CSMS
- Handle connection pooling for multiple stations
- Automatic reconnection with backoff
- TLS/SSL support with certificate validation
- Subprotocol negotiation (ocpp1.6, ocpp2.0.1, ocpp2.1)

### 2. Frontend (Web UI)

#### Dashboard
- Overview of all active stations
- Connection status indicators
- Active transactions count
- Recent message activity
- Quick actions (start/stop stations)

#### Station Manager View
- **List View**: Display all configured stations with status indicators
- **Create Station**: Form to create new station with all configuration options:
  - Basic info (ID, name, vendor, model, serial number)
  - Protocol version selection
  - Connector configuration (count, type, power)
  - Supported OCPP profiles/features
  - CSMS connection settings
  - Simulation behavior settings
  - Tags and metadata
- **Edit Station**: Modify existing station configuration
- **Delete Station**: Remove station from database
- **Station Controls**:
  - Start/Stop individual stations
  - Enable/Disable stations
  - Reset station state
  - View detailed station state
- **Bulk Operations**:
  - Import/Export stations (JSON format)
  - Start/Stop multiple stations
  - Clone station configuration
- **Templates**: Pre-configured station templates for quick setup
- **Real-time Status**: Live connection status, heartbeat indicator

#### Message Inspector
- Real-time message feed with filtering
- Message direction indicators (sent/received)
- Syntax highlighting for JSON/XML
- Message type categorization
- Search and filter by:
  - Station ID
  - Message type
  - Time range
  - Content
- Message details panel with request/response correlation

#### Custom Message Crafter
- Template-based message creation
- JSON editor with schema validation
- Send arbitrary messages to test edge cases
- Save message templates for reuse
- Test malformed messages for robustness testing

#### Scenario Runner
- Pre-defined test scenarios:
  - Complete charging session
  - Authorization failure handling
  - Network interruption recovery
  - Firmware update simulation
  - Error condition testing
- Scenario step visualization
- Pause/resume/stop controls

#### Configuration Panel
- **Application Settings** (read-only display of config.yaml values):
  - Server info (port, host)
  - MongoDB connection status
  - CSMS default settings
  - Application limits
- **Station Templates Management**:
  - Create/edit/delete station templates
  - Pre-configured templates for common scenarios
- **Import/Export**:
  - Import stations from JSON file
  - Export all stations to JSON
  - Backup/restore functionality
- **System Status**:
  - MongoDB connection health
  - Active WebSocket connections
  - Memory usage and performance metrics

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **HTTP Server**: standard library `net/http`
- **WebSocket**: `gorilla/websocket` (for WebSocket protocol handling)
- **OCPP Protocol**: Custom implementation based on official OCPP specifications
  - OCPP 1.6, 2.0.1, 2.1 message types and validation logic
  - No third-party OCPP libraries - full control over protocol implementation
- **Database**: MongoDB (`go.mongodb.org/mongo-driver`)
  - Message logging and history
  - Transaction records
  - Station configurations
  - Session persistence
- **Configuration**: `viper` for config management or standard library
- **Logging**: standard library `log/slog` for structured logging
- **Testing**: `testing` (standard library) with optional `testify` for assertions

### Frontend
- **Framework**: React or Vue.js
- **UI Components**: Material-UI, Ant Design, or Tailwind CSS
- **State Management**: Redux or Zustand
- **WebSocket Client**: Native WebSocket API or socket.io-client
- **Code Editor**: Monaco Editor (VS Code editor) for message crafting
- **Charts**: Recharts or Chart.js for visualizations

### Development Tools
- **Build**: Make or Task
- **Container**: Docker & Docker Compose
- **Database**: MongoDB 7.0+
- **Database Tools**: MongoDB Compass (GUI), mongosh (CLI)
- **API Documentation**: OpenAPI/Swagger
- **Version Control**: Git

## Custom OCPP Implementation

Since we're building a custom OCPP implementation without third-party libraries, the implementation will follow the official OCPP specifications directly:

### OCPP Message Format (JSON over WebSocket)
All OCPP messages follow the JSON-RPC 2.0 format:

**Call (Client → Server)**
```json
[2, "uniqueId", "Action", {payload}]
```

**CallResult (Server → Client)**
```json
[3, "uniqueId", {payload}]
```

**CallError (Server → Client)**
```json
[4, "uniqueId", "ErrorCode", "ErrorDescription", {details}]
```

### Implementation Approach
- Define Go structs for each OCPP message type (aligned with spec JSON schemas)
- Custom JSON marshaling/unmarshaling logic
- Message validation against specification rules
- Protocol state machines for each OCPP version
- No external OCPP dependencies - full implementation control

### Benefits of Custom Implementation
- **Full Control**: Complete understanding of protocol implementation
- **Debugging**: Easier to debug and modify protocol behavior
- **Testing**: Can craft invalid/malformed messages for edge case testing
- **Learning**: Deep understanding of OCPP specifications
- **Flexibility**: Easy to add custom extensions or modifications
- **No Library Constraints**: Not limited by third-party library design decisions
- **Performance**: Optimized for our specific use case (emulation)

## Project Structure

```
ocpp-emu/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── station/
│   │   ├── manager.go              # Station lifecycle management
│   │   ├── state.go                # Station state machine
│   │   └── config.go               # Station configuration
│   ├── ocpp/
│   │   ├── v16/
│   │   │   ├── handler.go          # OCPP 1.6 message handlers (custom)
│   │   │   ├── types.go            # OCPP 1.6 data types (custom structs)
│   │   │   ├── profiles.go         # Feature profiles
│   │   │   └── validation.go       # Message validation logic
│   │   ├── v201/
│   │   │   ├── handler.go          # OCPP 2.0.1 handlers (custom)
│   │   │   ├── types.go            # OCPP 2.0.1 data types
│   │   │   ├── device_model.go     # Device model implementation
│   │   │   └── validation.go       # Message validation
│   │   ├── v21/
│   │   │   └── ...                 # OCPP 2.1 implementation
│   │   ├── common.go               # Shared OCPP logic
│   │   ├── message.go              # Call/CallResult/CallError framing
│   │   └── soap.go                 # SOAP/XML support for OCPP 1.6
│   ├── connection/
│   │   ├── websocket.go            # WebSocket client management
│   │   ├── pool.go                 # Connection pooling
│   │   └── tls.go                  # TLS configuration
│   ├── logger/
│   │   ├── message_logger.go       # OCPP message logging (slog-based)
│   │   └── storage.go              # Log storage/retrieval
│   ├── api/
│   │   ├── handlers.go             # HTTP/WebSocket API handlers (net/http)
│   │   ├── routes.go               # API routing with ServeMux
│   │   └── middleware.go           # CORS, auth, etc.
│   └── storage/
│       ├── memory.go               # In-memory storage for active sessions
│       ├── mongodb.go              # MongoDB client and operations
│       ├── models.go               # MongoDB document models
│       └── repository.go           # Data access layer interface
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Dashboard/
│   │   │   ├── StationManager/
│   │   │   ├── MessageInspector/
│   │   │   ├── MessageCrafter/
│   │   │   └── ScenarioRunner/
│   │   ├── services/
│   │   │   ├── api.js              # Backend API client
│   │   │   └── websocket.js        # WebSocket connection
│   │   ├── store/                  # State management
│   │   └── App.jsx
│   ├── public/
│   └── package.json
├── configs/
│   ├── config.yaml                 # Application configuration
│   ├── config.dev.yaml             # Development overrides
│   └── config.prod.yaml            # Production overrides
├── testdata/
│   ├── scenarios/                  # Test scenario definitions
│   ├── certificates/               # Test certificates
│   └── seed/                       # MongoDB seed data
│       └── stations.json           # Sample stations for development
├── docs/
│   ├── API.md                      # API documentation
│   ├── OCPP_SUPPORT.md            # Supported OCPP features
│   └── USAGE.md                    # User guide
├── scripts/
│   └── build.sh                    # Build scripts
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
**Goal**: Basic project setup and core infrastructure

**Backend:**
- [x] **1.1** Initialize Go module and project structure
- [x] **1.2** Set up MongoDB connection and client (go.mongodb.org/mongo-driver)
- [x] **1.3** Design MongoDB schema and collections (messages, transactions, stations, sessions)
- [x] **1.4** Create MongoDB indexes and setup scripts
- [x] **1.5** Implement configuration loader for config.yaml (using viper or standard library)
- [x] **1.6** Set up basic HTTP/WebSocket server using standard `net/http`
- [x] **1.7** Implement WebSocket connection manager with gorilla/websocket
- [x] **1.8** Design OCPP message structure (Call, CallResult, CallError) for custom implementation
- [x] **1.9** Create station manager with:
  - Load stations from MongoDB on startup
  - Initialize station state machines
  - Auto-start logic for enabled stations
- [x] **1.10** Design and implement message logging infrastructure using `log/slog`
- [x] **1.11** Implement hybrid storage layer (memory + MongoDB)
- [x] **1.12** Build Station CRUD API endpoints:
  - GET /api/stations (list all)
  - GET /api/stations/:id (get one)
  - POST /api/stations (create)
  - PUT /api/stations/:id (update)
  - DELETE /api/stations/:id (delete)
  - PATCH /api/stations/:id/start (start station)
  - PATCH /api/stations/:id/stop (stop station)

**Frontend:**
- [x] **1.13** Set up basic React frontend with routing
- [x] **1.14** Implement WebSocket communication between frontend and backend
- [x] **1.15** Create simple dashboard view
- [x] **1.16** Build Station Manager UI (list view with basic CRUD)

**DevOps:**
- [x] **1.17** Set up Docker Compose with MongoDB
- [x] **1.18** Create seed data for sample stations (testdata/seed/stations.json)

**Deliverable**: Application starts with MongoDB connection, loads stations from DB, frontend can create/edit/delete stations

### Phase 2: OCPP 1.6 Support (Weeks 3-4)
**Goal**: Full OCPP 1.6 implementation with custom protocol handlers

**OCPP 1.6 Protocol:**
- [ ] **2.1** Define custom OCPP 1.6 message types (structs) based on specification
- [ ] **2.2** Implement OCPP 1.6 Core Profile message handlers:
  - [ ] 2.2a: BootNotification
  - [ ] 2.2b: Heartbeat
  - [ ] 2.2c: StatusNotification
  - [ ] 2.2d: Authorize
  - [ ] 2.2e: StartTransaction
  - [ ] 2.2f: StopTransaction
  - [ ] 2.2g: MeterValues
  - [ ] 2.2h: DataTransfer
- [ ] **2.3** Implement custom message encoding/decoding (JSON marshaling/unmarshaling)
- [ ] **2.4** Add SOAP/XML support for OCPP 1.6 (custom XML parsing)
- [ ] **2.5** Create station state machine for charging sessions

**Backend Storage:**
- [x] **2.6** Persist messages to MongoDB (messages collection) - *(Completed in Phase 1)*
- [ ] **2.7** Persist transactions to MongoDB (transactions collection)
- [x] **2.8** Implement station runtime state sync to MongoDB - *(Completed in Phase 1)*

**Frontend Enhancements:**
- [ ] **2.9** Enhance Station Manager UI:
  - [ ] 2.9a: Full create/edit form with all station properties
  - [ ] 2.9b: Station templates feature
  - [ ] 2.9c: Import/Export functionality
- [x] **2.10** Build Message Inspector UI component - *(Basic version completed in Phase 1)*
- [ ] **2.11** Add real-time message streaming to frontend with slog integration

**Testing:**
- [ ] **2.12** Test complete charging session with station loaded from DB

**Deliverable**: Emulator can simulate complete OCPP 1.6 charging session with stations managed via Web UI

### Phase 3: Enhanced Features (Weeks 5-6)
**Goal**: Message crafting and advanced debugging

**Frontend Features:**
- [ ] **3.1** Implement Custom Message Crafter UI
- [ ] **3.2** Add JSON editor with syntax highlighting
- [ ] **3.3** Create message templates library
- [ ] **3.4** Build message validation (optional validation mode)
- [ ] **3.5** Add message filtering and search in inspector (MongoDB queries)
- [ ] **3.6** Implement log export functionality (JSON, CSV)
- [ ] **3.7** Add configuration management UI

**Backend Features:**
- [ ] **3.8** Implement MongoDB aggregation pipelines for analytics
- [x] **3.9** Support multiple simultaneous station connections - *(Basic support in Phase 1)*
- [x] **3.10** Implement connection retry logic with backoff - *(Basic support in Phase 1)*
- [ ] **3.11** Set up MongoDB Change Streams for real-time UI updates

**Deliverable**: Users can craft and send custom messages, inspect all traffic with advanced filtering

### Phase 4: OCPP 2.0.1 Support (Weeks 7-9)
**Goal**: OCPP 2.0.1 protocol implementation

**OCPP 2.0.1 Protocol:**
- [ ] **4.1** Implement OCPP 2.0.1 core functionality:
  - [ ] 4.1a: Enhanced BootNotification with StatusInfo
  - [ ] 4.1b: TransactionEvent (replaces Start/StopTransaction)
  - [ ] 4.1c: Get/Set Variables (device model)
  - [ ] 4.1d: Enhanced authorization
  - [ ] 4.1e: Certificate management messages
  - [ ] 4.1f: Security event notifications
- [ ] **4.2** Implement device model system
- [ ] **4.3** Add ISO 15118 certificate handling

**Frontend Updates:**
- [ ] **4.4** Update UI to support OCPP 2.0.1 specific features
- [ ] **4.5** Add protocol version selector per station

**Deliverable**: Full OCPP 2.0.1 support with device model

### Phase 5: OCPP 2.1 & Advanced Features (Weeks 10-11)
**Goal**: OCPP 2.1 support and scenario automation

**OCPP 2.1 Protocol:**
- [ ] **5.1** Implement OCPP 2.1 enhancements:
  - [ ] 5.1a: Cost and tariff messages
  - [ ] 5.1b: Display messages
  - [ ] 5.1c: Additional security features
  - [ ] 5.1d: Enhanced reservation system

**Scenario Testing:**
- [ ] **5.2** Create Scenario Runner framework
- [ ] **5.3** Implement pre-defined test scenarios:
  - [ ] 5.3a: Happy path charging session
  - [ ] 5.3b: Authorization failures
  - [ ] 5.3c: Network disconnection/reconnection
  - [ ] 5.3d: Concurrent transactions
  - [ ] 5.3e: Error handling scenarios
- [ ] **5.4** Add scenario editor UI
- [ ] **5.5** Implement scenario playback controls

**Deliverable**: OCPP 2.1 support, automated scenario testing

### Phase 6: Testing & Documentation (Week 12)
**Goal**: Quality assurance and documentation

**Testing:**
- [ ] **6.1** Write comprehensive unit tests (target: 70%+ coverage)
- [ ] **6.2** Integration testing with real CSMS (if available)
- [ ] **6.3** Performance testing (10 simultaneous stations)

**Documentation:**
- [ ] **6.4** Create user documentation
- [ ] **6.5** Document API endpoints
- [ ] **6.6** Create video tutorials/screenshots

**DevOps:**
- [x] **6.7** Docker containerization - *(Completed in Phase 1)*
- [ ] **6.8** Deployment documentation

**Deliverable**: Production-ready application with documentation

### Phase 7: Polish & Deployment (Week 13)
**Goal**: Final touches and release

**UI Enhancements:**
- [ ] **7.1** UI/UX improvements based on testing
- [ ] **7.2** Enhance station templates library with common configurations
- [ ] **7.3** Add bulk operations UI (bulk start/stop, bulk edit)
- [ ] **7.4** Add station cloning feature
- [ ] **7.5** Add configuration backup/restore functionality
- [ ] **7.6** Add TLS/SSL certificate management UI
- [ ] **7.7** Implement configuration history/audit log (track changes to stations)

**Quality & Release:**
- [ ] **7.8** Performance optimizations
- [ ] **7.9** Security audit
- [ ] **7.10** Create release builds
- [ ] **7.11** Set up CI/CD pipeline
- [ ] **7.12** Document API endpoints for station management

**Deliverable**: v1.0 release

## Key Features Detail

### Station Configuration Management
- **Database-Driven**: All station configurations stored in MongoDB, not config files
- **Web-Based Management**: Create, edit, delete stations through Web UI
- **No Restart Required**: Add or modify stations without restarting the application
- **Hot-Reload**: Changes to station configuration applied immediately
- **Persistence**: Stations persist across application restarts
- **Auto-Start**: Configure stations to auto-connect on startup
- **Templates**: Pre-configured templates for common station types (AC 22kW, DC 50kW, etc.)
- **Bulk Operations**: Start/stop multiple stations, import/export configurations
- **Cloning**: Duplicate station configuration with modifications
- **Tags & Organization**: Group stations by tags for easier management
- **Runtime State Sync**: Connection status, heartbeats, errors synced to database
- **Import/Export**: JSON-based import/export for backup or migration
- **Configuration History**: Track who created/modified stations and when (future enhancement)

### Message Logging & Debugging
- **Structured Logs**: Every message stored with metadata (timestamp, direction, station ID, message type, protocol version)
- **Real-time Streaming**: WebSocket push to frontend for live monitoring
- **Filtering**: By station, message type, time range, content search
- **Export**: JSON, CSV, or pretty-printed text
- **Correlation**: Link requests with responses using unique message IDs
- **Color Coding**: Visual distinction between message types and success/error states

### Custom Message Crafting
- **Template Library**: Pre-built templates for all OCPP message types
- **JSON Editor**: Monaco editor with OCPP schema validation
- **Quick Edit**: Modify template parameters without editing raw JSON
- **Validation**: Optional schema validation (can be disabled for testing invalid messages)
- **Send Options**: Send immediately, schedule, or add to scenario
- **Response Viewer**: See server response to custom messages
- **History**: Keep track of previously sent custom messages

### Station Simulation
- **Multi-Connector Support**: Simulate stations with 1-4 connectors
- **State Management**: Proper state transitions (Available → Preparing → Charging → Finishing)
- **Realistic Timing**: Configurable delays for state transitions
- **Meter Values**: Simulated power consumption with realistic patterns
- **Error Injection**: Simulate hardware errors, communication failures
- **IdTag Management**: Configure authorization tags per station

### Connection Management
- **Auto-Reconnect**: Configurable retry logic with exponential backoff
- **TLS Support**: Client certificates for secure connections
- **Subprotocol Handling**: Automatic negotiation of OCPP version
- **Connection Health**: Heartbeat monitoring and timeout detection
- **Proxy Support**: HTTP/HTTPS proxy configuration

## OCPP Protocol Coverage

### OCPP 1.6 (JSON over WebSocket)

**Core Profile** (Priority 1)
- BootNotification
- Heartbeat
- StatusNotification
- Authorize
- StartTransaction
- StopTransaction
- MeterValues
- DataTransfer

**Firmware Management** (Priority 2)
- GetDiagnostics
- DiagnosticsStatusNotification
- UpdateFirmware
- FirmwareStatusNotification

**Remote Control** (Priority 2)
- RemoteStartTransaction
- RemoteStopTransaction
- UnlockConnector
- Reset
- ChangeAvailability

**Reservation** (Priority 3)
- ReserveNow
- CancelReservation

**Smart Charging** (Priority 3)
- GetCompositeSchedule
- SetChargingProfile
- ClearChargingProfile

### OCPP 2.0.1 (JSON over WebSocket)

**Core Functionality** (Priority 1)
- BootNotification
- Heartbeat
- StatusNotification
- TransactionEvent
- Authorize
- MeterValues (enhanced)
- NotifyEvent
- NotifyReport
- GetVariables
- SetVariables

**Security** (Priority 1)
- CertificateSigned
- SecurityEventNotification
- SignCertificate
- Get15118EVCertificate

**Device Management** (Priority 2)
- GetBaseReport
- GetReport
- SetMonitoringBase
- SetMonitoringLevel
- SetVariableMonitoring
- ClearVariableMonitoring

**Transactions** (Priority 2)
- RequestStartTransaction
- RequestStopTransaction
- GetTransactionStatus

### OCPP 2.1 Enhancements (Priority 3)
- CostUpdated
- NotifyDisplayMessages
- NotifyEVChargingSchedule
- Additional tariff messages

## Configuration Strategy

The application uses a **hybrid configuration approach**:

### 1. Static Configuration (config.yaml)
Application-level settings that rarely change, managed by DevOps/administrators:
- Server settings (port, host, TLS)
- MongoDB connection
- Logging configuration
- CSMS connection defaults
- Application limits

### 2. Dynamic Configuration (MongoDB)
Station-specific settings managed through the **Web Interface**:
- Station definitions (ID, vendor, model, etc.)
- Connector configurations
- Protocol versions
- Supported features/profiles
- Meter value settings
- CSMS URLs per station

**Benefits:**
- No app restart needed to add/modify stations
- Stations persist across restarts
- Web-based station management
- Easy backup/restore via MongoDB
- Multi-user station management
- Configuration history tracking (future enhancement)

## Configuration

### Application Configuration (config.yaml)
```yaml
server:
  port: 8080
  host: "0.0.0.0"
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json or text
  output: "stdout"       # stdout, stderr, or file path

mongodb:
  uri: "mongodb://localhost:27017"
  database: "ocpp_emu"
  connection_timeout: 10s
  max_pool_size: 100

  # Collection names (can be customized)
  collections:
    messages: "messages"
    transactions: "transactions"
    stations: "stations"
    sessions: "sessions"
    meter_values: "meter_values"

  # Time-series collection for meter values
  timeseries:
    enabled: true
    granularity: "seconds"  # seconds, minutes, hours

csms:
  # Default CSMS connection settings (can be overridden per station)
  default_url: "ws://localhost:9000"
  connection_timeout: 30s
  heartbeat_interval: 60s
  max_reconnect_attempts: 5
  reconnect_backoff: 10s

  # TLS/Certificate settings for secure connections
  tls:
    enabled: false
    ca_cert: ""
    client_cert: ""
    client_key: ""
    insecure_skip_verify: false

application:
  # Maximum number of concurrent stations
  max_stations: 10

  # Cache TTL for in-memory session data
  cache_ttl: 3600s

  # Enable debug features (custom messages, protocol violations)
  debug_mode: true

  # Maximum messages to keep in memory before flushing to DB
  message_buffer_size: 1000

  # Batch insert interval for performance
  batch_insert_interval: 5s
```

### Station Configuration (Managed via Web UI & MongoDB)

Stations are stored in the `stations` MongoDB collection and managed entirely through the web interface. On application startup:

1. **Load all stations** from MongoDB
2. **Initialize station state machines** in memory
3. **Auto-connect** stations marked as `auto_start: true`
4. **Monitor for changes** via web interface or API

**Example Station Document (MongoDB):**
```javascript
{
  _id: ObjectId("..."),

  // Identity
  station_id: "CP001",              // Unique charge point ID
  name: "Station 1 - Main Entrance", // Friendly name for UI
  enabled: true,                     // Enable/disable station
  auto_start: true,                  // Auto-connect on app startup

  // Protocol Configuration
  protocol_version: "ocpp1.6",       // "1.6", "2.0.1", "2.1"

  // Hardware Info
  vendor: "VendorName",
  model: "ModelX",
  serial_number: "SN123456",
  firmware_version: "1.0.0",
  iccid: "89310410106543789301",
  imsi: "310410123456789",

  // Connectors
  connectors: [
    {
      id: 1,
      type: "Type2",                 // Type2, CCS, CHAdeMO, etc.
      max_power: 22000,              // Watts
      status: "Available",           // Current status
      current_transaction_id: null
    },
    {
      id: 2,
      type: "Type2",
      max_power: 22000,
      status: "Available",
      current_transaction_id: null
    }
  ],

  // OCPP Features
  supported_profiles: [
    "Core",
    "FirmwareManagement",
    "RemoteControl",
    "SmartCharging"
  ],

  // Meter Values Configuration
  meter_values_config: {
    interval: 60,                    // Seconds between samples
    measurands: [
      "Energy.Active.Import.Register",
      "Power.Active.Import",
      "Current.Import",
      "Voltage",
      "SoC"
    ],
    aligned_data_interval: 900       // 15 minutes
  },

  // CSMS Connection (overrides defaults)
  csms_url: "ws://localhost:9000/ocpp/CP001",
  csms_auth: {
    type: "basic",                   // basic, bearer, certificate
    username: "cp001",
    password: "secret123"
  },

  // Simulation Behavior
  simulation: {
    // Realistic behavior patterns
    boot_delay: 5,                   // Seconds to wait before boot
    heartbeat_interval: 60,          // Seconds
    status_notification_on_change: true,

    // Transaction simulation
    default_id_tag: "TAG123456",
    energy_delivery_rate: 7000,      // Watts (charging speed)
    randomize_meter_values: true,
    meter_value_variance: 0.05       // ±5% variance
  },

  // Metadata
  connection_status: "disconnected", // "connected", "disconnected", "error"
  last_heartbeat: null,
  created_at: ISODate("2025-01-06T10:00:00Z"),
  updated_at: ISODate("2025-01-06T10:00:00Z"),
  created_by: "admin",               // Future: user management

  // Tags for organization
  tags: ["test", "development", "building-a"]
}
```

### Configuration Management APIs

RESTful API endpoints for station configuration (called by Web UI):

```
GET    /api/stations              - List all stations
GET    /api/stations/:id          - Get station details
POST   /api/stations              - Create new station
PUT    /api/stations/:id          - Update station configuration
DELETE /api/stations/:id          - Delete station
PATCH  /api/stations/:id/start    - Start/connect station
PATCH  /api/stations/:id/stop     - Stop/disconnect station
POST   /api/stations/:id/reset    - Reset station state
POST   /api/stations/:id/clone    - Clone station configuration
POST   /api/stations/import       - Import stations from JSON
GET    /api/stations/export       - Export all stations to JSON
GET    /api/stations/templates    - Get available station templates
POST   /api/stations/templates/:name - Create station from template
```

### Application Startup Flow

The application follows this initialization sequence:

```
1. Load config.yaml
   ├─ Parse server settings (port, host, TLS)
   ├─ Parse MongoDB connection settings
   ├─ Parse logging configuration
   └─ Parse CSMS default settings

2. Initialize MongoDB Connection
   ├─ Connect to MongoDB using URI from config
   ├─ Verify database exists (create if needed)
   ├─ Create/verify collections exist
   ├─ Create/verify indexes
   └─ Run optional seed data (if in dev mode)

3. Initialize Station Manager
   ├─ Query MongoDB for all stations
   ├─ Load station configurations into memory
   ├─ Initialize state machine for each station
   └─ Filter stations where enabled=true

4. Auto-Start Stations
   ├─ Filter stations where auto_start=true
   ├─ For each auto-start station:
   │  ├─ Create CSMS WebSocket connection
   │  ├─ Send BootNotification
   │  ├─ Start heartbeat timer
   │  └─ Update connection_status in MongoDB
   └─ Log startup results

5. Start HTTP/WebSocket Server
   ├─ Initialize API routes (including station CRUD)
   ├─ Set up WebSocket handler for frontend
   ├─ Start listening on configured port
   └─ Ready to accept requests

6. Runtime Operation
   ├─ Web UI can create/edit/delete stations (API)
   ├─ Station Manager reacts to configuration changes
   ├─ New stations can be started dynamically
   ├─ Running stations sync state to MongoDB
   └─ All OCPP messages persisted to MongoDB
```

**Key Benefits:**
- **Zero Downtime Configuration**: Modify stations without restart
- **Database as Source of Truth**: All config in one place
- **Audit Trail**: Track all configuration changes in MongoDB
- **Easy Scaling**: Add more stations on-the-fly
- **Backup/Recovery**: Export/import configurations easily

## MongoDB Schema Design

### Why MongoDB?

MongoDB is an excellent choice for this OCPP emulator for several reasons:

- **Flexible Schema**: OCPP messages have varying payloads depending on message type and version. MongoDB's document model handles this variability naturally.
- **Time-Series Data**: MongoDB 5.0+ has native time-series collections, perfect for storing meter values and metrics.
- **Scalability**: Can easily handle high-throughput message logging from multiple stations.
- **JSON-Native**: OCPP uses JSON messages, which map directly to MongoDB's BSON documents.
- **Powerful Queries**: Complex filtering and aggregation for message analysis and debugging.
- **Indexing**: Fast lookups for message correlation, station queries, and transaction searches.
- **Change Streams**: Real-time data synchronization for WebSocket updates to frontend.
- **Horizontal Scaling**: Can scale out if load testing requires many stations.

### Collections

#### messages
Stores all OCPP messages exchanged between stations and CSMS.

```javascript
{
  _id: ObjectId,
  station_id: String,           // Charging station ID
  direction: String,            // "sent" or "received"
  message_type: String,         // "Call", "CallResult", "CallError"
  action: String,               // e.g., "BootNotification", "Heartbeat"
  message_id: String,           // Unique message ID
  protocol_version: String,     // "1.6", "2.0.1", "2.1"
  payload: Object,              // Message payload
  timestamp: ISODate,           // Message timestamp
  correlation_id: String,       // Link request with response
  error_code: String,           // For CallError messages
  error_description: String,    // For CallError messages
  created_at: ISODate
}

// Indexes
db.messages.createIndex({ station_id: 1, timestamp: -1 })
db.messages.createIndex({ message_id: 1 })
db.messages.createIndex({ correlation_id: 1 })
db.messages.createIndex({ action: 1, timestamp: -1 })
db.messages.createIndex({ timestamp: -1 })
```

#### transactions
Stores charging transaction records.

```javascript
{
  _id: ObjectId,
  transaction_id: Number,       // OCPP transaction ID
  station_id: String,
  connector_id: Number,
  id_tag: String,               // Authorization ID
  start_timestamp: ISODate,
  stop_timestamp: ISODate,
  meter_start: Number,          // Wh
  meter_stop: Number,           // Wh
  energy_consumed: Number,      // Wh
  reason: String,               // Stop reason
  status: String,               // "active", "completed", "failed"
  protocol_version: String,
  created_at: ISODate,
  updated_at: ISODate
}

// Indexes
db.transactions.createIndex({ transaction_id: 1 }, { unique: true })
db.transactions.createIndex({ station_id: 1, start_timestamp: -1 })
db.transactions.createIndex({ status: 1 })
db.transactions.createIndex({ id_tag: 1 })
```

#### stations
Stores station configurations and state. **This is the source of truth for all station definitions.**

Stations are managed through the Web UI and loaded on application startup.

```javascript
{
  _id: ObjectId,

  // Identity & Management
  station_id: String,              // Unique charge point ID (required, unique)
  name: String,                    // Friendly name for UI
  enabled: Boolean,                // Enable/disable station
  auto_start: Boolean,             // Auto-connect on app startup

  // Protocol Configuration
  protocol_version: String,        // "1.6", "2.0.1", "2.1"

  // Hardware Information
  vendor: String,
  model: String,
  serial_number: String,
  firmware_version: String,
  iccid: String,                   // SIM card ID (optional)
  imsi: String,                    // Mobile subscriber ID (optional)

  // Connectors
  connectors: [
    {
      id: Number,
      type: String,                // "Type2", "CCS", "CHAdeMO", etc.
      max_power: Number,           // Watts
      status: String,              // "Available", "Occupied", "Faulted", etc.
      current_transaction_id: Number
    }
  ],

  // OCPP Features
  supported_profiles: [String],    // ["Core", "FirmwareManagement", etc.]

  // Meter Values Configuration
  meter_values_config: {
    interval: Number,              // Seconds between samples
    measurands: [String],          // List of measurand types
    aligned_data_interval: Number  // Clock-aligned interval (seconds)
  },

  // CSMS Connection
  csms_url: String,                // Override default CSMS URL
  csms_auth: {
    type: String,                  // "basic", "bearer", "certificate"
    username: String,
    password: String
  },

  // Simulation Behavior
  simulation: {
    boot_delay: Number,            // Seconds
    heartbeat_interval: Number,    // Seconds
    status_notification_on_change: Boolean,
    default_id_tag: String,
    energy_delivery_rate: Number,  // Watts
    randomize_meter_values: Boolean,
    meter_value_variance: Number   // Percentage (0.0-1.0)
  },

  // Runtime State
  connection_status: String,       // "connected", "disconnected", "error"
  last_heartbeat: ISODate,
  last_error: String,

  // Metadata
  created_at: ISODate,
  updated_at: ISODate,
  created_by: String,              // Future: user who created it
  tags: [String]                   // For organization/filtering
}

// Indexes
db.stations.createIndex({ station_id: 1 }, { unique: true })
db.stations.createIndex({ connection_status: 1 })
db.stations.createIndex({ enabled: 1, auto_start: 1 })
db.stations.createIndex({ tags: 1 })
db.stations.createIndex({ protocol_version: 1 })
```

#### sessions
Stores WebSocket session information.

```javascript
{
  _id: ObjectId,
  station_id: String,
  csms_url: String,
  connected_at: ISODate,
  disconnected_at: ISODate,
  status: String,               // "active", "disconnected"
  reconnect_attempts: Number,
  last_message_at: ISODate,
  protocol_version: String,
  subprotocol: String,
  created_at: ISODate,
  updated_at: ISODate
}

// Indexes
db.sessions.createIndex({ station_id: 1, status: 1 })
db.sessions.createIndex({ status: 1 })
```

#### meter_values (Time-Series Collection)
Optimized for storing meter value samples over time.

```javascript
// Time-series collection (MongoDB 5.0+)
db.createCollection("meter_values", {
  timeseries: {
    timeField: "timestamp",
    metaField: "metadata",
    granularity: "seconds"
  }
})

{
  _id: ObjectId,
  timestamp: ISODate,
  metadata: {
    station_id: String,
    connector_id: Number,
    transaction_id: Number,
    measurand: String           // "Energy.Active.Import.Register", etc.
  },
  value: Number,
  unit: String,
  context: String,              // "Sample.Periodic", "Transaction.Begin", etc.
  format: String,               // "Raw", "SignedData"
  location: String              // "Outlet", "Inlet", "Body"
}
```

### Docker Compose Setup

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    container_name: ocpp-emu-mongodb
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_DATABASE: ocpp_emu
    volumes:
      - mongodb_data:/data/db
      - ./scripts/mongo-init.js:/docker-entrypoint-initdb.d/mongo-init.js:ro
    networks:
      - ocpp-network
    healthcheck:
      test: echo 'db.runCommand("ping").ok' | mongosh localhost:27017/ocpp_emu --quiet
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: ocpp-emu-backend
    ports:
      - "8080:8080"
    environment:
      MONGODB_URI: mongodb://mongodb:27017
      MONGODB_DATABASE: ocpp_emu
    depends_on:
      mongodb:
        condition: service_healthy
    networks:
      - ocpp-network

  frontend:
    build:
      context: ./web
      dockerfile: Dockerfile
    container_name: ocpp-emu-frontend
    ports:
      - "3000:3000"
    depends_on:
      - backend
    networks:
      - ocpp-network

volumes:
  mongodb_data:

networks:
  ocpp-network:
    driver: bridge
```

### MongoDB Advanced Features

#### Change Streams
Use MongoDB Change Streams to push real-time updates to the frontend without polling:

```go
// Example: Watch for new messages
pipeline := mongo.Pipeline{
  bson.D{{Key: "$match", Value: bson.D{
    {Key: "operationType", Value: "insert"},
    {Key: "fullDocument.station_id", Value: stationID},
  }}},
}
changeStream, err := messagesCollection.Watch(ctx, pipeline)
// Stream changes to WebSocket clients
```

#### Aggregation Pipelines
Complex queries for analytics and debugging:

```javascript
// Example: Message statistics by action type
db.messages.aggregate([
  {
    $match: {
      timestamp: { $gte: ISODate("2025-01-01") }
    }
  },
  {
    $group: {
      _id: "$action",
      count: { $sum: 1 },
      avgResponseTime: { $avg: "$response_time" }
    }
  },
  { $sort: { count: -1 } }
])

// Example: Energy consumption by station
db.meter_values.aggregate([
  {
    $match: {
      "metadata.measurand": "Energy.Active.Import.Register",
      timestamp: { $gte: ISODate("2025-01-01") }
    }
  },
  {
    $group: {
      _id: "$metadata.station_id",
      totalEnergy: { $sum: "$value" },
      samples: { $sum: 1 }
    }
  }
])
```

#### Text Search
Enable text search for message content debugging:

```javascript
// Create text index
db.messages.createIndex({
  action: "text",
  error_description: "text"
})

// Search messages
db.messages.find({
  $text: { $search: "authentication failed" }
})
```

#### TTL Indexes
Automatic cleanup of old messages:

```javascript
// Auto-delete messages older than 30 days
db.messages.createIndex(
  { created_at: 1 },
  { expireAfterSeconds: 2592000 }  // 30 days
)
```

## Testing Strategy

### Unit Tests
- OCPP message encoding/decoding
- State machine transitions
- Message validation
- Connection management logic
- Storage operations (memory and MongoDB)
- MongoDB repository layer
- Data model transformations

### Integration Tests
- Complete charging session flow
- Multi-station scenarios
- Connection failure recovery
- Protocol version switching
- Custom message injection

### Manual Testing Checklist
- [ ] Connect to real CSMS and complete charging session
- [ ] Test all OCPP 1.6 core messages
- [ ] Test all OCPP 2.0.1 core messages
- [ ] Verify message logging accuracy
- [ ] Test custom message crafting with valid/invalid messages
- [ ] Simulate network interruptions
- [ ] Test multiple simultaneous stations (10 stations)
- [ ] Verify TLS/certificate authentication
- [ ] Test UI responsiveness and real-time updates
- [ ] Export and verify log files

## Success Criteria

- ✅ Support OCPP 1.6, 2.0.1, and 2.1 protocols
- ✅ Simulate 1-10 charging stations simultaneously
- ✅ Complete message logging with filtering and export
- ✅ Custom message crafting with template system
- ✅ Web UI for station management and monitoring
- ✅ Real-time message inspector
- ✅ Successful completion of full charging session
- ✅ TLS/certificate support for secure connections
- ✅ Comprehensive documentation
- ✅ Docker deployment support

## Future Enhancements (Post v1.0)

- **Load Testing Mode**: Simulate 100+ stations for stress testing
- **API for External Control**: REST API for programmatic control
- **Scenario Marketplace**: Share and import test scenarios
- **Advanced Analytics**: Message statistics, performance metrics
- **Plugin System**: Custom message handlers via plugins
- **Cloud Deployment**: SaaS version for remote testing
- **Mobile App**: Monitor stations from mobile devices
- **AI-Powered Testing**: Automated anomaly detection in server responses
- **Protocol Fuzzing**: Automated generation of edge case messages
- **Multi-CSMS Support**: Connect to multiple servers simultaneously

## Resources

### OCPP Specifications
- OCPP 1.6 Specification: [Open Charge Alliance](https://www.openchargealliance.org/protocols/ocpp-16/)
- OCPP 2.0.1 Specification: [Open Charge Alliance](https://www.openchargealliance.org/protocols/ocpp-201/)
- OCPP 2.1 Specification: [Open Charge Alliance](https://www.openchargealliance.org/protocols/ocpp-21/)

### Reference Implementations
- thoughtworks/maeve-csms: Open source CSMS for testing against
- lorenzodonini/ocpp-go: Reference for message structures (not used as dependency)

### Development Tools
- Postman/Insomnia: API testing
- Wireshark: Network traffic analysis for OCPP message inspection
- Docker Desktop: Containerization
- OCPP Message Validator: Online tools for message validation

## Timeline Summary
- **Total Duration**: ~13 weeks
- **Phase 1-2**: Foundation + OCPP 1.6 (4 weeks)
- **Phase 3-4**: Enhanced features + OCPP 2.0.1 (5 weeks)
- **Phase 5**: OCPP 2.1 + scenarios (2 weeks)
- **Phase 6-7**: Testing + polish (2 weeks)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| OCPP spec complexity | High | Focus on core features first, iterate |
| WebSocket connection stability | Medium | Implement robust retry logic, extensive testing |
| Multi-station performance | Medium | Profile early, optimize connection pooling |
| Protocol version differences | High | Modular architecture per version |
| UI complexity | Low | Use component library, iterative development |

## Next Steps

1. Review and approve this plan
2. Set up development environment:
   - Go 1.21+
   - Node.js (for frontend)
   - Docker & Docker Compose
   - MongoDB 7.0+ (local or Docker)
   - MongoDB Compass (optional, for database visualization)
3. Initialize Git repository
4. Create initial project structure
5. Set up MongoDB and create collections with indexes
6. Start Phase 1 implementation
7. Set up project management (issues, milestones)

---

**Document Version**: 1.3
**Last Updated**: 2025-11-06
**Status**: Draft - Awaiting Review

**Changelog**:
- v1.3: Revised configuration strategy - app config in YAML, station configs in MongoDB managed via Web UI
  - Stations loaded from database on startup
  - Web-based station management (create/edit/delete)
  - No restart required for station changes
  - Added Station CRUD API endpoints
  - Enhanced station schema with auto-start, simulation settings, and tags
  - Added station templates and import/export features
  - Updated implementation phases to reflect database-driven configuration
- v1.2: Added MongoDB as database with complete schema design, collections, indexes, and Docker Compose setup
- v1.1: Updated technology stack to use standard `net/http`, custom OCPP protocol implementation, and `log/slog` for logging
- v1.0: Initial plan
