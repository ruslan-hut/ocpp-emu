package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/storage"
)

// AnalyticsHandler handles analytics-related API requests
type AnalyticsHandler struct {
	db     *storage.MongoDBClient
	logger *slog.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(db *storage.MongoDBClient, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:     db,
		logger: logger,
	}
}

// GetMessageStats returns aggregated message statistics
func (h *AnalyticsHandler) GetMessageStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	stationID := query.Get("stationId")

	// Parse 'since' parameter (e.g., 24h, 7d, 30d)
	var since time.Time
	if sinceStr := query.Get("since"); sinceStr != "" {
		since = parseDuration(sinceStr)
	}

	stats, err := h.db.GetMessageStats(r.Context(), stationID, since)
	if err != nil {
		h.logger.Error("Failed to get message stats", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get message stats: %v", err), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetTransactionStats returns aggregated transaction statistics
func (h *AnalyticsHandler) GetTransactionStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	stationID := query.Get("stationId")

	// Parse 'since' parameter
	var since time.Time
	if sinceStr := query.Get("since"); sinceStr != "" {
		since = parseDuration(sinceStr)
	}

	stats, err := h.db.GetTransactionStats(r.Context(), stationID, since)
	if err != nil {
		h.logger.Error("Failed to get transaction stats", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get transaction stats: %v", err), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetErrorStats returns error statistics
func (h *AnalyticsHandler) GetErrorStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	stationID := query.Get("stationId")

	// Parse 'since' parameter
	var since time.Time
	if sinceStr := query.Get("since"); sinceStr != "" {
		since = parseDuration(sinceStr)
	}

	stats, err := h.db.GetErrorStats(r.Context(), stationID, since)
	if err != nil {
		h.logger.Error("Failed to get error stats", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get error stats: %v", err), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetDashboardStats returns combined statistics for the dashboard
func (h *AnalyticsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get stats for last 24 hours by default
	since := time.Now().Add(-24 * time.Hour)

	// Get all stats in parallel
	type result struct {
		messages     *storage.MessageStats
		transactions *storage.TransactionStats
		errors       *storage.ErrorStats
		err          error
	}

	ch := make(chan result, 3)

	// Fetch message stats
	go func() {
		stats, err := h.db.GetMessageStats(ctx, "", since)
		ch <- result{messages: stats, err: err}
	}()

	// Fetch transaction stats
	go func() {
		stats, err := h.db.GetTransactionStats(ctx, "", since)
		ch <- result{transactions: stats, err: err}
	}()

	// Fetch error stats
	go func() {
		stats, err := h.db.GetErrorStats(ctx, "", since)
		ch <- result{errors: stats, err: err}
	}()

	// Collect results
	var messageStats *storage.MessageStats
	var transactionStats *storage.TransactionStats
	var errorStats *storage.ErrorStats

	for i := 0; i < 3; i++ {
		res := <-ch
		if res.err != nil {
			h.logger.Warn("Failed to get stats", "error", res.err)
		}
		if res.messages != nil {
			messageStats = res.messages
		}
		if res.transactions != nil {
			transactionStats = res.transactions
		}
		if res.errors != nil {
			errorStats = res.errors
		}
	}

	response := map[string]interface{}{
		"period":       "24h",
		"timestamp":    time.Now(),
		"messages":     messageStats,
		"transactions": transactionStats,
		"errors":       errorStats,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// parseDuration parses a duration string like "24h", "7d", "30d" into a time.Time
func parseDuration(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	var duration time.Duration

	switch s {
	case "1h":
		duration = time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "12h":
		duration = 12 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "90d":
		duration = 90 * 24 * time.Hour
	default:
		// Try to parse as Go duration
		if d, err := time.ParseDuration(s); err == nil {
			duration = d
		} else {
			return time.Time{}
		}
	}

	return time.Now().Add(-duration)
}
