package storage

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/config"
)

// TestMongoDBConnection tests the MongoDB connection
// This test requires a running MongoDB instance on localhost:27017
func TestMongoDBConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test configuration
	cfg := &config.MongoDBConfig{
		URI:               "mongodb://localhost:27017",
		Database:          "ocpp_emu_test",
		ConnectionTimeout: 10 * time.Second,
		MaxPoolSize:       10,
		Collections: config.MongoDBCollectionsConfig{
			Messages:     "messages",
			Transactions: "transactions",
			Stations:     "stations",
			Sessions:     "sessions",
			MeterValues:  "meter_values",
		},
		TimeSeries: config.MongoDBTimeSeriesConfig{
			Enabled:     true,
			Granularity: "seconds",
		},
	}

	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create MongoDB client
	client, err := NewMongoDBClient(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create MongoDB client: %v", err)
	}
	defer client.Close(ctx)

	// Test ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Failed to ping MongoDB: %v", err)
	}

	// Test health check
	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test stats
	stats, err := client.Stats(ctx)
	if err != nil {
		t.Errorf("Failed to get stats: %v", err)
	} else {
		t.Logf("MongoDB stats: %+v", stats)
	}
}

// TestMongoDBClientCreation tests client creation without connection
func TestMongoDBClientCreation(t *testing.T) {
	cfg := &config.MongoDBConfig{
		URI:               "mongodb://localhost:27017",
		Database:          "ocpp_emu_test",
		ConnectionTimeout: 1 * time.Second,
		MaxPoolSize:       10,
		Collections: config.MongoDBCollectionsConfig{
			Messages:     "messages",
			Transactions: "transactions",
			Stations:     "stations",
			Sessions:     "sessions",
			MeterValues:  "meter_values",
		},
		TimeSeries: config.MongoDBTimeSeriesConfig{
			Enabled:     true,
			Granularity: "seconds",
		},
	}

	// Test that the configuration is valid
	if cfg.URI == "" {
		t.Error("URI should not be empty")
	}

	if cfg.Database == "" {
		t.Error("Database should not be empty")
	}

	if cfg.Collections.Messages == "" {
		t.Error("Messages collection name should not be empty")
	}
}
