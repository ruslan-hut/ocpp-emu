package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
)

// StationController defines the interface for controlling stations during scenario execution.
type StationController interface {
	StartStation(ctx context.Context, stationID string) error
	StopStation(ctx context.Context, stationID string) error
	StartCharging(ctx context.Context, stationID string, connectorID int, idTag string) error
	StopCharging(ctx context.Context, stationID string, connectorID int, reason string) error
	SendCustomMessage(ctx context.Context, stationID string, messageJSON []byte) error
	GetConnectors(ctx context.Context, stationID string) ([]map[string]interface{}, error)
	IsStationConnected(stationID string) bool
}

// MessageListener defines interface for subscribing to OCPP messages.
type MessageListener interface {
	AddListener(callback logging.MessageListener) string
	RemoveListener(id string)
}

// ProgressBroadcaster defines interface for broadcasting execution progress.
type ProgressBroadcaster interface {
	BroadcastScenarioProgress(progress interface{})
}

// Runner manages scenario executions.
type Runner struct {
	storage     *Storage
	controller  StationController
	msgListener MessageListener
	broadcaster ProgressBroadcaster
	logger      *slog.Logger

	executions map[string]*activeExecution
	mu         sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// activeExecution tracks a running scenario execution.
type activeExecution struct {
	execution *Execution
	scenario  *Scenario
	cancel    context.CancelFunc
	pauseCh   chan struct{}
	resumeCh  chan struct{}
	isPaused  bool
	mu        sync.RWMutex

	// Message capturing
	messageCh  chan logging.MessageEntry
	listenerID string
}

// NewRunner creates a new scenario runner.
func NewRunner(
	storage *Storage,
	controller StationController,
	msgListener MessageListener,
	broadcaster ProgressBroadcaster,
	logger *slog.Logger,
) *Runner {
	ctx, cancel := context.WithCancel(context.Background())

	return &Runner{
		storage:     storage,
		controller:  controller,
		msgListener: msgListener,
		broadcaster: broadcaster,
		logger:      logger,
		executions:  make(map[string]*activeExecution),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// StartScenario starts executing a scenario.
func (r *Runner) StartScenario(ctx context.Context, scenarioID, stationID string) (*Execution, error) {
	// Get scenario
	scenario, err := r.storage.GetScenario(ctx, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}

	// Use scenario's default station if not specified
	if stationID == "" {
		stationID = scenario.StationID
	}
	if stationID == "" {
		return nil, fmt.Errorf("station ID is required")
	}

	// Create execution
	executionID := uuid.New().String()
	execution := NewExecution(executionID, scenario, stationID)

	// Save to storage
	if err := r.storage.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Create active execution
	execCtx, execCancel := context.WithCancel(r.ctx)
	active := &activeExecution{
		execution: execution,
		scenario:  scenario,
		cancel:    execCancel,
		pauseCh:   make(chan struct{}),
		resumeCh:  make(chan struct{}),
		messageCh: make(chan logging.MessageEntry, 100),
	}

	// Register message listener
	if r.msgListener != nil {
		active.listenerID = r.msgListener.AddListener(func(entry logging.MessageEntry) {
			if entry.StationID == stationID {
				select {
				case active.messageCh <- entry:
				default:
					// Buffer full, drop message
				}
			}
		})
	}

	// Store active execution
	r.mu.Lock()
	r.executions[executionID] = active
	r.mu.Unlock()

	// Start execution in background
	go r.runExecution(execCtx, active)

	r.logger.Info("Started scenario execution",
		"execution_id", executionID,
		"scenario_id", scenarioID,
		"station_id", stationID,
	)

	return execution, nil
}

// PauseExecution pauses a running execution.
func (r *Runner) PauseExecution(executionID string) error {
	r.mu.RLock()
	active, exists := r.executions[executionID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	if active.isPaused {
		return fmt.Errorf("execution is already paused")
	}

	if active.execution.Status != ExecutionStatusRunning {
		return fmt.Errorf("can only pause running executions")
	}

	active.isPaused = true
	active.execution.Status = ExecutionStatusPaused
	close(active.pauseCh)

	// Update storage
	if err := r.storage.UpdateExecutionStatus(context.Background(), executionID, ExecutionStatusPaused, active.execution.CurrentStep); err != nil {
		r.logger.Error("Failed to update execution status", "error", err)
	}

	r.broadcastProgress(active)

	r.logger.Info("Paused execution", "execution_id", executionID)
	return nil
}

// ResumeExecution resumes a paused execution.
func (r *Runner) ResumeExecution(executionID string) error {
	r.mu.RLock()
	active, exists := r.executions[executionID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	if !active.isPaused {
		return fmt.Errorf("execution is not paused")
	}

	active.isPaused = false
	active.execution.Status = ExecutionStatusRunning
	active.pauseCh = make(chan struct{})

	// Signal resume
	select {
	case active.resumeCh <- struct{}{}:
	default:
	}

	// Update storage
	if err := r.storage.UpdateExecutionStatus(context.Background(), executionID, ExecutionStatusRunning, active.execution.CurrentStep); err != nil {
		r.logger.Error("Failed to update execution status", "error", err)
	}

	r.broadcastProgress(active)

	r.logger.Info("Resumed execution", "execution_id", executionID)
	return nil
}

// StopExecution stops/cancels a running or paused execution.
func (r *Runner) StopExecution(executionID string) error {
	r.mu.Lock()
	active, exists := r.executions[executionID]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("execution not found: %s", executionID)
	}
	r.mu.Unlock()

	// Cancel execution context
	active.cancel()

	// If paused, signal resume to unblock
	active.mu.Lock()
	if active.isPaused {
		select {
		case active.resumeCh <- struct{}{}:
		default:
		}
	}
	active.mu.Unlock()

	r.logger.Info("Stopped execution", "execution_id", executionID)
	return nil
}

// GetExecution returns the current state of an execution.
func (r *Runner) GetExecution(executionID string) (*Execution, error) {
	r.mu.RLock()
	active, exists := r.executions[executionID]
	r.mu.RUnlock()

	if exists {
		active.mu.RLock()
		exec := *active.execution
		active.mu.RUnlock()
		return &exec, nil
	}

	// Try storage
	return r.storage.GetExecution(context.Background(), executionID)
}

// runExecution executes the scenario steps.
func (r *Runner) runExecution(ctx context.Context, active *activeExecution) {
	defer func() {
		// Cleanup
		if active.listenerID != "" && r.msgListener != nil {
			r.msgListener.RemoveListener(active.listenerID)
		}

		r.mu.Lock()
		delete(r.executions, active.execution.ExecutionID)
		r.mu.Unlock()
	}()

	// Mark as running
	active.mu.Lock()
	active.execution.Status = ExecutionStatusRunning
	active.execution.StartTime = time.Now()
	active.mu.Unlock()

	if err := r.storage.UpdateExecutionStatus(ctx, active.execution.ExecutionID, ExecutionStatusRunning, 0); err != nil {
		r.logger.Error("Failed to update execution status", "error", err)
	}

	r.broadcastProgress(active)

	// Execute steps
	for i, step := range active.scenario.Steps {
		// Check for cancellation
		select {
		case <-ctx.Done():
			r.completeExecution(active, ExecutionStatusCancelled, "execution cancelled")
			return
		default:
		}

		// Check for pause
		active.mu.RLock()
		isPaused := active.isPaused
		pauseCh := active.pauseCh
		active.mu.RUnlock()

		if isPaused {
			// Wait for resume or cancel
			select {
			case <-active.resumeCh:
			case <-ctx.Done():
				r.completeExecution(active, ExecutionStatusCancelled, "execution cancelled while paused")
				return
			}
		}

		// Update current step
		active.mu.Lock()
		active.execution.CurrentStep = i
		active.execution.Results[i].Status = StepStatusRunning
		active.execution.Results[i].StartTime = time.Now()
		active.mu.Unlock()

		if err := r.storage.UpdateStepResult(ctx, active.execution.ExecutionID, i, active.execution.Results[i]); err != nil {
			r.logger.Error("Failed to update step result", "error", err)
		}

		r.broadcastProgress(active)

		// Execute step
		stepResult, err := r.executeStep(ctx, active, step, pauseCh)

		// Update step result
		now := time.Now()
		active.mu.Lock()
		active.execution.Results[i].EndTime = &now
		active.execution.Results[i].Duration = now.Sub(active.execution.Results[i].StartTime).Milliseconds()

		if err != nil {
			active.execution.Results[i].Status = StepStatusFailed
			active.execution.Results[i].Error = err.Error()
			active.mu.Unlock()

			if err := r.storage.UpdateStepResult(ctx, active.execution.ExecutionID, i, active.execution.Results[i]); err != nil {
				r.logger.Error("Failed to update step result", "error", err)
			}

			r.broadcastProgress(active)
			r.completeExecution(active, ExecutionStatusFailed, err.Error())
			return
		}

		active.execution.Results[i].Status = StepStatusSuccess
		if stepResult != nil {
			active.execution.Results[i].Output = stepResult
		}
		active.mu.Unlock()

		if err := r.storage.UpdateStepResult(ctx, active.execution.ExecutionID, i, active.execution.Results[i]); err != nil {
			r.logger.Error("Failed to update step result", "error", err)
		}

		r.broadcastProgress(active)
	}

	// All steps completed successfully
	r.completeExecution(active, ExecutionStatusCompleted, "")
}

// executeStep executes a single step.
func (r *Runner) executeStep(ctx context.Context, active *activeExecution, step Step, pauseCh <-chan struct{}) (interface{}, error) {
	// Create step context with timeout
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Millisecond)
		defer cancel()
	}

	switch step.Type {
	case StepTypeAPICall:
		return r.executeAPICall(stepCtx, active, step)
	case StepTypeWaitForMessage:
		return r.executeWaitForMessage(stepCtx, active, step)
	case StepTypeWaitForState:
		return r.executeWaitForState(stepCtx, active, step)
	case StepTypeDelay:
		return r.executeDelay(stepCtx, step)
	case StepTypeWaitCondition:
		return r.executeWaitCondition(stepCtx, active, step)
	case StepTypeSendMessage:
		return r.executeSendMessage(stepCtx, active, step)
	case StepTypeAssert:
		return r.executeAssert(stepCtx, active, step)
	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// executeAPICall executes an API call step.
func (r *Runner) executeAPICall(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	actionStr, _ := step.Params["action"].(string)
	stationID := active.execution.StationID
	if sid, ok := step.Params["stationId"].(string); ok && sid != "" {
		stationID = sid
	}

	switch APIAction(actionStr) {
	case APIActionStartStation:
		return nil, r.controller.StartStation(ctx, stationID)

	case APIActionStopStation:
		return nil, r.controller.StopStation(ctx, stationID)

	case APIActionStartCharging:
		connectorID := 1
		if cid, ok := step.Params["connectorId"].(float64); ok {
			connectorID = int(cid)
		}
		idTag := "DEFAULT_TAG"
		if tag, ok := step.Params["idTag"].(string); ok {
			idTag = tag
		}
		return nil, r.controller.StartCharging(ctx, stationID, connectorID, idTag)

	case APIActionStopCharging:
		connectorID := 1
		if cid, ok := step.Params["connectorId"].(float64); ok {
			connectorID = int(cid)
		}
		reason := "Local"
		if r, ok := step.Params["reason"].(string); ok {
			reason = r
		}
		return nil, r.controller.StopCharging(ctx, stationID, connectorID, reason)

	default:
		return nil, fmt.Errorf("unknown API action: %s", actionStr)
	}
}

// executeWaitForMessage waits for a specific OCPP message.
func (r *Runner) executeWaitForMessage(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	direction, _ := step.Params["direction"].(string)
	action, _ := step.Params["action"].(string)

	r.logger.Debug("Waiting for message",
		"direction", direction,
		"action", action,
	)

	// Listen for matching message
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message: %s %s", direction, action)

		case msg := <-active.messageCh:
			// Check if message matches
			if (direction == "" || msg.Direction == direction) &&
				(action == "" || msg.Action == action) {
				// Validate message if needed
				if step.Validate != nil {
					if err := r.validateMessage(msg, step.Validate); err != nil {
						continue // Message doesn't match validation, keep waiting
					}
				}

				return &CapturedMessage{
					Direction:   msg.Direction,
					MessageType: getMessageTypeInt(msg.MessageType),
					MessageID:   msg.MessageID,
					Action:      msg.Action,
					Payload:     msg.Payload,
					Timestamp:   msg.Timestamp,
				}, nil
			}
		}
	}
}

// executeWaitForState waits for a station/connector state.
func (r *Runner) executeWaitForState(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	target, _ := step.Params["target"].(string)
	desiredState, _ := step.Params["state"].(string)
	stationID := active.execution.StationID
	if sid, ok := step.Params["stationId"].(string); ok && sid != "" {
		stationID = sid
	}

	connectorID := 0
	if cid, ok := step.Params["connectorId"].(float64); ok {
		connectorID = int(cid)
	}

	// Poll for state
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for state: %s %s", target, desiredState)

		case <-ticker.C:
			var currentState string
			var err error

			switch target {
			case "station":
				connected := r.controller.IsStationConnected(stationID)
				if connected {
					currentState = "connected"
				} else {
					currentState = "disconnected"
				}

			case "connector":
				connectors, err := r.controller.GetConnectors(ctx, stationID)
				if err != nil {
					continue
				}
				for _, c := range connectors {
					if cid, ok := c["id"].(int); ok && cid == connectorID {
						if status, ok := c["status"].(string); ok {
							currentState = status
						}
					}
				}

			default:
				return nil, fmt.Errorf("unknown target: %s", target)
			}

			if err == nil && currentState == desiredState {
				return map[string]interface{}{
					"target":       target,
					"state":        currentState,
					"station_id":   stationID,
					"connector_id": connectorID,
				}, nil
			}
		}
	}
}

// executeDelay waits for a fixed duration.
func (r *Runner) executeDelay(ctx context.Context, step Step) (interface{}, error) {
	duration := 1000 // default 1 second
	if d, ok := step.Params["duration"].(float64); ok {
		duration = int(d)
	}

	timer := time.NewTimer(time.Duration(duration) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return map[string]interface{}{
			"waited_ms": duration,
		}, nil
	}
}

// executeWaitCondition waits for a condition to be true.
func (r *Runner) executeWaitCondition(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	conditionStr, _ := step.Params["condition"].(string)
	condition := ConditionType(conditionStr)
	stationID := active.execution.StationID
	if sid, ok := step.Params["stationId"].(string); ok && sid != "" {
		stationID = sid
	}

	connectorID := 0
	if cid, ok := step.Params["connectorId"].(float64); ok {
		connectorID = int(cid)
	}

	// Poll for condition
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for condition: %s", condition)

		case <-ticker.C:
			met := false

			switch condition {
			case ConditionStationConnected:
				met = r.controller.IsStationConnected(stationID)

			case ConditionStationDisconnected:
				met = !r.controller.IsStationConnected(stationID)

			case ConditionConnectorAvailable:
				connectors, err := r.controller.GetConnectors(ctx, stationID)
				if err == nil {
					for _, c := range connectors {
						if cid, ok := c["id"].(int); ok && cid == connectorID {
							if status, ok := c["status"].(string); ok {
								met = status == "Available"
							}
						}
					}
				}

			case ConditionConnectorCharging:
				connectors, err := r.controller.GetConnectors(ctx, stationID)
				if err == nil {
					for _, c := range connectors {
						if cid, ok := c["id"].(int); ok && cid == connectorID {
							if status, ok := c["status"].(string); ok {
								met = status == "Charging"
							}
						}
					}
				}

			case ConditionTransactionActive:
				connectors, err := r.controller.GetConnectors(ctx, stationID)
				if err == nil {
					for _, c := range connectors {
						if cid, ok := c["id"].(int); ok && cid == connectorID {
							if txID, ok := c["currentTransactionId"]; ok && txID != nil {
								met = true
							}
						}
					}
				}

			default:
				return nil, fmt.Errorf("unknown condition: %s", condition)
			}

			if met {
				return map[string]interface{}{
					"condition":  condition,
					"station_id": stationID,
				}, nil
			}
		}
	}
}

// executeSendMessage sends a custom OCPP message.
func (r *Runner) executeSendMessage(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	stationID := active.execution.StationID
	if sid, ok := step.Params["stationId"].(string); ok && sid != "" {
		stationID = sid
	}

	messageType := 2 // Call
	if mt, ok := step.Params["messageType"].(float64); ok {
		messageType = int(mt)
	}

	action, _ := step.Params["action"].(string)
	payload := step.Params["payload"]

	// Build OCPP message array
	messageID := uuid.New().String()
	var message interface{}

	switch messageType {
	case 2: // Call
		message = []interface{}{2, messageID, action, payload}
	case 3: // CallResult
		message = []interface{}{3, messageID, payload}
	case 4: // CallError
		errorCode, _ := step.Params["errorCode"].(string)
		errorDesc, _ := step.Params["errorDescription"].(string)
		message = []interface{}{4, messageID, errorCode, errorDesc, payload}
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := r.controller.SendCustomMessage(ctx, stationID, messageJSON); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return map[string]interface{}{
		"message_id": messageID,
		"action":     action,
	}, nil
}

// executeAssert validates a condition.
func (r *Runner) executeAssert(ctx context.Context, active *activeExecution, step Step) (interface{}, error) {
	// This is a simple assertion framework
	condition, _ := step.Params["condition"].(string)
	expected, _ := step.Params["expected"]
	actual, _ := step.Params["actual"]

	switch condition {
	case "equals":
		if expected != actual {
			return nil, fmt.Errorf("assertion failed: expected %v, got %v", expected, actual)
		}
	case "not_equals":
		if expected == actual {
			return nil, fmt.Errorf("assertion failed: expected not %v", expected)
		}
	case "exists":
		if actual == nil {
			return nil, fmt.Errorf("assertion failed: value does not exist")
		}
	default:
		return nil, fmt.Errorf("unknown assertion condition: %s", condition)
	}

	return map[string]interface{}{
		"assertion": "passed",
		"condition": condition,
	}, nil
}

// validateMessage validates a captured message against expected values.
func (r *Runner) validateMessage(msg logging.MessageEntry, validate map[string]interface{}) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("payload is not a map")
	}

	for key, expected := range validate {
		actual, exists := payload[key]
		if !exists {
			return fmt.Errorf("field %s not found in message", key)
		}
		if actual != expected {
			return fmt.Errorf("field %s: expected %v, got %v", key, expected, actual)
		}
	}

	return nil
}

// completeExecution marks an execution as completed or failed.
func (r *Runner) completeExecution(active *activeExecution, status ExecutionStatus, errorMsg string) {
	active.mu.Lock()
	active.execution.Status = status
	now := time.Now()
	active.execution.CompletedAt = &now
	active.execution.Error = errorMsg
	active.mu.Unlock()

	if err := r.storage.CompleteExecution(context.Background(), active.execution.ExecutionID, status, errorMsg); err != nil {
		r.logger.Error("Failed to complete execution", "error", err)
	}

	r.broadcastProgress(active)

	r.logger.Info("Execution completed",
		"execution_id", active.execution.ExecutionID,
		"status", status,
	)
}

// broadcastProgress broadcasts execution progress.
func (r *Runner) broadcastProgress(active *activeExecution) {
	if r.broadcaster == nil {
		return
	}

	active.mu.RLock()
	progress := active.execution.GetProgress()
	active.mu.RUnlock()

	r.broadcaster.BroadcastScenarioProgress(progress)
}

// Shutdown gracefully shuts down the runner.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down scenario runner")

	// Cancel all executions
	r.mu.Lock()
	for _, active := range r.executions {
		active.cancel()
	}
	r.mu.Unlock()

	// Wait for context or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Force cleanup
	}

	r.cancel()
	return nil
}

// ListActiveExecutions returns all currently running executions.
func (r *Runner) ListActiveExecutions() []*Execution {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executions := make([]*Execution, 0, len(r.executions))
	for _, active := range r.executions {
		active.mu.RLock()
		exec := *active.execution
		active.mu.RUnlock()
		executions = append(executions, &exec)
	}

	return executions
}

// Helper functions

func getMessageTypeInt(msgType string) int {
	switch msgType {
	case "Call":
		return 2
	case "CallResult":
		return 3
	case "CallError":
		return 4
	default:
		return 0
	}
}
