package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/config"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
)

const (
	appName    = "ocpp-emu"
	appVersion = "0.1.0"
)

func main() {
	// Initialize logger
	logger := initLogger()
	logger.Info("Starting OCPP Emulator",
		slog.String("version", appVersion),
		slog.String("app", appName))

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	// Initialize MongoDB connection
	ctx := context.Background()
	mongoClient, err := storage.NewMongoDBClient(ctx, &cfg.MongoDB, logger)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("MongoDB connection established successfully")

	// Perform health check
	if err := mongoClient.HealthCheck(ctx); err != nil {
		logger.Error("MongoDB health check failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Get and log MongoDB statistics
	stats, err := mongoClient.Stats(ctx)
	if err != nil {
		logger.Warn("Failed to get MongoDB stats", slog.String("error", err.Error()))
	} else {
		logger.Info("MongoDB statistics", slog.Any("stats", stats))
	}

	// Initialize Station Manager
	// TODO: Implement Station Manager

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check MongoDB health
		if err := mongoClient.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","version":"%s","error":"MongoDB connection failed"}`, appVersion)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s","database":"connected"}`, appVersion)
	})

	// TODO: Set up API routes
	// TODO: Set up WebSocket handlers

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server", slog.String("address", serverAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("OCPP Emulator started successfully", slog.String("address", serverAddr))

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	// Close MongoDB connection
	if err := mongoClient.Close(ctx); err != nil {
		logger.Error("Failed to close MongoDB connection", slog.String("error", err.Error()))
	}

	// TODO: Stop all running stations

	logger.Info("Server stopped")
}

// initLogger initializes the structured logger using slog
func initLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// Note: Configuration structs and loading logic have been moved to internal/config package
