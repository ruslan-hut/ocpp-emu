package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ruslanhut/ocpp-emu/internal/station"
)

// StationHandler handles station-related API requests
type StationHandler struct {
	manager *station.Manager
	logger  *slog.Logger
}

// NewStationHandler creates a new station handler
func NewStationHandler(manager *station.Manager, logger *slog.Logger) *StationHandler {
	return &StationHandler{
		manager: manager,
		logger:  logger,
	}
}

// StationResponse represents the API response for a station
type StationResponse struct {
	ID                string                    `json:"id"`
	StationID         string                    `json:"stationId"`
	Name              string                    `json:"name"`
	Enabled           bool                      `json:"enabled"`
	AutoStart         bool                      `json:"autoStart"`
	ProtocolVersion   string                    `json:"protocolVersion"`
	Vendor            string                    `json:"vendor"`
	Model             string                    `json:"model"`
	SerialNumber      string                    `json:"serialNumber"`
	FirmwareVersion   string                    `json:"firmwareVersion"`
	ICCID             string                    `json:"iccid,omitempty"`
	IMSI              string                    `json:"imsi,omitempty"`
	Connectors        []ConnectorResponse       `json:"connectors"`
	SupportedProfiles []string                  `json:"supportedProfiles"`
	MeterValuesConfig MeterValuesConfigResponse `json:"meterValuesConfig"`
	CSMSURL           string                    `json:"csmsUrl"`
	CSMSAuth          *CSMSAuthResponse         `json:"csmsAuth,omitempty"`
	Simulation        SimulationConfigResponse  `json:"simulation"`
	RuntimeState      *RuntimeStateResponse     `json:"runtimeState,omitempty"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
	Tags              []string                  `json:"tags,omitempty"`
}

// ConnectorResponse represents a connector in API response
type ConnectorResponse struct {
	ID                   int    `json:"id"`
	Type                 string `json:"type"`
	MaxPower             int    `json:"maxPower"`
	Status               string `json:"status"`
	CurrentTransactionID *int   `json:"currentTransactionId,omitempty"`
}

// MeterValuesConfigResponse represents meter values config in API response
type MeterValuesConfigResponse struct {
	Interval            int      `json:"interval"`
	Measurands          []string `json:"measurands"`
	AlignedDataInterval int      `json:"alignedDataInterval"`
}

// CSMSAuthResponse represents CSMS auth config in API response
type CSMSAuthResponse struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// SimulationConfigResponse represents simulation config in API response
type SimulationConfigResponse struct {
	BootDelay                  int     `json:"bootDelay"`
	HeartbeatInterval          int     `json:"heartbeatInterval"`
	StatusNotificationOnChange bool    `json:"statusNotificationOnChange"`
	DefaultIDTag               string  `json:"defaultIdTag"`
	EnergyDeliveryRate         int     `json:"energyDeliveryRate"`
	RandomizeMeterValues       bool    `json:"randomizeMeterValues"`
	MeterValueVariance         float64 `json:"meterValueVariance"`
}

// RuntimeStateResponse represents runtime state in API response
type RuntimeStateResponse struct {
	State            string     `json:"state"`
	ConnectionStatus string     `json:"connectionStatus"`
	LastHeartbeat    *time.Time `json:"lastHeartbeat,omitempty"`
	LastError        string     `json:"lastError,omitempty"`
	ConnectedAt      *time.Time `json:"connectedAt,omitempty"`
	TransactionID    *int       `json:"transactionId,omitempty"`
}

// CreateStationRequest represents the request to create a new station
type CreateStationRequest struct {
	StationID         string                   `json:"stationId"`
	Name              string                   `json:"name"`
	Enabled           bool                     `json:"enabled"`
	AutoStart         bool                     `json:"autoStart"`
	ProtocolVersion   string                   `json:"protocolVersion"`
	Vendor            string                   `json:"vendor"`
	Model             string                   `json:"model"`
	SerialNumber      string                   `json:"serialNumber"`
	FirmwareVersion   string                   `json:"firmwareVersion"`
	ICCID             string                   `json:"iccid,omitempty"`
	IMSI              string                   `json:"imsi,omitempty"`
	Connectors        []ConnectorRequest       `json:"connectors"`
	SupportedProfiles []string                 `json:"supportedProfiles"`
	MeterValuesConfig MeterValuesConfigRequest `json:"meterValuesConfig"`
	CSMSURL           string                   `json:"csmsUrl"`
	CSMSAuth          *CSMSAuthRequest         `json:"csmsAuth,omitempty"`
	Simulation        SimulationConfigRequest  `json:"simulation"`
	Tags              []string                 `json:"tags,omitempty"`
}

// ConnectorRequest represents a connector in create/update request
type ConnectorRequest struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	MaxPower int    `json:"maxPower"`
}

// MeterValuesConfigRequest represents meter values config in request
type MeterValuesConfigRequest struct {
	Interval            int      `json:"interval"`
	Measurands          []string `json:"measurands"`
	AlignedDataInterval int      `json:"alignedDataInterval"`
}

// CSMSAuthRequest represents CSMS auth config in request
type CSMSAuthRequest struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// SimulationConfigRequest represents simulation config in request
type SimulationConfigRequest struct {
	BootDelay                  int     `json:"bootDelay"`
	HeartbeatInterval          int     `json:"heartbeatInterval"`
	StatusNotificationOnChange bool    `json:"statusNotificationOnChange"`
	DefaultIDTag               string  `json:"defaultIdTag"`
	EnergyDeliveryRate         int     `json:"energyDeliveryRate"`
	RandomizeMeterValues       bool    `json:"randomizeMeterValues"`
	MeterValueVariance         float64 `json:"meterValueVariance"`
}

// ListStations handles GET /api/stations
func (h *StationHandler) ListStations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stations := h.manager.GetAllStations()
	response := make([]StationResponse, 0, len(stations))

	for _, st := range stations {
		response = append(response, h.convertToResponse(st))
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"stations": response,
		"count":    len(response),
	})
}

// GetStation handles GET /api/stations/:id
func (h *StationHandler) GetStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stationID := h.extractStationID(r.URL.Path)
	if stationID == "" {
		h.sendError(w, http.StatusBadRequest, "Station ID is required")
		return
	}

	st, err := h.manager.GetStation(stationID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Station not found: %s", stationID))
		return
	}

	response := h.convertToResponse(st)
	h.sendJSON(w, http.StatusOK, response)
}

// CreateStation handles POST /api/stations
func (h *StationHandler) CreateStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if err := h.validateCreateRequest(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert request to station config
	config := h.convertToConfig(&req)
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	// Add station
	if err := h.manager.AddStation(r.Context(), config); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.sendError(w, http.StatusConflict, err.Error())
		} else {
			h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create station: %v", err))
		}
		return
	}

	// Get the created station
	st, err := h.manager.GetStation(config.StationID)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Station created but failed to retrieve")
		return
	}

	response := h.convertToResponse(st)
	h.sendJSON(w, http.StatusCreated, response)
}

// UpdateStation handles PUT /api/stations/:id
func (h *StationHandler) UpdateStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stationID := h.extractStationID(r.URL.Path)
	if stationID == "" {
		h.sendError(w, http.StatusBadRequest, "Station ID is required")
		return
	}

	var req CreateStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if err := h.validateCreateRequest(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Station ID in URL must match request body
	if req.StationID != stationID {
		h.sendError(w, http.StatusBadRequest, "Station ID in URL does not match request body")
		return
	}

	// Get existing station to preserve created_at
	existing, err := h.manager.GetStation(stationID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Station not found: %s", stationID))
		return
	}

	// Convert request to station config
	config := h.convertToConfig(&req)
	config.CreatedAt = existing.Config.CreatedAt
	config.UpdatedAt = time.Now()

	// Update station
	if err := h.manager.UpdateStation(r.Context(), stationID, config); err != nil {
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update station: %v", err))
		return
	}

	// Get the updated station
	st, err := h.manager.GetStation(stationID)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Station updated but failed to retrieve")
		return
	}

	response := h.convertToResponse(st)
	h.sendJSON(w, http.StatusOK, response)
}

// DeleteStation handles DELETE /api/stations/:id
func (h *StationHandler) DeleteStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stationID := h.extractStationID(r.URL.Path)
	if stationID == "" {
		h.sendError(w, http.StatusBadRequest, "Station ID is required")
		return
	}

	if err := h.manager.RemoveStation(r.Context(), stationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, fmt.Sprintf("Station not found: %s", stationID))
		} else {
			h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete station: %v", err))
		}
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Station deleted successfully",
		"stationId": stationID,
	})
}

// StartStation handles PATCH /api/stations/:id/start
func (h *StationHandler) StartStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stationID := h.extractStationIDFromAction(r.URL.Path, "/start")
	if stationID == "" {
		h.sendError(w, http.StatusBadRequest, "Station ID is required")
		return
	}

	if err := h.manager.StartStation(r.Context(), stationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, fmt.Sprintf("Station not found: %s", stationID))
		} else if strings.Contains(err.Error(), "already connected") {
			h.sendError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "disabled") {
			h.sendError(w, http.StatusBadRequest, err.Error())
		} else {
			h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start station: %v", err))
		}
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Station started successfully",
		"stationId": stationID,
	})
}

// StopStation handles PATCH /api/stations/:id/stop
func (h *StationHandler) StopStation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stationID := h.extractStationIDFromAction(r.URL.Path, "/stop")
	if stationID == "" {
		h.sendError(w, http.StatusBadRequest, "Station ID is required")
		return
	}

	if err := h.manager.StopStation(r.Context(), stationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, fmt.Sprintf("Station not found: %s", stationID))
		} else {
			h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stop station: %v", err))
		}
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Station stopped successfully",
		"stationId": stationID,
	})
}

// Helper functions

func (h *StationHandler) extractStationID(path string) string {
	// Path format: /api/stations/:id
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (h *StationHandler) extractStationIDFromAction(path, action string) string {
	// Path format: /api/stations/:id/start or /api/stations/:id/stop
	path = strings.TrimSuffix(path, action)
	return h.extractStationID(path)
}

func (h *StationHandler) convertToResponse(st *station.Station) StationResponse {
	// Get thread-safe copy of station data
	config, runtimeState := st.GetData()

	connectors := make([]ConnectorResponse, len(config.Connectors))
	for i, c := range config.Connectors {
		connectors[i] = ConnectorResponse{
			ID:                   c.ID,
			Type:                 c.Type,
			MaxPower:             c.MaxPower,
			Status:               c.Status,
			CurrentTransactionID: c.CurrentTransactionID,
		}
	}

	var csmsAuth *CSMSAuthResponse
	if config.CSMSAuth != nil {
		csmsAuth = &CSMSAuthResponse{
			Type:     config.CSMSAuth.Type,
			Username: config.CSMSAuth.Username,
			Password: config.CSMSAuth.Password,
			Token:    config.CSMSAuth.Token,
		}
	}

	return StationResponse{
		ID:                config.ID,
		StationID:         config.StationID,
		Name:              config.Name,
		Enabled:           config.Enabled,
		AutoStart:         config.AutoStart,
		ProtocolVersion:   config.ProtocolVersion,
		Vendor:            config.Vendor,
		Model:             config.Model,
		SerialNumber:      config.SerialNumber,
		FirmwareVersion:   config.FirmwareVersion,
		ICCID:             config.ICCID,
		IMSI:              config.IMSI,
		Connectors:        connectors,
		SupportedProfiles: config.SupportedProfiles,
		MeterValuesConfig: MeterValuesConfigResponse{
			Interval:            config.MeterValuesConfig.Interval,
			Measurands:          config.MeterValuesConfig.Measurands,
			AlignedDataInterval: config.MeterValuesConfig.AlignedDataInterval,
		},
		CSMSURL:  config.CSMSURL,
		CSMSAuth: csmsAuth,
		Simulation: SimulationConfigResponse{
			BootDelay:                  config.Simulation.BootDelay,
			HeartbeatInterval:          config.Simulation.HeartbeatInterval,
			StatusNotificationOnChange: config.Simulation.StatusNotificationOnChange,
			DefaultIDTag:               config.Simulation.DefaultIDTag,
			EnergyDeliveryRate:         config.Simulation.EnergyDeliveryRate,
			RandomizeMeterValues:       config.Simulation.RandomizeMeterValues,
			MeterValueVariance:         config.Simulation.MeterValueVariance,
		},
		RuntimeState: &RuntimeStateResponse{
			State:            string(runtimeState.State),
			ConnectionStatus: runtimeState.ConnectionStatus,
			LastHeartbeat:    runtimeState.LastHeartbeat,
			LastError:        runtimeState.LastError,
			ConnectedAt:      runtimeState.ConnectedAt,
			TransactionID:    runtimeState.TransactionID,
		},
		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
		Tags:      config.Tags,
	}
}

func (h *StationHandler) convertToConfig(req *CreateStationRequest) station.Config {
	connectors := make([]station.ConnectorConfig, len(req.Connectors))
	for i, c := range req.Connectors {
		connectors[i] = station.ConnectorConfig{
			ID:       c.ID,
			Type:     c.Type,
			MaxPower: c.MaxPower,
			Status:   "Available", // Default status for new connectors
		}
	}

	var csmsAuth *station.CSMSAuthConfig
	if req.CSMSAuth != nil {
		csmsAuth = &station.CSMSAuthConfig{
			Type:     req.CSMSAuth.Type,
			Username: req.CSMSAuth.Username,
			Password: req.CSMSAuth.Password,
			Token:    req.CSMSAuth.Token,
		}
	}

	return station.Config{
		StationID:         req.StationID,
		Name:              req.Name,
		Enabled:           req.Enabled,
		AutoStart:         req.AutoStart,
		ProtocolVersion:   req.ProtocolVersion,
		Vendor:            req.Vendor,
		Model:             req.Model,
		SerialNumber:      req.SerialNumber,
		FirmwareVersion:   req.FirmwareVersion,
		ICCID:             req.ICCID,
		IMSI:              req.IMSI,
		Connectors:        connectors,
		SupportedProfiles: req.SupportedProfiles,
		MeterValuesConfig: station.MeterValuesConfig{
			Interval:            req.MeterValuesConfig.Interval,
			Measurands:          req.MeterValuesConfig.Measurands,
			AlignedDataInterval: req.MeterValuesConfig.AlignedDataInterval,
		},
		CSMSURL:  req.CSMSURL,
		CSMSAuth: csmsAuth,
		Simulation: station.SimulationConfig{
			BootDelay:                  req.Simulation.BootDelay,
			HeartbeatInterval:          req.Simulation.HeartbeatInterval,
			StatusNotificationOnChange: req.Simulation.StatusNotificationOnChange,
			DefaultIDTag:               req.Simulation.DefaultIDTag,
			EnergyDeliveryRate:         req.Simulation.EnergyDeliveryRate,
			RandomizeMeterValues:       req.Simulation.RandomizeMeterValues,
			MeterValueVariance:         req.Simulation.MeterValueVariance,
		},
		Tags: req.Tags,
	}
}

func (h *StationHandler) validateCreateRequest(req *CreateStationRequest) error {
	if req.StationID == "" {
		return fmt.Errorf("stationId is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.ProtocolVersion == "" {
		return fmt.Errorf("protocolVersion is required")
	}
	if req.Vendor == "" {
		return fmt.Errorf("vendor is required")
	}
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if req.CSMSURL == "" {
		return fmt.Errorf("csmsUrl is required")
	}
	if len(req.Connectors) == 0 {
		return fmt.Errorf("at least one connector is required")
	}

	// Validate connectors
	for i, c := range req.Connectors {
		if c.ID <= 0 {
			return fmt.Errorf("connector %d: id must be positive", i)
		}
		if c.Type == "" {
			return fmt.Errorf("connector %d: type is required", i)
		}
		if c.MaxPower <= 0 {
			return fmt.Errorf("connector %d: maxPower must be positive", i)
		}
	}

	return nil
}

func (h *StationHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *StationHandler) sendError(w http.ResponseWriter, status int, message string) {
	h.logger.Warn("API error", "status", status, "message", message)
	h.sendJSON(w, status, map[string]interface{}{
		"error":  message,
		"status": status,
	})
}
