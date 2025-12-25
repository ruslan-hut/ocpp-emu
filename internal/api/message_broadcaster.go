package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ruslanhut/ocpp-emu/internal/logging"
	"github.com/ruslanhut/ocpp-emu/internal/storage"
)

// MessageBroadcaster manages WebSocket connections and broadcasts messages to clients
type MessageBroadcaster struct {
	clients    map[*Client]bool
	clientsMu  sync.RWMutex
	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stats      BroadcasterStats
	statsMu    sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	ID          string
	conn        *websocket.Conn
	send        chan []byte
	filter      MessageFilterConfig
	broadcaster *MessageBroadcaster
}

// MessageFilterConfig contains client-specific message filtering
type MessageFilterConfig struct {
	StationID   string
	Direction   string
	MessageType string
}

// BroadcastMessage represents a message to be broadcast to clients
type BroadcastMessage struct {
	Type      string               `json:"type"`
	Timestamp time.Time            `json:"timestamp"`
	Message   logging.MessageEntry `json:"message"`
}

// BroadcasterStats tracks broadcaster statistics
type BroadcasterStats struct {
	ConnectedClients  int
	TotalMessages     int64
	BroadcastMessages int64
	DroppedMessages   int64
}

// NewMessageBroadcaster creates a new message broadcaster
func NewMessageBroadcaster(logger *slog.Logger) *MessageBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	return &MessageBroadcaster{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan BroadcastMessage, 1000),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the broadcaster's main loop
func (mb *MessageBroadcaster) Start() {
	mb.logger.Info("Starting message broadcaster")

	mb.wg.Add(1)
	go mb.run()
}

// run is the main loop for the broadcaster
func (mb *MessageBroadcaster) run() {
	defer mb.wg.Done()

	for {
		select {
		case <-mb.ctx.Done():
			mb.logger.Info("Stopping message broadcaster")
			// Close all client connections
			mb.clientsMu.Lock()
			for client := range mb.clients {
				client.close()
			}
			mb.clientsMu.Unlock()
			return

		case client := <-mb.register:
			mb.clientsMu.Lock()
			mb.clients[client] = true
			mb.clientsMu.Unlock()
			mb.updateClientCount()
			mb.logger.Info("Client registered", "clientId", client.ID)

		case client := <-mb.unregister:
			mb.clientsMu.Lock()
			if _, ok := mb.clients[client]; ok {
				delete(mb.clients, client)
				close(client.send)
			}
			mb.clientsMu.Unlock()
			mb.updateClientCount()
			mb.logger.Info("Client unregistered", "clientId", client.ID)

		case message := <-mb.broadcast:
			mb.incrementTotalMessages()
			mb.broadcastToClients(message)
		}
	}
}

// broadcastToClients sends a message to all connected clients
func (mb *MessageBroadcaster) broadcastToClients(message BroadcastMessage) {
	// Marshal message once for all clients
	data, err := json.Marshal(message)
	if err != nil {
		mb.logger.Error("Failed to marshal broadcast message", "error", err)
		return
	}

	mb.clientsMu.RLock()
	defer mb.clientsMu.RUnlock()

	broadcastCount := 0
	for client := range mb.clients {
		// Apply client-specific filtering
		if !client.shouldReceive(message) {
			continue
		}

		select {
		case client.send <- data:
			broadcastCount++
		default:
			// Client's send channel is full, skip this message
			mb.incrementDropped()
			mb.logger.Warn("Client send buffer full, dropping message", "clientId", client.ID)
		}
	}

	if broadcastCount > 0 {
		mb.incrementBroadcast()
	}
}

// BroadcastMessageEntry broadcasts a message entry to all connected clients
func (mb *MessageBroadcaster) BroadcastMessageEntry(entry logging.MessageEntry) {
	msg := BroadcastMessage{
		Type:      "ocpp_message",
		Timestamp: time.Now(),
		Message:   entry,
	}

	select {
	case mb.broadcast <- msg:
		// Successfully queued for broadcast
	default:
		// Broadcast channel full, drop message
		mb.incrementDropped()
		mb.logger.Warn("Broadcast channel full, dropping message")
	}
}

// ChangeEventMessage represents a change event message for WebSocket broadcast
type ChangeEventMessage struct {
	Type          string                 `json:"type"`
	Category      string                 `json:"category"`  // "station", "transaction", etc.
	EventType     string                 `json:"eventType"` // "insert", "update", "delete"
	Collection    string                 `json:"collection"`
	DocumentID    string                 `json:"documentId,omitempty"`
	Document      map[string]interface{} `json:"document,omitempty"`
	UpdatedFields map[string]interface{} `json:"updatedFields,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ScenarioProgressMessage represents a scenario execution progress update
type ScenarioProgressMessage struct {
	Type     string      `json:"type"`
	Progress interface{} `json:"progress"`
}

// BroadcastScenarioProgress broadcasts scenario execution progress to all clients
func (mb *MessageBroadcaster) BroadcastScenarioProgress(progress interface{}) {
	msg := ScenarioProgressMessage{
		Type:     "scenario_progress",
		Progress: progress,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		mb.logger.Error("Failed to marshal scenario progress", "error", err)
		return
	}

	mb.clientsMu.RLock()
	defer mb.clientsMu.RUnlock()

	for client := range mb.clients {
		select {
		case client.send <- data:
			// Message sent
		default:
			// Client buffer full, skip
			mb.incrementDropped()
		}
	}
}

// BroadcastChange broadcasts a database change event to all connected clients
func (mb *MessageBroadcaster) BroadcastChange(category string, event interface{}) {
	var changeMsg ChangeEventMessage

	// Handle storage.ChangeEvent type
	if e, ok := event.(storage.ChangeEvent); ok {
		changeMsg = ChangeEventMessage{
			Type:          "change_event",
			Category:      category,
			EventType:     string(e.Type),
			Collection:    e.Collection,
			DocumentID:    e.DocumentID,
			Document:      e.FullDocument,
			UpdatedFields: e.UpdatedFields,
			Timestamp:     e.Timestamp,
		}
	} else {
		// Fallback for other types
		changeMsg = ChangeEventMessage{
			Type:      "change_event",
			Category:  category,
			Timestamp: time.Now(),
		}
	}

	if changeMsg.Timestamp.IsZero() {
		changeMsg.Timestamp = time.Now()
	}

	// Marshal and broadcast directly to all clients (no filtering for change events)
	data, err := json.Marshal(changeMsg)
	if err != nil {
		mb.logger.Error("Failed to marshal change event", "error", err)
		return
	}

	mb.clientsMu.RLock()
	defer mb.clientsMu.RUnlock()

	for client := range mb.clients {
		select {
		case client.send <- data:
			// Message sent
		default:
			// Client buffer full, skip
			mb.incrementDropped()
		}
	}
}

// RegisterClient adds a new client connection
func (mb *MessageBroadcaster) RegisterClient(conn *websocket.Conn, clientID string, filter MessageFilterConfig) *Client {
	client := &Client{
		ID:          clientID,
		conn:        conn,
		send:        make(chan []byte, 256),
		filter:      filter,
		broadcaster: mb,
	}

	mb.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	return client
}

// GetStats returns broadcaster statistics
func (mb *MessageBroadcaster) GetStats() BroadcasterStats {
	mb.statsMu.RLock()
	defer mb.statsMu.RUnlock()

	stats := mb.stats
	mb.clientsMu.RLock()
	stats.ConnectedClients = len(mb.clients)
	mb.clientsMu.RUnlock()

	return stats
}

// Shutdown gracefully shuts down the broadcaster
func (mb *MessageBroadcaster) Shutdown() error {
	mb.logger.Info("Shutting down message broadcaster")

	mb.cancel()
	mb.wg.Wait()

	close(mb.broadcast)
	close(mb.register)
	close(mb.unregister)

	return nil
}

// Statistics helpers
func (mb *MessageBroadcaster) updateClientCount() {
	mb.statsMu.Lock()
	defer mb.statsMu.Unlock()
	mb.clientsMu.RLock()
	mb.stats.ConnectedClients = len(mb.clients)
	mb.clientsMu.RUnlock()
}

func (mb *MessageBroadcaster) incrementTotalMessages() {
	mb.statsMu.Lock()
	defer mb.statsMu.Unlock()
	mb.stats.TotalMessages++
}

func (mb *MessageBroadcaster) incrementBroadcast() {
	mb.statsMu.Lock()
	defer mb.statsMu.Unlock()
	mb.stats.BroadcastMessages++
}

func (mb *MessageBroadcaster) incrementDropped() {
	mb.statsMu.Lock()
	defer mb.statsMu.Unlock()
	mb.stats.DroppedMessages++
}

// Client methods

// shouldReceive determines if a client should receive a message based on filters
func (c *Client) shouldReceive(msg BroadcastMessage) bool {
	// Apply station ID filter
	if c.filter.StationID != "" && msg.Message.StationID != c.filter.StationID {
		return false
	}

	// Apply direction filter
	if c.filter.Direction != "" && msg.Message.Direction != c.filter.Direction {
		return false
	}

	// Apply message type filter
	if c.filter.MessageType != "" && msg.Message.MessageType != c.filter.MessageType {
		return false
	}

	return true
}

// writePump pumps messages from the send channel to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.broadcaster.logger.Error("Failed to write message to client",
					"clientId", c.ID,
					"error", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.broadcaster.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.broadcaster.logger.Error("WebSocket read error",
					"clientId", c.ID,
					"error", err)
			}
			break
		}

		// Handle incoming messages from client (e.g., filter updates)
		c.handleClientMessage(message)
	}
}

// handleClientMessage processes messages received from the client
func (c *Client) handleClientMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		c.broadcaster.logger.Warn("Failed to parse client message",
			"clientId", c.ID,
			"error", err)
		return
	}

	// Handle filter updates
	if msgType, ok := msg["type"].(string); ok && msgType == "update_filter" {
		if stationID, ok := msg["stationId"].(string); ok {
			c.filter.StationID = stationID
			c.broadcaster.logger.Info("Updated client filter",
				"clientId", c.ID,
				"stationId", stationID)
		}
	}
}

// close closes the client connection
func (c *Client) close() {
	c.conn.Close()
}
