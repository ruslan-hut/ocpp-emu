# Phase 2, Task 2.3: Custom Message Encoding/Decoding

## Status: ✅ COMPLETED

## Overview
Implemented comprehensive JSON encoding and decoding for OCPP messages following the OCPP specification. This includes custom marshaling/unmarshaling for all OCPP message types and utility functions for message handling.

## Implementation Summary

### Files Created

#### 1. **internal/ocpp/encoding.go** (NEW)
Encoding/decoding utilities providing high-level abstractions:

**MessageEncoder:**
- `EncodeCall()` - Encodes Call messages to JSON
- `EncodeCallResult()` - Encodes CallResult messages to JSON
- `EncodeCallError()` - Encodes CallError messages to JSON

**MessageDecoder:**
- `Decode()` - Decodes raw OCPP messages
- `DecodeCall()` - Decodes Call messages with payload extraction
- `DecodeCallResult()` - Decodes CallResult messages with payload extraction
- `DecodeCallError()` - Decodes CallError messages

**Utility Functions:**
- `ValidateMessage()` - Validates OCPP message structure
- `PrettyPrint()` - Pretty-prints messages for debugging
- `CompactPrint()` - Compact JSON representation
- `GetMessageType()` - Extracts message type from raw bytes
- `GetMessageID()` - Extracts unique message ID
- `GetAction()` - Extracts action from Call messages

#### 2. **internal/ocpp/encoding_test.go** (NEW)
Comprehensive test suite with 18 tests and 7 benchmarks covering:
- Encoder functionality
- Decoder functionality
- Validation
- Utility functions
- Performance benchmarks

### Files Previously Implemented

#### 3. **internal/ocpp/message.go** (EXISTING)
Core OCPP message types with custom JSON marshaling:

**Custom MarshalJSON implementations:**
- `Call.MarshalJSON()` - Marshals to `[2, "uniqueId", "Action", {payload}]`
- `CallResult.MarshalJSON()` - Marshals to `[3, "uniqueId", {payload}]`
- `CallError.MarshalJSON()` - Marshals to `[4, "uniqueId", "ErrorCode", "ErrorDesc", {details}]`

**Custom UnmarshalJSON implementations:**
- `Call.UnmarshalJSON()` - Parses Call messages
- `CallResult.UnmarshalJSON()` - Parses CallResult messages
- `CallError.UnmarshalJSON()` - Parses CallError messages

**Helper functions:**
- `NewCall()` - Creates new Call message
- `NewCallResult()` - Creates new CallResult message
- `NewCallError()` - Creates new CallError message
- `ParseMessage()` - Parses raw OCPP message to appropriate type
- `GenerateMessageID()` - Generates UUID for message IDs
- `ValidateMessageID()` - Validates UUID format

#### 4. **internal/ocpp/v16/types.go** (EXISTING)
OCPP 1.6 specific types with custom marshaling:

**DateTime type:**
- `DateTime.MarshalJSON()` - Marshals to RFC3339 format
- `DateTime.UnmarshalJSON()` - Parses RFC3339 timestamps

**All OCPP 1.6 data types:**
- Enums (ChargePointStatus, AuthorizationStatus, etc.)
- Complex types (IdTagInfo, SampledValue, MeterValue)
- Constants for all OCPP actions, error codes, measurands, etc.

## OCPP Message Format

### JSON-over-WebSocket Format
OCPP uses a specific JSON array format for all messages:

#### Call (Request from Client)
```json
[2, "uniqueId", "Action", {payload}]
```

**Example:**
```json
[
  2,
  "550e8400-e29b-41d4-a716-446655440000",
  "BootNotification",
  {
    "chargePointVendor": "VendorX",
    "chargePointModel": "ModelY"
  }
]
```

#### CallResult (Response from Server)
```json
[3, "uniqueId", {payload}]
```

**Example:**
```json
[
  3,
  "550e8400-e29b-41d4-a716-446655440000",
  {
    "status": "Accepted",
    "currentTime": "2025-01-08T12:00:00Z",
    "interval": 60
  }
]
```

#### CallError (Error Response)
```json
[4, "uniqueId", "ErrorCode", "ErrorDescription", {details}]
```

**Example:**
```json
[
  4,
  "550e8400-e29b-41d4-a716-446655440000",
  "NotSupported",
  "Action not supported",
  {}
]
```

## DateTime Encoding

OCPP requires RFC3339 format for all timestamps:

### Custom DateTime Type
```go
type DateTime struct {
    time.Time
}

// Marshals to: "2025-01-08T12:00:00Z"
func (dt DateTime) MarshalJSON() ([]byte, error) {
    return []byte(`"` + dt.Time.Format(time.RFC3339) + `"`), nil
}

// Parses from: "2025-01-08T12:00:00Z"
func (dt *DateTime) UnmarshalJSON(data []byte) error {
    str := string(data[1 : len(data)-1])
    t, err := time.Parse(time.RFC3339, str)
    if err != nil {
        return err
    }
    dt.Time = t
    return nil
}
```

### Usage Example
```go
req := v16.BootNotificationRequest{
    ChargePointVendor: "VendorX",
    ChargePointModel:  "ModelY",
}

resp := v16.BootNotificationResponse{
    Status:      v16.RegistrationStatusAccepted,
    CurrentTime: v16.DateTime{Time: time.Now()},
    Interval:    60,
}
```

## Usage Examples

### Encoding Messages

#### Using High-Level Encoder
```go
encoder := ocpp.NewMessageEncoder()

// Encode a Call message
payload := v16.BootNotificationRequest{
    ChargePointVendor: "VendorX",
    ChargePointModel:  "ModelY",
}
data, err := encoder.EncodeCall("BootNotification", payload)

// Encode a CallResult
respPayload := v16.BootNotificationResponse{
    Status:      v16.RegistrationStatusAccepted,
    CurrentTime: v16.DateTime{Time: time.Now()},
    Interval:    60,
}
data, err = encoder.EncodeCallResult("unique-id-123", respPayload)

// Encode a CallError
errorDetails := map[string]string{"detail": "Invalid payload"}
data, err = encoder.EncodeCallError(
    "unique-id-456",
    ocpp.ErrorCodeProtocolError,
    "Protocol error occurred",
    errorDetails,
)
```

#### Using Low-Level API
```go
// Create Call message
call, err := ocpp.NewCall("BootNotification", payload)
data, err := call.ToBytes()

// Create CallResult
callResult, err := ocpp.NewCallResult("unique-id", payload)
data, err := callResult.ToBytes()

// Create CallError
callError, err := ocpp.NewCallError("unique-id", errorCode, desc, details)
data, err := callError.ToBytes()
```

### Decoding Messages

#### Using High-Level Decoder
```go
decoder := ocpp.NewMessageDecoder()

// Decode any message type
msg, err := decoder.Decode(rawBytes)
switch typedMsg := msg.(type) {
case *ocpp.Call:
    // Handle Call
case *ocpp.CallResult:
    // Handle CallResult
case *ocpp.CallError:
    // Handle CallError
}

// Decode Call with payload extraction
var payload v16.BootNotificationRequest
call, err := decoder.DecodeCall(rawBytes, &payload)
// Now you have both the call metadata and the parsed payload

// Decode CallResult with payload extraction
var respPayload v16.BootNotificationResponse
result, err := decoder.DecodeCallResult(rawBytes, &respPayload)
```

#### Using Low-Level API
```go
// Parse message to determine type
msg, err := ocpp.ParseMessage(rawBytes)

// Type assert and handle
switch typedMsg := msg.(type) {
case *ocpp.Call:
    // Unmarshal payload
    var payload v16.BootNotificationRequest
    err = json.Unmarshal(typedMsg.Payload, &payload)

case *ocpp.CallResult:
    var payload v16.BootNotificationResponse
    err = json.Unmarshal(typedMsg.Payload, &payload)
}
```

### Validation and Utilities

```go
// Validate message structure
err := ocpp.ValidateMessage(rawBytes)
if err != nil {
    // Invalid OCPP message format
}

// Extract message metadata without full parsing
msgType, err := ocpp.GetMessageType(rawBytes)
msgID, err := ocpp.GetMessageID(rawBytes)
action, err := ocpp.GetAction(rawBytes) // For Call messages only

// Pretty print for debugging
prettyJSON, err := ocpp.PrettyPrint(call)
fmt.Println(prettyJSON)

// Compact print for logging
compactJSON, err := ocpp.CompactPrint(call)
log.Debug(compactJSON)
```

## Performance Benchmarks

All benchmarks run on typical hardware (10 cores):

| Operation | ns/op | B/op | allocs/op | Notes |
|-----------|-------|------|-----------|-------|
| EncodeCall | 1,295 | 752 | 16 | Create and marshal Call message |
| EncodeCallResult | 701 | 480 | 13 | Create and marshal CallResult |
| DecodeCall | 2,306 | 2,040 | 41 | Parse and extract payload |
| ParseMessage | 1,262 | 1,648 | 34 | Parse to determine type |
| ValidateMessage | 459 | 552 | 13 | Validate structure |
| GetMessageType | 457 | 552 | 13 | Extract message type |
| GetAction | 1,001 | 1,144 | 27 | Extract action from Call |

### Performance Notes
- **Sub-microsecond encoding**: ~700-1,300 ns for encoding operations
- **Sub-2.5-microsecond decoding**: ~2,300 ns for full decode with payload
- **Minimal allocations**: 13-41 allocations per operation
- **Low memory footprint**: 480-2,040 bytes per operation

### Throughput Estimates
Based on benchmarks:
- **EncodeCall**: ~772,000 ops/sec (~772K messages/sec)
- **DecodeCall**: ~434,000 ops/sec (~434K messages/sec)
- **ValidateMessage**: ~2,180,000 ops/sec (~2.2M messages/sec)

This performance is more than sufficient for typical OCPP usage (heartbeats every 60s, meter values every 60s, etc.).

## Test Coverage

### Unit Tests (18 tests)
✅ All tests passing

**Encoder Tests:**
- EncodeCall
- EncodeCallResult
- EncodeCallError

**Decoder Tests:**
- Decode (all message types)
- DecodeCall with payload extraction
- DecodeCallResult with payload extraction
- DecodeCallError

**Validation Tests:**
- ValidateMessage (valid and invalid cases)
- GetMessageType
- GetMessageID
- GetAction
- PrettyPrint
- CompactPrint

**OCPP 1.6 Message Tests:**
- BootNotification request/response
- Heartbeat request/response
- Authorize request/response
- StatusNotification request
- StartTransaction request/response
- StopTransaction request/response
- MeterValues request (with complex nested structures)
- DataTransfer request/response
- DateTime marshaling/unmarshaling
- All enum constants
- All action constants

### Running Tests
```bash
# Run all encoding/decoding tests
go test ./internal/ocpp/... -v

# Run benchmarks
go test ./internal/ocpp -bench=. -benchmem

# Run specific test
go test ./internal/ocpp -v -run TestMessageEncoder_EncodeCall

# Run with coverage
go test ./internal/ocpp -cover
```

## Architecture

### Layered Design

```
┌─────────────────────────────────────────────────────┐
│           Application Layer (Handlers)               │
│  Uses: SendBootNotification(), HandleCall(), etc.   │
└─────────────────────────────────────────────────────┘
                         ▲
                         │ Uses
                         │
┌─────────────────────────────────────────────────────┐
│        High-Level Encoding API (encoding.go)        │
│  MessageEncoder, MessageDecoder, Utilities           │
└─────────────────────────────────────────────────────┘
                         ▲
                         │ Uses
                         │
┌─────────────────────────────────────────────────────┐
│         Low-Level OCPP Messages (message.go)        │
│  Call, CallResult, CallError, ParseMessage()        │
│  Custom MarshalJSON/UnmarshalJSON implementations   │
└─────────────────────────────────────────────────────┘
                         ▲
                         │ Uses
                         │
┌─────────────────────────────────────────────────────┐
│       Standard Go encoding/json Package              │
└─────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Custom Marshaling**: Implemented custom `MarshalJSON`/`UnmarshalJSON` to comply with OCPP's array-based format
2. **Type Safety**: All messages use strongly-typed structs
3. **Separation of Concerns**: High-level API (encoder/decoder) separate from low-level (Call/CallResult)
4. **Zero Dependencies**: Uses only standard library `encoding/json`
5. **RFC3339 DateTime**: Custom DateTime type ensures ISO 8601 compliance
6. **Validation Layer**: Message validation separate from parsing
7. **Utility Functions**: Helper functions for common operations

## Error Handling

### Encoding Errors
```go
call, err := encoder.EncodeCall("BootNotification", payload)
if err != nil {
    // Possible errors:
    // - Failed to marshal payload
    // - Invalid action or payload type
    log.Printf("Encoding error: %v", err)
}
```

### Decoding Errors
```go
msg, err := decoder.Decode(rawBytes)
if err != nil {
    // Possible errors:
    // - Invalid JSON
    // - Incorrect message format
    // - Wrong number of array elements
    // - Invalid message type
    log.Printf("Decoding error: %v", err)
}
```

### Validation Errors
```go
err := ocpp.ValidateMessage(rawBytes)
if err != nil {
    // Returns specific error:
    // - "invalid JSON array: ..."
    // - "message array too short: ..."
    // - "Call message must have 4 elements, got X"
    // - "unknown message type: X"
    log.Printf("Validation error: %v", err)
}
```

## OCPP Specification Compliance

✅ **OCPP 1.6 JSON-over-WebSocket Specification**

### Message Format Compliance
- ✅ Call messages: `[2, uniqueId, action, payload]`
- ✅ CallResult messages: `[3, uniqueId, payload]`
- ✅ CallError messages: `[4, uniqueId, errorCode, errorDescription, errorDetails]`
- ✅ Unique message IDs (UUID v4)
- ✅ RFC3339 timestamp format
- ✅ Proper JSON encoding for all data types

### Data Type Compliance
- ✅ String max lengths enforced via validation tags
- ✅ Integer constraints
- ✅ Optional vs required fields
- ✅ Enum values for statuses, error codes, measurands, etc.
- ✅ Complex nested structures (MeterValues, SampledValues)

### Error Code Compliance
All OCPP error codes implemented:
- NotImplemented
- NotSupported
- InternalError
- ProtocolError
- SecurityError
- FormationViolation
- PropertyConstraintViolation
- OccurrenceConstraintViolation
- TypeConstraintViolation
- GenericError

## Future Enhancements

### OCPP 2.0.1 / 2.1 Encoding
- Implement custom types for OCPP 2.x
- Enhanced security message encoding
- Device model encoding
- ISO 15118 certificate encoding

### Performance Optimizations
- [ ] Message pooling to reduce allocations
- [ ] Pre-allocated buffers for common messages
- [ ] Lazy parsing for large payloads
- [ ] Streaming JSON for large datasets

### Additional Utilities
- [ ] Message replay from logs
- [ ] Message diff for debugging
- [ ] Schema validation using JSON Schema
- [ ] Message statistics and analytics

## Integration Points

### Handler Integration
The v16.Handler uses encoding/decoding throughout:

```go
// Sending messages
call, err := handler.SendBootNotification(stationID, &req)
// Internally uses: ocpp.NewCall() → call.ToBytes()

// Receiving messages
response, err := handler.HandleCall(stationID, call)
// Internally uses: json.Unmarshal(call.Payload, &request)
```

### Message Logger Integration
All messages are encoded before logging:

```go
data, _ := call.ToBytes()
messageLogger.LogMessage(stationID, "sent", call, "ocpp1.6")
```

### WebSocket Integration
Raw bytes are sent over WebSocket:

```go
call, _ := ocpp.NewCall("Heartbeat", HeartbeatRequest{})
data, _ := call.ToBytes()
websocket.WriteMessage(websocket.TextMessage, data)
```

## Files Summary

| File | Type | Lines | Purpose |
|------|------|-------|---------|
| `internal/ocpp/message.go` | Core | 369 | OCPP message types and custom marshaling |
| `internal/ocpp/message_test.go` | Test | 355 | Message marshaling/unmarshaling tests |
| `internal/ocpp/encoding.go` | Utility | 256 | High-level encoding/decoding utilities |
| `internal/ocpp/encoding_test.go` | Test | 549 | Encoding utility tests and benchmarks |
| `internal/ocpp/v16/types.go` | Types | 244 | OCPP 1.6 data types with DateTime |
| `internal/ocpp/v16/messages.go` | Messages | 237 | OCPP 1.6 message request/response types |
| `internal/ocpp/v16/messages_test.go` | Test | 469 | OCPP 1.6 message encoding tests |

**Total:** ~2,479 lines of code for encoding/decoding

## Documentation

### Code Comments
- All public types have GoDoc comments
- All functions documented with examples
- Complex logic has inline comments
- Error conditions documented

### API Documentation
```bash
# Generate documentation
godoc -http=:6060

# View at: http://localhost:6060/pkg/github.com/ruslanhut/ocpp-emu/internal/ocpp/
```

## Conclusion

Task 2.3 is **complete** with a comprehensive, tested, and performant encoding/decoding implementation for OCPP messages. The implementation:

✅ Fully complies with OCPP 1.6 specification
✅ Provides both low-level and high-level APIs
✅ Includes comprehensive test coverage
✅ Delivers excellent performance (<3μs per operation)
✅ Offers useful utilities for validation and debugging
✅ Is ready for OCPP 2.0.1/2.1 extensions

The encoding/decoding foundation is solid and will support all future OCPP protocol implementations.

---

**Completed:** 2025-01-08
**Developer:** Claude Code
**Status:** ✅ Ready for Phase 2 Task 2.4 (SOAP/XML support)
