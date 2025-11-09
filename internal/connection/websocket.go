package connection

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient represents a WebSocket client connection to a CSMS
type WebSocketClient struct {
	config ConnectionConfig
	logger *slog.Logger

	// Connection state
	conn           *websocket.Conn
	state          ConnectionState
	stateMu        sync.RWMutex
	reconnectCount int
	connectedAt    *time.Time
	disconnectedAt *time.Time
	lastMessageAt  *time.Time

	// Statistics
	messagesSent     int64
	messagesReceived int64
	bytesSent        int64
	bytesReceived    int64
	statsMu          sync.RWMutex

	// Control channels
	ctx       context.Context
	cancel    context.CancelFunc
	sendQueue chan Message
	closeChan chan struct{}
	closeOnce sync.Once

	// Error tracking
	lastError   string
	lastErrorMu sync.RWMutex
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(config ConnectionConfig, logger *slog.Logger) *WebSocketClient {
	if logger == nil {
		logger = slog.Default()
	}

	// Set defaults
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.PingInterval == 0 {
		config.PingInterval = 30 * time.Second
	}
	if config.PongTimeout == 0 {
		config.PongTimeout = 10 * time.Second
	}
	if config.MaxReconnectAttempts == 0 {
		config.MaxReconnectAttempts = 5
	}
	if config.ReconnectBackoff == 0 {
		config.ReconnectBackoff = 5 * time.Second
	}
	if config.ReconnectMaxBackoff == 0 {
		config.ReconnectMaxBackoff = 60 * time.Second
	}

	// Determine subprotocol from protocol version
	if config.Subprotocol == "" {
		config.Subprotocol = getSubprotocol(config.ProtocolVersion)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketClient{
		config:    config,
		logger:    logger,
		state:     StateDisconnected,
		ctx:       ctx,
		cancel:    cancel,
		sendQueue: make(chan Message, 100),
		closeChan: make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the CSMS
func (c *WebSocketClient) Connect() error {
	c.setState(StateConnecting)

	c.logger.Info("Connecting to CSMS",
		"station_id", c.config.StationID,
		"url", c.config.URL,
		"protocol", c.config.ProtocolVersion,
		"subprotocol", c.config.Subprotocol,
	)

	// Create HTTP headers
	headers := http.Header{}
	if c.config.BasicAuthUsername != "" {
		headers.Set("Authorization", basicAuth(c.config.BasicAuthUsername, c.config.BasicAuthPassword))
	} else if c.config.BearerToken != "" {
		headers.Set("Authorization", "Bearer "+c.config.BearerToken)
	}

	// Create WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectionTimeout,
		Subprotocols:     []string{c.config.Subprotocol},
	}

	// Configure TLS if enabled
	if c.config.TLSEnabled {
		tlsConfig, err := c.createTLSConfig()
		if err != nil {
			c.setError(fmt.Errorf("failed to create TLS config: %w", err))
			c.setState(StateError)
			return err
		}
		dialer.TLSClientConfig = tlsConfig
	}

	// Establish connection
	conn, resp, err := dialer.Dial(c.config.URL, headers)
	if err != nil {
		c.setError(fmt.Errorf("failed to dial: %w", err))
		c.setState(StateError)
		return err
	}
	defer resp.Body.Close()

	c.conn = conn
	now := time.Now()
	c.connectedAt = &now
	c.setState(StateConnected)
	c.reconnectCount = 0

	c.logger.Info("Connected to CSMS",
		"station_id", c.config.StationID,
		"subprotocol", conn.Subprotocol(),
	)

	// Trigger connected callback
	if c.config.OnConnected != nil {
		c.config.OnConnected()
	}

	// Start read/write goroutines
	go c.readPump()
	go c.writePump()
	go c.pingPump()

	return nil
}

// Disconnect closes the WebSocket connection
func (c *WebSocketClient) Disconnect() error {
	c.closeOnce.Do(func() {
		c.logger.Info("Disconnecting from CSMS", "station_id", c.config.StationID)

		c.cancel()
		close(c.closeChan)

		if c.conn != nil {
			// Send close message
			err := c.conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			if err != nil {
				c.logger.Warn("Failed to send close message", "error", err)
			}

			// Close connection
			if err := c.conn.Close(); err != nil {
				c.logger.Warn("Failed to close connection", "error", err)
			}
		}

		now := time.Now()
		c.disconnectedAt = &now
		c.setState(StateClosed)

		c.logger.Info("Disconnected from CSMS", "station_id", c.config.StationID)
	})

	return nil
}

// Send queues a message to be sent
func (c *WebSocketClient) Send(data []byte) error {
	if c.GetState() != StateConnected {
		return fmt.Errorf("connection not established")
	}

	select {
	case c.sendQueue <- Message{Type: TextMessage, Data: data}:
		return nil
	case <-c.ctx.Done():
		return fmt.Errorf("connection closed")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send queue full")
	}
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.handleDisconnect(fmt.Errorf("read pump stopped"))
	}()

	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))

	// Set pong handler
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Error("WebSocket read error", "error", err)
			}
			c.handleDisconnect(err)
			return
		}

		// Update statistics
		c.statsMu.Lock()
		c.messagesReceived++
		c.bytesReceived += int64(len(message))
		now := time.Now()
		c.lastMessageAt = &now
		c.statsMu.Unlock()

		// Handle message based on type
		switch messageType {
		case websocket.TextMessage:
			c.logger.Debug("Received message",
				"station_id", c.config.StationID,
				"size", len(message),
			)

			if c.config.OnMessage != nil {
				c.config.OnMessage(message)
			}

		case websocket.BinaryMessage:
			c.logger.Warn("Received unexpected binary message", "station_id", c.config.StationID)

		case websocket.CloseMessage:
			c.logger.Info("Received close message", "station_id", c.config.StationID)
			c.handleDisconnect(nil)
			return
		}

		// Reset read deadline
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	}
}

// writePump writes messages from the send queue to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case message, ok := <-c.sendQueue:
			if !ok {
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))

			err := c.conn.WriteMessage(int(message.Type), message.Data)
			if err != nil {
				c.logger.Error("Failed to write message", "error", err)
				c.handleDisconnect(err)
				return
			}

			// Update statistics
			c.statsMu.Lock()
			c.messagesSent++
			c.bytesSent += int64(len(message.Data))
			c.statsMu.Unlock()

			c.logger.Debug("Sent message",
				"station_id", c.config.StationID,
				"size", len(message.Data),
			)
		}
	}
}

// pingPump sends periodic ping messages
func (c *WebSocketClient) pingPump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Warn("Failed to send ping", "error", err)
				c.handleDisconnect(err)
				return
			}
		}
	}
}

// handleDisconnect handles connection disconnection and triggers reconnection if needed
func (c *WebSocketClient) handleDisconnect(err error) {
	c.stateMu.Lock()
	if c.state == StateClosed {
		c.stateMu.Unlock()
		return
	}
	c.stateMu.Unlock()

	now := time.Now()
	c.disconnectedAt = &now
	c.setState(StateDisconnected)

	if err != nil {
		c.setError(err)
		c.logger.Warn("Connection disconnected", "station_id", c.config.StationID, "error", err)
	} else {
		c.logger.Info("Connection disconnected", "station_id", c.config.StationID)
	}

	// Trigger disconnected callback
	if c.config.OnDisconnected != nil {
		c.config.OnDisconnected(err)
	}

	// Check if disconnection was intentional (context cancelled means explicit Disconnect() call)
	select {
	case <-c.ctx.Done():
		// Context cancelled - this was an intentional disconnect, don't reconnect
		c.logger.Info("Connection closed intentionally, not attempting reconnect", "station_id", c.config.StationID)
		c.setState(StateClosed)
		return
	default:
		// Context still active - this was an unexpected disconnect, attempt reconnection
	}

	// Attempt reconnection
	if c.reconnectCount < c.config.MaxReconnectAttempts {
		go c.reconnect()
	} else {
		c.logger.Error("Max reconnect attempts reached", "station_id", c.config.StationID)
		c.setState(StateError)
	}
}

// reconnect attempts to reconnect with exponential backoff
func (c *WebSocketClient) reconnect() {
	c.setState(StateReconnecting)
	c.reconnectCount++

	// Calculate backoff with exponential increase
	backoff := c.config.ReconnectBackoff * time.Duration(1<<uint(c.reconnectCount-1))
	if backoff > c.config.ReconnectMaxBackoff {
		backoff = c.config.ReconnectMaxBackoff
	}

	c.logger.Info("Attempting to reconnect",
		"station_id", c.config.StationID,
		"attempt", c.reconnectCount,
		"backoff", backoff,
	)

	time.Sleep(backoff)

	if err := c.Connect(); err != nil {
		c.logger.Error("Reconnection failed",
			"station_id", c.config.StationID,
			"error", err,
		)
	}
}

// GetState returns the current connection state
func (c *WebSocketClient) GetState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// setState sets the connection state
func (c *WebSocketClient) setState(state ConnectionState) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

// GetStats returns connection statistics
func (c *WebSocketClient) GetStats() ConnectionStats {
	c.statsMu.RLock()
	c.stateMu.RLock()
	c.lastErrorMu.RLock()

	stats := ConnectionStats{
		StationID:         c.config.StationID,
		State:             c.state,
		ConnectedAt:       c.connectedAt,
		DisconnectedAt:    c.disconnectedAt,
		LastMessageAt:     c.lastMessageAt,
		ReconnectAttempts: c.reconnectCount,
		MessagesSent:      c.messagesSent,
		MessagesReceived:  c.messagesReceived,
		BytesSent:         c.bytesSent,
		BytesReceived:     c.bytesReceived,
		LastError:         c.lastError,
	}

	c.lastErrorMu.RUnlock()
	c.stateMu.RUnlock()
	c.statsMu.RUnlock()

	return stats
}

// setError sets the last error
func (c *WebSocketClient) setError(err error) {
	c.lastErrorMu.Lock()
	if err != nil {
		c.lastError = err.Error()
		if c.config.OnError != nil {
			c.config.OnError(err)
		}
	}
	c.lastErrorMu.Unlock()
}

// createTLSConfig creates TLS configuration
func (c *WebSocketClient) createTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.TLSSkipVerify,
	}

	// Load CA certificate
	if c.config.TLSCACert != "" {
		caCert, err := os.ReadFile(c.config.TLSCACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate
	if c.config.TLSClientCert != "" && c.config.TLSClientKey != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSClientCert, c.config.TLSClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// getSubprotocol returns the OCPP subprotocol for the given version
func getSubprotocol(version string) string {
	switch version {
	case "1.6":
		return "ocpp1.6"
	case "2.0.1":
		return "ocpp2.0.1"
	case "2.1":
		return "ocpp2.1"
	default:
		return "ocpp1.6"
	}
}

// basicAuth creates a basic auth header value
func basicAuth(username, password string) string {
	return "Basic " + base64Encode(username+":"+password)
}

// base64Encode encodes a string to base64
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
