package logging

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/ocpp"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MessageLogger handles OCPP message logging with buffering and real-time streaming
type MessageLogger struct {
	db            *storage.MongoDBClient
	logger        *slog.Logger
	messageBuffer chan MessageEntry
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	config        LoggerConfig
	stats         LoggerStats
	statsMu       sync.RWMutex
}

// LoggerConfig represents the message logger configuration
type LoggerConfig struct {
	BufferSize      int           // Size of message buffer
	BatchSize       int           // Number of messages to batch for MongoDB insert
	FlushInterval   time.Duration // How often to flush buffer
	EnableFiltering bool          // Enable message filtering
	LogLevel        string        // Minimum log level
}

// MessageEntry represents a message to be logged
type MessageEntry struct {
	StationID       string
	Direction       string // "sent" or "received"
	MessageType     string // "Call", "CallResult", "CallError"
	Action          string
	MessageID       string
	ProtocolVersion string
	Payload         interface{}
	RawMessage      []byte
	Timestamp       time.Time
	CorrelationID   string
	ErrorCode       string
	ErrorDesc       string
}

// LoggerStats tracks message logging statistics
type LoggerStats struct {
	TotalMessages      int64
	SentMessages       int64
	ReceivedMessages   int64
	CallMessages       int64
	CallResultMessages int64
	CallErrorMessages  int64
	BufferedMessages   int
	DroppedMessages    int64
	LastFlush          time.Time
	FlushCount         int64
}

// NewMessageLogger creates a new message logger
func NewMessageLogger(
	db *storage.MongoDBClient,
	logger *slog.Logger,
	config LoggerConfig,
) *MessageLogger {
	// Set defaults
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	ctx, cancel := context.WithCancel(context.Background())

	ml := &MessageLogger{
		db:            db,
		logger:        logger,
		messageBuffer: make(chan MessageEntry, config.BufferSize),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
	}

	return ml
}

// Start begins the message logging process
func (ml *MessageLogger) Start() {
	ml.logger.Info("Starting message logger",
		"bufferSize", ml.config.BufferSize,
		"batchSize", ml.config.BatchSize,
		"flushInterval", ml.config.FlushInterval,
	)

	ml.wg.Add(1)
	go ml.processMessages()
}

// processMessages processes messages from the buffer and writes to MongoDB
func (ml *MessageLogger) processMessages() {
	defer ml.wg.Done()

	ticker := time.NewTicker(ml.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]MessageEntry, 0, ml.config.BatchSize)

	for {
		select {
		case <-ml.ctx.Done():
			ml.logger.Info("Stopping message logger")
			// Flush remaining messages
			if len(batch) > 0 {
				ml.flushBatch(batch)
			}
			return

		case msg := <-ml.messageBuffer:
			batch = append(batch, msg)

			// Flush if batch is full
			if len(batch) >= ml.config.BatchSize {
				ml.flushBatch(batch)
				batch = make([]MessageEntry, 0, ml.config.BatchSize)
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				ml.flushBatch(batch)
				batch = make([]MessageEntry, 0, ml.config.BatchSize)
			}
		}
	}
}

// flushBatch writes a batch of messages to MongoDB
func (ml *MessageLogger) flushBatch(batch []MessageEntry) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert to storage.Message format
	documents := make([]interface{}, len(batch))
	for i, entry := range batch {
		documents[i] = ml.convertToStorageMessage(entry)
	}

	// Bulk insert
	collection := ml.db.MessagesCollection
	_, err := collection.InsertMany(ctx, documents)
	if err != nil {
		ml.logger.Error("Failed to insert message batch",
			"count", len(batch),
			"error", err,
		)
		ml.incrementDropped(int64(len(batch)))
		return
	}

	ml.logger.Debug("Flushed message batch",
		"count", len(batch),
	)

	ml.updateStats(batch)
}

// convertToStorageMessage converts MessageEntry to storage.Message
func (ml *MessageLogger) convertToStorageMessage(entry MessageEntry) storage.Message {
	// Convert payload to map[string]interface{}
	var payloadMap map[string]interface{}
	if entry.Payload != nil {
		if m, ok := entry.Payload.(map[string]interface{}); ok {
			payloadMap = m
		} else {
			payloadMap = make(map[string]interface{})
			payloadMap["data"] = entry.Payload
		}
	}

	return storage.Message{
		StationID:        entry.StationID,
		Direction:        entry.Direction,
		MessageType:      entry.MessageType,
		Action:           entry.Action,
		MessageID:        entry.MessageID,
		ProtocolVersion:  entry.ProtocolVersion,
		Payload:          payloadMap,
		Timestamp:        entry.Timestamp,
		CorrelationID:    entry.CorrelationID,
		ErrorCode:        entry.ErrorCode,
		ErrorDescription: entry.ErrorDesc,
		CreatedAt:        time.Now(),
	}
}

// LogMessage logs an OCPP message
func (ml *MessageLogger) LogMessage(
	stationID string,
	direction string,
	message interface{},
	protocolVersion string,
) error {
	entry := ml.createMessageEntry(stationID, direction, message, protocolVersion)

	select {
	case ml.messageBuffer <- entry:
		// Successfully buffered
		return nil
	default:
		// Buffer full, drop message
		ml.logger.Warn("Message buffer full, dropping message",
			"stationId", stationID,
			"direction", direction,
		)
		ml.incrementDropped(1)
		return fmt.Errorf("message buffer full")
	}
}

// createMessageEntry creates a MessageEntry from an OCPP message
func (ml *MessageLogger) createMessageEntry(
	stationID string,
	direction string,
	message interface{},
	protocolVersion string,
) MessageEntry {
	entry := MessageEntry{
		StationID:       stationID,
		Direction:       direction,
		ProtocolVersion: protocolVersion,
		Timestamp:       time.Now(),
	}

	switch msg := message.(type) {
	case *ocpp.Call:
		entry.MessageType = "Call"
		entry.Action = msg.Action
		entry.MessageID = msg.UniqueID
		entry.Payload = msg.Payload

	case *ocpp.CallResult:
		entry.MessageType = "CallResult"
		entry.MessageID = msg.UniqueID
		entry.Payload = msg.Payload

	case *ocpp.CallError:
		entry.MessageType = "CallError"
		entry.MessageID = msg.UniqueID
		entry.ErrorCode = string(msg.ErrorCode)
		entry.ErrorDesc = msg.ErrorDesc
		errorPayload := make(map[string]interface{})
		errorPayload["errorCode"] = msg.ErrorCode
		errorPayload["errorDesc"] = msg.ErrorDesc
		errorPayload["details"] = msg.ErrorDetails
		entry.Payload = errorPayload

	case []byte:
		// Raw message
		entry.RawMessage = msg
		// Try to parse
		if parsed, err := ocpp.ParseMessage(msg); err == nil {
			return ml.createMessageEntry(stationID, direction, parsed, protocolVersion)
		}
	}

	return entry
}

// updateStats updates logger statistics
func (ml *MessageLogger) updateStats(batch []MessageEntry) {
	ml.statsMu.Lock()
	defer ml.statsMu.Unlock()

	ml.stats.TotalMessages += int64(len(batch))
	ml.stats.LastFlush = time.Now()
	ml.stats.FlushCount++

	for _, entry := range batch {
		if entry.Direction == "sent" {
			ml.stats.SentMessages++
		} else {
			ml.stats.ReceivedMessages++
		}

		switch entry.MessageType {
		case "Call":
			ml.stats.CallMessages++
		case "CallResult":
			ml.stats.CallResultMessages++
		case "CallError":
			ml.stats.CallErrorMessages++
		}
	}

	ml.stats.BufferedMessages = len(ml.messageBuffer)
}

// incrementDropped increments dropped message count
func (ml *MessageLogger) incrementDropped(count int64) {
	ml.statsMu.Lock()
	defer ml.statsMu.Unlock()
	ml.stats.DroppedMessages += count
}

// GetStats returns current logger statistics
func (ml *MessageLogger) GetStats() LoggerStats {
	ml.statsMu.RLock()
	defer ml.statsMu.RUnlock()

	stats := ml.stats
	stats.BufferedMessages = len(ml.messageBuffer)
	return stats
}

// GetMessages retrieves messages from MongoDB with filtering
func (ml *MessageLogger) GetMessages(ctx context.Context, filter MessageFilter) ([]storage.Message, error) {
	query := bson.M{}

	// Apply filters
	if filter.StationID != "" {
		query["station_id"] = filter.StationID
	}
	if filter.Direction != "" {
		query["direction"] = filter.Direction
	}
	if filter.MessageType != "" {
		query["message_type"] = filter.MessageType
	}
	if filter.Action != "" {
		query["action"] = filter.Action
	}
	if !filter.StartTime.IsZero() {
		query["timestamp"] = bson.M{"$gte": filter.StartTime}
	}
	if !filter.EndTime.IsZero() {
		if existingTime, ok := query["timestamp"].(bson.M); ok {
			existingTime["$lte"] = filter.EndTime
		} else {
			query["timestamp"] = bson.M{"$lte": filter.EndTime}
		}
	}

	// Set defaults
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if filter.Skip < 0 {
		filter.Skip = 0
	}

	// Query options
	opts := options.Find().
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Skip)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}}) // Most recent first

	collection := ml.db.MessagesCollection
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer cursor.Close(ctx)

	messages := make([]storage.Message, 0)
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// CountMessages counts messages matching the filter
func (ml *MessageLogger) CountMessages(ctx context.Context, filter MessageFilter) (int64, error) {
	query := bson.M{}

	if filter.StationID != "" {
		query["station_id"] = filter.StationID
	}
	if filter.Direction != "" {
		query["direction"] = filter.Direction
	}
	if filter.MessageType != "" {
		query["message_type"] = filter.MessageType
	}
	if filter.Action != "" {
		query["action"] = filter.Action
	}
	if !filter.StartTime.IsZero() {
		query["timestamp"] = bson.M{"$gte": filter.StartTime}
	}
	if !filter.EndTime.IsZero() {
		if existingTime, ok := query["timestamp"].(bson.M); ok {
			existingTime["$lte"] = filter.EndTime
		} else {
			query["timestamp"] = bson.M{"$lte": filter.EndTime}
		}
	}

	collection := ml.db.MessagesCollection
	count, err := collection.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// SearchMessages searches messages by text
func (ml *MessageLogger) SearchMessages(ctx context.Context, searchTerm string, filter MessageFilter) ([]storage.Message, error) {
	query := bson.M{
		"$or": []bson.M{
			{"action": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"message_id": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"station_id": bson.M{"$regex": searchTerm, "$options": "i"}},
		},
	}

	// Apply additional filters
	if filter.StationID != "" {
		query["station_id"] = filter.StationID
	}
	if filter.Direction != "" {
		query["direction"] = filter.Direction
	}
	if filter.MessageType != "" {
		query["message_type"] = filter.MessageType
	}

	if filter.Limit == 0 {
		filter.Limit = 100
	}

	opts := options.Find().
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Skip)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	collection := ml.db.MessagesCollection
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer cursor.Close(ctx)

	messages := make([]storage.Message, 0)
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// DeleteOldMessages deletes messages older than the specified duration
func (ml *MessageLogger) DeleteOldMessages(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	collection := ml.db.MessagesCollection
	result, err := collection.DeleteMany(ctx, bson.M{
		"timestamp": bson.M{"$lt": cutoffTime},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}

	ml.logger.Info("Deleted old messages",
		"count", result.DeletedCount,
		"olderThan", olderThan,
	)

	return result.DeletedCount, nil
}

// Shutdown gracefully shuts down the message logger
func (ml *MessageLogger) Shutdown() error {
	ml.logger.Info("Shutting down message logger")

	// Cancel context to stop processing
	ml.cancel()

	// Wait for processing to finish
	ml.wg.Wait()

	// Close buffer channel
	close(ml.messageBuffer)

	stats := ml.GetStats()
	ml.logger.Info("Message logger shutdown complete",
		"totalMessages", stats.TotalMessages,
		"droppedMessages", stats.DroppedMessages,
		"flushCount", stats.FlushCount,
	)

	return nil
}

// MessageFilter represents message filtering criteria
type MessageFilter struct {
	StationID   string
	Direction   string // "sent" or "received"
	MessageType string // "Call", "CallResult", "CallError"
	Action      string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Skip        int
}
