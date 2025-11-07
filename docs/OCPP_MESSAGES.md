## OCPP Message Structures

Complete implementation of OCPP message framing and payloads for protocol versions 1.6, 2.0.1, and 2.1.

## Overview

The OCPP Emulator implements a custom OCPP message handling system following the official OCPP specifications from the Open Charge Alliance.

### Message Types

All OCPP messages use JSON-RPC 2.0 format over WebSocket with three message types:

| Type | ID | Format | Description |
|------|----|----|-------------|
| **Call** | 2 | `[2, "uniqueId", "Action", {payload}]` | Request from client to server |
| **CallResult** | 3 | `[3, "uniqueId", {payload}]` | Successful response |
| **CallError** | 4 | `[4, "uniqueId", "ErrorCode", "ErrorDescription", {details}]` | Error response |

## Architecture

```
┌─────────────────────────────────────────────────┐
│            OCPP Message Layer                    │
│  ┌───────────────────────────────────────────┐  │
│  │   Common Message Framing (message.go)     │  │
│  │   - Call, CallResult, CallError           │  │
│  │   - Message Parser                        │  │
│  │   - ID Generation                         │  │
│  └───────────────────────────────────────────┘  │
│                       │                          │
│          ┌────────────┼────────────┐            │
│          │            │            │            │
│  ┌───────▼──────┐ ┌──▼──────┐ ┌──▼──────┐     │
│  │  OCPP 1.6    │ │ OCPP    │ │ OCPP    │     │
│  │   v16/       │ │ 2.0.1   │ │  2.1    │     │
│  │              │ │ v201/   │ │  v21/   │     │
│  └──────────────┘ └─────────┘ └─────────┘     │
└─────────────────────────────────────────────────┘
```

## Core Components

### 1. Message Framing (`internal/ocpp/message.go`)

#### Call Message

Represents a request from charge point to CSMS:

```go
type Call struct {
    MessageTypeID MessageType
    UniqueID      string
    Action        string
    Payload       json.RawMessage
}
```

**Example JSON:**
```json
[2, "19223201", "BootNotification", {
  "chargePointVendor": "VendorX",
  "chargePointModel": "SingleSocketCharger"
}]
```

**Creating a Call:**
```go
payload := v16.BootNotificationRequest{
    ChargePointVendor: "VendorX",
    ChargePointModel: "SingleSocketCharger",
}

call, err := ocpp.NewCall("BootNotification", payload)
if err != nil {
    // Handle error
}

bytes, _ := call.ToBytes()
// Send bytes over WebSocket
```

#### CallResult Message

Represents a successful response from CSMS:

```go
type CallResult struct {
    MessageTypeID MessageType
    UniqueID      string
    Payload       json.RawMessage
}
```

**Example JSON:**
```json
[3, "19223201", {
  "status": "Accepted",
  "currentTime": "2025-01-07T12:00:00Z",
  "interval": 300
}]
```

**Creating a CallResult:**
```go
payload := v16.BootNotificationResponse{
    Status: v16.RegistrationStatusAccepted,
    CurrentTime: ocpp.DateTime{Time: time.Now()},
    Interval: 300,
}

result, err := ocpp.NewCallResult(call.UniqueID, payload)
bytes, _ := result.ToBytes()
```

#### CallError Message

Represents an error response:

```go
type CallError struct {
    MessageTypeID MessageType
    UniqueID      string
    ErrorCode     ErrorCode
    ErrorDesc     string
    ErrorDetails  json.RawMessage
}
```

**Example JSON:**
```json
[4, "19223201", "NotSupported", "BootNotification is not supported", {}]
```

**Creating a CallError:**
```go
callError, err := ocpp.NewCallError(
    call.UniqueID,
    ocpp.ErrorCodeNotSupported,
    "Action not supported",
    nil,
)
```

### 2. Error Codes

Standard OCPP error codes:

| Error Code | Description |
|-----------|-------------|
| `NotImplemented` | Action is not known by receiver |
| `NotSupported` | Action is recognized but not supported |
| `InternalError` | Internal error occurred |
| `ProtocolError` | Payload is incomplete |
| `SecurityError` | Security issue occurred |
| `FormationViolation` | Payload is syntactically incorrect |
| `PropertyConstraintViolation` | Payload violates constraints |
| `OccurrenceConstraintViolation` | Field contains invalid value |
| `TypeConstraintViolation` | Field violates data type constraints |
| `GenericError` | Any other error |

### 3. Message Parsing

Parse any OCPP message:

```go
data := []byte(`[2,"123","Heartbeat",{}]`)

message, err := ocpp.ParseMessage(data)
if err != nil {
    // Handle parse error
}

switch msg := message.(type) {
case *ocpp.Call:
    // Handle Call
    fmt.Printf("Action: %s\n", msg.Action)

case *ocpp.CallResult:
    // Handle CallResult
    fmt.Printf("Response for: %s\n", msg.UniqueID)

case *ocpp.CallError:
    // Handle CallError
    fmt.Printf("Error: %s\n", msg.ErrorCode)
}
```

### 4. Message ID Generation

Unique message IDs are automatically generated:

```go
// Generate new UUID
id := ocpp.GenerateMessageID()
// Output: "550e8400-e29b-41d4-a716-446655440000"

// Validate message ID
if ocpp.ValidateMessageID(id) {
    // Valid UUID format
}
```

## OCPP 1.6 Core Profile

### Message Types (`internal/ocpp/v16/types.go`)

Comprehensive type definitions for OCPP 1.6:

- **Actions** - All OCPP 1.6 actions (Core, Firmware, Smart Charging, etc.)
- **Status Enums** - ChargePointStatus, RegistrationStatus, AuthorizationStatus
- **Error Codes** - Charge point error codes
- **Measurands** - Energy, power, current, voltage, etc.
- **Context** - Reading context (periodic, transaction, etc.)
- **Custom Types** - DateTime, IdTagInfo, SampledValue, MeterValue

### Core Profile Messages (`internal/ocpp/v16/messages.go`)

#### Authorize

```go
// Request
type AuthorizeRequest struct {
    IdTag string `json:"idTag"`
}

// Response
type AuthorizeResponse struct {
    IdTagInfo IdTagInfo `json:"idTagInfo"`
}

// Example
req := v16.AuthorizeRequest{IdTag: "RFID123"}
call, _ := ocpp.NewCall(string(v16.ActionAuthorize), req)
```

#### BootNotification

```go
// Request
type BootNotificationRequest struct {
    ChargePointVendor       string
    ChargePointModel        string
    ChargePointSerialNumber string
    // ... other fields
}

// Response
type BootNotificationResponse struct {
    Status      RegistrationStatus
    CurrentTime DateTime
    Interval    int
}

// Example
req := v16.BootNotificationRequest{
    ChargePointVendor: "Acme",
    ChargePointModel: "Model X",
}
call, _ := ocpp.NewCall(string(v16.ActionBootNotification), req)
```

#### Heartbeat

```go
// Request (empty payload)
type HeartbeatRequest struct{}

// Response
type HeartbeatResponse struct {
    CurrentTime DateTime
}
```

#### StatusNotification

```go
// Request
type StatusNotificationRequest struct {
    ConnectorId int
    ErrorCode   ChargePointErrorCode
    Status      ChargePointStatus
    Timestamp   *DateTime
    // ... optional fields
}

// Response (empty)
type StatusNotificationResponse struct{}

// Example
req := v16.StatusNotificationRequest{
    ConnectorId: 1,
    ErrorCode: v16.ChargePointErrorNoError,
    Status: v16.ChargePointStatusAvailable,
}
```

#### StartTransaction

```go
// Request
type StartTransactionRequest struct {
    ConnectorId   int
    IdTag         string
    MeterStart    int
    Timestamp     DateTime
    ReservationId *int
}

// Response
type StartTransactionResponse struct {
    IdTagInfo     IdTagInfo
    TransactionId int
}
```

#### StopTransaction

```go
// Request
type StopTransactionRequest struct {
    TransactionId   int
    IdTag           string
    MeterStop       int
    Timestamp       DateTime
    Reason          Reason
    TransactionData []MeterValue
}

// Response
type StopTransactionResponse struct {
    IdTagInfo *IdTagInfo
}
```

#### MeterValues

```go
// Request
type MeterValuesRequest struct {
    ConnectorId   int
    TransactionId *int
    MeterValue    []MeterValue
}

// Response (empty)
type MeterValuesResponse struct{}

// Example
meterValue := v16.MeterValue{
    Timestamp: v16.DateTime{Time: time.Now()},
    SampledValue: []v16.SampledValue{
        {
            Value: "15000",
            Unit: v16.UnitOfMeasureWh,
            Measurand: v16.MeasurandEnergyActiveImportRegister,
        },
    },
}

req := v16.MeterValuesRequest{
    ConnectorId: 1,
    MeterValue: []v16.MeterValue{meterValue},
}
```

### Remote Operations

#### RemoteStartTransaction

```go
type RemoteStartTransactionRequest struct {
    ConnectorId     *int
    IdTag           string
    ChargingProfile interface{}
}

type RemoteStartTransactionResponse struct {
    Status string // "Accepted" or "Rejected"
}
```

#### RemoteStopTransaction

```go
type RemoteStopTransactionRequest struct {
    TransactionId int
}

type RemoteStopTransactionResponse struct {
    Status string // "Accepted" or "Rejected"
}
```

#### Reset

```go
type ResetRequest struct {
    Type string // "Hard" or "Soft"
}

type ResetResponse struct {
    Status string // "Accepted" or "Rejected"
}
```

### Configuration Operations

#### GetConfiguration

```go
type GetConfigurationRequest struct {
    Key []string // Optional: specific keys to retrieve
}

type GetConfigurationResponse struct {
    ConfigurationKey []KeyValue
    UnknownKey       []string
}
```

#### ChangeConfiguration

```go
type ChangeConfigurationRequest struct {
    Key   string
    Value string
}

type ChangeConfigurationResponse struct {
    Status string // Accepted, Rejected, RebootRequired, NotSupported
}
```

## Custom DateTime Type

OCPP uses ISO 8601 format for date-time:

```go
type DateTime struct {
    time.Time
}

// Creating
dt := v16.DateTime{Time: time.Now()}

// JSON marshaling
// Output: "2025-01-07T12:00:00Z"

// Parsing
var dt v16.DateTime
json.Unmarshal([]byte(`"2025-01-07T12:00:00Z"`), &dt)
```

## Usage Examples

### Complete BootNotification Flow

```go
// 1. Create BootNotification request
bootReq := v16.BootNotificationRequest{
    ChargePointVendor: "Acme Corp",
    ChargePointModel: "AC-001",
    FirmwareVersion: "1.0.0",
}

// 2. Create Call message
call, err := ocpp.NewCall(string(v16.ActionBootNotification), bootReq)
if err != nil {
    log.Fatal(err)
}

// 3. Convert to bytes
messageBytes, _ := call.ToBytes()

// 4. Send over WebSocket
conn.SendMessage(stationID, messageBytes)

// 5. Receive response
responseBytes := <-responseChan

// 6. Parse response
message, _ := ocpp.ParseMessage(responseBytes)
callResult := message.(*ocpp.CallResult)

// 7. Parse payload
var bootResp v16.BootNotificationResponse
json.Unmarshal(callResult.Payload, &bootResp)

// 8. Handle response
if bootResp.Status == v16.RegistrationStatusAccepted {
    log.Printf("Accepted! Heartbeat interval: %d", bootResp.Interval)
}
```

### Complete Charging Session

```go
// 1. Authorize
authReq := v16.AuthorizeRequest{IdTag: "RFID-12345"}
authCall, _ := ocpp.NewCall(string(v16.ActionAuthorize), authReq)
// ... send and get response

// 2. Start Transaction
startReq := v16.StartTransactionRequest{
    ConnectorId: 1,
    IdTag: "RFID-12345",
    MeterStart: 0,
    Timestamp: v16.DateTime{Time: time.Now()},
}
startCall, _ := ocpp.NewCall(string(v16.ActionStartTransaction), startReq)
// ... send and get transaction ID

// 3. Send periodic meter values
for {
    meterReq := v16.MeterValuesRequest{
        ConnectorId: 1,
        TransactionId: &transactionID,
        MeterValue: []v16.MeterValue{
            {
                Timestamp: v16.DateTime{Time: time.Now()},
                SampledValue: []v16.SampledValue{
                    {Value: "5000", Unit: v16.UnitOfMeasureWh},
                },
            },
        },
    }
    // ... send meter values
    time.Sleep(60 * time.Second)
}

// 4. Stop Transaction
stopReq := v16.StopTransactionRequest{
    TransactionId: transactionID,
    MeterStop: 10000,
    Timestamp: v16.DateTime{Time: time.Now()},
    Reason: v16.ReasonLocal,
}
stopCall, _ := ocpp.NewCall(string(v16.ActionStopTransaction), stopReq)
// ... send and complete
```

## Testing

### Unit Tests

All message types have comprehensive unit tests:

```bash
# Run all OCPP message tests
go test ./internal/ocpp/... -v

# Run specific tests
go test ./internal/ocpp -run TestCall
go test ./internal/ocpp -run TestParse
go test ./internal/ocpp -run TestGenerate
```

### Test Examples

```go
// Test Call message
func TestCallMessage(t *testing.T) {
    call, _ := ocpp.NewCall("Heartbeat", struct{}{})
    bytes, _ := call.ToBytes()

    // Verify format
    var arr []interface{}
    json.Unmarshal(bytes, &arr)
    assert.Equal(t, 4, len(arr))
    assert.Equal(t, 2, int(arr[0].(float64)))
}
```

## Message Validation

### Field Validation

OCPP messages include validation tags:

```go
type BootNotificationRequest struct {
    ChargePointVendor string `json:"chargePointVendor" validate:"required,max=20"`
    ChargePointModel  string `json:"chargePointModel" validate:"required,max=20"`
    // ...
}
```

**Validation rules:**
- `required` - Field is mandatory
- `max=N` - Maximum string length
- `gte=N` - Greater than or equal (numbers)
- `gt=N` - Greater than (numbers)

### Error Handling

```go
message, err := ocpp.ParseMessage(data)
if err != nil {
    // Send CallError response
    callError, _ := ocpp.NewCallError(
        uniqueID,
        ocpp.ErrorCodeFormationViolation,
        "Invalid message format",
        nil,
    )
    // Send callError back
}
```

## Best Practices

### 1. Message Correlation

Always correlate requests with responses using UniqueID:

```go
// Store pending requests
pendingCalls := make(map[string]*ocpp.Call)

// When sending
call, _ := ocpp.NewCall("Heartbeat", struct{}{})
pendingCalls[call.UniqueID] = call
// ... send

// When receiving response
callResult := message.(*ocpp.CallResult)
originalCall := pendingCalls[callResult.UniqueID]
delete(pendingCalls, callResult.UniqueID)
```

### 2. Timeout Handling

```go
timeout := time.After(30 * time.Second)
select {
case response := <-responseChan:
    // Handle response
case <-timeout:
    // Send timeout error
    callError, _ := ocpp.NewCallError(
        call.UniqueID,
        ocpp.ErrorCodeInternalError,
        "Request timeout",
        nil,
    )
}
```

### 3. Error Responses

Always respond with appropriate error codes:

```go
if err := validatePayload(payload); err != nil {
    return ocpp.NewCallError(
        uniqueID,
        ocpp.ErrorCodePropertyConstraintViolation,
        err.Error(),
        nil,
    )
}
```

## Future Enhancements

- [ ] OCPP 2.0.1 message types
- [ ] OCPP 2.1 message types
- [ ] JSON schema validation
- [ ] Message encryption support
- [ ] Message signing/verification
- [ ] Protocol version negotiation
- [ ] Advanced validation rules

## References

- [OCPP 1.6 Specification](https://www.openchargealliance.org/protocols/ocpp-16/)
- [OCPP 2.0.1 Specification](https://www.openchargealliance.org/protocols/ocpp-201/)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)

---

**Implementation Status**: OCPP 1.6 Core Profile - ✅ Complete
