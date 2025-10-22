package websocket

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Client represents a WebSocket client connection
type Client struct {
	// Unique client identifier
	ID string

	// User ID (if authenticated)
	UserID string

	// WebSocket connection
	conn *websocket.Conn

	// Hub reference
	hub *Hub

	// Buffered channel of outbound messages
	send chan []byte

	// Context for cancellation
	ctx context.Context
	cancel context.CancelFunc

	// Last heartbeat timestamp
	lastHeartbeat time.Time
	heartbeatMu sync.RWMutex

	// Custom metadata
	metadata map[string]interface{}
	metadataMu sync.RWMutex

	// Connected timestamp
	connectedAt time.Time

	// Atomic flag to track if client is closed
	closed atomic.Bool
}

// NewClient creates a new Client instance
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	ctx, cancel := context.WithCancel(hub.ctx)

	return &Client{
		ID:            id,
		conn:          conn,
		hub:           hub,
		send:          make(chan []byte, 256),
		ctx:           ctx,
		cancel:        cancel,
		lastHeartbeat: time.Now(),
		metadata:      make(map[string]interface{}),
		connectedAt:   time.Now(),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.updateHeartbeat()
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for client %s: %v", c.ID, err)
				}
				return
			}

			c.updateHeartbeat()

			// Process message through hub
			if err := c.hub.HandleMessage(c.ctx, c, message); err != nil {
				log.Printf("Error handling message from client %s: %v", c.ID, err)
				c.SendError(err.Error())
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(message *Message) (err error) {
	// Protect against send on closed channel
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("client closed")
		}
	}()

	// Double-check closed status with short-circuit
	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	data, marshalErr := marshalMessage(message)
	if marshalErr != nil {
		return marshalErr
	}

	// Check again right before send to minimize race window
	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	select {
	case c.send <- data:
		return nil
	case <-c.ctx.Done():
		return context.Canceled
	default:
		log.Printf("Client %s send channel full, dropping message", c.ID)
		return fmt.Errorf("send channel full")
	}
}

// SendError sends an error message to the client
func (c *Client) SendError(errorMsg string) {
	// Ignore error if send fails - client may be closed
	_ = c.Send(&Message{
		Type: "error",
		Payload: map[string]string{
			"message": errorMsg,
		},
	})
}

// SendJSON sends a JSON message to the client
func (c *Client) SendJSON(messageType string, payload interface{}) error {
	return c.Send(&Message{
		Type:    messageType,
		Payload: payload,
	})
}

// JoinRoom adds the client to a room
func (c *Client) JoinRoom(room string) {
	c.hub.JoinRoom(c, room)
}

// LeaveRoom removes the client from a room
func (c *Client) LeaveRoom(room string) {
	c.hub.LeaveRoom(c, room)
}

// SetMetadata sets custom metadata for the client
func (c *Client) SetMetadata(key string, value interface{}) {
	c.metadataMu.Lock()
	defer c.metadataMu.Unlock()
	c.metadata[key] = value
}

// GetMetadata retrieves custom metadata for the client
func (c *Client) GetMetadata(key string) (interface{}, bool) {
	c.metadataMu.RLock()
	defer c.metadataMu.RUnlock()
	value, ok := c.metadata[key]
	return value, ok
}

// updateHeartbeat updates the last heartbeat timestamp
func (c *Client) updateHeartbeat() {
	c.heartbeatMu.Lock()
	defer c.heartbeatMu.Unlock()
	c.lastHeartbeat = time.Now()
}

// GetLastHeartbeat returns the last heartbeat timestamp
func (c *Client) GetLastHeartbeat() time.Time {
	c.heartbeatMu.RLock()
	defer c.heartbeatMu.RUnlock()
	return c.lastHeartbeat
}

// ConnectionDuration returns how long the client has been connected
func (c *Client) ConnectionDuration() time.Duration {
	return time.Since(c.connectedAt)
}

// Close gracefully closes the client connection
func (c *Client) Close() {
	c.closed.Store(true)
	c.cancel()
	c.hub.unregister <- c
}
