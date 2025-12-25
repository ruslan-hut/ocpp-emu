// Package scenario provides scenario testing framework for OCPP emulator.
package scenario

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepType defines the type of step in a scenario.
type StepType string

const (
	// StepTypeAPICall executes an API call (start station, start charging, etc.)
	StepTypeAPICall StepType = "api_call"
	// StepTypeWaitForMessage waits for a specific OCPP message
	StepTypeWaitForMessage StepType = "wait_for_message"
	// StepTypeWaitForState waits for a station/connector state
	StepTypeWaitForState StepType = "wait_for_state"
	// StepTypeDelay waits for a fixed duration
	StepTypeDelay StepType = "delay"
	// StepTypeWaitCondition waits for a condition to be true
	StepTypeWaitCondition StepType = "wait_condition"
	// StepTypeSendMessage sends a custom OCPP message
	StepTypeSendMessage StepType = "send_message"
	// StepTypeAssert validates a condition
	StepTypeAssert StepType = "assert"
)

// APIAction defines API actions that can be called in scenarios.
type APIAction string

const (
	APIActionStartStation  APIAction = "start_station"
	APIActionStopStation   APIAction = "stop_station"
	APIActionStartCharging APIAction = "start_charging"
	APIActionStopCharging  APIAction = "stop_charging"
	APIActionSendHeartbeat APIAction = "send_heartbeat"
	APIActionReset         APIAction = "reset"
)

// ConditionType defines types of conditions for wait_condition steps.
type ConditionType string

const (
	ConditionStationConnected    ConditionType = "station_connected"
	ConditionStationDisconnected ConditionType = "station_disconnected"
	ConditionConnectorAvailable  ConditionType = "connector_available"
	ConditionConnectorCharging   ConditionType = "connector_charging"
	ConditionTransactionActive   ConditionType = "transaction_active"
)

// ExecutionStatus defines the status of a scenario execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusPaused    ExecutionStatus = "paused"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// StepStatus defines the status of a step execution.
type StepStatus string

const (
	StepStatusPending StepStatus = "pending"
	StepStatusRunning StepStatus = "running"
	StepStatusSuccess StepStatus = "success"
	StepStatusFailed  StepStatus = "failed"
	StepStatusSkipped StepStatus = "skipped"
)

// Scenario represents a test scenario definition.
type Scenario struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ScenarioID  string             `json:"scenarioId" bson:"scenario_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	StationID   string             `json:"stationId,omitempty" bson:"station_id,omitempty"`
	Steps       []Step             `json:"steps" bson:"steps"`
	Tags        []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updated_at"`
	Version     string             `json:"version,omitempty" bson:"version,omitempty"`
	IsBuiltin   bool               `json:"isBuiltin" bson:"is_builtin"`
}

// Step represents a single step in a scenario.
type Step struct {
	Type        StepType               `json:"type" bson:"type"`
	Description string                 `json:"description,omitempty" bson:"description,omitempty"`
	Timeout     int                    `json:"timeout,omitempty" bson:"timeout,omitempty"` // milliseconds
	Params      map[string]interface{} `json:"params,omitempty" bson:"params,omitempty"`
	Validate    map[string]interface{} `json:"validate,omitempty" bson:"validate,omitempty"`
	OnSuccess   string                 `json:"onSuccess,omitempty" bson:"on_success,omitempty"`
	OnFailure   string                 `json:"onFailure,omitempty" bson:"on_failure,omitempty"`
}

// APICallParams defines parameters for API call steps.
type APICallParams struct {
	Action      APIAction `json:"action"`
	StationID   string    `json:"stationId,omitempty"`
	ConnectorID int       `json:"connectorId,omitempty"`
	IDTag       string    `json:"idTag,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// WaitForMessageParams defines parameters for wait_for_message steps.
type WaitForMessageParams struct {
	Direction string `json:"direction"` // "sent" or "received"
	Action    string `json:"action"`    // OCPP action name
	StationID string `json:"stationId,omitempty"`
}

// WaitForStateParams defines parameters for wait_for_state steps.
type WaitForStateParams struct {
	Target      string `json:"target"` // "station" or "connector"
	StationID   string `json:"stationId,omitempty"`
	ConnectorID int    `json:"connectorId,omitempty"`
	State       string `json:"state"`
}

// DelayParams defines parameters for delay steps.
type DelayParams struct {
	Duration int `json:"duration"` // milliseconds
}

// WaitConditionParams defines parameters for wait_condition steps.
type WaitConditionParams struct {
	Condition   ConditionType `json:"condition"`
	StationID   string        `json:"stationId,omitempty"`
	ConnectorID int           `json:"connectorId,omitempty"`
}

// SendMessageParams defines parameters for send_message steps.
type SendMessageParams struct {
	StationID   string      `json:"stationId"`
	MessageType int         `json:"messageType"` // 2=Call, 3=CallResult, 4=CallError
	Action      string      `json:"action"`
	Payload     interface{} `json:"payload"`
}

// Execution represents a scenario execution instance.
type Execution struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ExecutionID  string             `json:"executionId" bson:"execution_id"`
	ScenarioID   string             `json:"scenarioId" bson:"scenario_id"`
	ScenarioName string             `json:"scenarioName" bson:"scenario_name"`
	StationID    string             `json:"stationId" bson:"station_id"`
	Status       ExecutionStatus    `json:"status" bson:"status"`
	CurrentStep  int                `json:"currentStep" bson:"current_step"`
	TotalSteps   int                `json:"totalSteps" bson:"total_steps"`
	Results      []StepResult       `json:"results" bson:"results"`
	StartTime    time.Time          `json:"startTime" bson:"start_time"`
	CompletedAt  *time.Time         `json:"completedAt,omitempty" bson:"completed_at,omitempty"`
	Error        string             `json:"error,omitempty" bson:"error,omitempty"`
	CreatedAt    time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updated_at"`
}

// StepResult represents the result of executing a single step.
type StepResult struct {
	StepIndex   int              `json:"stepIndex" bson:"step_index"`
	StepType    StepType         `json:"stepType" bson:"step_type"`
	Description string           `json:"description" bson:"description"`
	Status      StepStatus       `json:"status" bson:"status"`
	StartTime   time.Time        `json:"startTime" bson:"start_time"`
	EndTime     *time.Time       `json:"endTime,omitempty" bson:"end_time,omitempty"`
	Duration    int64            `json:"duration,omitempty" bson:"duration,omitempty"` // milliseconds
	Error       string           `json:"error,omitempty" bson:"error,omitempty"`
	Output      interface{}      `json:"output,omitempty" bson:"output,omitempty"`
	MessageData *CapturedMessage `json:"messageData,omitempty" bson:"message_data,omitempty"`
}

// CapturedMessage represents an OCPP message captured during step execution.
type CapturedMessage struct {
	Direction   string      `json:"direction" bson:"direction"`
	MessageType int         `json:"messageType" bson:"message_type"`
	MessageID   string      `json:"messageId" bson:"message_id"`
	Action      string      `json:"action" bson:"action"`
	Payload     interface{} `json:"payload" bson:"payload"`
	Timestamp   time.Time   `json:"timestamp" bson:"timestamp"`
}

// ExecutionProgress provides real-time progress information.
type ExecutionProgress struct {
	ExecutionID     string          `json:"executionId"`
	ScenarioName    string          `json:"scenarioName"`
	Status          ExecutionStatus `json:"status"`
	CurrentStep     int             `json:"currentStep"`
	TotalSteps      int             `json:"totalSteps"`
	Percentage      float64         `json:"percentage"`
	CurrentStepDesc string          `json:"currentStepDesc,omitempty"`
	ElapsedTime     int64           `json:"elapsedTime"` // milliseconds
	Error           string          `json:"error,omitempty"`
}

// NewScenario creates a new scenario with defaults.
func NewScenario(name, description string) *Scenario {
	now := time.Now()
	return &Scenario{
		Name:        name,
		Description: description,
		Steps:       []Step{},
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     "1.0",
		IsBuiltin:   false,
	}
}

// NewExecution creates a new execution for a scenario.
func NewExecution(executionID string, scenario *Scenario, stationID string) *Execution {
	now := time.Now()
	results := make([]StepResult, len(scenario.Steps))
	for i, step := range scenario.Steps {
		results[i] = StepResult{
			StepIndex:   i,
			StepType:    step.Type,
			Description: step.Description,
			Status:      StepStatusPending,
		}
	}

	return &Execution{
		ExecutionID:  executionID,
		ScenarioID:   scenario.ScenarioID,
		ScenarioName: scenario.Name,
		StationID:    stationID,
		Status:       ExecutionStatusPending,
		CurrentStep:  0,
		TotalSteps:   len(scenario.Steps),
		Results:      results,
		StartTime:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// GetProgress returns the current execution progress.
func (e *Execution) GetProgress() ExecutionProgress {
	percentage := 0.0
	if e.TotalSteps > 0 {
		completed := 0
		for _, r := range e.Results {
			if r.Status == StepStatusSuccess || r.Status == StepStatusFailed || r.Status == StepStatusSkipped {
				completed++
			}
		}
		percentage = float64(completed) / float64(e.TotalSteps) * 100
	}

	currentDesc := ""
	if e.CurrentStep < len(e.Results) {
		currentDesc = e.Results[e.CurrentStep].Description
	}

	elapsed := int64(0)
	if !e.StartTime.IsZero() {
		if e.CompletedAt != nil {
			elapsed = e.CompletedAt.Sub(e.StartTime).Milliseconds()
		} else {
			elapsed = time.Since(e.StartTime).Milliseconds()
		}
	}

	return ExecutionProgress{
		ExecutionID:     e.ExecutionID,
		ScenarioName:    e.ScenarioName,
		Status:          e.Status,
		CurrentStep:     e.CurrentStep,
		TotalSteps:      e.TotalSteps,
		Percentage:      percentage,
		CurrentStepDesc: currentDesc,
		ElapsedTime:     elapsed,
		Error:           e.Error,
	}
}
