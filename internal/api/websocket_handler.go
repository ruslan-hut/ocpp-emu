package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// TODO: Restrict origins in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections for real-time message streaming
type WebSocketHandler struct {
	broadcaster *MessageBroadcaster
	logger      *slog.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(broadcaster *MessageBroadcaster, logger *slog.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		broadcaster: broadcaster,
		logger:      logger,
	}
}

// HandleMessages handles WebSocket connections for message streaming
// Endpoint: /api/ws/messages
func (h *WebSocketHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	// Generate unique client ID
	clientID := uuid.New().String()

	// Parse query parameters for filtering
	query := r.URL.Query()
	filter := MessageFilterConfig{
		StationID:   query.Get("stationId"),
		Direction:   query.Get("direction"),
		MessageType: query.Get("messageType"),
	}

	h.logger.Info("New WebSocket connection",
		"clientId", clientID,
		"remoteAddr", r.RemoteAddr,
		"filter", filter,
	)

	// Register client with broadcaster
	client := h.broadcaster.RegisterClient(conn, clientID, filter)

	// Send welcome message
	h.sendWelcomeMessage(client)
}

// sendWelcomeMessage sends an initial welcome message to the client
func (h *WebSocketHandler) sendWelcomeMessage(client *Client) {
	welcomeMsg := map[string]interface{}{
		"type":      "welcome",
		"message":   "Connected to OCPP message stream",
		"clientId":  client.ID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Marshal and send
	data, err := marshalJSON(welcomeMsg)
	if err != nil {
		h.logger.Error("Failed to marshal welcome message", "error", err)
		return
	}

	select {
	case client.send <- data:
		// Welcome message sent
	default:
		h.logger.Warn("Failed to send welcome message", "clientId", client.ID)
	}
}

// HandleBroadcasterStats handles requests for broadcaster statistics
// Endpoint: /api/ws/stats
func (h *WebSocketHandler) HandleBroadcasterStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.broadcaster.GetStats()

	w.Header().Set("Content-Type", "application/json")
	data, err := marshalJSON(map[string]interface{}{
		"connectedClients":  stats.ConnectedClients,
		"totalMessages":     stats.TotalMessages,
		"broadcastMessages": stats.BroadcastMessages,
		"droppedMessages":   stats.DroppedMessages,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// marshalJSON is a helper function to marshal data to JSON
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
