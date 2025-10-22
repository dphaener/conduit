package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	assert.Equal(t, "test-id", client.ID)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.metadata)
	assert.Equal(t, hub, client.hub)
}

func TestClientSend(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"content": "hello",
		},
	}

	err := client.Send(msg)
	assert.NoError(t, err)

	// Message should be in send channel
	assert.Equal(t, 1, len(client.send))
}

func TestClientSendJSON(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	err := client.SendJSON("test", map[string]string{"key": "value"})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(client.send))
}

func TestClientSendError(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	client.SendError("test error message")

	assert.Equal(t, 1, len(client.send))
}

func TestClientMetadata(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	// Set metadata
	client.SetMetadata("key1", "value1")
	client.SetMetadata("key2", 42)

	// Get metadata
	val1, ok1 := client.GetMetadata("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", val1)

	val2, ok2 := client.GetMetadata("key2")
	assert.True(t, ok2)
	assert.Equal(t, 42, val2)

	// Non-existent key
	_, ok3 := client.GetMetadata("key3")
	assert.False(t, ok3)
}

func TestClientHeartbeat(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	initialHeartbeat := client.GetLastHeartbeat()

	time.Sleep(10 * time.Millisecond)

	client.updateHeartbeat()

	updatedHeartbeat := client.GetLastHeartbeat()

	assert.True(t, updatedHeartbeat.After(initialHeartbeat))
}

func TestClientConnectionDuration(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	client := NewClient("test-id", nil, hub)

	time.Sleep(100 * time.Millisecond)

	duration := client.ConnectionDuration()

	assert.Greater(t, duration, 50*time.Millisecond)
}

func TestClientJoinLeaveRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)

	hub.register <- client

	time.Sleep(50 * time.Millisecond)

	// Join room
	client.JoinRoom("test-room")

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.RoomCount())

	clients := hub.GetRoomClients("test-room")
	assert.Equal(t, 1, len(clients))
	assert.Equal(t, client, clients[0])

	// Leave room
	client.LeaveRoom("test-room")

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.RoomCount())
}

func TestClientSendChannelFull(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	// Create client with small buffer
	client := &Client{
		ID:       "test-id",
		hub:      hub,
		send:     make(chan []byte, 1),
		ctx:      context.Background(),
		metadata: make(map[string]interface{}),
	}

	// Fill the channel
	client.send <- []byte("message 1")

	// Try to send when channel is full
	msg := &Message{
		Type:    "test",
		Payload: "test",
	}

	err := client.Send(msg)
	// Should return error when channel is full
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send channel full")
}

func TestClientClose(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)

	hub.register <- client

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Close client
	client.Close()

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}
