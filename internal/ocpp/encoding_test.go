package ocpp

import (
	"encoding/json"
	"testing"
)

func TestMessageEncoder_EncodeCall(t *testing.T) {
	encoder := NewMessageEncoder()

	payload := map[string]string{
		"chargePointVendor": "TestVendor",
		"chargePointModel":  "TestModel",
	}

	data, err := encoder.EncodeCall("BootNotification", payload)
	if err != nil {
		t.Fatalf("Failed to encode Call: %v", err)
	}

	// Verify it's valid JSON
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(arr) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(arr))
	}

	if int(arr[0].(float64)) != int(MessageTypeCall) {
		t.Errorf("Expected message type %d, got %v", MessageTypeCall, arr[0])
	}

	if arr[2] != "BootNotification" {
		t.Errorf("Expected action 'BootNotification', got %v", arr[2])
	}
}

func TestMessageEncoder_EncodeCallResult(t *testing.T) {
	encoder := NewMessageEncoder()

	payload := map[string]interface{}{
		"status":   "Accepted",
		"interval": 60,
	}

	data, err := encoder.EncodeCallResult("test-123", payload)
	if err != nil {
		t.Fatalf("Failed to encode CallResult: %v", err)
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	if arr[1] != "test-123" {
		t.Errorf("Expected unique ID 'test-123', got %v", arr[1])
	}
}

func TestMessageEncoder_EncodeCallError(t *testing.T) {
	encoder := NewMessageEncoder()

	details := map[string]string{
		"detail": "Test error",
	}

	data, err := encoder.EncodeCallError("test-456", ErrorCodeInternalError, "An error occurred", details)
	if err != nil {
		t.Fatalf("Failed to encode CallError: %v", err)
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}

	if arr[2] != string(ErrorCodeInternalError) {
		t.Errorf("Expected error code %s, got %v", ErrorCodeInternalError, arr[2])
	}
}

func TestMessageDecoder_Decode(t *testing.T) {
	decoder := NewMessageDecoder()

	tests := []struct {
		name     string
		data     string
		wantType interface{}
	}{
		{
			name:     "Call message",
			data:     `[2,"test-1","Heartbeat",{}]`,
			wantType: &Call{},
		},
		{
			name:     "CallResult message",
			data:     `[3,"test-2",{"status":"Accepted"}]`,
			wantType: &CallResult{},
		},
		{
			name:     "CallError message",
			data:     `[4,"test-3","NotSupported","Not supported",{}]`,
			wantType: &CallError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := decoder.Decode([]byte(tt.data))
			if err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}

			switch tt.wantType.(type) {
			case *Call:
				if _, ok := msg.(*Call); !ok {
					t.Errorf("Expected *Call, got %T", msg)
				}
			case *CallResult:
				if _, ok := msg.(*CallResult); !ok {
					t.Errorf("Expected *CallResult, got %T", msg)
				}
			case *CallError:
				if _, ok := msg.(*CallError); !ok {
					t.Errorf("Expected *CallError, got %T", msg)
				}
			}
		})
	}
}

func TestMessageDecoder_DecodeCall(t *testing.T) {
	decoder := NewMessageDecoder()
	data := []byte(`[2,"test-1","BootNotification",{"chargePointVendor":"Vendor","chargePointModel":"Model"}]`)

	type BootNotificationPayload struct {
		ChargePointVendor string `json:"chargePointVendor"`
		ChargePointModel  string `json:"chargePointModel"`
	}

	var payload BootNotificationPayload
	call, err := decoder.DecodeCall(data, &payload)
	if err != nil {
		t.Fatalf("Failed to decode Call: %v", err)
	}

	if call.Action != "BootNotification" {
		t.Errorf("Expected action 'BootNotification', got %s", call.Action)
	}

	if payload.ChargePointVendor != "Vendor" {
		t.Errorf("Expected vendor 'Vendor', got %s", payload.ChargePointVendor)
	}

	if payload.ChargePointModel != "Model" {
		t.Errorf("Expected model 'Model', got %s", payload.ChargePointModel)
	}
}

func TestMessageDecoder_DecodeCallResult(t *testing.T) {
	decoder := NewMessageDecoder()
	data := []byte(`[3,"test-2",{"status":"Accepted","interval":60}]`)

	type BootNotificationResponse struct {
		Status   string `json:"status"`
		Interval int    `json:"interval"`
	}

	var payload BootNotificationResponse
	result, err := decoder.DecodeCallResult(data, &payload)
	if err != nil {
		t.Fatalf("Failed to decode CallResult: %v", err)
	}

	if result.UniqueID != "test-2" {
		t.Errorf("Expected unique ID 'test-2', got %s", result.UniqueID)
	}

	if payload.Status != "Accepted" {
		t.Errorf("Expected status 'Accepted', got %s", payload.Status)
	}

	if payload.Interval != 60 {
		t.Errorf("Expected interval 60, got %d", payload.Interval)
	}
}

func TestMessageDecoder_DecodeCallError(t *testing.T) {
	decoder := NewMessageDecoder()
	data := []byte(`[4,"test-3","InternalError","An error occurred",{"detail":"System failure"}]`)

	callError, err := decoder.DecodeCallError(data)
	if err != nil {
		t.Fatalf("Failed to decode CallError: %v", err)
	}

	if callError.UniqueID != "test-3" {
		t.Errorf("Expected unique ID 'test-3', got %s", callError.UniqueID)
	}

	if callError.ErrorCode != ErrorCodeInternalError {
		t.Errorf("Expected error code %s, got %s", ErrorCodeInternalError, callError.ErrorCode)
	}

	if callError.ErrorDesc != "An error occurred" {
		t.Errorf("Expected error desc 'An error occurred', got %s", callError.ErrorDesc)
	}
}

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Valid Call",
			data:    `[2,"test-1","Heartbeat",{}]`,
			wantErr: false,
		},
		{
			name:    "Valid CallResult",
			data:    `[3,"test-2",{}]`,
			wantErr: false,
		},
		{
			name:    "Valid CallError",
			data:    `[4,"test-3","NotSupported","Not supported",{}]`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			data:    `invalid json`,
			wantErr: true,
		},
		{
			name:    "Too few elements",
			data:    `[2,"test"]`,
			wantErr: true,
		},
		{
			name:    "Invalid message type",
			data:    `[99,"test","Action",{}]`,
			wantErr: true,
		},
		{
			name:    "Call with wrong element count",
			data:    `[2,"test","Action"]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetMessageType(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantType MessageType
		wantErr  bool
	}{
		{
			name:     "Call message",
			data:     `[2,"test","Action",{}]`,
			wantType: MessageTypeCall,
			wantErr:  false,
		},
		{
			name:     "CallResult message",
			data:     `[3,"test",{}]`,
			wantType: MessageTypeCallResult,
			wantErr:  false,
		},
		{
			name:     "CallError message",
			data:     `[4,"test","Error","Desc",{}]`,
			wantType: MessageTypeCallError,
			wantErr:  false,
		},
		{
			name:    "Invalid JSON",
			data:    `invalid`,
			wantErr: true,
		},
		{
			name:    "Empty array",
			data:    `[]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgType, err := GetMessageType([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessageType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && msgType != tt.wantType {
				t.Errorf("GetMessageType() = %v, want %v", msgType, tt.wantType)
			}
		})
	}
}

func TestGetMessageID(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantID  string
		wantErr bool
	}{
		{
			name:    "Call message",
			data:    `[2,"unique-123","Action",{}]`,
			wantID:  "unique-123",
			wantErr: false,
		},
		{
			name:    "CallResult message",
			data:    `[3,"unique-456",{}]`,
			wantID:  "unique-456",
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			data:    `invalid`,
			wantErr: true,
		},
		{
			name:    "Too short array",
			data:    `[2]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GetMessageID([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id != tt.wantID {
				t.Errorf("GetMessageID() = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestGetAction(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantAction string
		wantErr    bool
	}{
		{
			name:       "Call message",
			data:       `[2,"test-1","BootNotification",{}]`,
			wantAction: "BootNotification",
			wantErr:    false,
		},
		{
			name:       "Call message with different action",
			data:       `[2,"test-2","Heartbeat",{}]`,
			wantAction: "Heartbeat",
			wantErr:    false,
		},
		{
			name:    "CallResult message",
			data:    `[3,"test-3",{}]`,
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			data:    `invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := GetAction([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && action != tt.wantAction {
				t.Errorf("GetAction() = %v, want %v", action, tt.wantAction)
			}
		})
	}
}

func TestPrettyPrint(t *testing.T) {
	call := Call{
		MessageTypeID: MessageTypeCall,
		UniqueID:      "test-123",
		Action:        "Heartbeat",
		Payload:       json.RawMessage(`{}`),
	}

	result, err := PrettyPrint(&call)
	if err != nil {
		t.Fatalf("PrettyPrint failed: %v", err)
	}

	if result == "" {
		t.Error("PrettyPrint returned empty string")
	}

	// Should be valid JSON
	var parsed []interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("PrettyPrint result is not valid JSON: %v", err)
	}
}

func TestCompactPrint(t *testing.T) {
	call := Call{
		MessageTypeID: MessageTypeCall,
		UniqueID:      "test-123",
		Action:        "Heartbeat",
		Payload:       json.RawMessage(`{}`),
	}

	result, err := CompactPrint(&call)
	if err != nil {
		t.Fatalf("CompactPrint failed: %v", err)
	}

	if result == "" {
		t.Error("CompactPrint returned empty string")
	}

	// Should be valid JSON
	var parsed []interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("CompactPrint result is not valid JSON: %v", err)
	}
}

// Benchmarks

func BenchmarkEncodeCall(b *testing.B) {
	encoder := NewMessageEncoder()
	payload := map[string]string{
		"chargePointVendor": "TestVendor",
		"chargePointModel":  "TestModel",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.EncodeCall("BootNotification", payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeCallResult(b *testing.B) {
	encoder := NewMessageEncoder()
	payload := map[string]interface{}{
		"status":   "Accepted",
		"interval": 60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.EncodeCallResult("test-123", payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeCall(b *testing.B) {
	decoder := NewMessageDecoder()
	data := []byte(`[2,"test-1","BootNotification",{"chargePointVendor":"Vendor","chargePointModel":"Model"}]`)

	type BootNotificationPayload struct {
		ChargePointVendor string `json:"chargePointVendor"`
		ChargePointModel  string `json:"chargePointModel"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var payload BootNotificationPayload
		_, err := decoder.DecodeCall(data, &payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMessage(b *testing.B) {
	data := []byte(`[2,"test-1","Heartbeat",{}]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateMessage(b *testing.B) {
	data := []byte(`[2,"test-1","Heartbeat",{}]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateMessage(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetMessageType(b *testing.B) {
	data := []byte(`[2,"test-1","Heartbeat",{}]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetMessageType(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAction(b *testing.B) {
	data := []byte(`[2,"test-1","BootNotification",{}]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetAction(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
