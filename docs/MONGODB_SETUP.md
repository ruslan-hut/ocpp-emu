# MongoDB Setup and Configuration

This document describes the MongoDB setup for the OCPP Emulator project.

## Overview

The OCPP Emulator uses MongoDB as its primary database for storing:
- OCPP messages (with full message history)
- Charging transactions
- Station configurations
- WebSocket sessions
- Meter values (time-series data)

## Architecture

### MongoDB Client

The MongoDB client is implemented in `internal/storage/mongodb.go` and provides:
- Connection pooling
- Automatic collection creation
- Index management
- Health checks and monitoring
- Time-series collection support for meter values

### Collections

The following collections are created automatically on startup:

1. **messages** - All OCPP messages exchanged
2. **transactions** - Charging transaction records
3. **stations** - Station configurations and state
4. **sessions** - WebSocket session information
5. **meter_values** - Time-series meter value samples

### Data Models

All data models are defined in `internal/storage/models.go`:
- `Message` - OCPP message with metadata
- `Transaction` - Charging transaction
- `Station` - Station configuration
- `Session` - WebSocket session
- `MeterValue` - Meter value sample

## Configuration

MongoDB configuration is loaded from `configs/config.yaml`:

```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "ocpp_emu"
  connection_timeout: 10s
  max_pool_size: 100

  collections:
    messages: "messages"
    transactions: "transactions"
    stations: "stations"
    sessions: "sessions"
    meter_values: "meter_values"

  timeseries:
    enabled: true
    granularity: "seconds"
```

### Environment Variable Overrides

You can override configuration using environment variables:
- `MONGODB_URI` - MongoDB connection URI
- `MONGODB_DATABASE` - Database name

Example:
```bash
export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DATABASE="ocpp_emu"
```

## Usage

### Starting MongoDB

Using Docker Compose:
```bash
docker compose up -d mongodb
```

Using local MongoDB:
```bash
mongod --dbpath /path/to/data
```

### Running the Application

```bash
# Build
make build

# Run
./bin/server

# Or use the Makefile
make run
```

### Verifying Connection

The application logs will show:
```json
{"level":"info","msg":"Connecting to MongoDB","uri":"mongodb://localhost:27017","database":"ocpp_emu"}
{"level":"info","msg":"Successfully connected to MongoDB"}
{"level":"info","msg":"MongoDB connection established successfully"}
```

You can also check the health endpoint:
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","version":"0.1.0","database":"connected"}
```

## Indexes

The following indexes are created automatically:

### Messages Collection
- `{station_id: 1, timestamp: -1}` - Query messages by station
- `{message_id: 1}` - Lookup by message ID
- `{correlation_id: 1}` - Link requests with responses
- `{action: 1, timestamp: -1}` - Query by message type
- `{timestamp: -1}` - Time-based queries

### Transactions Collection
- `{transaction_id: 1}` (unique) - Lookup by transaction ID
- `{station_id: 1, start_timestamp: -1}` - Query by station
- `{status: 1}` - Filter by status
- `{id_tag: 1}` - Query by authorization tag

### Stations Collection
- `{station_id: 1}` (unique) - Lookup by station ID
- `{connection_status: 1}` - Filter by connection status
- `{enabled: 1, auto_start: 1}` - Query enabled stations
- `{tags: 1}` - Filter by tags
- `{protocol_version: 1}` - Filter by OCPP version

### Sessions Collection
- `{station_id: 1, status: 1}` - Query sessions by station
- `{status: 1}` - Filter by status

## Time-Series Collection

Meter values are stored in a time-series collection for optimal performance:

```javascript
db.createCollection("meter_values", {
  timeseries: {
    timeField: "timestamp",
    metaField: "metadata",
    granularity: "seconds"
  }
})
```

Benefits:
- Optimized storage for time-series data
- Efficient queries by time range
- Better compression
- Improved query performance

## MongoDB Client API

The `MongoDBClient` provides the following methods:

```go
// Create new client
client, err := storage.NewMongoDBClient(ctx, &cfg.MongoDB, logger)

// Ping the database
err := client.Ping(ctx)

// Health check (verifies collections exist)
err := client.HealthCheck(ctx)

// Get statistics
stats, err := client.Stats(ctx)

// Close connection
err := client.Close(ctx)

// Access collections directly
client.MessagesCollection
client.TransactionsCollection
client.StationsCollection
client.SessionsCollection
client.MeterValuesCollection
```

## Testing

### Unit Tests

Run unit tests (no MongoDB required):
```bash
go test -v ./internal/storage/... -run TestMongoDBClientCreation
```

### Integration Tests

Run integration tests (requires MongoDB):
```bash
# Start MongoDB first
docker compose up -d mongodb

# Run tests
go test -v ./internal/storage/... -run TestMongoDBConnection
```

### Manual Testing

1. Start MongoDB:
   ```bash
   docker compose up -d mongodb
   ```

2. Build and run the application:
   ```bash
   make build
   ./bin/server
   ```

3. Check health endpoint:
   ```bash
   curl http://localhost:8080/health
   ```

4. View logs to verify connection

## Troubleshooting

### Connection Failed

If you see `"Failed to connect to MongoDB"`:

1. Verify MongoDB is running:
   ```bash
   docker compose ps mongodb
   ```

2. Check MongoDB logs:
   ```bash
   docker compose logs mongodb
   ```

3. Test connection manually:
   ```bash
   mongosh mongodb://localhost:27017/ocpp_emu
   ```

### Collection Creation Failed

If indexes fail to create:
1. Check MongoDB version (requires 5.0+ for time-series)
2. Verify database permissions
3. Check disk space

### Health Check Failed

If health check fails but connection works:
1. Check if all collections were created
2. Verify MongoDB user has read permissions
3. Check application logs for specific errors

## Production Considerations

### Connection Pooling

The application uses connection pooling with configurable pool size:
```yaml
mongodb:
  max_pool_size: 100  # Adjust based on load
```

### Connection Timeout

Configure appropriate timeouts:
```yaml
mongodb:
  connection_timeout: 10s  # Increase for slow networks
```

### Security

For production deployments:

1. Use authentication:
   ```yaml
   mongodb:
     uri: "mongodb://username:password@host:27017/ocpp_emu?authSource=admin"
   ```

2. Enable TLS/SSL:
   ```yaml
   mongodb:
     uri: "mongodb://host:27017/ocpp_emu?tls=true&tlsCAFile=/path/to/ca.pem"
   ```

3. Use replica sets for high availability:
   ```yaml
   mongodb:
     uri: "mongodb://host1:27017,host2:27017,host3:27017/ocpp_emu?replicaSet=rs0"
   ```

### Monitoring

Use the stats endpoint to monitor database health:
```go
stats, err := client.Stats(ctx)
// Returns database stats and collection counts
```

### Backup

Regular backups are essential:
```bash
# Backup
mongodump --uri="mongodb://localhost:27017/ocpp_emu" --out=/backup/path

# Restore
mongorestore --uri="mongodb://localhost:27017/ocpp_emu" /backup/path/ocpp_emu
```

## Next Steps

- [ ] Implement repository pattern for data access
- [ ] Add MongoDB Change Streams for real-time updates
- [ ] Implement data retention policies (TTL indexes)
- [ ] Add aggregation pipelines for analytics
- [ ] Implement bulk insert for message logging

## References

- [MongoDB Go Driver Documentation](https://pkg.go.dev/go.mongodb.org/mongo-driver)
- [MongoDB Time-Series Collections](https://www.mongodb.com/docs/manual/core/timeseries-collections/)
- [MongoDB Indexes](https://www.mongodb.com/docs/manual/indexes/)
