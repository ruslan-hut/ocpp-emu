package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON writes data as a JSON response with the given status code. Encoding
// failures are logged rather than surfaced, since the status line has already
// been written by the time encoding runs.
func writeJSON(w http.ResponseWriter, status int, data interface{}, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil && logger != nil {
		logger.Error("Failed to encode JSON response", "error", err)
	}
}

// writeError writes a JSON error response of the form {"error": ..., "status": ...}.
func writeError(w http.ResponseWriter, status int, message string, logger *slog.Logger) {
	if logger != nil {
		logger.Warn("API error", "status", status, "message", message)
	}
	writeJSON(w, status, map[string]interface{}{
		"error":  message,
		"status": status,
	}, logger)
}
