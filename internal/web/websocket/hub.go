package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool
	clientsMu sync.RWMutex

	// Rooms for grouping clients
	rooms map[string]map[*Client]bool
	roomsMu sync.RWMutex

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan *Message

	// Broadcast messages to specific room
	roomBroadcast chan *RoomMessage

	// Message handlers
	handlers map[string]MessageHandler
	handlersMu sync.RWMutex

	// Authentication handler
	authHandler AuthHandler

	// Shutdown channel
	shutdown chan struct{}

	// Wait group for graceful shutdown
	wg sync.WaitGroup

	// Context for cancellation
	ctx context.Context
	cancel context.CancelFunc
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Payload interface{}     `json:"-"`
}

// RoomMessage represents a message to be broadcast to a specific room
type RoomMessage struct {
	Room    string
	Message *Message
}

// MessageHandler is a function that handles incoming messages
type MessageHandler func(ctx context.Context, client *Client, message *Message) error

// AuthHandler is a function that authenticates WebSocket connections
type AuthHandler func(ctx context.Context, token string) (userID string, err error)

// NewHub creates a new Hub instance
func NewHub(ctx context.Context) *Hub {
	hubCtx, cancel := context.WithCancel(ctx)

	return &Hub{
		clients:       make(map[*Client]bool),
		rooms:         make(map[string]map[*Client]bool),
		register:      make(chan *Client, 256),
		unregister:    make(chan *Client, 256),
		broadcast:     make(chan *Message, 1024),
		roomBroadcast: make(chan *RoomMessage, 1024),
		handlers:      make(map[string]MessageHandler),
		shutdown:      make(chan struct{}),
		ctx:           hubCtx,
		cancel:        cancel,
	}
}

// RegisterHandler registers a message handler for a specific message type
func (h *Hub) RegisterHandler(messageType string, handler MessageHandler) {
	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()
	h.handlers[messageType] = handler
}

// SetAuthHandler sets the authentication handler
func (h *Hub) SetAuthHandler(handler AuthHandler) {
	h.authHandler = handler
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	h.wg.Add(1)
	defer h.wg.Done()

	// Start cleanup ticker
	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			h.cleanup()
			return

		case <-h.shutdown:
			h.cleanup()
			return

		case client := <-h.register:
			h.clientsMu.Lock()
			h.clients[client] = true
			h.clientsMu.Unlock()
			log.Printf("Client registered: %s (total: %d)", client.ID, h.ClientCount())

		case client := <-h.unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closed.Store(true)
				close(client.send)
			}
			h.clientsMu.Unlock()

			// Remove from all rooms
			h.roomsMu.Lock()
			for roomName, clients := range h.rooms {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.rooms, roomName)
					}
				}
			}
			h.roomsMu.Unlock()

			log.Printf("Client unregistered: %s (total: %d)", client.ID, h.ClientCount())

		case message := <-h.broadcast:
			h.broadcastToAll(message)

		case roomMsg := <-h.roomBroadcast:
			h.broadcastToRoom(roomMsg.Room, roomMsg.Message)

		case <-cleanupTicker.C:
			h.cleanupStaleConnections()
		}
	}
}

// broadcastToAll sends a message to all connected clients
func (h *Hub) broadcastToAll(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			// Client's send channel is full, skip
			log.Printf("Skipping client %s: send channel full", client.ID)
		}
	}
}

// broadcastToRoom sends a message to all clients in a specific room
func (h *Hub) broadcastToRoom(room string, message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.roomsMu.RLock()
	clientMap, ok := h.rooms[room]
	if !ok {
		h.roomsMu.RUnlock()
		return
	}

	// Copy client pointers while holding lock
	clientList := make([]*Client, 0, len(clientMap))
	for client := range clientMap {
		clientList = append(clientList, client)
	}
	h.roomsMu.RUnlock()

	// Now iterate over the copy without holding lock
	for _, client := range clientList {
		select {
		case client.send <- data:
		default:
			log.Printf("Skipping client %s in room %s: send channel full", client.ID, room)
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message *Message) {
	select {
	case h.broadcast <- message:
	case <-h.ctx.Done():
		log.Printf("Hub context done, cannot broadcast")
	default:
		log.Printf("Broadcast channel full, message dropped")
	}
}

// BroadcastToRoom sends a message to all clients in a specific room
func (h *Hub) BroadcastToRoom(room string, message *Message) {
	select {
	case h.roomBroadcast <- &RoomMessage{Room: room, Message: message}:
	case <-h.ctx.Done():
		log.Printf("Hub context done, cannot broadcast to room")
	default:
		log.Printf("Room broadcast channel full, message dropped")
	}
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, room string) {
	h.roomsMu.Lock()
	defer h.roomsMu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true

	log.Printf("Client %s joined room %s", client.ID, room)
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, room string) {
	h.roomsMu.Lock()
	defer h.roomsMu.Unlock()

	if clients, ok := h.rooms[room]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
		log.Printf("Client %s left room %s", client.ID, room)
	}
}

// GetRoomClients returns all clients in a room
func (h *Hub) GetRoomClients(room string) []*Client {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	clients, ok := h.rooms[room]
	if !ok {
		return nil
	}

	result := make([]*Client, 0, len(clients))
	for client := range clients {
		result = append(result, client)
	}
	return result
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// RoomCount returns the number of active rooms
func (h *Hub) RoomCount() int {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()
	return len(h.rooms)
}

// GetClients returns all connected clients
func (h *Hub) GetClients() []*Client {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// HandleMessage processes an incoming message from a client
func (h *Hub) HandleMessage(ctx context.Context, client *Client, data []byte) error {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}

	h.handlersMu.RLock()
	handler, ok := h.handlers[message.Type]
	h.handlersMu.RUnlock()

	if !ok {
		log.Printf("No handler registered for message type: %s", message.Type)
		return nil
	}

	return handler(ctx, client, &message)
}

// cleanup closes all client connections and cleans up resources
func (h *Hub) cleanup() {
	log.Printf("Hub shutting down, disconnecting %d clients", h.ClientCount())

	h.clientsMu.Lock()
	for client := range h.clients {
		client.closed.Store(true)
		// Don't close send channel here - let WritePump handle it via context
		if client.conn != nil {
			client.conn.Close()
		}
	}
	h.clients = make(map[*Client]bool)
	h.clientsMu.Unlock()

	h.roomsMu.Lock()
	h.rooms = make(map[string]map[*Client]bool)
	h.roomsMu.Unlock()
}

// cleanupStaleConnections removes clients that haven't sent a heartbeat recently
func (h *Hub) cleanupStaleConnections() {
	h.clientsMu.RLock()
	staleClients := make([]*Client, 0)

	for client := range h.clients {
		if time.Since(client.lastHeartbeat) > 90*time.Second {
			staleClients = append(staleClients, client)
		}
	}
	h.clientsMu.RUnlock()

	for _, client := range staleClients {
		log.Printf("Removing stale client: %s", client.ID)
		h.unregister <- client
	}
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	log.Printf("Hub shutdown initiated")
	h.cancel()
	close(h.shutdown)
	h.wg.Wait()
	log.Printf("Hub shutdown complete")
}
