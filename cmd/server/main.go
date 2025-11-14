package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/api"
	"github.com/ruslanhut/ocpp-emu/internal/config"
	"github.com/ruslanhut/ocpp-emu/internal/connection"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
	"github.com/ruslanhut/ocpp-emu/internal/station"
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

	// Initialize WebSocket Connection Manager
	connManager := connection.NewManager(&cfg.CSMS, logger)
	logger.Info("WebSocket connection manager initialized")

	// Message Broadcaster for real-time WebSocket streaming
	// Testing if this is causing the backend to become unresponsive
	messageBroadcaster := api.NewMessageBroadcaster(logger)
	messageBroadcaster.Start()
	logger.Info("Message broadcaster initialized and started")

	// Message Logger
	// Testing if this is causing the backend to become unresponsive
	messageLogger := logging.NewMessageLogger(
		mongoClient,
		logger,
		logging.LoggerConfig{
			BufferSize:    1000,
			BatchSize:     100,
			FlushInterval: 5 * time.Second,
			LogLevel:      "info",
		},
	)
	messageLogger.SetBroadcaster(messageBroadcaster)
	messageLogger.Start()
	logger.Info("Message logger initialized and started")

	// Initialize Station Manager
	stationManager := station.NewManager(
		mongoClient,
		connManager,
		messageLogger,
		logger,
		station.ManagerConfig{
			SyncInterval: 30 * time.Second,
		},
	)
	logger.Info("Station manager initialized")

	// Load stations from MongoDB
	if err := stationManager.LoadStations(ctx); err != nil {
		logger.Error("Failed to load stations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Stations loaded from MongoDB")

	// Reconcile station data with active transactions
	if err := stationManager.ReconcileStationData(ctx); err != nil {
		logger.Error("Failed to reconcile station data", slog.String("error", err.Error()))
		// Don't exit - continue with potentially inconsistent state
	}

	// Set up connection callbacks to route through station manager
	connManager.OnMessageReceived = func(stationID string, message []byte) {
		stationManager.OnMessageReceived(stationID, message)
	}

	connManager.OnStationConnected = func(stationID string) {
		stationManager.OnStationConnected(stationID)
	}

	connManager.OnStationDisconnected = func(stationID string, err error) {
		stationManager.OnStationDisconnected(stationID, err)
	}

	connManager.OnStationError = func(stationID string, err error) {
		logger.Error("Station connection error",
			slog.String("station_id", stationID),
			slog.String("error", err.Error()),
		)
	}

	// Background state synchronization
	stationManager.StartSync()
	logger.Info("Station state synchronization started")

	// Auto-start enabled stations
	if err := stationManager.AutoStart(ctx); err != nil {
		logger.Error("Failed to auto-start stations", slog.String("error", err.Error()))
	}

	// Set up HTTP server
	mux := http.NewServeMux()

	// CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from frontend dev server
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Health check endpoint
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check MongoDB health
		if err := mongoClient.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","version":"%s","error":"MongoDB connection failed"}`, appVersion)
			return
		}

		// Get station manager stats
		stats := stationManager.GetStats()

		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"status":   "healthy",
			"version":  appVersion,
			"database": "connected",
			"stations": stats,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Connection status endpoint
	mux.HandleFunc("/api/connections", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		stats := connManager.GetAllConnectionStats()

		// Convert to JSON
		response := make(map[string]interface{})
		response["total"] = connManager.GetTotalCount()
		response["connected"] = connManager.GetConnectedCount()
		response["stations"] = stats

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Message history endpoint
	mux.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if messageLogger == nil {
			if r.Method == http.MethodDelete {
				response := map[string]interface{}{
					"success": false,
					"message": "Message logger is disabled",
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			response := map[string]interface{}{
				"messages": []interface{}{},
				"total":    0,
				"count":    0,
				"limit":    100,
				"skip":     0,
				"disabled": true,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle DELETE request to clear all messages
		if r.Method == http.MethodDelete {
			deletedCount, err := messageLogger.ClearAllMessages(r.Context())
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to clear messages: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"success":      true,
				"message":      "All messages cleared successfully",
				"deletedCount": deletedCount,
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
			return
		}

		// Handle GET request to retrieve messages
		// Parse query parameters
		query := r.URL.Query()
		filter := logging.MessageFilter{
			StationID:   query.Get("stationId"),
			Direction:   query.Get("direction"),
			MessageType: query.Get("messageType"),
			Action:      query.Get("action"),
			Limit:       100, // default
			Skip:        0,
		}

		// Parse limit and skip
		if limitStr := query.Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &filter.Limit)
		}
		if skipStr := query.Get("skip"); skipStr != "" {
			fmt.Sscanf(skipStr, "%d", &filter.Skip)
		}

		// Parse time range
		if startStr := query.Get("startTime"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				filter.StartTime = t
			}
		}
		if endStr := query.Get("endTime"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				filter.EndTime = t
			}
		}

		// Get messages
		messages, err := messageLogger.GetMessages(r.Context(), filter)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get messages: %v", err), http.StatusInternalServerError)
			return
		}

		// Get total count
		totalCount, err := messageLogger.CountMessages(r.Context(), filter)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to count messages: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"messages": messages,
			"total":    totalCount,
			"count":    len(messages),
			"limit":    filter.Limit,
			"skip":     filter.Skip,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Message search endpoint
	mux.HandleFunc("/api/messages/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if messageLogger == nil {
			response := map[string]interface{}{
				"messages":   []interface{}{},
				"count":      0,
				"searchTerm": r.URL.Query().Get("q"),
				"disabled":   true,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		query := r.URL.Query()
		searchTerm := query.Get("q")
		if searchTerm == "" {
			http.Error(w, "Search term required", http.StatusBadRequest)
			return
		}

		filter := logging.MessageFilter{
			StationID:   query.Get("stationId"),
			Direction:   query.Get("direction"),
			MessageType: query.Get("messageType"),
			Limit:       100,
			Skip:        0,
		}

		if limitStr := query.Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &filter.Limit)
		}
		if skipStr := query.Get("skip"); skipStr != "" {
			fmt.Sscanf(skipStr, "%d", &filter.Skip)
		}

		messages, err := messageLogger.SearchMessages(r.Context(), searchTerm, filter)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to search messages: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"messages":   messages,
			"count":      len(messages),
			"searchTerm": searchTerm,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Message logger stats endpoint
	mux.HandleFunc("/api/messages/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if messageLogger == nil {
			response := map[string]interface{}{
				"total":              0,
				"sent":               0,
				"received":           0,
				"buffered":           0,
				"dropped":            0,
				"callMessages":       0,
				"callResultMessages": 0,
				"callErrorMessages":  0,
				"disabled":           true,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		stats := messageLogger.GetStats()

		response := map[string]interface{}{
			"total":              stats.TotalMessages,
			"sent":               stats.SentMessages,
			"received":           stats.ReceivedMessages,
			"buffered":           stats.BufferedMessages,
			"dropped":            stats.DroppedMessages,
			"callMessages":       stats.CallMessages,
			"callResultMessages": stats.CallResultMessages,
			"callErrorMessages":  stats.CallErrorMessages,
			"lastFlush":          stats.LastFlush,
			"flushCount":         stats.FlushCount,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Initialize Station API Handler
	stationHandler := api.NewStationHandler(stationManager, logger)
	logger.Info("Station API handler initialized")

	// WebSocket Handler for real-time message streaming
	wsHandler := api.NewWebSocketHandler(messageBroadcaster, logger)
	logger.Info("WebSocket handler initialized")

	// Station CRUD endpoints
	mux.HandleFunc("/api/stations", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			stationHandler.ListStations(w, r)
		case http.MethodPost:
			stationHandler.CreateStation(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Station detail endpoints (with path-based routing)
	mux.HandleFunc("/api/stations/", func(w http.ResponseWriter, r *http.Request) {
		// Check if path ends with /start or /stop
		if strings.HasSuffix(r.URL.Path, "/start") {
			stationHandler.StartStation(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stop") {
			stationHandler.StopStation(w, r)
			return
		}

		// Check if path ends with /connectors
		if strings.HasSuffix(r.URL.Path, "/connectors") {
			stationHandler.GetConnectors(w, r)
			return
		}

		// Check if path ends with /charge
		if strings.HasSuffix(r.URL.Path, "/charge") {
			stationHandler.StartCharging(w, r)
			return
		}

		// Check if path ends with /stop-charge
		if strings.HasSuffix(r.URL.Path, "/stop-charge") {
			stationHandler.StopCharging(w, r)
			return
		}

		// Otherwise, handle CRUD operations on individual stations
		switch r.Method {
		case http.MethodGet:
			stationHandler.GetStation(w, r)
		case http.MethodPut:
			stationHandler.UpdateStation(w, r)
		case http.MethodDelete:
			stationHandler.DeleteStation(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// WebSocket endpoints
	mux.HandleFunc("/api/ws/messages", wsHandler.HandleMessages)
	mux.HandleFunc("/api/ws/stats", wsHandler.HandleBroadcasterStats)
	logger.Info("WebSocket endpoints registered")

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      corsMiddleware(mux),
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	// Shutdown station manager (stops all stations)
	if err := stationManager.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown station manager", slog.String("error", err.Error()))
	} else {
		logger.Info("Station manager shutdown complete")
	}

	// Shutdown connection manager
	if err := connManager.Shutdown(); err != nil {
		logger.Error("Failed to shutdown connection manager", slog.String("error", err.Error()))
	}

	// Shutdown message logger
	if err := messageLogger.Shutdown(); err != nil {
		logger.Error("Failed to shutdown message logger", slog.String("error", err.Error()))
	}

	// Shutdown message broadcaster
	if err := messageBroadcaster.Shutdown(); err != nil {
		logger.Error("Failed to shutdown message broadcaster", slog.String("error", err.Error()))
	}

	// Close MongoDB connection
	if err := mongoClient.Close(shutdownCtx); err != nil {
		logger.Error("Failed to close MongoDB connection", slog.String("error", err.Error()))
	}

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
