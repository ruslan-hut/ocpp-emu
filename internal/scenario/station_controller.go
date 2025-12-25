package scenario

import (
	"context"

	"github.com/ruslanhut/ocpp-emu/internal/station"
)

// StationManagerController wraps a station.Manager to implement the StationController interface.
type StationManagerController struct {
	manager *station.Manager
}

// NewStationManagerController creates a new station controller wrapping the station manager.
func NewStationManagerController(manager *station.Manager) *StationManagerController {
	return &StationManagerController{
		manager: manager,
	}
}

// StartStation starts a station.
func (c *StationManagerController) StartStation(ctx context.Context, stationID string) error {
	return c.manager.StartStation(ctx, stationID)
}

// StopStation stops a station.
func (c *StationManagerController) StopStation(ctx context.Context, stationID string) error {
	return c.manager.StopStation(ctx, stationID)
}

// StartCharging starts a charging session.
func (c *StationManagerController) StartCharging(ctx context.Context, stationID string, connectorID int, idTag string) error {
	return c.manager.StartCharging(ctx, stationID, connectorID, idTag)
}

// StopCharging stops a charging session.
func (c *StationManagerController) StopCharging(ctx context.Context, stationID string, connectorID int, reason string) error {
	return c.manager.StopCharging(ctx, stationID, connectorID, reason)
}

// SendCustomMessage sends a custom OCPP message.
func (c *StationManagerController) SendCustomMessage(ctx context.Context, stationID string, messageJSON []byte) error {
	return c.manager.SendCustomMessage(ctx, stationID, messageJSON)
}

// GetConnectors returns connector information for a station.
func (c *StationManagerController) GetConnectors(ctx context.Context, stationID string) ([]map[string]interface{}, error) {
	return c.manager.GetConnectors(ctx, stationID)
}

// IsStationConnected checks if a station is connected.
func (c *StationManagerController) IsStationConnected(stationID string) bool {
	s, err := c.manager.GetStation(stationID)
	if err != nil {
		return false
	}
	return s.RuntimeState.ConnectionStatus == "connected"
}
