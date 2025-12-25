package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MessageStats represents aggregated message statistics
type MessageStats struct {
	TotalMessages    int64            `json:"totalMessages" bson:"totalMessages"`
	SentMessages     int64            `json:"sentMessages" bson:"sentMessages"`
	ReceivedMessages int64            `json:"receivedMessages" bson:"receivedMessages"`
	CallMessages     int64            `json:"callMessages" bson:"callMessages"`
	CallResults      int64            `json:"callResults" bson:"callResults"`
	CallErrors       int64            `json:"callErrors" bson:"callErrors"`
	ByAction         []ActionCount    `json:"byAction" bson:"byAction"`
	ByStation        []StationCount   `json:"byStation" bson:"byStation"`
	ByHour           []TimeSeriesData `json:"byHour,omitempty" bson:"byHour,omitempty"`
}

// ActionCount represents message count per action type
type ActionCount struct {
	Action string `json:"action" bson:"_id"`
	Count  int64  `json:"count" bson:"count"`
}

// StationCount represents message count per station
type StationCount struct {
	StationID string `json:"stationId" bson:"_id"`
	Count     int64  `json:"count" bson:"count"`
	Sent      int64  `json:"sent" bson:"sent"`
	Received  int64  `json:"received" bson:"received"`
}

// TimeSeriesData represents time-bucketed data
type TimeSeriesData struct {
	Timestamp time.Time `json:"timestamp" bson:"_id"`
	Count     int64     `json:"count" bson:"count"`
}

// TransactionStats represents aggregated transaction statistics
type TransactionStats struct {
	TotalTransactions     int64                  `json:"totalTransactions"`
	ActiveTransactions    int64                  `json:"activeTransactions"`
	CompletedTransactions int64                  `json:"completedTransactions"`
	FailedTransactions    int64                  `json:"failedTransactions"`
	TotalEnergyConsumed   float64                `json:"totalEnergyConsumed"` // kWh
	AvgEnergyPerSession   float64                `json:"avgEnergyPerSession"` // kWh
	AvgSessionDuration    float64                `json:"avgSessionDuration"`  // minutes
	ByStation             []StationTransactions  `json:"byStation"`
	ByDay                 []DailyTransactionData `json:"byDay,omitempty"`
}

// StationTransactions represents transaction stats per station
type StationTransactions struct {
	StationID       string  `json:"stationId" bson:"_id"`
	Count           int64   `json:"count" bson:"count"`
	TotalEnergy     float64 `json:"totalEnergy" bson:"totalEnergy"` // Wh
	AvgEnergy       float64 `json:"avgEnergy" bson:"avgEnergy"`     // Wh
	AvgDurationMins float64 `json:"avgDurationMins" bson:"avgDurationMins"`
}

// DailyTransactionData represents daily transaction summary
type DailyTransactionData struct {
	Date        time.Time `json:"date" bson:"_id"`
	Count       int64     `json:"count" bson:"count"`
	TotalEnergy float64   `json:"totalEnergy" bson:"totalEnergy"` // Wh
}

// ErrorStats represents error statistics
type ErrorStats struct {
	TotalErrors int64        `json:"totalErrors"`
	ErrorRate   float64      `json:"errorRate"` // percentage
	ByErrorCode []ErrorCount `json:"byErrorCode"`
	ByStation   []ErrorCount `json:"byStation"`
}

// ErrorCount represents error count by category
type ErrorCount struct {
	Category string `json:"category" bson:"_id"`
	Count    int64  `json:"count" bson:"count"`
}

// GetMessageStats returns aggregated message statistics
func (m *MongoDBClient) GetMessageStats(ctx context.Context, stationID string, since time.Time) (*MessageStats, error) {
	stats := &MessageStats{}

	// Build match stage
	matchStage := bson.M{}
	if stationID != "" {
		matchStage["station_id"] = stationID
	}
	if !since.IsZero() {
		matchStage["timestamp"] = bson.M{"$gte": since}
	}

	// Get total counts by direction and type
	countPipeline := mongo.Pipeline{}
	if len(matchStage) > 0 {
		countPipeline = append(countPipeline, bson.D{{Key: "$match", Value: matchStage}})
	}
	countPipeline = append(countPipeline, bson.D{{Key: "$group", Value: bson.M{
		"_id":   nil,
		"total": bson.M{"$sum": 1},
		"sent": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$direction", "sent"}}, 1, 0},
		}},
		"received": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$direction", "received"}}, 1, 0},
		}},
		"call": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$message_type", "Call"}}, 1, 0},
		}},
		"callResult": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$message_type", "CallResult"}}, 1, 0},
		}},
		"callError": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$message_type", "CallError"}}, 1, 0},
		}},
	}}})

	cursor, err := m.MessagesCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate message counts: %w", err)
	}
	defer cursor.Close(ctx)

	var countResults []struct {
		Total      int64 `bson:"total"`
		Sent       int64 `bson:"sent"`
		Received   int64 `bson:"received"`
		Call       int64 `bson:"call"`
		CallResult int64 `bson:"callResult"`
		CallError  int64 `bson:"callError"`
	}
	if err := cursor.All(ctx, &countResults); err != nil {
		return nil, fmt.Errorf("failed to decode count results: %w", err)
	}

	if len(countResults) > 0 {
		stats.TotalMessages = countResults[0].Total
		stats.SentMessages = countResults[0].Sent
		stats.ReceivedMessages = countResults[0].Received
		stats.CallMessages = countResults[0].Call
		stats.CallResults = countResults[0].CallResult
		stats.CallErrors = countResults[0].CallError
	}

	// Get counts by action
	actionPipeline := mongo.Pipeline{}
	if len(matchStage) > 0 {
		actionPipeline = append(actionPipeline, bson.D{{Key: "$match", Value: matchStage}})
	}
	actionPipeline = append(actionPipeline,
		bson.D{{Key: "$match", Value: bson.M{"action": bson.M{"$ne": ""}}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$action",
			"count": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
		bson.D{{Key: "$limit", Value: 20}},
	)

	cursor, err = m.MessagesCollection.Aggregate(ctx, actionPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by action: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &stats.ByAction); err != nil {
		return nil, fmt.Errorf("failed to decode action results: %w", err)
	}

	// Get counts by station (only if not filtering by station)
	if stationID == "" {
		stationPipeline := mongo.Pipeline{}
		if !since.IsZero() {
			stationPipeline = append(stationPipeline, bson.D{{Key: "$match", Value: bson.M{
				"timestamp": bson.M{"$gte": since},
			}}})
		}
		stationPipeline = append(stationPipeline,
			bson.D{{Key: "$group", Value: bson.M{
				"_id":   "$station_id",
				"count": bson.M{"$sum": 1},
				"sent": bson.M{"$sum": bson.M{
					"$cond": bson.A{bson.M{"$eq": bson.A{"$direction", "sent"}}, 1, 0},
				}},
				"received": bson.M{"$sum": bson.M{
					"$cond": bson.A{bson.M{"$eq": bson.A{"$direction", "received"}}, 1, 0},
				}},
			}}},
			bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
			bson.D{{Key: "$limit", Value: 10}},
		)

		cursor, err = m.MessagesCollection.Aggregate(ctx, stationPipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate by station: %w", err)
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &stats.ByStation); err != nil {
			return nil, fmt.Errorf("failed to decode station results: %w", err)
		}
	}

	// Get hourly breakdown (last 24 hours)
	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour)
	}
	hourlyPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"timestamp": bson.M{"$gte": since},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateTrunc": bson.M{
					"date": "$timestamp",
					"unit": "hour",
				},
			},
			"count": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}
	if stationID != "" {
		hourlyPipeline[0] = bson.D{{Key: "$match", Value: bson.M{
			"station_id": stationID,
			"timestamp":  bson.M{"$gte": since},
		}}}
	}

	cursor, err = m.MessagesCollection.Aggregate(ctx, hourlyPipeline)
	if err != nil {
		// Log error but don't fail - hourly stats are optional
		m.logger.Warn("Failed to get hourly stats", "error", err)
	} else {
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &stats.ByHour); err != nil {
			m.logger.Warn("Failed to decode hourly stats", "error", err)
		}
	}

	return stats, nil
}

// GetTransactionStats returns aggregated transaction statistics
func (m *MongoDBClient) GetTransactionStats(ctx context.Context, stationID string, since time.Time) (*TransactionStats, error) {
	stats := &TransactionStats{}

	// Build match stage
	matchStage := bson.M{}
	if stationID != "" {
		matchStage["station_id"] = stationID
	}
	if !since.IsZero() {
		matchStage["start_timestamp"] = bson.M{"$gte": since}
	}

	// Get total counts and averages
	pipeline := mongo.Pipeline{}
	if len(matchStage) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchStage}})
	}
	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.M{
		"_id":   nil,
		"total": bson.M{"$sum": 1},
		"active": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "active"}}, 1, 0},
		}},
		"completed": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "completed"}}, 1, 0},
		}},
		"failed": bson.M{"$sum": bson.M{
			"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "failed"}}, 1, 0},
		}},
		"totalEnergy": bson.M{"$sum": "$energy_consumed"},
		"avgEnergy":   bson.M{"$avg": "$energy_consumed"},
	}}})

	cursor, err := m.TransactionsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		Total       int64   `bson:"total"`
		Active      int64   `bson:"active"`
		Completed   int64   `bson:"completed"`
		Failed      int64   `bson:"failed"`
		TotalEnergy float64 `bson:"totalEnergy"`
		AvgEnergy   float64 `bson:"avgEnergy"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode transaction results: %w", err)
	}

	if len(results) > 0 {
		stats.TotalTransactions = results[0].Total
		stats.ActiveTransactions = results[0].Active
		stats.CompletedTransactions = results[0].Completed
		stats.FailedTransactions = results[0].Failed
		stats.TotalEnergyConsumed = results[0].TotalEnergy / 1000 // Convert Wh to kWh
		stats.AvgEnergyPerSession = results[0].AvgEnergy / 1000   // Convert Wh to kWh
	}

	// Get average session duration for completed transactions
	durationPipeline := mongo.Pipeline{}
	durationMatch := bson.M{"status": "completed"}
	if stationID != "" {
		durationMatch["station_id"] = stationID
	}
	if !since.IsZero() {
		durationMatch["start_timestamp"] = bson.M{"$gte": since}
	}
	durationPipeline = append(durationPipeline,
		bson.D{{Key: "$match", Value: durationMatch}},
		bson.D{{Key: "$project", Value: bson.M{
			"duration": bson.M{
				"$divide": bson.A{
					bson.M{"$subtract": bson.A{"$stop_timestamp", "$start_timestamp"}},
					60000, // Convert ms to minutes
				},
			},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"avgDuration": bson.M{"$avg": "$duration"},
		}}},
	)

	cursor, err = m.TransactionsCollection.Aggregate(ctx, durationPipeline)
	if err == nil {
		defer cursor.Close(ctx)
		var durationResults []struct {
			AvgDuration float64 `bson:"avgDuration"`
		}
		if err := cursor.All(ctx, &durationResults); err == nil && len(durationResults) > 0 {
			stats.AvgSessionDuration = durationResults[0].AvgDuration
		}
	}

	// Get stats by station (only if not filtering by station)
	if stationID == "" {
		stationPipeline := mongo.Pipeline{}
		if !since.IsZero() {
			stationPipeline = append(stationPipeline, bson.D{{Key: "$match", Value: bson.M{
				"start_timestamp": bson.M{"$gte": since},
			}}})
		}
		stationPipeline = append(stationPipeline,
			bson.D{{Key: "$group", Value: bson.M{
				"_id":         "$station_id",
				"count":       bson.M{"$sum": 1},
				"totalEnergy": bson.M{"$sum": "$energy_consumed"},
				"avgEnergy":   bson.M{"$avg": "$energy_consumed"},
			}}},
			bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
			bson.D{{Key: "$limit", Value: 10}},
		)

		cursor, err = m.TransactionsCollection.Aggregate(ctx, stationPipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate by station: %w", err)
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &stats.ByStation); err != nil {
			return nil, fmt.Errorf("failed to decode station results: %w", err)
		}
	}

	// Get daily breakdown (last 30 days)
	dailySince := since
	if dailySince.IsZero() {
		dailySince = time.Now().Add(-30 * 24 * time.Hour)
	}
	dailyPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"start_timestamp": bson.M{"$gte": dailySince},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateTrunc": bson.M{
					"date": "$start_timestamp",
					"unit": "day",
				},
			},
			"count":       bson.M{"$sum": 1},
			"totalEnergy": bson.M{"$sum": "$energy_consumed"},
		}}},
		bson.D{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}
	if stationID != "" {
		dailyPipeline[0] = bson.D{{Key: "$match", Value: bson.M{
			"station_id":      stationID,
			"start_timestamp": bson.M{"$gte": dailySince},
		}}}
	}

	cursor, err = m.TransactionsCollection.Aggregate(ctx, dailyPipeline)
	if err != nil {
		m.logger.Warn("Failed to get daily stats", "error", err)
	} else {
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &stats.ByDay); err != nil {
			m.logger.Warn("Failed to decode daily stats", "error", err)
		}
	}

	return stats, nil
}

// GetErrorStats returns error statistics
func (m *MongoDBClient) GetErrorStats(ctx context.Context, stationID string, since time.Time) (*ErrorStats, error) {
	stats := &ErrorStats{}

	// Build match stage for CallError messages
	matchStage := bson.M{"message_type": "CallError"}
	if stationID != "" {
		matchStage["station_id"] = stationID
	}
	if !since.IsZero() {
		matchStage["timestamp"] = bson.M{"$gte": since}
	}

	// Count errors by error code
	byCodePipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: matchStage}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$error_code",
			"count": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := m.MessagesCollection.Aggregate(ctx, byCodePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate errors by code: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &stats.ByErrorCode); err != nil {
		return nil, fmt.Errorf("failed to decode error code results: %w", err)
	}

	// Calculate total errors
	for _, ec := range stats.ByErrorCode {
		stats.TotalErrors += ec.Count
	}

	// Count errors by station (only if not filtering by station)
	if stationID == "" {
		byStationPipeline := mongo.Pipeline{
			bson.D{{Key: "$match", Value: bson.M{"message_type": "CallError"}}},
		}
		if !since.IsZero() {
			byStationPipeline[0] = bson.D{{Key: "$match", Value: bson.M{
				"message_type": "CallError",
				"timestamp":    bson.M{"$gte": since},
			}}}
		}
		byStationPipeline = append(byStationPipeline,
			bson.D{{Key: "$group", Value: bson.M{
				"_id":   "$station_id",
				"count": bson.M{"$sum": 1},
			}}},
			bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
			bson.D{{Key: "$limit", Value: 10}},
		)

		cursor, err = m.MessagesCollection.Aggregate(ctx, byStationPipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate errors by station: %w", err)
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &stats.ByStation); err != nil {
			return nil, fmt.Errorf("failed to decode station error results: %w", err)
		}
	}

	// Calculate error rate
	totalMatch := bson.M{}
	if stationID != "" {
		totalMatch["station_id"] = stationID
	}
	if !since.IsZero() {
		totalMatch["timestamp"] = bson.M{"$gte": since}
	}

	totalMessages, err := m.MessagesCollection.CountDocuments(ctx, totalMatch)
	if err == nil && totalMessages > 0 {
		stats.ErrorRate = float64(stats.TotalErrors) / float64(totalMessages) * 100
	}

	return stats, nil
}
