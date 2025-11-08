# OCPP Emulator Seed Data

This directory contains seed data for quickly populating the OCPP Emulator with test stations.

## Files

### `stations.json`

Contains 10 pre-configured charging stations with diverse configurations for testing various scenarios.

## Station Overview

| ID | Name | Protocol | Vendor | Power | Auto-Start | Status | Use Case |
|----|------|----------|--------|-------|------------|--------|----------|
| CP001 | Main Entrance | OCPP 1.6 | ABB | 22 kW AC | ✅ | Enabled | Production - Dual AC charger |
| CP002 | Parking Lot B | OCPP 1.6 | Schneider | 11 kW AC | ❌ | Enabled | Production - Single AC charger |
| CP003 | DC Fast Charger | OCPP 1.6 | ChargePoint | 50 kW DC | ✅ | Enabled | Production - Multi-standard fast |
| CP004 | Tesla Compatible | OCPP 1.6 | Tesla | 22 kW AC | ❌ | Disabled | Test - Wall connector |
| CP005 | OCPP 2.0.1 Test | OCPP 2.0.1 | Siemens | 22 kW AC | ❌ | Enabled | Development - Protocol testing |
| CP006 | Multi-Standard | OCPP 1.6 | EVBox | 100 kW DC | ✅ | Enabled | Production - Ultra-fast multi-plug |
| CP007 | Home Charger | OCPP 1.6 | Wallbox | 7.4 kW AC | ❌ | Enabled | Residential - Basic home charging |
| CP008 | Fleet Management | OCPP 1.6 | ABB | 175 kW DC | ✅ | Enabled | Production - Fleet depot charging |
| CP009 | Smart Charging | OCPP 1.6 | Enel X | 22 kW AC | ❌ | Enabled | Test - Smart charging features |
| CP010 | Diagnostic | OCPP 1.6 | Generic | 3.7 kW AC | ❌ | Disabled | Development - Testing/debugging |

## Loading Seed Data

### Method 1: Using MongoDB Import (Recommended)

```bash
# Import all stations directly into MongoDB
docker exec -i ocpp-emu-mongodb mongosh ocpp_emu --eval '
  db.stations.deleteMany({});
' < /dev/null

docker exec -i ocpp-emu-mongodb mongoimport \
  --db ocpp_emu \
  --collection stations \
  --file /tmp/stations.json \
  --jsonArray
```

Or copy the file into the container first:

```bash
# Copy file to container
docker cp testdata/seed/stations.json ocpp-emu-mongodb:/tmp/stations.json

# Import into MongoDB
docker exec ocpp-emu-mongodb mongoimport \
  --db ocpp_emu \
  --collection stations \
  --file /tmp/stations.json \
  --jsonArray
```

### Method 2: Using MongoDB Compass (GUI)

1. Open MongoDB Compass
2. Connect to `mongodb://localhost:27017`
3. Navigate to database `ocpp_emu` > collection `stations`
4. Click "Add Data" > "Import JSON or CSV file"
5. Select `testdata/seed/stations.json`
6. Click "Import"

### Method 3: Using the REST API

You can import stations one by one using the API:

```bash
# Read the JSON file and POST each station
cat testdata/seed/stations.json | jq -c '.[]' | while read station; do
  curl -X POST http://localhost:8080/api/stations \
    -H "Content-Type: application/json" \
    -d "$station"
done
```

### Method 4: Using mongosh CLI

```bash
# Connect to MongoDB
docker exec -it ocpp-emu-mongodb mongosh ocpp_emu

# Load the file (if copied to container)
load('/tmp/stations.json')

# Or paste the JSON directly:
db.stations.insertMany([
  // ... paste JSON content here
])
```

## Verifying Imported Data

After importing, verify the stations were loaded:

```bash
# Check station count
curl http://localhost:8080/api/stations | jq '.count'

# List all stations
curl http://localhost:8080/api/stations | jq '.stations[] | {stationId, name, enabled}'

# Check auto-start stations
curl http://localhost:8080/api/stations | jq '.stations[] | select(.autoStart == true) | .stationId'
```

Or via MongoDB:

```bash
# Count stations
docker exec ocpp-emu-mongodb mongosh ocpp_emu --eval 'db.stations.countDocuments()'

# List enabled stations
docker exec ocpp-emu-mongodb mongosh ocpp_emu --eval '
  db.stations.find({enabled: true}, {station_id: 1, name: 1, auto_start: 1})
'
```

## Station Details

### CP001 - Main Entrance (Production)
- **Type**: AC Level 2 charging
- **Connectors**: 2x Type2 (22 kW each)
- **Features**: Full profile support (Smart Charging)
- **Auto-start**: Yes
- **Meter interval**: 60s
- **Use case**: Public parking, office building

### CP002 - Parking Lot B (Production)
- **Type**: AC Level 2 charging
- **Connectors**: 1x Type2 (11 kW)
- **Features**: Basic profiles
- **Auto-start**: No
- **Meter interval**: 30s
- **Use case**: Secondary parking area

### CP003 - DC Fast Charger (Production)
- **Type**: DC fast charging
- **Connectors**: 1x CCS (50 kW) + 1x CHAdeMO (50 kW)
- **Features**: Full profile support
- **Auto-start**: Yes
- **Meter interval**: 15s
- **Use case**: Highway rest stop, commercial

### CP004 - Tesla Compatible (Disabled)
- **Type**: AC Level 2 charging
- **Connectors**: 1x Type2 (22 kW)
- **Features**: Basic profiles
- **Auto-start**: No
- **Status**: Disabled for testing
- **Use case**: Tesla-specific installations

### CP005 - OCPP 2.0.1 Test (Development)
- **Type**: AC Level 2 charging
- **Protocol**: OCPP 2.0.1
- **Connectors**: 3x Type2 (2x 22kW, 1x 11kW)
- **Features**: Advanced profiles + LocalAuthList
- **Auto-start**: No
- **Use case**: Protocol compliance testing

### CP006 - Multi-Standard (Production)
- **Type**: DC ultra-fast charging
- **Connectors**: 1x CCS (100 kW) + 1x CHAdeMO (50 kW) + 1x Type2 (43 kW)
- **Features**: Multi-standard support
- **Auto-start**: Yes
- **Meter interval**: 10s
- **Use case**: Highway, commercial fleet

### CP007 - Home Charger (Residential)
- **Type**: AC Level 1/2 charging
- **Connectors**: 1x Type2 (7.4 kW)
- **Features**: Basic profiles
- **Auto-start**: No
- **Meter interval**: 120s
- **Use case**: Residential home charging

### CP008 - Fleet Management (Production)
- **Type**: DC ultra-fast charging
- **Connectors**: 2x CCS (175 kW each)
- **Features**: Full profile support
- **Auto-start**: Yes
- **Meter interval**: 5s (high frequency)
- **Use case**: Fleet depot, commercial trucks/buses

### CP009 - Smart Charging Test (Test)
- **Type**: AC Level 2 charging
- **Connectors**: 1x Type2 (22 kW)
- **Features**: Smart Charging profile
- **Auto-start**: No
- **Meter interval**: 30s
- **Use case**: V2G testing, demand response

### CP010 - Diagnostic Station (Disabled)
- **Type**: AC Level 1 charging
- **Connectors**: 1x Type2 (3.7 kW)
- **Features**: All profiles (for testing)
- **Auto-start**: No
- **Status**: Disabled
- **Meter interval**: 10s
- **Use case**: Development, diagnostics

## Configuration Notes

### Auto-Start Stations
The following stations will automatically connect on server startup:
- CP001 (Main Entrance)
- CP003 (DC Fast Charger)
- CP006 (Multi-Standard)
- CP008 (Fleet Management)

### Disabled Stations
These stations are disabled and won't appear in active lists:
- CP004 (Tesla Compatible) - Testing
- CP010 (Diagnostic) - Development

### Protocol Versions
- **OCPP 1.6**: CP001, CP002, CP003, CP004, CP006, CP007, CP008, CP009, CP010
- **OCPP 2.0.1**: CP005

### Connector Types
- **Type2**: Standard European AC connector
- **CCS**: Combined Charging System (DC fast)
- **CHAdeMO**: Japanese DC fast charging standard

## Customization

To customize the seed data:

1. Edit `testdata/seed/stations.json`
2. Modify fields as needed:
   - `enabled`: Set to `true` or `false`
   - `auto_start`: Enable/disable auto-connection
   - `csms_url`: Change CSMS endpoint
   - `simulation`: Adjust behavior parameters
   - `tags`: Add custom tags for filtering

3. Re-import using one of the methods above

## Testing Scenarios

### Scenario 1: Load Testing
Enable all 10 stations with auto-start to test multiple simultaneous connections.

### Scenario 2: Protocol Testing
Use CP005 to test OCPP 2.0.1 compatibility.

### Scenario 3: Multi-Connector Testing
Use CP001, CP003, CP005, CP006, or CP008 for multi-connector scenarios.

### Scenario 4: Power Diversity
Stations range from 3.7 kW to 175 kW for testing various power levels.

## Cleaning Up

To remove all seed data:

```bash
# Via MongoDB
docker exec ocpp-emu-mongodb mongosh ocpp_emu --eval 'db.stations.deleteMany({})'

# Via API (one by one)
for id in CP001 CP002 CP003 CP004 CP005 CP006 CP007 CP008 CP009 CP010; do
  curl -X DELETE http://localhost:8080/api/stations/$id
done
```

## Support

For issues or questions about seed data:
1. Check the main PLAN.md for architecture details
2. Review PHASE1_STATION_API.md for API documentation
3. Consult MongoDB schema in PHASE1_MONGODB.md
