package websocket

import (
	"fmt"
	"sync"
)

// Room represents a WebSocket room (channel) for grouping clients
type Room struct {
	Name    string
	clients map[*Client]bool
	mu      sync.RWMutex
}

// NewRoom creates a new Room
func NewRoom(name string) *Room {
	return &Room{
		Name:    name,
		clients: make(map[*Client]bool),
	}
}

// Add adds a client to the room
func (r *Room) Add(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client] = true
}

// Remove removes a client from the room
func (r *Room) Remove(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, client)
}

// Clients returns all clients in the room
func (r *Room) Clients() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}

// Count returns the number of clients in the room
func (r *Room) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Broadcast sends a message to all clients in the room
func (r *Room) Broadcast(message *Message) error {
	data, err := marshalMessage(message)
	if err != nil {
		return err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for client := range r.clients {
		select {
		case client.send <- data:
		default:
			// Skip if send channel is full
		}
	}

	return nil
}

// BroadcastExcept sends a message to all clients in the room except the specified client
func (r *Room) BroadcastExcept(message *Message, exceptClient *Client) error {
	data, err := marshalMessage(message)
	if err != nil {
		return err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for client := range r.clients {
		if client == exceptClient {
			continue
		}

		select {
		case client.send <- data:
		default:
			// Skip if send channel is full
		}
	}

	return nil
}

// RoomManager manages multiple rooms
type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

// NewRoomManager creates a new RoomManager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// GetOrCreate gets an existing room or creates a new one
func (rm *RoomManager) GetOrCreate(name string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, ok := rm.rooms[name]; ok {
		return room
	}

	room := NewRoom(name)
	rm.rooms[name] = room
	return room
}

// Get returns a room by name
func (rm *RoomManager) Get(name string) (*Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[name]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", name)
	}

	return room, nil
}

// Delete deletes a room
func (rm *RoomManager) Delete(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, name)
}

// List returns all room names
func (rm *RoomManager) List() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	names := make([]string, 0, len(rm.rooms))
	for name := range rm.rooms {
		names = append(names, name)
	}
	return names
}

// Count returns the total number of rooms
func (rm *RoomManager) Count() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// Cleanup removes empty rooms
func (rm *RoomManager) Cleanup() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for name, room := range rm.rooms {
		if room.Count() == 0 {
			delete(rm.rooms, name)
		}
	}
}

// Stats returns statistics about all rooms
func (rm *RoomManager) Stats() map[string]int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := make(map[string]int)
	for name, room := range rm.rooms {
		stats[name] = room.Count()
	}
	return stats
}
