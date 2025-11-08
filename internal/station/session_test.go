package station

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp/v16"
)

func TestNewSessionManager(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000},
		{ID: 2, Type: "Type2", MaxPower: 22000},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	if sm.stationID != "CP001" {
		t.Errorf("Expected station ID 'CP001', got %s", sm.stationID)
	}

	if len(sm.connectors) != 2 {
		t.Errorf("Expected 2 connectors, got %d", len(sm.connectors))
	}

	connector1, err := sm.GetConnector(1)
	if err != nil {
		t.Fatalf("Failed to get connector 1: %v", err)
	}

	if connector1.Type != "Type2" {
		t.Errorf("Expected connector type 'Type2', got %s", connector1.Type)
	}
}

func TestSessionManager_GetConnector(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Valid connector
	connector, err := sm.GetConnector(1)
	if err != nil {
		t.Fatalf("Failed to get connector 1: %v", err)
	}

	if connector.ID != 1 {
		t.Errorf("Expected connector ID 1, got %d", connector.ID)
	}

	// Invalid connector
	_, err = sm.GetConnector(99)
	if err == nil {
		t.Error("Expected error for non-existent connector")
	}
}

func TestSessionManager_Authorize(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Test offline authorization (no callback set)
	idTagInfo, err := sm.Authorize("TAG123")
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if idTagInfo.Status != v16.AuthorizationStatusAccepted {
		t.Errorf("Expected status Accepted, got %s", idTagInfo.Status)
	}

	// Test with callback
	sm.SendAuthorize = func(idTag string) (*v16.AuthorizeResponse, error) {
		return &v16.AuthorizeResponse{
			IdTagInfo: v16.IdTagInfo{
				Status: v16.AuthorizationStatusAccepted,
			},
		}, nil
	}

	idTagInfo, err = sm.Authorize("TAG456")
	if err != nil {
		t.Fatalf("Authorize with callback failed: %v", err)
	}

	if idTagInfo.Status != v16.AuthorizationStatusAccepted {
		t.Errorf("Expected status Accepted, got %s", idTagInfo.Status)
	}
}

func TestSessionManager_StartCharging(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Set up callbacks
	var startTxCalled bool
	var statusNotifCalled int

	sm.SendStartTransaction = func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error) {
		startTxCalled = true
		return &v16.StartTransactionResponse{
			IdTagInfo: v16.IdTagInfo{
				Status: v16.AuthorizationStatusAccepted,
			},
			TransactionId: 12345,
		}, nil
	}

	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		statusNotifCalled++
		return nil
	}

	// Start charging
	transactionID, err := sm.StartCharging(1, "TAG123")
	if err != nil {
		t.Fatalf("StartCharging failed: %v", err)
	}

	if transactionID != 12345 {
		t.Errorf("Expected transaction ID 12345, got %d", transactionID)
	}

	if !startTxCalled {
		t.Error("Expected SendStartTransaction to be called")
	}

	if statusNotifCalled != 2 {
		t.Errorf("Expected 2 status notifications (Preparing, Charging), got %d", statusNotifCalled)
	}

	// Verify connector state
	connector, _ := sm.GetConnector(1)
	if connector.GetState() != ConnectorStateCharging {
		t.Errorf("Expected connector state Charging, got %s", connector.GetState())
	}

	if !connector.HasActiveTransaction() {
		t.Error("Expected connector to have active transaction")
	}
}

func TestSessionManager_StopCharging(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Set up callbacks
	sm.SendStartTransaction = func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error) {
		return &v16.StartTransactionResponse{
			IdTagInfo: v16.IdTagInfo{
				Status: v16.AuthorizationStatusAccepted,
			},
			TransactionId: 12345,
		}, nil
	}

	var stopTxCalled bool
	sm.SendStopTransaction = func(transactionID int, idTag string, meterStop int, timestamp time.Time, reason v16.Reason) (*v16.StopTransactionResponse, error) {
		stopTxCalled = true
		if transactionID != 12345 {
			t.Errorf("Expected transaction ID 12345, got %d", transactionID)
		}
		if reason != v16.ReasonLocal {
			t.Errorf("Expected reason Local, got %s", reason)
		}
		return &v16.StopTransactionResponse{}, nil
	}

	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		return nil
	}

	// Start charging
	_, err := sm.StartCharging(1, "TAG123")
	if err != nil {
		t.Fatalf("StartCharging failed: %v", err)
	}

	// Stop charging
	err = sm.StopCharging(1, v16.ReasonLocal)
	if err != nil {
		t.Fatalf("StopCharging failed: %v", err)
	}

	if !stopTxCalled {
		t.Error("Expected SendStopTransaction to be called")
	}

	// Verify connector state
	connector, _ := sm.GetConnector(1)
	if connector.GetState() != ConnectorStateAvailable {
		t.Errorf("Expected connector state Available, got %s", connector.GetState())
	}

	if connector.HasActiveTransaction() {
		t.Error("Expected connector to have no active transaction")
	}
}

func TestSessionManager_ChangeAvailability(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
		{ID: 2, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	var statusNotifications []v16.ChargePointStatus
	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		statusNotifications = append(statusNotifications, status)
		return nil
	}

	// Change single connector to Inoperative
	err := sm.ChangeAvailability(1, "Inoperative")
	if err != nil {
		t.Fatalf("ChangeAvailability failed: %v", err)
	}

	connector1, _ := sm.GetConnector(1)
	if connector1.GetState() != ConnectorStateUnavailable {
		t.Errorf("Expected connector 1 state Unavailable, got %s", connector1.GetState())
	}

	// Verify status notification was sent
	if len(statusNotifications) < 1 {
		t.Error("Expected at least 1 status notification")
	} else if statusNotifications[len(statusNotifications)-1] != v16.ChargePointStatusUnavailable {
		t.Errorf("Expected last status Unavailable, got %s", statusNotifications[len(statusNotifications)-1])
	}

	// Change single connector back to Operative
	err = sm.ChangeAvailability(1, "Operative")
	if err != nil {
		t.Fatalf("ChangeAvailability failed: %v", err)
	}

	connector1, _ = sm.GetConnector(1)
	if connector1.GetState() != ConnectorStateAvailable {
		t.Errorf("Expected connector 1 state Available, got %s", connector1.GetState())
	}

	// Verify connector 2 is still available
	connector2, _ := sm.GetConnector(2)
	if connector2.GetState() != ConnectorStateAvailable {
		t.Errorf("Expected connector 2 state Available, got %s", connector2.GetState())
	}
}

func TestSessionManager_ChangeAvailability_WhileCharging(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Set up callbacks
	sm.SendStartTransaction = func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error) {
		return &v16.StartTransactionResponse{
			IdTagInfo: v16.IdTagInfo{
				Status: v16.AuthorizationStatusAccepted,
			},
			TransactionId: 12345,
		}, nil
	}

	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		return nil
	}

	// Start charging
	_, err := sm.StartCharging(1, "TAG123")
	if err != nil {
		t.Fatalf("StartCharging failed: %v", err)
	}

	// Try to change availability while charging
	err = sm.ChangeAvailability(1, "Inoperative")
	if err == nil {
		t.Error("Expected error when changing availability while charging")
	}
}

func TestSessionManager_Shutdown(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Set up callbacks
	sm.SendStartTransaction = func(connectorID int, idTag string, meterStart int, timestamp time.Time) (*v16.StartTransactionResponse, error) {
		return &v16.StartTransactionResponse{
			IdTagInfo: v16.IdTagInfo{
				Status: v16.AuthorizationStatusAccepted,
			},
			TransactionId: 12345,
		}, nil
	}

	var stopTxReason v16.Reason
	sm.SendStopTransaction = func(transactionID int, idTag string, meterStop int, timestamp time.Time, reason v16.Reason) (*v16.StopTransactionResponse, error) {
		stopTxReason = reason
		return &v16.StopTransactionResponse{}, nil
	}

	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		return nil
	}

	// Start charging
	_, err := sm.StartCharging(1, "TAG123")
	if err != nil {
		t.Fatalf("StartCharging failed: %v", err)
	}

	// Shutdown
	ctx := context.Background()
	err = sm.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify transaction was stopped with Reboot reason
	if stopTxReason != v16.ReasonReboot {
		t.Errorf("Expected stop reason Reboot, got %s", stopTxReason)
	}

	// Verify no active transactions
	connector, _ := sm.GetConnector(1)
	if connector.HasActiveTransaction() {
		t.Error("Expected no active transaction after shutdown")
	}
}

func TestSessionManager_GetAllConnectors(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000},
		{ID: 2, Type: "CCS", MaxPower: 50000},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	connectors := sm.GetAllConnectors()

	if len(connectors) != 2 {
		t.Errorf("Expected 2 connectors, got %d", len(connectors))
	}
}

func TestSessionManager_StartCharging_ConnectorNotFound(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	_, err := sm.StartCharging(99, "TAG123")
	if err == nil {
		t.Error("Expected error for non-existent connector")
	}
}

func TestSessionManager_StopCharging_NoActiveTransaction(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	err := sm.StopCharging(1, v16.ReasonLocal)
	if err == nil {
		t.Error("Expected error when stopping charging with no active transaction")
	}
}

func TestSessionManager_TransactionIDIncrement(t *testing.T) {
	connectorConfigs := []ConnectorConfig{
		{ID: 1, Type: "Type2", MaxPower: 22000, Status: "Available"},
		{ID: 2, Type: "Type2", MaxPower: 22000, Status: "Available"},
	}

	sm := NewSessionManager("CP001", connectorConfigs, slog.Default())

	// Override to not use CSMS transaction ID
	sm.SendStartTransaction = nil
	sm.SendStatusNotification = func(connectorID int, status v16.ChargePointStatus, errorCode v16.ChargePointErrorCode, info string) error {
		return nil
	}

	// Start charging on connector 1
	txID1, err := sm.StartCharging(1, "TAG123")
	if err != nil {
		t.Fatalf("StartCharging on connector 1 failed: %v", err)
	}

	// Stop charging on connector 1
	sm.StopCharging(1, v16.ReasonLocal)

	// Start charging on connector 2
	txID2, err := sm.StartCharging(2, "TAG456")
	if err != nil {
		t.Fatalf("StartCharging on connector 2 failed: %v", err)
	}

	if txID2 <= txID1 {
		t.Errorf("Expected transaction ID to increment, got %d then %d", txID1, txID2)
	}
}
