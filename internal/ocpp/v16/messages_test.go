package v16

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBootNotificationRequest tests BootNotification request marshaling
func TestBootNotificationRequest(t *testing.T) {
	req := BootNotificationRequest{
		ChargePointVendor:       "VendorX",
		ChargePointModel:        "ModelY",
		ChargePointSerialNumber: "SN123456",
		FirmwareVersion:         "1.0.0",
		Iccid:                   "89310410106543789301",
		Imsi:                    "310410123456789",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal BootNotificationRequest: %v", err)
	}

	// Unmarshal and verify
	var parsed BootNotificationRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal BootNotificationRequest: %v", err)
	}

	if parsed.ChargePointVendor != req.ChargePointVendor {
		t.Errorf("ChargePointVendor mismatch: expected %s, got %s", req.ChargePointVendor, parsed.ChargePointVendor)
	}

	if parsed.FirmwareVersion != req.FirmwareVersion {
		t.Errorf("FirmwareVersion mismatch: expected %s, got %s", req.FirmwareVersion, parsed.FirmwareVersion)
	}
}

// TestBootNotificationResponse tests BootNotification response marshaling
func TestBootNotificationResponse(t *testing.T) {
	now := time.Now()
	resp := BootNotificationResponse{
		Status:      RegistrationStatusAccepted,
		CurrentTime: DateTime{Time: now},
		Interval:    60,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal BootNotificationResponse: %v", err)
	}

	// Unmarshal and verify
	var parsed BootNotificationResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal BootNotificationResponse: %v", err)
	}

	if parsed.Status != resp.Status {
		t.Errorf("Status mismatch: expected %s, got %s", resp.Status, parsed.Status)
	}

	if parsed.Interval != resp.Interval {
		t.Errorf("Interval mismatch: expected %d, got %d", resp.Interval, parsed.Interval)
	}
}

// TestHeartbeatMessages tests Heartbeat request and response
func TestHeartbeatMessages(t *testing.T) {
	// Request (empty)
	req := HeartbeatRequest{}
	reqData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal HeartbeatRequest: %v", err)
	}

	// Should be empty object
	if string(reqData) != "{}" {
		t.Errorf("Expected empty object, got %s", string(reqData))
	}

	// Response
	resp := HeartbeatResponse{
		CurrentTime: DateTime{Time: time.Now()},
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal HeartbeatResponse: %v", err)
	}

	var parsed HeartbeatResponse
	if err := json.Unmarshal(respData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal HeartbeatResponse: %v", err)
	}
}

// TestAuthorizeMessages tests Authorize request and response
func TestAuthorizeMessages(t *testing.T) {
	// Request
	req := AuthorizeRequest{
		IdTag: "TAG123456",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal AuthorizeRequest: %v", err)
	}

	var parsed AuthorizeRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AuthorizeRequest: %v", err)
	}

	if parsed.IdTag != req.IdTag {
		t.Errorf("IdTag mismatch: expected %s, got %s", req.IdTag, parsed.IdTag)
	}

	// Response
	expiry := DateTime{Time: time.Now().Add(24 * time.Hour)}
	resp := AuthorizeResponse{
		IdTagInfo: IdTagInfo{
			Status:      AuthorizationStatusAccepted,
			ExpiryDate:  &expiry,
			ParentIdTag: "PARENT123",
		},
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal AuthorizeResponse: %v", err)
	}

	var parsedResp AuthorizeResponse
	if err := json.Unmarshal(respData, &parsedResp); err != nil {
		t.Fatalf("Failed to unmarshal AuthorizeResponse: %v", err)
	}

	if parsedResp.IdTagInfo.Status != resp.IdTagInfo.Status {
		t.Errorf("Status mismatch: expected %s, got %s", resp.IdTagInfo.Status, parsedResp.IdTagInfo.Status)
	}
}

// TestStatusNotificationRequest tests StatusNotification request
func TestStatusNotificationRequest(t *testing.T) {
	now := DateTime{Time: time.Now()}
	req := StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   ChargePointErrorNoError,
		Status:      ChargePointStatusAvailable,
		Timestamp:   &now,
		Info:        "Ready to charge",
		VendorId:    "VendorX",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal StatusNotificationRequest: %v", err)
	}

	var parsed StatusNotificationRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal StatusNotificationRequest: %v", err)
	}

	if parsed.ConnectorId != req.ConnectorId {
		t.Errorf("ConnectorId mismatch: expected %d, got %d", req.ConnectorId, parsed.ConnectorId)
	}

	if parsed.Status != req.Status {
		t.Errorf("Status mismatch: expected %s, got %s", req.Status, parsed.Status)
	}

	if parsed.ErrorCode != req.ErrorCode {
		t.Errorf("ErrorCode mismatch: expected %s, got %s", req.ErrorCode, parsed.ErrorCode)
	}
}

// TestStartTransactionMessages tests StartTransaction request and response
func TestStartTransactionMessages(t *testing.T) {
	req := StartTransactionRequest{
		ConnectorId: 1,
		IdTag:       "TAG123",
		MeterStart:  0,
		Timestamp:   DateTime{Time: time.Now()},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal StartTransactionRequest: %v", err)
	}

	var parsed StartTransactionRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal StartTransactionRequest: %v", err)
	}

	if parsed.IdTag != req.IdTag {
		t.Errorf("IdTag mismatch: expected %s, got %s", req.IdTag, parsed.IdTag)
	}

	// Response
	resp := StartTransactionResponse{
		IdTagInfo: IdTagInfo{
			Status: AuthorizationStatusAccepted,
		},
		TransactionId: 12345,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal StartTransactionResponse: %v", err)
	}

	var parsedResp StartTransactionResponse
	if err := json.Unmarshal(respData, &parsedResp); err != nil {
		t.Fatalf("Failed to unmarshal StartTransactionResponse: %v", err)
	}

	if parsedResp.TransactionId != resp.TransactionId {
		t.Errorf("TransactionId mismatch: expected %d, got %d", resp.TransactionId, parsedResp.TransactionId)
	}
}

// TestStopTransactionMessages tests StopTransaction request and response
func TestStopTransactionMessages(t *testing.T) {
	req := StopTransactionRequest{
		IdTag:         "TAG123",
		MeterStop:     15000,
		Timestamp:     DateTime{Time: time.Now()},
		TransactionId: 12345,
		Reason:        ReasonLocal,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal StopTransactionRequest: %v", err)
	}

	var parsed StopTransactionRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal StopTransactionRequest: %v", err)
	}

	if parsed.TransactionId != req.TransactionId {
		t.Errorf("TransactionId mismatch: expected %d, got %d", req.TransactionId, parsed.TransactionId)
	}

	if parsed.MeterStop != req.MeterStop {
		t.Errorf("MeterStop mismatch: expected %d, got %d", req.MeterStop, parsed.MeterStop)
	}
}

// TestMeterValuesRequest tests MeterValues request with sampled values
func TestMeterValuesRequest(t *testing.T) {
	transactionId := 12345
	req := MeterValuesRequest{
		ConnectorId:   1,
		TransactionId: &transactionId,
		MeterValue: []MeterValue{
			{
				Timestamp: DateTime{Time: time.Now()},
				SampledValue: []SampledValue{
					{
						Value:     "7200",
						Context:   ReadingContextSamplePeriodic,
						Measurand: MeasurandPowerActiveImport,
						Unit:      UnitOfMeasureW,
						Location:  LocationOutlet,
					},
					{
						Value:     "2500",
						Context:   ReadingContextSamplePeriodic,
						Measurand: MeasurandEnergyActiveImportRegister,
						Unit:      UnitOfMeasureWh,
					},
				},
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal MeterValuesRequest: %v", err)
	}

	var parsed MeterValuesRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal MeterValuesRequest: %v", err)
	}

	if parsed.ConnectorId != req.ConnectorId {
		t.Errorf("ConnectorId mismatch: expected %d, got %d", req.ConnectorId, parsed.ConnectorId)
	}

	if len(parsed.MeterValue) != len(req.MeterValue) {
		t.Errorf("MeterValue count mismatch: expected %d, got %d", len(req.MeterValue), len(parsed.MeterValue))
	}

	if len(parsed.MeterValue) > 0 && len(parsed.MeterValue[0].SampledValue) != 2 {
		t.Errorf("SampledValue count mismatch: expected 2, got %d", len(parsed.MeterValue[0].SampledValue))
	}

	if len(parsed.MeterValue[0].SampledValue) > 0 {
		sv := parsed.MeterValue[0].SampledValue[0]
		if sv.Value != "7200" {
			t.Errorf("SampledValue.Value mismatch: expected 7200, got %s", sv.Value)
		}
		if sv.Measurand != MeasurandPowerActiveImport {
			t.Errorf("SampledValue.Measurand mismatch: expected %s, got %s", MeasurandPowerActiveImport, sv.Measurand)
		}
	}
}

// TestDataTransferMessages tests DataTransfer request and response
func TestDataTransferMessages(t *testing.T) {
	req := DataTransferRequest{
		VendorId:  "VendorX",
		MessageId: "CustomMessage",
		Data:      `{"key":"value"}`,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal DataTransferRequest: %v", err)
	}

	var parsed DataTransferRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal DataTransferRequest: %v", err)
	}

	if parsed.VendorId != req.VendorId {
		t.Errorf("VendorId mismatch: expected %s, got %s", req.VendorId, parsed.VendorId)
	}

	// Response
	resp := DataTransferResponse{
		Status: "Accepted",
		Data:   `{"response":"ok"}`,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal DataTransferResponse: %v", err)
	}

	var parsedResp DataTransferResponse
	if err := json.Unmarshal(respData, &parsedResp); err != nil {
		t.Fatalf("Failed to unmarshal DataTransferResponse: %v", err)
	}

	if parsedResp.Status != resp.Status {
		t.Errorf("Status mismatch: expected %s, got %s", resp.Status, parsedResp.Status)
	}
}

// TestDateTimeMarshalUnmarshal tests custom DateTime type
func TestDateTimeMarshalUnmarshal(t *testing.T) {
	now := time.Date(2025, 11, 8, 12, 30, 45, 0, time.UTC)
	dt := DateTime{Time: now}

	// Marshal
	data, err := json.Marshal(dt)
	if err != nil {
		t.Fatalf("Failed to marshal DateTime: %v", err)
	}

	// Should be RFC3339 format in quotes
	expected := `"2025-11-08T12:30:45Z"`
	if string(data) != expected {
		t.Errorf("DateTime format mismatch: expected %s, got %s", expected, string(data))
	}

	// Unmarshal
	var parsed DateTime
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal DateTime: %v", err)
	}

	if !parsed.Time.Equal(now) {
		t.Errorf("DateTime mismatch: expected %v, got %v", now, parsed.Time)
	}
}

// TestChargePointStatuses tests all charge point status constants
func TestChargePointStatuses(t *testing.T) {
	statuses := []ChargePointStatus{
		ChargePointStatusAvailable,
		ChargePointStatusPreparing,
		ChargePointStatusCharging,
		ChargePointStatusSuspendedEVSE,
		ChargePointStatusSuspendedEV,
		ChargePointStatusFinishing,
		ChargePointStatusReserved,
		ChargePointStatusUnavailable,
		ChargePointStatusFaulted,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("ChargePointStatus should not be empty: %v", status)
		}
	}
}

// TestAuthorizationStatuses tests all authorization status constants
func TestAuthorizationStatuses(t *testing.T) {
	statuses := []AuthorizationStatus{
		AuthorizationStatusAccepted,
		AuthorizationStatusBlocked,
		AuthorizationStatusExpired,
		AuthorizationStatusInvalid,
		AuthorizationStatusConcurrentTx,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("AuthorizationStatus should not be empty: %v", status)
		}
	}
}

// TestMeasurands tests measurand constants
func TestMeasurands(t *testing.T) {
	measurands := []Measurand{
		MeasurandEnergyActiveImportRegister,
		MeasurandPowerActiveImport,
		MeasurandVoltage,
		MeasurandCurrentImport,
		MeasurandSoC,
		MeasurandTemperature,
	}

	for _, measurand := range measurands {
		if string(measurand) == "" {
			t.Errorf("Measurand should not be empty: %v", measurand)
		}
	}
}

// TestActionConstants tests that all action constants are defined
func TestActionConstants(t *testing.T) {
	actions := []Action{
		ActionAuthorize,
		ActionBootNotification,
		ActionChangeAvailability,
		ActionChangeConfiguration,
		ActionClearCache,
		ActionDataTransfer,
		ActionGetConfiguration,
		ActionHeartbeat,
		ActionMeterValues,
		ActionRemoteStartTransaction,
		ActionRemoteStopTransaction,
		ActionReset,
		ActionStartTransaction,
		ActionStatusNotification,
		ActionStopTransaction,
		ActionUnlockConnector,
	}

	for _, action := range actions {
		if string(action) == "" {
			t.Errorf("Action should not be empty: %v", action)
		}
	}
}
