package v16

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
)

func TestHandler_HandleCall_RemoteStartTransaction(t *testing.T) {
	handler := NewHandler(slog.Default())

	// Set up callback
	var receivedReq *RemoteStartTransactionRequest
	handler.OnRemoteStartTransaction = func(stationID string, req *RemoteStartTransactionRequest) (*RemoteStartTransactionResponse, error) {
		receivedReq = req
		return &RemoteStartTransactionResponse{Status: "Accepted"}, nil
	}

	// Create request
	req := RemoteStartTransactionRequest{
		IdTag: "TAG123",
	}
	reqBytes, _ := json.Marshal(req)

	call := &ocpp.Call{
		MessageTypeID: ocpp.MessageTypeCall,
		UniqueID:      "test-123",
		Action:        string(ActionRemoteStartTransaction),
		Payload:       reqBytes,
	}

	// Handle call
	resp, err := handler.HandleCall("CP001", call)
	if err != nil {
		t.Fatalf("HandleCall failed: %v", err)
	}

	// Verify response
	remoteStartResp, ok := resp.(*RemoteStartTransactionResponse)
	if !ok {
		t.Fatalf("Expected *RemoteStartTransactionResponse, got %T", resp)
	}

	if remoteStartResp.Status != "Accepted" {
		t.Errorf("Expected status 'Accepted', got '%s'", remoteStartResp.Status)
	}

	if receivedReq == nil {
		t.Fatal("Callback was not called")
	}

	if receivedReq.IdTag != "TAG123" {
		t.Errorf("Expected IdTag 'TAG123', got '%s'", receivedReq.IdTag)
	}
}

func TestHandler_HandleCall_RemoteStopTransaction(t *testing.T) {
	handler := NewHandler(slog.Default())

	handler.OnRemoteStopTransaction = func(stationID string, req *RemoteStopTransactionRequest) (*RemoteStopTransactionResponse, error) {
		if req.TransactionId != 42 {
			t.Errorf("Expected TransactionId 42, got %d", req.TransactionId)
		}
		return &RemoteStopTransactionResponse{Status: "Accepted"}, nil
	}

	req := RemoteStopTransactionRequest{
		TransactionId: 42,
	}
	reqBytes, _ := json.Marshal(req)

	call := &ocpp.Call{
		MessageTypeID: ocpp.MessageTypeCall,
		UniqueID:      "test-456",
		Action:        string(ActionRemoteStopTransaction),
		Payload:       reqBytes,
	}

	resp, err := handler.HandleCall("CP001", call)
	if err != nil {
		t.Fatalf("HandleCall failed: %v", err)
	}

	remoteStopResp, ok := resp.(*RemoteStopTransactionResponse)
	if !ok {
		t.Fatalf("Expected *RemoteStopTransactionResponse, got %T", resp)
	}

	if remoteStopResp.Status != "Accepted" {
		t.Errorf("Expected status 'Accepted', got '%s'", remoteStopResp.Status)
	}
}

func TestHandler_HandleCall_Reset(t *testing.T) {
	handler := NewHandler(slog.Default())

	handler.OnReset = func(stationID string, req *ResetRequest) (*ResetResponse, error) {
		if req.Type != "Soft" {
			t.Errorf("Expected reset type 'Soft', got '%s'", req.Type)
		}
		return &ResetResponse{Status: "Accepted"}, nil
	}

	req := ResetRequest{
		Type: "Soft",
	}
	reqBytes, _ := json.Marshal(req)

	call := &ocpp.Call{
		MessageTypeID: ocpp.MessageTypeCall,
		UniqueID:      "test-789",
		Action:        string(ActionReset),
		Payload:       reqBytes,
	}

	resp, err := handler.HandleCall("CP001", call)
	if err != nil {
		t.Fatalf("HandleCall failed: %v", err)
	}

	resetResp, ok := resp.(*ResetResponse)
	if !ok {
		t.Fatalf("Expected *ResetResponse, got %T", resp)
	}

	if resetResp.Status != "Accepted" {
		t.Errorf("Expected status 'Accepted', got '%s'", resetResp.Status)
	}
}

func TestHandler_HandleCall_UnknownAction(t *testing.T) {
	handler := NewHandler(slog.Default())

	call := &ocpp.Call{
		MessageTypeID: ocpp.MessageTypeCall,
		UniqueID:      "test-unknown",
		Action:        "UnknownAction",
		Payload:       json.RawMessage("{}"),
	}

	_, err := handler.HandleCall("CP001", call)
	if err == nil {
		t.Fatal("Expected error for unknown action, got nil")
	}
}

func TestHandler_SendBootNotification(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &BootNotificationRequest{
		ChargePointVendor: "Vendor",
		ChargePointModel:  "Model",
	}

	call, err := handler.SendBootNotification("CP001", req)
	if err != nil {
		t.Fatalf("SendBootNotification failed: %v", err)
	}

	if call.Action != string(ActionBootNotification) {
		t.Errorf("Expected action 'BootNotification', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}

	// Verify sent data can be parsed
	var parsedCall ocpp.Call
	if err := json.Unmarshal(sentData, &parsedCall); err != nil {
		t.Fatalf("Failed to parse sent data: %v", err)
	}

	if parsedCall.Action != string(ActionBootNotification) {
		t.Errorf("Expected action 'BootNotification', got '%s'", parsedCall.Action)
	}
}

func TestHandler_SendHeartbeat(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	call, err := handler.SendHeartbeat("CP001")
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}

	if call.Action != string(ActionHeartbeat) {
		t.Errorf("Expected action 'Heartbeat', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_SendStatusNotification(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   ChargePointErrorNoError,
		Status:      ChargePointStatusAvailable,
	}

	call, err := handler.SendStatusNotification("CP001", req)
	if err != nil {
		t.Fatalf("SendStatusNotification failed: %v", err)
	}

	if call.Action != string(ActionStatusNotification) {
		t.Errorf("Expected action 'StatusNotification', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}

	// Verify timestamp was set
	var parsedCall ocpp.Call
	json.Unmarshal(sentData, &parsedCall)
	var parsedReq StatusNotificationRequest
	json.Unmarshal(parsedCall.Payload, &parsedReq)

	if parsedReq.Timestamp == nil {
		t.Error("Expected Timestamp to be set")
	}
}

func TestHandler_SendAuthorize(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &AuthorizeRequest{
		IdTag: "TAG123",
	}

	call, err := handler.SendAuthorize("CP001", req)
	if err != nil {
		t.Fatalf("SendAuthorize failed: %v", err)
	}

	if call.Action != string(ActionAuthorize) {
		t.Errorf("Expected action 'Authorize', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_SendStartTransaction(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &StartTransactionRequest{
		ConnectorId: 1,
		IdTag:       "TAG123",
		MeterStart:  0,
		Timestamp:   DateTime{Time: time.Now()},
	}

	call, err := handler.SendStartTransaction("CP001", req)
	if err != nil {
		t.Fatalf("SendStartTransaction failed: %v", err)
	}

	if call.Action != string(ActionStartTransaction) {
		t.Errorf("Expected action 'StartTransaction', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_SendStopTransaction(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &StopTransactionRequest{
		TransactionId: 42,
		MeterStop:     1000,
		Timestamp:     DateTime{Time: time.Now()},
	}

	call, err := handler.SendStopTransaction("CP001", req)
	if err != nil {
		t.Fatalf("SendStopTransaction failed: %v", err)
	}

	if call.Action != string(ActionStopTransaction) {
		t.Errorf("Expected action 'StopTransaction', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_SendMeterValues(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &MeterValuesRequest{
		ConnectorId: 1,
		MeterValue: []MeterValue{
			{
				Timestamp: DateTime{Time: time.Now()},
				SampledValue: []SampledValue{
					{
						Value:     "1000",
						Measurand: MeasurandEnergyActiveImportRegister,
						Unit:      UnitOfMeasureWh,
					},
				},
			},
		},
	}

	call, err := handler.SendMeterValues("CP001", req)
	if err != nil {
		t.Fatalf("SendMeterValues failed: %v", err)
	}

	if call.Action != string(ActionMeterValues) {
		t.Errorf("Expected action 'MeterValues', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_SendDataTransfer(t *testing.T) {
	handler := NewHandler(slog.Default())

	var sentData []byte
	handler.SendMessage = func(stationID string, data []byte) error {
		sentData = data
		return nil
	}

	req := &DataTransferRequest{
		VendorId:  "TestVendor",
		MessageId: "TestMessage",
		Data:      "TestData",
	}

	call, err := handler.SendDataTransfer("CP001", req)
	if err != nil {
		t.Fatalf("SendDataTransfer failed: %v", err)
	}

	if call.Action != string(ActionDataTransfer) {
		t.Errorf("Expected action 'DataTransfer', got '%s'", call.Action)
	}

	if sentData == nil {
		t.Fatal("SendMessage was not called")
	}
}

func TestHandler_HandleCallResult_BootNotification(t *testing.T) {
	handler := NewHandler(slog.Default())

	resp := BootNotificationResponse{
		Status:      RegistrationStatusAccepted,
		CurrentTime: DateTime{Time: time.Now()},
		Interval:    300,
	}
	respBytes, _ := json.Marshal(resp)

	callResult := &ocpp.CallResult{
		MessageTypeID: ocpp.MessageTypeCallResult,
		UniqueID:      "test-123",
		Payload:       respBytes,
	}

	result, err := handler.HandleCallResult("CP001", callResult, ActionBootNotification)
	if err != nil {
		t.Fatalf("HandleCallResult failed: %v", err)
	}

	bootResp, ok := result.(*BootNotificationResponse)
	if !ok {
		t.Fatalf("Expected *BootNotificationResponse, got %T", result)
	}

	if bootResp.Status != RegistrationStatusAccepted {
		t.Errorf("Expected status 'Accepted', got '%s'", bootResp.Status)
	}

	if bootResp.Interval != 300 {
		t.Errorf("Expected interval 300, got %d", bootResp.Interval)
	}
}

func TestHandler_HandleCallResult_Heartbeat(t *testing.T) {
	handler := NewHandler(slog.Default())

	resp := HeartbeatResponse{
		CurrentTime: DateTime{Time: time.Now()},
	}
	respBytes, _ := json.Marshal(resp)

	callResult := &ocpp.CallResult{
		MessageTypeID: ocpp.MessageTypeCallResult,
		UniqueID:      "test-456",
		Payload:       respBytes,
	}

	result, err := handler.HandleCallResult("CP001", callResult, ActionHeartbeat)
	if err != nil {
		t.Fatalf("HandleCallResult failed: %v", err)
	}

	heartbeatResp, ok := result.(*HeartbeatResponse)
	if !ok {
		t.Fatalf("Expected *HeartbeatResponse, got %T", result)
	}

	if heartbeatResp.CurrentTime.IsZero() {
		t.Error("Expected CurrentTime to be set")
	}
}

func TestHandler_HandleCallResult_Authorize(t *testing.T) {
	handler := NewHandler(slog.Default())

	resp := AuthorizeResponse{
		IdTagInfo: IdTagInfo{
			Status: AuthorizationStatusAccepted,
		},
	}
	respBytes, _ := json.Marshal(resp)

	callResult := &ocpp.CallResult{
		MessageTypeID: ocpp.MessageTypeCallResult,
		UniqueID:      "test-789",
		Payload:       respBytes,
	}

	result, err := handler.HandleCallResult("CP001", callResult, ActionAuthorize)
	if err != nil {
		t.Fatalf("HandleCallResult failed: %v", err)
	}

	authResp, ok := result.(*AuthorizeResponse)
	if !ok {
		t.Fatalf("Expected *AuthorizeResponse, got %T", result)
	}

	if authResp.IdTagInfo.Status != AuthorizationStatusAccepted {
		t.Errorf("Expected status 'Accepted', got '%s'", authResp.IdTagInfo.Status)
	}
}

func TestHandler_HandleCallResult_StartTransaction(t *testing.T) {
	handler := NewHandler(slog.Default())

	resp := StartTransactionResponse{
		TransactionId: 42,
		IdTagInfo: IdTagInfo{
			Status: AuthorizationStatusAccepted,
		},
	}
	respBytes, _ := json.Marshal(resp)

	callResult := &ocpp.CallResult{
		MessageTypeID: ocpp.MessageTypeCallResult,
		UniqueID:      "test-abc",
		Payload:       respBytes,
	}

	result, err := handler.HandleCallResult("CP001", callResult, ActionStartTransaction)
	if err != nil {
		t.Fatalf("HandleCallResult failed: %v", err)
	}

	startTxResp, ok := result.(*StartTransactionResponse)
	if !ok {
		t.Fatalf("Expected *StartTransactionResponse, got %T", result)
	}

	if startTxResp.TransactionId != 42 {
		t.Errorf("Expected TransactionId 42, got %d", startTxResp.TransactionId)
	}

	if startTxResp.IdTagInfo.Status != AuthorizationStatusAccepted {
		t.Errorf("Expected status 'Accepted', got '%s'", startTxResp.IdTagInfo.Status)
	}
}
