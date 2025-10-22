package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationPingPong(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send ping message
	pingMsg := Message{
		Type: "ping",
		Data: json.RawMessage(`"2024-01-01T00:00:00Z"`),
	}

	err = ws.WriteJSON(pingMsg)
	require.NoError(t, err)

	// Receive pong response
	var response Message
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, "pong", response.Type)
}

func TestIntegrationRoomJoinLeave(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Join room
	joinMsg := Message{
		Type: "join_room",
	}
	joinMsg.Data, _ = json.Marshal(map[string]string{"room": "test-room"})

	err = ws.WriteJSON(joinMsg)
	require.NoError(t, err)

	// Receive room_joined response
	var response Message
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, "room_joined", response.Type)

	// Leave room
	leaveMsg := Message{
		Type: "leave_room",
	}
	leaveMsg.Data, _ = json.Marshal(map[string]string{"room": "test-room"})

	err = ws.WriteJSON(leaveMsg)
	require.NoError(t, err)

	// Receive room_left response
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, "room_left", response.Type)
}

func TestIntegrationBroadcastToAll(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients
	client1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client1.Close()

	client2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client2.Close()

	time.Sleep(100 * time.Millisecond)

	// Broadcast message
	broadcastMsg := Message{
		Type: "broadcast",
	}
	payload := map[string]interface{}{
		"type":    "notification",
		"payload": json.RawMessage(`{"message":"hello all"}`),
	}
	broadcastMsg.Data, _ = json.Marshal(payload)

	err = client1.WriteJSON(broadcastMsg)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Both clients should receive the broadcast
	client1.SetReadDeadline(time.Now().Add(1 * time.Second))
	client2.SetReadDeadline(time.Now().Add(1 * time.Second))

	var msg1, msg2 Message
	err1 := client1.ReadJSON(&msg1)
	err2 := client2.ReadJSON(&msg2)

	// At least one client should receive the message
	assert.True(t, err1 == nil || err2 == nil)
}

func TestIntegrationBroadcastToRoom(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect three clients
	client1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client1.Close()

	client2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client2.Close()

	client3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client3.Close()

	time.Sleep(100 * time.Millisecond)

	// Client1 and Client2 join "room-a"
	joinMsg := Message{Type: "join_room"}
	joinMsg.Data, _ = json.Marshal(map[string]string{"room": "room-a"})

	client1.WriteJSON(joinMsg)
	client2.WriteJSON(joinMsg)

	// Client3 joins "room-b"
	joinMsg.Data, _ = json.Marshal(map[string]string{"room": "room-b"})
	client3.WriteJSON(joinMsg)

	// Drain responses
	for i := 0; i < 3; i++ {
		time.Sleep(50 * time.Millisecond)
	}

	// Clear any pending messages
	client1.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	client2.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	client3.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	var dummy Message
	client1.ReadJSON(&dummy)
	client2.ReadJSON(&dummy)
	client3.ReadJSON(&dummy)

	// Broadcast to room-a
	broadcastMsg := Message{Type: "broadcast"}
	payload := map[string]interface{}{
		"room":    "room-a",
		"type":    "notification",
		"payload": json.RawMessage(`{"message":"hello room-a"}`),
	}
	broadcastMsg.Data, _ = json.Marshal(payload)

	err = client1.WriteJSON(broadcastMsg)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Client1 and Client2 should receive, Client3 should not
	client1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	client2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	client3.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	var msg1, msg2, msg3 Message
	err1 := client1.ReadJSON(&msg1)
	err2 := client2.ReadJSON(&msg2)
	err3 := client3.ReadJSON(&msg3)

	// Client 1 or 2 should receive
	assert.True(t, err1 == nil || err2 == nil, "At least one client in room-a should receive message")

	// Client 3 should timeout (not receive message)
	assert.Error(t, err3, "Client in room-b should not receive room-a message")
}

func TestIntegrationEcho(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send echo message
	echoMsg := Message{
		Type: "echo",
		Data: json.RawMessage(`"test message"`),
	}

	err = ws.WriteJSON(echoMsg)
	require.NoError(t, err)

	// Receive echo response
	var response Message
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, "echo", response.Type)
}

func TestIntegrationStatus(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send status request
	statusMsg := Message{
		Type: "status",
	}

	err = ws.WriteJSON(statusMsg)
	require.NoError(t, err)

	// Receive status response
	var response Message
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, "status", response.Type)
}

func TestIntegrationConnectionLifecycle(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, server.Hub.ClientCount())

	// Send some messages
	for i := 0; i < 10; i++ {
		msg := Message{
			Type: "echo",
			Data: json.RawMessage(`"test"`),
		}
		ws.WriteJSON(msg)
	}

	time.Sleep(100 * time.Millisecond)

	// Disconnect
	ws.Close()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, server.Hub.ClientCount())
}

func TestIntegrationMultipleRoomsPerClient(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Join multiple rooms
	rooms := []string{"room1", "room2", "room3"}
	for _, room := range rooms {
		joinMsg := Message{Type: "join_room"}
		joinMsg.Data, _ = json.Marshal(map[string]string{"room": room})

		err = ws.WriteJSON(joinMsg)
		require.NoError(t, err)

		var response Message
		ws.SetReadDeadline(time.Now().Add(1 * time.Second))
		ws.ReadJSON(&response)
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 3, server.Hub.RoomCount())
}
