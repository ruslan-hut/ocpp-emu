package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ruslanhut/ocpp-emu/internal/scenario"
)

// ScenarioHandler handles scenario-related API requests
type ScenarioHandler struct {
	runner  *scenario.Runner
	storage *scenario.Storage
	logger  *slog.Logger
}

// NewScenarioHandler creates a new scenario handler
func NewScenarioHandler(runner *scenario.Runner, storage *scenario.Storage, logger *slog.Logger) *ScenarioHandler {
	return &ScenarioHandler{
		runner:  runner,
		storage: storage,
		logger:  logger,
	}
}

// ScenarioResponse represents the API response for a scenario
type ScenarioResponse struct {
	ID          string          `json:"id"`
	ScenarioID  string          `json:"scenarioId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	StationID   string          `json:"stationId,omitempty"`
	Steps       []scenario.Step `json:"steps"`
	Tags        []string        `json:"tags,omitempty"`
	Version     string          `json:"version,omitempty"`
	IsBuiltin   bool            `json:"isBuiltin"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

// CreateScenarioRequest represents the request to create a scenario
type CreateScenarioRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	StationID   string          `json:"stationId,omitempty"`
	Steps       []scenario.Step `json:"steps"`
	Tags        []string        `json:"tags,omitempty"`
}

// ExecuteScenarioRequest represents the request to execute a scenario
type ExecuteScenarioRequest struct {
	StationID string `json:"stationId,omitempty"`
}

// HandleScenarios handles GET /api/scenarios and POST /api/scenarios
func (h *ScenarioHandler) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listScenarios(w, r)
	case http.MethodPost:
		h.createScenario(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleScenario handles GET/PUT/DELETE /api/scenarios/{id}
func (h *ScenarioHandler) HandleScenario(w http.ResponseWriter, r *http.Request) {
	// Extract scenario ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/scenarios/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Scenario ID required", http.StatusBadRequest)
		return
	}

	scenarioID := parts[0]

	// Check for sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "execute":
			h.executeScenario(w, r, scenarioID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getScenario(w, r, scenarioID)
	case http.MethodPut:
		h.updateScenario(w, r, scenarioID)
	case http.MethodDelete:
		h.deleteScenario(w, r, scenarioID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleExecutions handles GET /api/executions and GET /api/executions/{id}
func (h *ScenarioHandler) HandleExecutions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/executions")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// List all executions
		h.listExecutions(w, r)
		return
	}

	parts := strings.Split(path, "/")
	executionID := parts[0]

	// Check for sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "pause":
			h.pauseExecution(w, r, executionID)
			return
		case "resume":
			h.resumeExecution(w, r, executionID)
			return
		case "stop":
			h.stopExecution(w, r, executionID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getExecution(w, r, executionID)
	case http.MethodDelete:
		h.deleteExecution(w, r, executionID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listScenarios returns all scenarios
func (h *ScenarioHandler) listScenarios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filter parameters
	filter := &scenario.ScenarioFilter{
		Tag:         r.URL.Query().Get("tag"),
		StationID:   r.URL.Query().Get("stationId"),
		BuiltinOnly: r.URL.Query().Get("builtin") == "true",
		CustomOnly:  r.URL.Query().Get("custom") == "true",
	}

	scenarios, err := h.storage.ListScenarios(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to list scenarios", "error", err)
		http.Error(w, "Failed to list scenarios", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := make([]ScenarioResponse, len(scenarios))
	for i, s := range scenarios {
		response[i] = h.scenarioToResponse(s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createScenario creates a new scenario
func (h *ScenarioHandler) createScenario(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if len(req.Steps) == 0 {
		http.Error(w, "At least one step is required", http.StatusBadRequest)
		return
	}

	s := scenario.NewScenario(req.Name, req.Description)
	s.StationID = req.StationID
	s.Steps = req.Steps
	s.Tags = req.Tags

	if err := h.storage.CreateScenario(ctx, s); err != nil {
		h.logger.Error("Failed to create scenario", "error", err)
		http.Error(w, "Failed to create scenario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(h.scenarioToResponse(s))
}

// getScenario returns a single scenario
func (h *ScenarioHandler) getScenario(w http.ResponseWriter, r *http.Request, scenarioID string) {
	ctx := r.Context()

	s, err := h.storage.GetScenario(ctx, scenarioID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get scenario", "error", err)
		http.Error(w, "Failed to get scenario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.scenarioToResponse(s))
}

// updateScenario updates an existing scenario
func (h *ScenarioHandler) updateScenario(w http.ResponseWriter, r *http.Request, scenarioID string) {
	ctx := r.Context()

	// Get existing scenario
	existing, err := h.storage.GetScenario(ctx, scenarioID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get scenario", "error", err)
		http.Error(w, "Failed to get scenario", http.StatusInternalServerError)
		return
	}

	if existing.IsBuiltin {
		http.Error(w, "Cannot modify built-in scenarios", http.StatusForbidden)
		return
	}

	var req CreateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields
	existing.Name = req.Name
	existing.Description = req.Description
	existing.StationID = req.StationID
	existing.Steps = req.Steps
	existing.Tags = req.Tags

	if err := h.storage.UpdateScenario(ctx, existing); err != nil {
		h.logger.Error("Failed to update scenario", "error", err)
		http.Error(w, "Failed to update scenario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.scenarioToResponse(existing))
}

// deleteScenario deletes a scenario
func (h *ScenarioHandler) deleteScenario(w http.ResponseWriter, r *http.Request, scenarioID string) {
	ctx := r.Context()

	// Check if builtin
	existing, err := h.storage.GetScenario(ctx, scenarioID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get scenario", "error", err)
		http.Error(w, "Failed to get scenario", http.StatusInternalServerError)
		return
	}

	if existing.IsBuiltin {
		http.Error(w, "Cannot delete built-in scenarios", http.StatusForbidden)
		return
	}

	if err := h.storage.DeleteScenario(ctx, scenarioID); err != nil {
		h.logger.Error("Failed to delete scenario", "error", err)
		http.Error(w, "Failed to delete scenario", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// executeScenario starts executing a scenario
func (h *ScenarioHandler) executeScenario(w http.ResponseWriter, r *http.Request, scenarioID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var req ExecuteScenarioRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	execution, err := h.runner.StartScenario(ctx, scenarioID, req.StationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to start scenario", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(execution)
}

// listExecutions returns all executions
func (h *ScenarioHandler) listExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := &scenario.ExecutionFilter{
		ScenarioID: r.URL.Query().Get("scenarioId"),
		StationID:  r.URL.Query().Get("stationId"),
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = scenario.ExecutionStatus(status)
	}

	executions, err := h.storage.ListExecutions(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to list executions", "error", err)
		http.Error(w, "Failed to list executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

// getExecution returns a single execution
func (h *ScenarioHandler) getExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	execution, err := h.runner.GetExecution(executionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get execution", "error", err)
		http.Error(w, "Failed to get execution", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// pauseExecution pauses a running execution
func (h *ScenarioHandler) pauseExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.runner.PauseExecution(executionID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to pause execution", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	execution, _ := h.runner.GetExecution(executionID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// resumeExecution resumes a paused execution
func (h *ScenarioHandler) resumeExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.runner.ResumeExecution(executionID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to resume execution", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	execution, _ := h.runner.GetExecution(executionID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// stopExecution stops a running or paused execution
func (h *ScenarioHandler) stopExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.runner.StopExecution(executionID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to stop execution", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteExecution deletes an execution record
func (h *ScenarioHandler) deleteExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	ctx := r.Context()

	if err := h.storage.DeleteExecution(ctx, executionID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to delete execution", "error", err)
		http.Error(w, "Failed to delete execution", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// scenarioToResponse converts a scenario to API response format
func (h *ScenarioHandler) scenarioToResponse(s *scenario.Scenario) ScenarioResponse {
	return ScenarioResponse{
		ID:          s.ID.Hex(),
		ScenarioID:  s.ScenarioID,
		Name:        s.Name,
		Description: s.Description,
		StationID:   s.StationID,
		Steps:       s.Steps,
		Tags:        s.Tags,
		Version:     s.Version,
		IsBuiltin:   s.IsBuiltin,
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
