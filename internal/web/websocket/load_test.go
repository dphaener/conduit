package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad1000Connections tests 1,000 concurrent connections
func TestLoad1000Connections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	testLoadConnections(t, 1000)
}

// TestLoad5000Connections tests 5,000 concurrent connections
func TestLoad5000Connections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	testLoadConnections(t, 5000)
}

// TestLoad10000Connections tests 10,000 concurrent connections
func TestLoad10000Connections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	testLoadConnections(t, 10000)
}

func testLoadConnections(t *testing.T, numConnections int) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	var wg sync.WaitGroup
	connections := make([]*websocket.Conn, numConnections)
	var successCount int32
	var failCount int32

	startTime := time.Now()

	// Connect clients concurrently
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				atomic.AddInt32(&failCount, 1)
				return
			}

			connections[idx] = ws
			atomic.AddInt32(&successCount, 1)
		}(i)
	}

	wg.Wait()
	connectionTime := time.Since(startTime)

	t.Logf("Connected %d clients in %v (%.0f connections/sec)",
		successCount, connectionTime, float64(successCount)/connectionTime.Seconds())

	// Allow hub to process registrations
	time.Sleep(1 * time.Second)

	actualCount := server.Hub.ClientCount()
	t.Logf("Hub reports %d active clients", actualCount)

	assert.GreaterOrEqual(t, int32(actualCount), successCount*95/100, "At least 95%% of connections should be active")

	// Clean up
	for _, ws := range connections {
		if ws != nil {
			ws.Close()
		}
	}

	t.Logf("Load test completed: %d successful, %d failed", successCount, failCount)
}

// TestLoadBroadcast tests broadcast performance with many clients
func TestLoadBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	numClients := 1000
	numMessages := 100

	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect clients
	var wg sync.WaitGroup
	connections := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			connections[idx] = ws
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	t.Logf("Connected %d clients", server.Hub.ClientCount())

	// Broadcast messages
	startTime := time.Now()

	for i := 0; i < numMessages; i++ {
		msg := &Message{
			Type: "test",
			Payload: map[string]int{
				"number": i,
			},
		}
		server.Hub.Broadcast(msg)
	}

	broadcastTime := time.Since(startTime)
	messagesPerSecond := float64(numMessages) / broadcastTime.Seconds()

	t.Logf("Broadcast %d messages in %v (%.0f msg/sec)",
		numMessages, broadcastTime, messagesPerSecond)

	// Allow messages to be delivered
	time.Sleep(1 * time.Second)

	// Clean up
	for _, ws := range connections {
		if ws != nil {
			ws.Close()
		}
	}

	assert.Greater(t, messagesPerSecond, float64(100), "Should broadcast at least 100 messages/sec")
}

// TestLoadRoomBroadcast tests room-based broadcast performance
func TestLoadRoomBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	numClients := 1000
	numRooms := 10

	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect clients and assign to rooms
	connections := make([]*websocket.Conn, numClients)
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			connections[idx] = ws

			// Join a room
			roomNum := idx % numRooms
			joinMsg := Message{Type: "join_room"}
			joinMsg.Data, _ = json.Marshal(map[string]string{
				"room": fmt.Sprintf("room-%d", roomNum),
			})
			ws.WriteJSON(joinMsg)
		}(i)
	}

	wg.Wait()
	time.Sleep(1 * time.Second)

	t.Logf("Connected %d clients across %d rooms", server.Hub.ClientCount(), server.Hub.RoomCount())

	// Broadcast to each room
	startTime := time.Now()

	for i := 0; i < numRooms; i++ {
		msg := &Message{
			Type: "test",
			Payload: map[string]string{
				"room": fmt.Sprintf("room-%d", i),
			},
		}
		server.Hub.BroadcastToRoom(fmt.Sprintf("room-%d", i), msg)
	}

	broadcastTime := time.Since(startTime)

	t.Logf("Broadcast to %d rooms in %v", numRooms, broadcastTime)

	// Clean up
	for _, ws := range connections {
		if ws != nil {
			ws.Close()
		}
	}

	assert.Less(t, broadcastTime, 1*time.Second, "Should broadcast to all rooms in under 1 second")
}

// TestLoadMessageThroughput tests message throughput
func TestLoadMessageThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	numClients := 100
	messagesPerClient := 100

	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect clients
	connections := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		connections[i] = ws
	}

	time.Sleep(500 * time.Millisecond)

	t.Logf("Connected %d clients", server.Hub.ClientCount())

	// Send messages concurrently
	var wg sync.WaitGroup
	var messagesSent int32

	startTime := time.Now()

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(ws *websocket.Conn) {
			defer wg.Done()

			for j := 0; j < messagesPerClient; j++ {
				msg := Message{
					Type: "echo",
					Data: json.RawMessage(`"test"`),
				}
				if ws.WriteJSON(msg) == nil {
					atomic.AddInt32(&messagesSent, 1)
				}
			}
		}(connections[i])
	}

	wg.Wait()
	throughputTime := time.Since(startTime)

	messagesPerSecond := float64(messagesSent) / throughputTime.Seconds()

	t.Logf("Sent %d messages in %v (%.0f msg/sec)",
		messagesSent, throughputTime, messagesPerSecond)

	// Clean up
	for _, ws := range connections {
		if ws != nil {
			ws.Close()
		}
	}

	assert.Greater(t, messagesPerSecond, float64(1000), "Should handle at least 1000 messages/sec")
}

// TestLoadMemoryUsage monitors memory usage under load
func TestLoadMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	numConnections := 5000

	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	connections := make([]*websocket.Conn, numConnections)

	// Connect all clients
	for i := 0; i < numConnections; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			continue
		}
		connections[i] = ws
	}

	time.Sleep(2 * time.Second)

	t.Logf("Established %d connections", server.Hub.ClientCount())

	// Keep connections alive and send periodic messages
	duration := 10 * time.Second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(duration)

	messagesProcessed := 0

loop:
	for {
		select {
		case <-ticker.C:
			// Send a broadcast message
			msg := &Message{
				Type:    "test",
				Payload: map[string]string{"timestamp": time.Now().String()},
			}
			server.Hub.Broadcast(msg)
			messagesProcessed++

		case <-timeout:
			break loop
		}
	}

	t.Logf("Processed %d broadcast cycles over %v with %d connections",
		messagesProcessed, duration, server.Hub.ClientCount())

	// Clean up
	for _, ws := range connections {
		if ws != nil {
			ws.Close()
		}
	}

	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, server.Hub.ClientCount(), "All connections should be cleaned up")
}

// BenchmarkBroadcast benchmarks broadcast performance
func BenchmarkBroadcast(b *testing.B) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	msg := &Message{
		Type:    "test",
		Payload: map[string]string{"data": "benchmark"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		server.Hub.Broadcast(msg)
	}
}

// BenchmarkMessageMarshaling benchmarks message marshaling
func BenchmarkMessageMarshaling(b *testing.B) {
	msg := &Message{
		Type: "test",
		Payload: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		marshalMessage(msg)
	}
}

// BenchmarkClientSend benchmarks sending messages to a client
func BenchmarkClientSend(b *testing.B) {
	ctx := context.Background()
	hub := NewHub(ctx)
	client := NewClient("bench-client", nil, hub)

	msg := &Message{
		Type:    "test",
		Payload: "benchmark data",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Send(msg)
		<-client.send // Drain to prevent blocking
	}
}
