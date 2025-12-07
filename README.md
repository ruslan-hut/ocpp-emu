# OCPP Charging Station Emulator

A web-based EV charging station emulator supporting OCPP 1.6, 2.0.1, and 2.1 protocols. Designed to test and diagnose OCPP-compliant remote servers (CSMS - Charging Station Management Systems).

## Features

- 🔌 **Multi-Protocol Support**: OCPP 1.6, 2.0.1, and 2.1
- 🌐 **Web-Based Management**: Create and manage stations through Web UI
- 📊 **Real-Time Message Streaming**: WebSocket-based live message inspector with filtering
- ✏️ **Custom Message Crafter**: Send arbitrary messages for edge case testing
- 🗄️ **MongoDB Backend**: Persistent storage for stations, messages, and transactions
- 🔄 **Hot-Reload**: Add/modify stations without restart
- 📈 **Time-Series Data**: Optimized meter value storage
- 🎯 **Zero Downtime**: Configuration changes without restart
- 🐳 **Docker Ready**: Complete containerization with docker-compose
- 📝 **Structured Logging**: JSON logging with slog for production-ready observability

## Quick Start

### Prerequisites

- Go 1.21+
- MongoDB 7.0+
- Docker & Docker Compose (optional)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/ruslanhut/ocpp-emu.git
cd ocpp-emu
```

2. Install dependencies:
```bash
go mod download
```

3. Start MongoDB (using Docker):
```bash
docker-compose up -d mongodb
```

4. Run the application:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

### Using Docker Compose (Recommended)

**Quick Start - Run everything in containers:**

```bash
# Build and start all services
docker-compose up --build

# Or run in background
docker-compose up -d
```

This will start:
- **MongoDB** on port 27017 (with auto-initialization)
- **Backend** on port 8080 (OCPP emulator API)
- **Frontend** on port 3000 (Web UI with Nginx)

**Access the application:**
- 🌐 Frontend: http://localhost:3000
- 🔌 Backend API: http://localhost:8080/api/health
- 📊 WebSocket Messages: ws://localhost:3000/api/ws/messages

**Useful Docker commands:**
```bash
# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Rebuild after code changes
docker-compose build backend
docker-compose up backend

# Access MongoDB shell
docker-compose exec mongodb mongosh ocpp_emu
```

📖 See [docs/DOCKER.md](docs/DOCKER.md) for detailed Docker setup guide and troubleshooting.

## Configuration

Configuration is split into two parts:

### 1. Application Configuration (configs/config.yaml)

Static application settings:
- Server port and host
- MongoDB connection
- Logging configuration
- CSMS defaults

### 2. Station Configuration (MongoDB)

Dynamic station configurations managed through Web UI:
- Station identity and hardware info
- Protocol version
- Connector configuration
- OCPP features
- Simulation behavior

See [docs/PLAN.md](docs/PLAN.md) for detailed configuration options.

## Project Structure

```
ocpp-emu/
├── cmd/server/          # Application entry point
├── internal/            # Internal packages
│   ├── station/         # Station management
│   ├── ocpp/           # OCPP protocol implementation
│   ├── connection/     # WebSocket management
│   ├── logger/         # Message logging
│   ├── api/            # HTTP/WebSocket API
│   └── storage/        # MongoDB integration
├── configs/            # Configuration files
├── web/                # Frontend application
├── testdata/           # Test data and scenarios
└── docker/             # Docker configurations
```

## API Endpoints

### Health Check
```
GET /api/health
```

### Station Management
```
GET    /api/stations              - List all stations
GET    /api/stations/:id          - Get station details
POST   /api/stations              - Create new station
PUT    /api/stations/:id          - Update station
DELETE /api/stations/:id          - Delete station
PATCH  /api/stations/:id/start    - Start station
PATCH  /api/stations/:id/stop     - Stop station
POST   /api/stations/:id/clone    - Clone station
GET    /api/stations/export       - Export stations
POST   /api/stations/import       - Import stations
```

### Message Streaming (WebSocket)
```
WS     /api/ws/messages           - Real-time message stream
GET    /api/ws/stats              - Broadcaster statistics
```

Query parameters for WebSocket:
- `stationId` - Filter by station ID
- `direction` - Filter by direction (sent/received)
- `messageType` - Filter by type (Call/CallResult/CallError)

### Message History
```
GET    /api/messages              - Get message history (with filters)
GET    /api/messages/search       - Search messages
GET    /api/messages/stats        - Message statistics
DELETE /api/messages              - Clear all messages
```

## Development

### Build
```bash
make build
```

### Run tests
```bash
make test
```

### Run with live reload
```bash
make dev
```

### Format code
```bash
make fmt
```

## Architecture

- **Backend**: Go with standard library HTTP, custom OCPP implementation
- **Database**: MongoDB for persistence
- **Frontend**: React 18 with Vite
- **Logging**: Standard library slog for structured logging
- **WebSocket**: gorilla/websocket for protocol handling

## OCPP Protocol Support

### OCPP 1.6 ✅ Implemented
- ✅ Core Profile (BootNotification, Heartbeat, StatusNotification, Authorize, StartTransaction, StopTransaction, MeterValues, DataTransfer)
- ⏳ Firmware Management (Planned)
- ⏳ Remote Control (Planned)
- ⏳ Smart Charging (Planned)

### OCPP 2.0.1 (Planned)
- Core functionality
- Security features
- Device management

### OCPP 2.1 (Planned)
- Enhanced features
- Cost and tariff messages

## Contributing

This project is in active development. See [docs/PLAN.md](docs/PLAN.md) for the development roadmap.

## License

MIT License - see LICENSE file for details

## Resources

- [OCPP Specifications](https://www.openchargealliance.org/)
- [Project Plan](docs/PLAN.md)
- [MongoDB Documentation](https://www.mongodb.com/docs/)

## Status

✅ **Phase 1 & 2 Complete** - OCPP 1.6 support implemented
🚧 **Phase 3 In Progress** - Enhanced features and message crafting

Current Version: 0.1.0

For detailed status and roadmap, see [docs/PLAN.md](docs/PLAN.md).
