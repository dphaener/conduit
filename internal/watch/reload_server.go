package watch

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ReloadServer manages WebSocket connections for live reload
type ReloadServer struct {
	connections map[*websocket.Conn]bool
	broadcast   chan *ReloadMessage
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
	done        chan struct{}
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
}

// ReloadMessage represents a reload message sent to browsers
type ReloadMessage struct {
	Type      string      `json:"type"`      // "reload", "error", "building", "success"
	Scope     string      `json:"scope"`     // "ui", "backend", "config"
	Timestamp int64       `json:"timestamp"` // Unix timestamp
	Error     *ErrorInfo  `json:"error,omitempty"`
	Files     []string    `json:"files,omitempty"`
	Duration  float64     `json:"duration,omitempty"` // Milliseconds
}

// ErrorInfo holds detailed error information
type ErrorInfo struct {
	Message  string           `json:"message"`
	File     string           `json:"file,omitempty"`
	Line     int              `json:"line,omitempty"`
	Column   int              `json:"column,omitempty"`
	Code     string           `json:"code,omitempty"`
	Phase    string           `json:"phase,omitempty"`
	Severity string           `json:"severity,omitempty"`
}

// NewReloadServer creates a new reload server
func NewReloadServer() *ReloadServer {
	rs := &ReloadServer{
		connections: make(map[*websocket.Conn]bool),
		broadcast:   make(chan *ReloadMessage, 256),
		register:    make(chan *websocket.Conn),
		unregister:  make(chan *websocket.Conn),
		done:        make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// Allow no origin (same-origin)
					return true
				}
				// Allow localhost only for security
				return strings.HasPrefix(origin, "http://localhost") ||
					strings.HasPrefix(origin, "https://localhost") ||
					strings.HasPrefix(origin, "http://127.0.0.1") ||
					strings.HasPrefix(origin, "https://127.0.0.1")
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	go rs.run()

	return rs
}

// run handles the WebSocket connection lifecycle
func (rs *ReloadServer) run() {
	for {
		select {
		case <-rs.done:
			// Shutdown signal received
			log.Printf("[Reload] Shutting down reload server")
			return

		case conn := <-rs.register:
			rs.mutex.Lock()
			rs.connections[conn] = true
			rs.mutex.Unlock()
			log.Printf("[Reload] Client connected (total: %d)", len(rs.connections))

		case conn := <-rs.unregister:
			rs.mutex.Lock()
			if _, ok := rs.connections[conn]; ok {
				delete(rs.connections, conn)
				conn.Close()
			}
			rs.mutex.Unlock()
			log.Printf("[Reload] Client disconnected (total: %d)", len(rs.connections))

		case message := <-rs.broadcast:
			rs.sendToAll(message)
		}
	}
}

// sendToAll sends a message to all connected clients
func (rs *ReloadServer) sendToAll(message *ReloadMessage) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("[Reload] Failed to marshal message: %v", err)
		return
	}

	// Collect failed connections while holding read lock
	rs.mutex.RLock()
	var failedConns []*websocket.Conn
	for conn := range rs.connections {
		err := conn.WriteMessage(websocket.TextMessage, messageJSON)
		if err != nil {
			log.Printf("[Reload] Failed to send message: %v", err)
			failedConns = append(failedConns, conn)
		}
	}
	rs.mutex.RUnlock()

	// Remove failed connections with write lock
	if len(failedConns) > 0 {
		rs.mutex.Lock()
		for _, conn := range failedConns {
			if _, ok := rs.connections[conn]; ok {
				conn.Close()
				delete(rs.connections, conn)
			}
		}
		rs.mutex.Unlock()
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket
func (rs *ReloadServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := rs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Reload] Failed to upgrade connection: %v", err)
		return
	}

	// Register connection
	rs.register <- conn

	// Start reading messages (mostly for keepalive)
	go rs.readMessages(conn)
}

// readMessages reads messages from the client (for keepalive)
func (rs *ReloadServer) readMessages(conn *websocket.Conn) {
	defer func() {
		rs.unregister <- conn
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Reload] WebSocket error: %v", err)
			}
			break
		}
	}
}

// NotifyBuilding sends a "building" message to clients
func (rs *ReloadServer) NotifyBuilding(files []string) {
	rs.broadcast <- &ReloadMessage{
		Type:      "building",
		Timestamp: time.Now().Unix(),
		Files:     files,
	}
}

// NotifySuccess sends a "success" message to clients
func (rs *ReloadServer) NotifySuccess(duration time.Duration) {
	rs.broadcast <- &ReloadMessage{
		Type:      "success",
		Timestamp: time.Now().Unix(),
		Duration:  float64(duration.Milliseconds()),
	}
}

// NotifyReload sends a reload message to clients
func (rs *ReloadServer) NotifyReload(scope string) {
	rs.broadcast <- &ReloadMessage{
		Type:      "reload",
		Scope:     scope,
		Timestamp: time.Now().Unix(),
	}
}

// NotifyError sends an error message to clients
func (rs *ReloadServer) NotifyError(errorInfo *ErrorInfo) {
	rs.broadcast <- &ReloadMessage{
		Type:      "error",
		Timestamp: time.Now().Unix(),
		Error:     errorInfo,
	}
}

// NotifyErrors sends multiple errors to clients
func (rs *ReloadServer) NotifyErrors(errors []*ErrorInfo) {
	// Send first error with full details
	if len(errors) > 0 {
		rs.broadcast <- &ReloadMessage{
			Type:      "error",
			Timestamp: time.Now().Unix(),
			Error:     errors[0],
		}
	}
}

// ConnectionCount returns the number of active connections
func (rs *ReloadServer) ConnectionCount() int {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()
	return len(rs.connections)
}

// Close closes all connections and stops the server
func (rs *ReloadServer) Close() {
	// Signal the run goroutine to stop
	close(rs.done)

	// Close all connections
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	for conn := range rs.connections {
		conn.Close()
	}
	rs.connections = make(map[*websocket.Conn]bool)
}
