package websocket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRoom(t *testing.T) {
	room := NewRoom("test-room")

	assert.Equal(t, "test-room", room.Name)
	assert.NotNil(t, room.clients)
	assert.Equal(t, 0, room.Count())
}

func TestRoomAddRemove(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	room := NewRoom("test-room")

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)

	// Add clients
	room.Add(client1)
	room.Add(client2)

	assert.Equal(t, 2, room.Count())

	// Remove client
	room.Remove(client1)

	assert.Equal(t, 1, room.Count())

	// Remove last client
	room.Remove(client2)

	assert.Equal(t, 0, room.Count())
}

func TestRoomClients(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	room := NewRoom("test-room")

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)

	room.Add(client1)
	room.Add(client2)

	clients := room.Clients()

	assert.Equal(t, 2, len(clients))
	assert.Contains(t, clients, client1)
	assert.Contains(t, clients, client2)
}

func TestRoomBroadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	room := NewRoom("test-room")

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)

	room.Add(client1)
	room.Add(client2)

	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "hello",
		},
	}

	err := room.Broadcast(msg)
	assert.NoError(t, err)

	// Both clients should receive the message
	assert.Equal(t, 1, len(client1.send))
	assert.Equal(t, 1, len(client2.send))
}

func TestRoomBroadcastExcept(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	room := NewRoom("test-room")

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)
	client3 := NewClient("client3", nil, hub)

	room.Add(client1)
	room.Add(client2)
	room.Add(client3)

	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "hello",
		},
	}

	err := room.BroadcastExcept(msg, client2)
	assert.NoError(t, err)

	// Client1 and Client3 should receive, but not Client2
	assert.Equal(t, 1, len(client1.send))
	assert.Equal(t, 0, len(client2.send))
	assert.Equal(t, 1, len(client3.send))
}

func TestNewRoomManager(t *testing.T) {
	rm := NewRoomManager()

	assert.NotNil(t, rm)
	assert.NotNil(t, rm.rooms)
	assert.Equal(t, 0, rm.Count())
}

func TestRoomManagerGetOrCreate(t *testing.T) {
	rm := NewRoomManager()

	// Create new room
	room1 := rm.GetOrCreate("room1")
	assert.Equal(t, "room1", room1.Name)
	assert.Equal(t, 1, rm.Count())

	// Get existing room
	room2 := rm.GetOrCreate("room1")
	assert.Equal(t, room1, room2)
	assert.Equal(t, 1, rm.Count())
}

func TestRoomManagerGet(t *testing.T) {
	rm := NewRoomManager()

	// Create room
	rm.GetOrCreate("room1")

	// Get existing room
	room, err := rm.Get("room1")
	assert.NoError(t, err)
	assert.Equal(t, "room1", room.Name)

	// Get non-existent room
	_, err = rm.Get("room2")
	assert.Error(t, err)
}

func TestRoomManagerDelete(t *testing.T) {
	rm := NewRoomManager()

	rm.GetOrCreate("room1")
	rm.GetOrCreate("room2")

	assert.Equal(t, 2, rm.Count())

	rm.Delete("room1")

	assert.Equal(t, 1, rm.Count())

	_, err := rm.Get("room1")
	assert.Error(t, err)
}

func TestRoomManagerList(t *testing.T) {
	rm := NewRoomManager()

	rm.GetOrCreate("room1")
	rm.GetOrCreate("room2")
	rm.GetOrCreate("room3")

	names := rm.List()

	assert.Equal(t, 3, len(names))
	assert.Contains(t, names, "room1")
	assert.Contains(t, names, "room2")
	assert.Contains(t, names, "room3")
}

func TestRoomManagerCleanup(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	rm := NewRoomManager()

	// Create rooms with clients
	room1 := rm.GetOrCreate("room1")
	room2 := rm.GetOrCreate("room2")
	_ = rm.GetOrCreate("room3") // room3 will be empty

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)

	// Add clients to some rooms
	room1.Add(client1)
	room2.Add(client2)
	// room3 is empty

	assert.Equal(t, 3, rm.Count())

	// Cleanup should remove empty rooms
	rm.Cleanup()

	assert.Equal(t, 2, rm.Count())

	names := rm.List()
	assert.Contains(t, names, "room1")
	assert.Contains(t, names, "room2")
	assert.NotContains(t, names, "room3")
}

func TestRoomManagerStats(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	rm := NewRoomManager()

	room1 := rm.GetOrCreate("room1")
	room2 := rm.GetOrCreate("room2")

	client1 := NewClient("client1", nil, hub)
	client2 := NewClient("client2", nil, hub)
	client3 := NewClient("client3", nil, hub)

	room1.Add(client1)
	room1.Add(client2)
	room2.Add(client3)

	stats := rm.Stats()

	assert.Equal(t, 2, stats["room1"])
	assert.Equal(t, 1, stats["room2"])
}

func TestRoomConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	room := NewRoom("concurrent-test")

	// Add multiple clients concurrently
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(n int) {
			client := NewClient(string(rune(n)), nil, hub)
			room.Add(client)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	assert.Equal(t, 100, room.Count())
}

func TestRoomManagerConcurrentAccess(t *testing.T) {
	rm := NewRoomManager()

	done := make(chan bool)

	// Create multiple rooms concurrently
	for i := 0; i < 100; i++ {
		go func(n int) {
			roomName := string(rune('A' + n%26))
			rm.GetOrCreate(roomName)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have 26 unique rooms (A-Z)
	assert.LessOrEqual(t, rm.Count(), 26)
}
