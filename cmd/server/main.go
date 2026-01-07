package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/api"
	"github.com/ruslanhut/ocpp-emu/internal/auth"
	"github.com/ruslanhut/ocpp-emu/internal/config"
	"github.com/ruslanhut/ocpp-emu/internal/connection"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
	"github.com/ruslanhut/ocpp-emu/internal/scenario"
	"github.com/ruslanhut/ocpp-emu/internal/station"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
)

const (
	appName    = "ocpp-emu"
	appVersion = "0.1.0"
)

func main() {
	configPath := flag.String("conf", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Error loading config: %v", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg)
	logger.Info("Starting OCPP Emulator",
		slog.String("version", appVersion),
		slog.String("conf", fmt.Sprintf("%v", configPath)),
		slog.String("app", appName))

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

	// Initialize Auth Service
	authConfig := convertAuthConfig(&cfg.Auth)
	authService := auth.NewService(authConfig, logger)
	authHandler := auth.NewHandler(authService, logger)
	if authService.IsEnabled() {
		logger.Info("Authentication enabled")
	} else {
		logger.Info("Authentication disabled")
	}

	// CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from frontend dev server
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Auth middleware that requires authentication
	requireAuth := func(next http.Handler) http.Handler {
		return authService.Middleware(next)
	}

	// Note: For admin-only checks, we check user.Role inline in handlers
	// to allow viewers to access GET endpoints while restricting write operations

	// Auth endpoints (public)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.Handle("/api/auth/logout", requireAuth(http.HandlerFunc(authHandler.HandleLogout)))
	mux.Handle("/api/auth/me", requireAuth(http.HandlerFunc(authHandler.HandleMe)))
	mux.Handle("/api/auth/refresh", requireAuth(http.HandlerFunc(authHandler.HandleRefresh)))
	logger.Info("Auth endpoints registered")

	// Health check endpoint (public)
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

	// Connection status endpoint (auth protected)
	mux.Handle("/api/connections", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})))

	// Message history endpoint (auth protected, DELETE requires admin)
	mux.Handle("/api/messages", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Handle DELETE request to clear all messages (admin only)
		if r.Method == http.MethodDelete {
			user := auth.GetUserFromContext(r.Context())
			if user == nil || user.Role != auth.RoleAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
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
	})))

	// Message search endpoint (auth protected)
	mux.Handle("/api/messages/search", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})))

	// Message logger stats endpoint (auth protected)
	mux.Handle("/api/messages/stats", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})))

	// Initialize Station API Handler
	stationHandler := api.NewStationHandler(stationManager, logger)
	logger.Info("Station API handler initialized")

	// WebSocket Handler for real-time message streaming
	wsHandler := api.NewWebSocketHandler(messageBroadcaster, logger)
	logger.Info("WebSocket handler initialized")

	// Station CRUD endpoints (auth protected)
	mux.Handle("/api/stations", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			stationHandler.ListStations(w, r)
		case http.MethodPost:
			// Admin only for create
			if user == nil || user.Role != auth.RoleAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.CreateStation(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Station detail endpoints (with path-based routing, auth protected)
	mux.Handle("/api/stations/", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		isAdmin := user != nil && user.Role == auth.RoleAdmin

		// Check if path ends with /start or /stop (admin only)
		if strings.HasSuffix(r.URL.Path, "/start") {
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.StartStation(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stop") {
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.StopStation(w, r)
			return
		}

		// Check if path ends with /connectors (viewer + admin)
		if strings.HasSuffix(r.URL.Path, "/connectors") {
			stationHandler.GetConnectors(w, r)
			return
		}

		// Check if path ends with /charge (admin only)
		if strings.HasSuffix(r.URL.Path, "/charge") {
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.StartCharging(w, r)
			return
		}

		// Check if path ends with /stop-charge (admin only)
		if strings.HasSuffix(r.URL.Path, "/stop-charge") {
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.StopCharging(w, r)
			return
		}

		// Check if path ends with /send-message (admin only)
		if strings.HasSuffix(r.URL.Path, "/send-message") {
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.SendCustomMessage(w, r)
			return
		}

		// Otherwise, handle CRUD operations on individual stations
		switch r.Method {
		case http.MethodGet:
			stationHandler.GetStation(w, r)
		case http.MethodPut:
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.UpdateStation(w, r)
		case http.MethodDelete:
			if !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}
			stationHandler.DeleteStation(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// WebSocket endpoints (auth protected)
	mux.Handle("/api/ws/messages", requireAuth(http.HandlerFunc(wsHandler.HandleMessages)))
	mux.Handle("/api/ws/stats", requireAuth(http.HandlerFunc(wsHandler.HandleBroadcasterStats)))
	logger.Info("WebSocket endpoints registered")

	// Analytics endpoints (auth protected, viewer + admin)
	analyticsHandler := api.NewAnalyticsHandler(mongoClient, logger)
	mux.Handle("/api/analytics/messages", requireAuth(http.HandlerFunc(analyticsHandler.GetMessageStats)))
	mux.Handle("/api/analytics/transactions", requireAuth(http.HandlerFunc(analyticsHandler.GetTransactionStats)))
	mux.Handle("/api/analytics/errors", requireAuth(http.HandlerFunc(analyticsHandler.GetErrorStats)))
	mux.Handle("/api/analytics/dashboard", requireAuth(http.HandlerFunc(analyticsHandler.GetDashboardStats)))
	logger.Info("Analytics endpoints registered")

	// Initialize Scenario Runner
	scenarioStorage, err := scenario.NewStorage(mongoClient.GetDatabase(), logger)
	if err != nil {
		logger.Error("Failed to initialize scenario storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	stationController := scenario.NewStationManagerController(stationManager)
	scenarioRunner := scenario.NewRunner(
		scenarioStorage,
		stationController,
		messageLogger,
		messageBroadcaster,
		logger,
	)
	logger.Info("Scenario runner initialized")

	// Load builtin scenarios
	scenarioLoader := scenario.NewLoader(scenarioStorage, logger)
	if err := scenarioLoader.LoadBuiltinScenarios(ctx, "testdata/scenarios"); err != nil {
		logger.Warn("Failed to load builtin scenarios", slog.String("error", err.Error()))
	}

	// Scenario API Handler
	scenarioHandler := api.NewScenarioHandler(scenarioRunner, scenarioStorage, logger)

	// Scenario endpoints (auth protected with role checks)
	mux.Handle("/api/scenarios", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		isAdmin := user != nil && user.Role == auth.RoleAdmin
		// POST requires admin
		if r.Method == http.MethodPost && !isAdmin {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		scenarioHandler.HandleScenarios(w, r)
	})))
	mux.Handle("/api/scenarios/", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		isAdmin := user != nil && user.Role == auth.RoleAdmin
		// PUT, DELETE, and execute (POST to /execute) require admin
		if (r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPost) && !isAdmin {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		scenarioHandler.HandleScenario(w, r)
	})))
	mux.Handle("/api/executions", requireAuth(http.HandlerFunc(scenarioHandler.HandleExecutions)))
	mux.Handle("/api/executions/", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		isAdmin := user != nil && user.Role == auth.RoleAdmin
		// POST (pause/resume/stop) and DELETE require admin
		if (r.Method == http.MethodPost || r.Method == http.MethodDelete) && !isAdmin {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		scenarioHandler.HandleExecutions(w, r)
	})))
	logger.Info("Scenario endpoints registered")

	// Initialize Change Stream Watcher for real-time updates
	changeStreamWatcher := storage.NewChangeStreamWatcher(mongoClient, logger)

	// Register handlers for station and transaction changes
	changeStreamWatcher.WatchStations(func(event storage.ChangeEvent) {
		// Broadcast station changes to connected clients
		messageBroadcaster.BroadcastChange("station", event)
	})

	changeStreamWatcher.WatchTransactions(func(event storage.ChangeEvent) {
		// Broadcast transaction changes to connected clients
		messageBroadcaster.BroadcastChange("transaction", event)
	})

	// Start change stream watcher
	if err := changeStreamWatcher.Start(); err != nil {
		logger.Warn("Failed to start change stream watcher (may require MongoDB replica set)",
			"error", err,
		)
	} else {
		logger.Info("Change stream watcher started")
	}

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

	// Shutdown scenario runner
	if err := scenarioRunner.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown scenario runner", slog.String("error", err.Error()))
	} else {
		logger.Info("Scenario runner shutdown complete")
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

	// Shutdown change stream watcher
	if err := changeStreamWatcher.Stop(); err != nil {
		logger.Error("Failed to shutdown change stream watcher", slog.String("error", err.Error()))
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
func initLogger(cfg *config.Config) *slog.Logger {
	var logger *slog.Logger
	var logFile *os.File
	var err error

	if cfg.Logging.Output != "stdout" {
		logFile, err = os.OpenFile(cfg.Logging.Output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("error opening log file: ", err)
		}
		log.Printf("env: %s; log file: %s", cfg.Logging.Level, cfg.Logging.Output)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	if cfg.Logging.Level == "info" {
		opts.Level = slog.LevelInfo
	}

	if cfg.Logging.Output == "stdout" {
		logger = slog.New(
			slog.NewTextHandler(os.Stdout, opts),
		)
	} else {
		logger = slog.New(
			slog.NewTextHandler(logFile, opts),
		)
	}

	return logger
}

// convertAuthConfig converts config.AuthConfig to auth.Config
func convertAuthConfig(cfg *config.AuthConfig) *auth.Config {
	authCfg := &auth.Config{
		Enabled:   cfg.Enabled,
		JWTSecret: cfg.JWTSecret,
		JWTExpiry: cfg.JWTExpiry,
		Users:     make([]auth.User, 0, len(cfg.Users)),
		APIKeys:   make([]auth.APIKey, 0, len(cfg.APIKeys)),
	}

	for _, u := range cfg.Users {
		authCfg.Users = append(authCfg.Users, auth.User{
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			Role:         auth.Role(u.Role),
			Enabled:      u.Enabled,
		})
	}

	for _, k := range cfg.APIKeys {
		apiKey := auth.APIKey{
			Name:    k.Name,
			KeyHash: k.KeyHash,
			Role:    auth.Role(k.Role),
			Enabled: k.Enabled,
		}
		if k.ExpiresAt != "" {
			if t, err := time.Parse(time.RFC3339, k.ExpiresAt); err == nil {
				apiKey.ExpiresAt = &t
			}
		}
		authCfg.APIKeys = append(authCfg.APIKeys, apiKey)
	}

	return authCfg
}

// Note: Configuration structs and loading logic have been moved to internal/config package
