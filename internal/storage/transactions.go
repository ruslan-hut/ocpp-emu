package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TransactionRepository handles transaction persistence operations
type TransactionRepository struct {
	collection *mongo.Collection
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *MongoDBClient) *TransactionRepository {
	return &TransactionRepository{
		collection: db.TransactionsCollection,
	}
}

// Create creates a new transaction in the database
func (r *TransactionRepository) Create(ctx context.Context, transaction Transaction) error {
	transaction.CreatedAt = time.Now()
	transaction.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, transaction)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// Update updates an existing transaction
func (r *TransactionRepository) Update(ctx context.Context, transactionID int, stationID string, updates bson.M) error {
	updates["updated_at"] = time.Now()

	filter := bson.M{
		"transaction_id": transactionID,
		"station_id":     stationID,
	}

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": updates})
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction not found: %d", transactionID)
	}

	return nil
}

// UpdateTransactionID updates the transaction ID (when CSMS assigns a different ID)
func (r *TransactionRepository) UpdateTransactionID(ctx context.Context, stationID string, oldTransactionID, newTransactionID int) error {
	filter := bson.M{
		"transaction_id": oldTransactionID,
		"station_id":     stationID,
	}

	updates := bson.M{
		"transaction_id": newTransactionID,
		"updated_at":     time.Now(),
	}

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": updates})
	if err != nil {
		return fmt.Errorf("failed to update transaction ID: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction not found: %d", oldTransactionID)
	}

	return nil
}

// Complete marks a transaction as completed
func (r *TransactionRepository) Complete(ctx context.Context, transactionID int, stationID string, meterStop int, reason string) error {
	updates := bson.M{
		"stop_timestamp": time.Now(),
		"meter_stop":     meterStop,
		"status":         "completed",
		"reason":         reason,
	}

	// Calculate energy consumed
	// First get the transaction to get meter_start
	transaction, err := r.GetByID(ctx, transactionID, stationID)
	if err != nil {
		return fmt.Errorf("failed to get transaction for completion: %w", err)
	}

	energyConsumed := meterStop - transaction.MeterStart
	updates["energy_consumed"] = energyConsumed

	return r.Update(ctx, transactionID, stationID, updates)
}

// MarkAsFailed marks a transaction as failed
func (r *TransactionRepository) MarkAsFailed(ctx context.Context, transactionID int, stationID string, reason string) error {
	updates := bson.M{
		"status": "failed",
		"reason": reason,
	}

	return r.Update(ctx, transactionID, stationID, updates)
}

// GetByID retrieves a transaction by its ID
func (r *TransactionRepository) GetByID(ctx context.Context, transactionID int, stationID string) (*Transaction, error) {
	filter := bson.M{
		"transaction_id": transactionID,
		"station_id":     stationID,
	}

	var transaction Transaction
	err := r.collection.FindOne(ctx, filter).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("transaction not found: %d", transactionID)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// GetActive retrieves all active transactions for a station
func (r *TransactionRepository) GetActive(ctx context.Context, stationID string) ([]Transaction, error) {
	filter := bson.M{
		"station_id": stationID,
		"status":     "active",
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query active transactions: %w", err)
	}
	defer cursor.Close(ctx)

	transactions := make([]Transaction, 0)
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}

// GetActiveByConnector retrieves the active transaction for a specific connector
func (r *TransactionRepository) GetActiveByConnector(ctx context.Context, stationID string, connectorID int) (*Transaction, error) {
	filter := bson.M{
		"station_id":   stationID,
		"connector_id": connectorID,
		"status":       "active",
	}

	var transaction Transaction
	err := r.collection.FindOne(ctx, filter).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active transaction
		}
		return nil, fmt.Errorf("failed to get active transaction: %w", err)
	}

	return &transaction, nil
}

// GetByStation retrieves all transactions for a station with pagination
func (r *TransactionRepository) GetByStation(ctx context.Context, stationID string, limit, skip int) ([]Transaction, error) {
	filter := bson.M{"station_id": stationID}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "start_timestamp", Value: -1}}) // Most recent first

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer cursor.Close(ctx)

	transactions := make([]Transaction, 0)
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}

// GetByIDTag retrieves all transactions for a specific ID tag
func (r *TransactionRepository) GetByIDTag(ctx context.Context, idTag string, limit, skip int) ([]Transaction, error) {
	filter := bson.M{"id_tag": idTag}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "start_timestamp", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer cursor.Close(ctx)

	transactions := make([]Transaction, 0)
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}

// GetByDateRange retrieves transactions within a date range
func (r *TransactionRepository) GetByDateRange(ctx context.Context, stationID string, startDate, endDate time.Time, limit, skip int) ([]Transaction, error) {
	filter := bson.M{
		"station_id": stationID,
		"start_timestamp": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "start_timestamp", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer cursor.Close(ctx)

	transactions := make([]Transaction, 0)
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}

// Count counts transactions matching the filter
func (r *TransactionRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// GetStats retrieves transaction statistics for a station
func (r *TransactionRepository) GetStats(ctx context.Context, stationID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total transactions
	totalCount, err := r.Count(ctx, bson.M{"station_id": stationID})
	if err != nil {
		return nil, err
	}
	stats["total"] = totalCount

	// Active transactions
	activeCount, err := r.Count(ctx, bson.M{"station_id": stationID, "status": "active"})
	if err != nil {
		return nil, err
	}
	stats["active"] = activeCount

	// Completed transactions
	completedCount, err := r.Count(ctx, bson.M{"station_id": stationID, "status": "completed"})
	if err != nil {
		return nil, err
	}
	stats["completed"] = completedCount

	// Failed transactions
	failedCount, err := r.Count(ctx, bson.M{"station_id": stationID, "status": "failed"})
	if err != nil {
		return nil, err
	}
	stats["failed"] = failedCount

	// Total energy consumed (using aggregation)
	pipeline := []bson.M{
		{"$match": bson.M{"station_id": stationID, "status": "completed"}},
		{"$group": bson.M{
			"_id":                nil,
			"total_energy":       bson.M{"$sum": "$energy_consumed"},
			"average_energy":     bson.M{"$avg": "$energy_consumed"},
			"total_transactions": bson.M{"$sum": 1},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate transaction stats: %w", err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
		stats["total_energy_wh"] = result["total_energy"]
		stats["average_energy_wh"] = result["average_energy"]
	} else {
		stats["total_energy_wh"] = 0
		stats["average_energy_wh"] = 0
	}

	return stats, nil
}

// Delete deletes a transaction
func (r *TransactionRepository) Delete(ctx context.Context, transactionID int, stationID string) error {
	filter := bson.M{
		"transaction_id": transactionID,
		"station_id":     stationID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("transaction not found: %d", transactionID)
	}

	return nil
}

// DeleteOld deletes transactions older than the specified duration
func (r *TransactionRepository) DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	filter := bson.M{
		"start_timestamp": bson.M{"$lt": cutoffTime},
		"status":          bson.M{"$in": []string{"completed", "failed"}},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old transactions: %w", err)
	}

	return result.DeletedCount, nil
}
