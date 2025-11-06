package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBClient represents a MongoDB client with all collections
type MongoDBClient struct {
	client   *mongo.Client
	database *mongo.Database
	cfg      *config.MongoDBConfig
	logger   *slog.Logger

	// Collections
	MessagesCollection     *mongo.Collection
	TransactionsCollection *mongo.Collection
	StationsCollection     *mongo.Collection
	SessionsCollection     *mongo.Collection
	MeterValuesCollection  *mongo.Collection
}

// NewMongoDBClient creates a new MongoDB client and establishes connection
func NewMongoDBClient(ctx context.Context, cfg *config.MongoDBConfig, logger *slog.Logger) (*MongoDBClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("Connecting to MongoDB",
		"uri", cfg.URI,
		"database", cfg.Database,
	)

	// Set client options
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetServerSelectionTimeout(cfg.ConnectionTimeout)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	ctxPing, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()

	if err := client.Ping(ctxPing, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Info("Successfully connected to MongoDB")

	// Get database reference
	database := client.Database(cfg.Database)

	// Create MongoDBClient instance
	mongoClient := &MongoDBClient{
		client:                 client,
		database:               database,
		cfg:                    cfg,
		logger:                 logger,
		MessagesCollection:     database.Collection(cfg.Collections.Messages),
		TransactionsCollection: database.Collection(cfg.Collections.Transactions),
		StationsCollection:     database.Collection(cfg.Collections.Stations),
		SessionsCollection:     database.Collection(cfg.Collections.Sessions),
		MeterValuesCollection:  database.Collection(cfg.Collections.MeterValues),
	}

	// Initialize collections and indexes
	if err := mongoClient.initializeCollections(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize collections: %w", err)
	}

	return mongoClient, nil
}

// initializeCollections creates collections and indexes
func (m *MongoDBClient) initializeCollections(ctx context.Context) error {
	m.logger.Info("Initializing MongoDB collections and indexes")

	// Create time-series collection for meter values if enabled
	if m.cfg.TimeSeries.Enabled {
		if err := m.createTimeSeriesCollection(ctx); err != nil {
			m.logger.Warn("Failed to create time-series collection, continuing with regular collection",
				"error", err,
			)
		}
	}

	// Create indexes
	if err := m.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	m.logger.Info("Successfully initialized MongoDB collections and indexes")
	return nil
}

// createTimeSeriesCollection creates a time-series collection for meter values
func (m *MongoDBClient) createTimeSeriesCollection(ctx context.Context) error {
	// Check if collection already exists
	collections, err := m.database.ListCollectionNames(ctx, bson.M{"name": m.cfg.Collections.MeterValues})
	if err != nil {
		return err
	}

	// If collection exists, skip creation
	if len(collections) > 0 {
		m.logger.Info("Meter values collection already exists")
		return nil
	}

	// Create time-series collection
	opts := options.CreateCollection().
		SetTimeSeriesOptions(options.TimeSeries().
			SetTimeField("timestamp").
			SetMetaField("metadata").
			SetGranularity(m.cfg.TimeSeries.Granularity))

	err = m.database.CreateCollection(ctx, m.cfg.Collections.MeterValues, opts)
	if err != nil {
		return fmt.Errorf("failed to create time-series collection: %w", err)
	}

	m.logger.Info("Created time-series collection for meter values")
	return nil
}

// createIndexes creates all necessary indexes
func (m *MongoDBClient) createIndexes(ctx context.Context) error {
	// Messages indexes
	messagesIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "station_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "message_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "correlation_id", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "action", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
	}

	if _, err := m.MessagesCollection.Indexes().CreateMany(ctx, messagesIndexes); err != nil {
		return fmt.Errorf("failed to create messages indexes: %w", err)
	}

	// Transactions indexes
	transactionsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "transaction_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "station_id", Value: 1},
				{Key: "start_timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "id_tag", Value: 1}},
		},
	}

	if _, err := m.TransactionsCollection.Indexes().CreateMany(ctx, transactionsIndexes); err != nil {
		return fmt.Errorf("failed to create transactions indexes: %w", err)
	}

	// Stations indexes
	stationsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "station_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "connection_status", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "enabled", Value: 1},
				{Key: "auto_start", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "protocol_version", Value: 1}},
		},
	}

	if _, err := m.StationsCollection.Indexes().CreateMany(ctx, stationsIndexes); err != nil {
		return fmt.Errorf("failed to create stations indexes: %w", err)
	}

	// Sessions indexes
	sessionsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "station_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	if _, err := m.SessionsCollection.Indexes().CreateMany(ctx, sessionsIndexes); err != nil {
		return fmt.Errorf("failed to create sessions indexes: %w", err)
	}

	m.logger.Info("Successfully created all indexes")
	return nil
}

// Ping checks if the MongoDB connection is alive
func (m *MongoDBClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return m.client.Ping(ctx, nil)
}

// Close closes the MongoDB connection
func (m *MongoDBClient) Close(ctx context.Context) error {
	m.logger.Info("Closing MongoDB connection")

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	m.logger.Info("Successfully closed MongoDB connection")
	return nil
}

// GetClient returns the underlying MongoDB client
func (m *MongoDBClient) GetClient() *mongo.Client {
	return m.client
}

// GetDatabase returns the database instance
func (m *MongoDBClient) GetDatabase() *mongo.Database {
	return m.database
}

// HealthCheck performs a health check on the MongoDB connection
func (m *MongoDBClient) HealthCheck(ctx context.Context) error {
	// Ping the database
	if err := m.Ping(ctx); err != nil {
		return fmt.Errorf("MongoDB health check failed: %w", err)
	}

	// Check if collections exist
	collections := []string{
		m.cfg.Collections.Messages,
		m.cfg.Collections.Transactions,
		m.cfg.Collections.Stations,
		m.cfg.Collections.Sessions,
		m.cfg.Collections.MeterValues,
	}

	dbCollections, err := m.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	collectionMap := make(map[string]bool)
	for _, col := range dbCollections {
		collectionMap[col] = true
	}

	for _, col := range collections {
		if !collectionMap[col] {
			return fmt.Errorf("collection %s does not exist", col)
		}
	}

	return nil
}

// Stats returns MongoDB connection statistics
func (m *MongoDBClient) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get database stats
	var dbStats bson.M
	if err := m.database.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStats); err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	stats["database"] = dbStats

	// Get collection counts
	collectionCounts := make(map[string]int64)

	if count, err := m.MessagesCollection.CountDocuments(ctx, bson.M{}); err == nil {
		collectionCounts["messages"] = count
	}

	if count, err := m.TransactionsCollection.CountDocuments(ctx, bson.M{}); err == nil {
		collectionCounts["transactions"] = count
	}

	if count, err := m.StationsCollection.CountDocuments(ctx, bson.M{}); err == nil {
		collectionCounts["stations"] = count
	}

	if count, err := m.SessionsCollection.CountDocuments(ctx, bson.M{}); err == nil {
		collectionCounts["sessions"] = count
	}

	if count, err := m.MeterValuesCollection.CountDocuments(ctx, bson.M{}); err == nil {
		collectionCounts["meter_values"] = count
	}

	stats["collection_counts"] = collectionCounts

	return stats, nil
}
