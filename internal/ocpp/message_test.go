package ocpp

import (
	"encoding/json"
	"testing"
)

// TestCallMessageMarshal tests marshaling of Call messages
func TestCallMessageMarshal(t *testing.T) {
	payload := map[string]interface{}{
		"chargePointVendor": "Acme",
		"chargePointModel":  "Model X",
	}

	call, err := NewCall("BootNotification", payload)
	if err != nil {
		t.Fatalf("Failed to create Call: %v", err)
	}

	data, err := call.ToBytes()
	if err != nil {
		t.Fatalf("Failed to marshal Call: %v", err)
	}

	// Verify format: [2, "uniqueId", "BootNotification", {...}]
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Failed to unmarshal Call: %v", err)
	}

	if len(arr) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(arr))
	}

	if int(arr[0].(float64)) != int(MessageTypeCall) {
		t.Errorf("Expected message type %d, got %v", MessageTypeCall, arr[0])
	}

	if arr[2] != "BootNotification" {
		t.Errorf("Expected action BootNotification, got %v", arr[2])
	}
}

// TestCallMessageUnmarshal tests unmarshaling of Call messages
func TestCallMessageUnmarshal(t *testing.T) {
	data := []byte(`[2,"test-123","Heartbeat",{}]`)

	var call Call
	if err := json.Unmarshal(data, &call); err != nil {
		t.Fatalf("Failed to unmarshal Call: %v", err)
	}

	if call.MessageTypeID != MessageTypeCall {
		t.Errorf("Expected message type %d, got %d", MessageTypeCall, call.MessageTypeID)
	}

	if call.UniqueID != "test-123" {
		t.Errorf("Expected unique ID 'test-123', got %s", call.UniqueID)
	}

	if call.Action != "Heartbeat" {
		t.Errorf("Expected action 'Heartbeat', got %s", call.Action)
	}
}

// TestCallResultMessageMarshal tests marshaling of CallResult messages
func TestCallResultMessageMarshal(t *testing.T) {
	payload := map[string]interface{}{
		"status":      "Accepted",
		"currentTime": "2025-01-01T10:00:00Z",
		"interval":    60,
	}

	callResult, err := NewCallResult("test-456", payload)
	if err != nil {
		t.Fatalf("Failed to create CallResult: %v", err)
	}

	data, err := callResult.ToBytes()
	if err != nil {
		t.Fatalf("Failed to marshal CallResult: %v", err)
	}

	// Verify format: [3, "uniqueId", {...}]
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Failed to unmarshal CallResult: %v", err)
	}

	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	if int(arr[0].(float64)) != int(MessageTypeCallResult) {
		t.Errorf("Expected message type %d, got %v", MessageTypeCallResult, arr[0])
	}

	if arr[1] != "test-456" {
		t.Errorf("Expected unique ID 'test-456', got %v", arr[1])
	}
}

// TestCallResultMessageUnmarshal tests unmarshaling of CallResult messages
func TestCallResultMessageUnmarshal(t *testing.T) {
	data := []byte(`[3,"test-789",{"status":"Accepted","interval":60}]`)

	var callResult CallResult
	if err := json.Unmarshal(data, &callResult); err != nil {
		t.Fatalf("Failed to unmarshal CallResult: %v", err)
	}

	if callResult.MessageTypeID != MessageTypeCallResult {
		t.Errorf("Expected message type %d, got %d", MessageTypeCallResult, callResult.MessageTypeID)
	}

	if callResult.UniqueID != "test-789" {
		t.Errorf("Expected unique ID 'test-789', got %s", callResult.UniqueID)
	}

	// Verify payload can be parsed
	var payload map[string]interface{}
	if err := json.Unmarshal(callResult.Payload, &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload["status"] != "Accepted" {
		t.Errorf("Expected status 'Accepted', got %v", payload["status"])
	}
}

// TestCallErrorMessageMarshal tests marshaling of CallError messages
func TestCallErrorMessageMarshal(t *testing.T) {
	errorDetails := map[string]string{
		"detail": "Invalid payload format",
	}

	callError, err := NewCallError("test-error", ErrorCodeProtocolError, "Protocol error occurred", errorDetails)
	if err != nil {
		t.Fatalf("Failed to create CallError: %v", err)
	}

	data, err := callError.ToBytes()
	if err != nil {
		t.Fatalf("Failed to marshal CallError: %v", err)
	}

	// Verify format: [4, "uniqueId", "ErrorCode", "ErrorDescription", {...}]
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Failed to unmarshal CallError: %v", err)
	}

	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}

	if int(arr[0].(float64)) != int(MessageTypeCallError) {
		t.Errorf("Expected message type %d, got %v", MessageTypeCallError, arr[0])
	}

	if arr[2] != string(ErrorCodeProtocolError) {
		t.Errorf("Expected error code %s, got %v", ErrorCodeProtocolError, arr[2])
	}
}

// TestCallErrorMessageUnmarshal tests unmarshaling of CallError messages
func TestCallErrorMessageUnmarshal(t *testing.T) {
	data := []byte(`[4,"err-123","InternalError","An internal error occurred",{"detail":"System failure"}]`)

	var callError CallError
	if err := json.Unmarshal(data, &callError); err != nil {
		t.Fatalf("Failed to unmarshal CallError: %v", err)
	}

	if callError.MessageTypeID != MessageTypeCallError {
		t.Errorf("Expected message type %d, got %d", MessageTypeCallError, callError.MessageTypeID)
	}

	if callError.UniqueID != "err-123" {
		t.Errorf("Expected unique ID 'err-123', got %s", callError.UniqueID)
	}

	if callError.ErrorCode != ErrorCodeInternalError {
		t.Errorf("Expected error code %s, got %s", ErrorCodeInternalError, callError.ErrorCode)
	}

	if callError.ErrorDesc != "An internal error occurred" {
		t.Errorf("Expected error description 'An internal error occurred', got %s", callError.ErrorDesc)
	}
}

// TestParseMessage tests parsing of different message types
func TestParseMessage(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		expectedType interface{}
	}{
		{
			name:         "Call message",
			data:         `[2,"test-1","BootNotification",{"chargePointVendor":"Acme"}]`,
			expectedType: &Call{},
		},
		{
			name:         "CallResult message",
			data:         `[3,"test-2",{"status":"Accepted"}]`,
			expectedType: &CallResult{},
		},
		{
			name:         "CallError message",
			data:         `[4,"test-3","NotSupported","Action not supported",{}]`,
			expectedType: &CallError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.data))
			if err != nil {
				t.Fatalf("Failed to parse message: %v", err)
			}

			switch tt.expectedType.(type) {
			case *Call:
				if _, ok := msg.(*Call); !ok {
					t.Errorf("Expected Call message, got %T", msg)
				}
			case *CallResult:
				if _, ok := msg.(*CallResult); !ok {
					t.Errorf("Expected CallResult message, got %T", msg)
				}
			case *CallError:
				if _, ok := msg.(*CallError); !ok {
					t.Errorf("Expected CallError message, got %T", msg)
				}
			}
		})
	}
}

// TestParseMessageInvalid tests parsing of invalid messages
func TestParseMessageInvalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"Invalid JSON", `invalid json`},
		{"Too few elements", `[2,"test"]`},
		{"Invalid message type", `[99,"test","Action",{}]`},
		{"Not an array", `{"test": "value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMessage([]byte(tt.data))
			if err == nil {
				t.Error("Expected error for invalid message, got nil")
			}
		})
	}
}

// TestGenerateMessageID tests message ID generation
func TestGenerateMessageID(t *testing.T) {
	id1 := GenerateMessageID()
	id2 := GenerateMessageID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if !ValidateMessageID(id1) {
		t.Error("Generated ID should be valid UUID")
	}
}

// TestValidateMessageID tests message ID validation
func TestValidateMessageID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true}, // Valid UUID
		{"invalid-id", false},
		{"", false},
		{"12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := ValidateMessageID(tt.id)
			if result != tt.valid {
				t.Errorf("Expected validation result %v for ID %s, got %v", tt.valid, tt.id, result)
			}
		})
	}
}

// TestRoundTripCall tests marshaling and unmarshaling a Call message
func TestRoundTripCall(t *testing.T) {
	original := Call{
		MessageTypeID: MessageTypeCall,
		UniqueID:      "round-trip-1",
		Action:        "StatusNotification",
		Payload:       json.RawMessage(`{"connectorId":1,"status":"Available"}`),
	}

	// Marshal
	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var parsed Call
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare
	if parsed.UniqueID != original.UniqueID {
		t.Errorf("UniqueID mismatch: expected %s, got %s", original.UniqueID, parsed.UniqueID)
	}

	if parsed.Action != original.Action {
		t.Errorf("Action mismatch: expected %s, got %s", original.Action, parsed.Action)
	}
}

// TestErrorCodes tests that all error codes are defined
func TestErrorCodes(t *testing.T) {
	errorCodes := []ErrorCode{
		ErrorCodeNotImplemented,
		ErrorCodeNotSupported,
		ErrorCodeInternalError,
		ErrorCodeProtocolError,
		ErrorCodeSecurityError,
		ErrorCodeFormationViolation,
		ErrorCodePropertyConstraintViolation,
		ErrorCodeOccurrenceConstraintViolation,
		ErrorCodeTypeConstraintViolation,
		ErrorCodeGenericError,
	}

	for _, code := range errorCodes {
		if string(code) == "" {
			t.Errorf("Error code should not be empty: %v", code)
		}
	}
}
