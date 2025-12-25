package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage handles persistence of scenarios and executions.
type Storage struct {
	scenariosCollection  *mongo.Collection
	executionsCollection *mongo.Collection
	logger               *slog.Logger
}

// NewStorage creates a new scenario storage.
func NewStorage(db *mongo.Database, logger *slog.Logger) (*Storage, error) {
	if logger == nil {
		logger = slog.Default()
	}

	storage := &Storage{
		scenariosCollection:  db.Collection("scenarios"),
		executionsCollection: db.Collection("scenario_executions"),
		logger:               logger,
	}

	// Create indexes
	if err := storage.createIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create scenario indexes: %w", err)
	}

	logger.Info("Scenario storage initialized")
	return storage, nil
}

// createIndexes creates necessary indexes for scenarios and executions.
func (s *Storage) createIndexes(ctx context.Context) error {
	// Scenario indexes
	scenarioIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "scenario_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_builtin", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
	}

	if _, err := s.scenariosCollection.Indexes().CreateMany(ctx, scenarioIndexes); err != nil {
		return fmt.Errorf("failed to create scenario indexes: %w", err)
	}

	// Execution indexes
	executionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "execution_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "scenario_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "station_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "start_time", Value: -1}},
		},
	}

	if _, err := s.executionsCollection.Indexes().CreateMany(ctx, executionIndexes); err != nil {
		return fmt.Errorf("failed to create execution indexes: %w", err)
	}

	return nil
}

// CreateScenario creates a new scenario.
func (s *Storage) CreateScenario(ctx context.Context, scenario *Scenario) error {
	if scenario.ScenarioID == "" {
		scenario.ScenarioID = primitive.NewObjectID().Hex()
	}
	scenario.CreatedAt = time.Now()
	scenario.UpdatedAt = scenario.CreatedAt

	result, err := s.scenariosCollection.InsertOne(ctx, scenario)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("scenario with ID %s already exists", scenario.ScenarioID)
		}
		return fmt.Errorf("failed to create scenario: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		scenario.ID = oid
	}

	s.logger.Info("Created scenario",
		"scenario_id", scenario.ScenarioID,
		"name", scenario.Name,
	)
	return nil
}

// GetScenario retrieves a scenario by its scenario_id.
func (s *Storage) GetScenario(ctx context.Context, scenarioID string) (*Scenario, error) {
	var scenario Scenario
	err := s.scenariosCollection.FindOne(ctx, bson.M{"scenario_id": scenarioID}).Decode(&scenario)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("scenario not found: %s", scenarioID)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return &scenario, nil
}

// GetScenarioByObjectID retrieves a scenario by its MongoDB ObjectID.
func (s *Storage) GetScenarioByObjectID(ctx context.Context, id primitive.ObjectID) (*Scenario, error) {
	var scenario Scenario
	err := s.scenariosCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&scenario)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("scenario not found: %s", id.Hex())
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return &scenario, nil
}

// ListScenarios retrieves all scenarios with optional filtering.
func (s *Storage) ListScenarios(ctx context.Context, filter *ScenarioFilter) ([]*Scenario, error) {
	query := bson.M{}

	if filter != nil {
		if filter.Tag != "" {
			query["tags"] = filter.Tag
		}
		if filter.BuiltinOnly {
			query["is_builtin"] = true
		}
		if filter.CustomOnly {
			query["is_builtin"] = false
		}
		if filter.StationID != "" {
			query["station_id"] = filter.StationID
		}
	}

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cursor, err := s.scenariosCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	defer cursor.Close(ctx)

	var scenarios []*Scenario
	if err := cursor.All(ctx, &scenarios); err != nil {
		return nil, fmt.Errorf("failed to decode scenarios: %w", err)
	}

	return scenarios, nil
}

// UpdateScenario updates an existing scenario.
func (s *Storage) UpdateScenario(ctx context.Context, scenario *Scenario) error {
	scenario.UpdatedAt = time.Now()

	result, err := s.scenariosCollection.UpdateOne(
		ctx,
		bson.M{"scenario_id": scenario.ScenarioID},
		bson.M{"$set": scenario},
	)
	if err != nil {
		return fmt.Errorf("failed to update scenario: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("scenario not found: %s", scenario.ScenarioID)
	}

	s.logger.Info("Updated scenario",
		"scenario_id", scenario.ScenarioID,
		"name", scenario.Name,
	)
	return nil
}

// DeleteScenario deletes a scenario by its scenario_id.
func (s *Storage) DeleteScenario(ctx context.Context, scenarioID string) error {
	result, err := s.scenariosCollection.DeleteOne(ctx, bson.M{"scenario_id": scenarioID})
	if err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("scenario not found: %s", scenarioID)
	}

	s.logger.Info("Deleted scenario", "scenario_id", scenarioID)
	return nil
}

// CreateExecution creates a new execution record.
func (s *Storage) CreateExecution(ctx context.Context, execution *Execution) error {
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = execution.CreatedAt

	result, err := s.executionsCollection.InsertOne(ctx, execution)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		execution.ID = oid
	}

	s.logger.Info("Created execution",
		"execution_id", execution.ExecutionID,
		"scenario_id", execution.ScenarioID,
		"station_id", execution.StationID,
	)
	return nil
}

// GetExecution retrieves an execution by its execution_id.
func (s *Storage) GetExecution(ctx context.Context, executionID string) (*Execution, error) {
	var execution Execution
	err := s.executionsCollection.FindOne(ctx, bson.M{"execution_id": executionID}).Decode(&execution)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}
	return &execution, nil
}

// UpdateExecution updates an existing execution record.
func (s *Storage) UpdateExecution(ctx context.Context, execution *Execution) error {
	execution.UpdatedAt = time.Now()

	result, err := s.executionsCollection.UpdateOne(
		ctx,
		bson.M{"execution_id": execution.ExecutionID},
		bson.M{"$set": execution},
	)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("execution not found: %s", execution.ExecutionID)
	}

	return nil
}

// UpdateExecutionStatus updates just the status and current step of an execution.
func (s *Storage) UpdateExecutionStatus(ctx context.Context, executionID string, status ExecutionStatus, currentStep int) error {
	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"current_step": currentStep,
			"updated_at":   time.Now(),
		},
	}

	result, err := s.executionsCollection.UpdateOne(
		ctx,
		bson.M{"execution_id": executionID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	return nil
}

// UpdateStepResult updates the result of a specific step in an execution.
func (s *Storage) UpdateStepResult(ctx context.Context, executionID string, stepIndex int, result StepResult) error {
	update := bson.M{
		"$set": bson.M{
			fmt.Sprintf("results.%d", stepIndex): result,
			"updated_at":                         time.Now(),
		},
	}

	res, err := s.executionsCollection.UpdateOne(
		ctx,
		bson.M{"execution_id": executionID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update step result: %w", err)
	}

	if res.MatchedCount == 0 {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	return nil
}

// CompleteExecution marks an execution as completed or failed.
func (s *Storage) CompleteExecution(ctx context.Context, executionID string, status ExecutionStatus, errorMsg string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"completed_at": now,
			"updated_at":   now,
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error"] = errorMsg
	}

	result, err := s.executionsCollection.UpdateOne(
		ctx,
		bson.M{"execution_id": executionID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to complete execution: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	s.logger.Info("Completed execution",
		"execution_id", executionID,
		"status", status,
	)
	return nil
}

// ListExecutions retrieves executions with optional filtering.
func (s *Storage) ListExecutions(ctx context.Context, filter *ExecutionFilter) ([]*Execution, error) {
	query := bson.M{}

	if filter != nil {
		if filter.ScenarioID != "" {
			query["scenario_id"] = filter.ScenarioID
		}
		if filter.StationID != "" {
			query["station_id"] = filter.StationID
		}
		if filter.Status != "" {
			query["status"] = filter.Status
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetLimit(100)

	cursor, err := s.executionsCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer cursor.Close(ctx)

	var executions []*Execution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, fmt.Errorf("failed to decode executions: %w", err)
	}

	return executions, nil
}

// GetActiveExecutions returns all currently running or paused executions.
func (s *Storage) GetActiveExecutions(ctx context.Context) ([]*Execution, error) {
	query := bson.M{
		"status": bson.M{
			"$in": []ExecutionStatus{ExecutionStatusRunning, ExecutionStatusPaused},
		},
	}

	cursor, err := s.executionsCollection.Find(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active executions: %w", err)
	}
	defer cursor.Close(ctx)

	var executions []*Execution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, fmt.Errorf("failed to decode executions: %w", err)
	}

	return executions, nil
}

// DeleteExecution deletes an execution by its execution_id.
func (s *Storage) DeleteExecution(ctx context.Context, executionID string) error {
	result, err := s.executionsCollection.DeleteOne(ctx, bson.M{"execution_id": executionID})
	if err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	s.logger.Info("Deleted execution", "execution_id", executionID)
	return nil
}

// DeleteOldExecutions deletes executions older than the specified duration.
func (s *Storage) DeleteOldExecutions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.executionsCollection.DeleteMany(ctx, bson.M{
		"start_time": bson.M{"$lt": cutoff},
		"status": bson.M{
			"$nin": []ExecutionStatus{ExecutionStatusRunning, ExecutionStatusPaused},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete old executions: %w", err)
	}

	if result.DeletedCount > 0 {
		s.logger.Info("Deleted old executions",
			"count", result.DeletedCount,
			"older_than", olderThan.String(),
		)
	}

	return result.DeletedCount, nil
}

// ScenarioFilter defines filtering options for listing scenarios.
type ScenarioFilter struct {
	Tag         string
	BuiltinOnly bool
	CustomOnly  bool
	StationID   string
}

// ExecutionFilter defines filtering options for listing executions.
type ExecutionFilter struct {
	ScenarioID string
	StationID  string
	Status     ExecutionStatus
}

// CountScenarios returns the total count of scenarios.
func (s *Storage) CountScenarios(ctx context.Context) (int64, error) {
	count, err := s.scenariosCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count scenarios: %w", err)
	}
	return count, nil
}

// CountExecutions returns the total count of executions.
func (s *Storage) CountExecutions(ctx context.Context) (int64, error) {
	count, err := s.executionsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}
	return count, nil
}
