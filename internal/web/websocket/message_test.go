package websocket

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalMessage(t *testing.T) {
	msg := &Message{
		Type: "test",
		Payload: map[string]string{
			"key": "value",
		},
	}

	data, err := marshalMessage(msg)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result["type"])
	assert.NotNil(t, result["data"])
}

func TestMessageRouter(t *testing.T) {
	router := NewMessageRouter()

	handlerCalled := false
	handler := func(ctx context.Context, client *Client, message *Message) error {
		handlerCalled = true
		return nil
	}

	router.Register("test", handler)

	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "test",
	}

	err := router.Route(ctx, client, msg)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestMessageRouterUnknownType(t *testing.T) {
	router := NewMessageRouter()

	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "unknown",
	}

	err := router.Route(ctx, client, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler for message type")
}

func TestPingHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "ping",
		Data: json.RawMessage(`"timestamp"`),
	}

	err := PingHandler(ctx, client, msg)
	assert.NoError(t, err)

	// Should send pong response
	assert.Equal(t, 1, len(client.send))
}

func TestJoinRoomHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)
	hub.register <- client

	reqData, _ := json.Marshal(map[string]string{
		"room": "test-room",
	})

	msg := &Message{
		Type: "join_room",
		Data: reqData,
	}

	err := JoinRoomHandler(ctx, client, msg)
	assert.NoError(t, err)

	// Should send room_joined response
	assert.Equal(t, 1, len(client.send))
}

func TestJoinRoomHandlerInvalidRequest(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "join_room",
		Data: json.RawMessage(`invalid json`),
	}

	err := JoinRoomHandler(ctx, client, msg)
	assert.Error(t, err)
}

func TestJoinRoomHandlerEmptyRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	reqData, _ := json.Marshal(map[string]string{
		"room": "",
	})

	msg := &Message{
		Type: "join_room",
		Data: reqData,
	}

	err := JoinRoomHandler(ctx, client, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "room name is required")
}

func TestLeaveRoomHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)
	hub.register <- client

	// First join a room
	hub.JoinRoom(client, "test-room")

	reqData, _ := json.Marshal(map[string]string{
		"room": "test-room",
	})

	msg := &Message{
		Type: "leave_room",
		Data: reqData,
	}

	err := LeaveRoomHandler(ctx, client, msg)
	assert.NoError(t, err)

	// Should send room_left response
	assert.Equal(t, 1, len(client.send))
}

func TestEchoHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)

	msg := &Message{
		Type: "echo",
		Data: json.RawMessage(`"test message"`),
	}

	err := EchoHandler(ctx, client, msg)
	assert.NoError(t, err)

	// Should send echo response
	assert.Equal(t, 1, len(client.send))
}

func TestStatusHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("test-id", nil, hub)
	client.UserID = "user-123"

	msg := &Message{
		Type: "status",
	}

	err := StatusHandler(ctx, client, msg)
	assert.NoError(t, err)

	// Should send status response
	assert.Equal(t, 1, len(client.send))

	// Check response content
	var response Message
	err = json.Unmarshal(<-client.send, &response)
	require.NoError(t, err)

	assert.Equal(t, "status", response.Type)
}

func TestBroadcastHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)
	hub.register <- client

	reqData, _ := json.Marshal(map[string]interface{}{
		"type":    "notification",
		"payload": json.RawMessage(`{"message":"hello"}`),
	})

	msg := &Message{
		Type: "broadcast",
		Data: reqData,
	}

	err := BroadcastHandler(ctx, client, msg)
	assert.NoError(t, err)
}

func TestBroadcastHandlerToRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Shutdown()

	client := NewClient("test-id", nil, hub)
	hub.register <- client

	reqData, _ := json.Marshal(map[string]interface{}{
		"room":    "test-room",
		"type":    "notification",
		"payload": json.RawMessage(`{"message":"hello"}`),
	})

	msg := &Message{
		Type: "broadcast",
		Data: reqData,
	}

	err := BroadcastHandler(ctx, client, msg)
	assert.NoError(t, err)
}

func TestRegisterDefaultHandlers(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	RegisterDefaultHandlers(hub)

	// Check all default handlers are registered
	hub.handlersMu.RLock()
	defer hub.handlersMu.RUnlock()

	assert.NotNil(t, hub.handlers["ping"])
	assert.NotNil(t, hub.handlers["join_room"])
	assert.NotNil(t, hub.handlers["leave_room"])
	assert.NotNil(t, hub.handlers["broadcast"])
	assert.NotNil(t, hub.handlers["echo"])
	assert.NotNil(t, hub.handlers["status"])
}
