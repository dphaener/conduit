package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.rooms)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.roomBroadcast)
	assert.NotNil(t, hub.handlers)
	assert.Equal(t, 0, hub.ClientCount())
	assert.Equal(t, 0, hub.RoomCount())
}

func TestHubRegisterHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	handler := func(ctx context.Context, client *Client, message *Message) error {
		return nil
	}

	hub.RegisterHandler("test", handler)

	hub.handlersMu.RLock()
	_, ok := hub.handlers["test"]
	hub.handlersMu.RUnlock()

	assert.True(t, ok, "Handler should be registered")
}

func TestHubClientRegistration(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	// Create mock client
	client := &Client{
		ID:   "test-client",
		send: make(chan []byte, 256),
	}

	hub.register <- client

	// Give hub time to process
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Unregister client
	hub.unregister <- client

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}

func TestHubBroadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	// Create mock clients
	client1 := &Client{
		ID:   "client1",
		send: make(chan []byte, 256),
	}
	client2 := &Client{
		ID:   "client2",
		send: make(chan []byte, 256),
	}

	hub.register <- client1
	hub.register <- client2

	time.Sleep(50 * time.Millisecond)

	// Broadcast message
	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "hello",
		},
	}

	hub.Broadcast(msg)

	time.Sleep(50 * time.Millisecond)

	// Check both clients received the message
	assert.Greater(t, len(client1.send), 0, "Client 1 should receive message")
	assert.Greater(t, len(client2.send), 0, "Client 2 should receive message")
}

func TestHubRoomManagement(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client1 := &Client{
		ID:   "client1",
		send: make(chan []byte, 256),
		hub:  hub,
	}
	client2 := &Client{
		ID:   "client2",
		send: make(chan []byte, 256),
		hub:  hub,
	}

	hub.register <- client1
	hub.register <- client2

	time.Sleep(50 * time.Millisecond)

	// Join room
	hub.JoinRoom(client1, "room1")
	hub.JoinRoom(client2, "room1")

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.RoomCount())

	clients := hub.GetRoomClients("room1")
	assert.Equal(t, 2, len(clients))

	// Leave room
	hub.LeaveRoom(client1, "room1")

	time.Sleep(50 * time.Millisecond)

	clients = hub.GetRoomClients("room1")
	assert.Equal(t, 1, len(clients))
}

func TestHubRoomBroadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client1 := &Client{
		ID:   "client1",
		send: make(chan []byte, 256),
		hub:  hub,
	}
	client2 := &Client{
		ID:   "client2",
		send: make(chan []byte, 256),
		hub:  hub,
	}
	client3 := &Client{
		ID:   "client3",
		send: make(chan []byte, 256),
		hub:  hub,
	}

	hub.register <- client1
	hub.register <- client2
	hub.register <- client3

	time.Sleep(50 * time.Millisecond)

	// Join rooms
	hub.JoinRoom(client1, "room1")
	hub.JoinRoom(client2, "room1")
	hub.JoinRoom(client3, "room2")

	time.Sleep(50 * time.Millisecond)

	// Broadcast to room1
	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "room1 message",
		},
	}

	hub.BroadcastToRoom("room1", msg)

	time.Sleep(50 * time.Millisecond)

	// Only client1 and client2 should receive the message
	assert.Greater(t, len(client1.send), 0, "Client 1 in room1 should receive message")
	assert.Greater(t, len(client2.send), 0, "Client 2 in room1 should receive message")
	assert.Equal(t, 0, len(client3.send), "Client 3 not in room1 should not receive message")
}

func TestHubHandleMessage(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	handlerCalled := false
	var receivedMessage *Message

	handler := func(ctx context.Context, client *Client, message *Message) error {
		handlerCalled = true
		receivedMessage = message
		return nil
	}

	hub.RegisterHandler("test", handler)

	client := &Client{
		ID:   "test-client",
		send: make(chan []byte, 256),
		hub:  hub,
	}

	message := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "hello",
		},
	}

	data, err := json.Marshal(message)
	require.NoError(t, err)

	err = hub.HandleMessage(ctx, client, data)
	assert.NoError(t, err)
	assert.True(t, handlerCalled, "Handler should be called")
	assert.Equal(t, "test", receivedMessage.Type)
}

func TestHubShutdown(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()

	// Add clients without actual connections
	client := &Client{
		ID:   "test-client",
		send: make(chan []byte, 256),
		conn: nil, // No actual connection in test
	}

	hub.register <- client

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Shutdown
	hub.Shutdown()

	time.Sleep(50 * time.Millisecond)

	// All clients should be disconnected
	assert.Equal(t, 0, hub.ClientCount())
	assert.Equal(t, 0, hub.RoomCount())
}

func TestHubCleanupStaleConnections(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	// Create client with old heartbeat
	client := &Client{
		ID:            "stale-client",
		send:          make(chan []byte, 256),
		lastHeartbeat: time.Now().Add(-2 * time.Minute),
	}

	hub.register <- client

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Trigger cleanup
	hub.cleanupStaleConnections()

	time.Sleep(50 * time.Millisecond)

	// Stale client should be removed
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHubMultipleRooms(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := &Client{
		ID:   "multi-room-client",
		send: make(chan []byte, 256),
		hub:  hub,
	}

	hub.register <- client

	time.Sleep(50 * time.Millisecond)

	// Join multiple rooms
	hub.JoinRoom(client, "room1")
	hub.JoinRoom(client, "room2")
	hub.JoinRoom(client, "room3")

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 3, hub.RoomCount())

	// Unregister client - should be removed from all rooms
	hub.unregister <- client

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.RoomCount(), "All rooms should be cleaned up")
}

func TestHubConcurrentBroadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	// Create multiple clients
	clients := make([]*Client, 10)
	for i := 0; i < 10; i++ {
		clients[i] = &Client{
			ID:   string(rune('A' + i)),
			send: make(chan []byte, 256),
		}
		hub.register <- clients[i]
	}

	time.Sleep(50 * time.Millisecond)

	// Broadcast multiple messages concurrently
	for i := 0; i < 100; i++ {
		go func(n int) {
			msg := &Message{
				Type: "test",
				Payload: map[string]int{
					"number": n,
				},
			}
			hub.Broadcast(msg)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	// All clients should receive messages
	for _, client := range clients {
		assert.Greater(t, len(client.send), 0, "Each client should receive messages")
	}
}
