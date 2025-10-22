package websocket

import (
	"context"
	"encoding/json"
	"fmt"
)

// marshalMessage converts a Message to JSON bytes
func marshalMessage(message *Message) ([]byte, error) {
	// If Payload is set, marshal it to Data
	if message.Payload != nil {
		data, err := json.Marshal(message.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		message.Data = data
	}

	return json.Marshal(message)
}

// MessageRouter routes messages based on type
type MessageRouter struct {
	handlers map[string]MessageHandler
}

// NewMessageRouter creates a new MessageRouter
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[string]MessageHandler),
	}
}

// Register registers a handler for a message type
func (r *MessageRouter) Register(messageType string, handler MessageHandler) {
	r.handlers[messageType] = handler
}

// Route routes a message to the appropriate handler
func (r *MessageRouter) Route(ctx context.Context, client *Client, message *Message) error {
	handler, ok := r.handlers[message.Type]
	if !ok {
		return fmt.Errorf("no handler for message type: %s", message.Type)
	}

	return handler(ctx, client, message)
}

// Built-in message handlers

// PingHandler handles ping messages
func PingHandler(ctx context.Context, client *Client, message *Message) error {
	return client.SendJSON("pong", map[string]interface{}{
		"timestamp": message.Data,
	})
}

// JoinRoomHandler handles room join requests
func JoinRoomHandler(ctx context.Context, client *Client, message *Message) error {
	var req struct {
		Room string `json:"room"`
	}

	if err := json.Unmarshal(message.Data, &req); err != nil {
		return fmt.Errorf("invalid join room request: %w", err)
	}

	if req.Room == "" {
		return fmt.Errorf("room name is required")
	}

	client.JoinRoom(req.Room)

	return client.SendJSON("room_joined", map[string]interface{}{
		"room": req.Room,
	})
}

// LeaveRoomHandler handles room leave requests
func LeaveRoomHandler(ctx context.Context, client *Client, message *Message) error {
	var req struct {
		Room string `json:"room"`
	}

	if err := json.Unmarshal(message.Data, &req); err != nil {
		return fmt.Errorf("invalid leave room request: %w", err)
	}

	if req.Room == "" {
		return fmt.Errorf("room name is required")
	}

	client.LeaveRoom(req.Room)

	return client.SendJSON("room_left", map[string]interface{}{
		"room": req.Room,
	})
}

// BroadcastHandler handles broadcast requests
func BroadcastHandler(ctx context.Context, client *Client, message *Message) error {
	var req struct {
		Room    string          `json:"room,omitempty"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(message.Data, &req); err != nil {
		return fmt.Errorf("invalid broadcast request: %w", err)
	}

	broadcastMsg := &Message{
		Type: req.Type,
		Data: req.Payload,
	}

	if req.Room != "" {
		client.hub.BroadcastToRoom(req.Room, broadcastMsg)
	} else {
		client.hub.Broadcast(broadcastMsg)
	}

	return nil
}

// EchoHandler echoes messages back to the sender
func EchoHandler(ctx context.Context, client *Client, message *Message) error {
	return client.SendJSON("echo", map[string]interface{}{
		"original": string(message.Data),
	})
}

// StatusHandler returns connection status
func StatusHandler(ctx context.Context, client *Client, message *Message) error {
	return client.SendJSON("status", map[string]interface{}{
		"client_id":          client.ID,
		"user_id":           client.UserID,
		"connected_at":      client.connectedAt,
		"connection_duration": client.ConnectionDuration().String(),
		"last_heartbeat":    client.GetLastHeartbeat(),
	})
}

// RegisterDefaultHandlers registers built-in message handlers
func RegisterDefaultHandlers(hub *Hub) {
	hub.RegisterHandler("ping", PingHandler)
	hub.RegisterHandler("join_room", JoinRoomHandler)
	hub.RegisterHandler("leave_room", LeaveRoomHandler)
	hub.RegisterHandler("broadcast", BroadcastHandler)
	hub.RegisterHandler("echo", EchoHandler)
	hub.RegisterHandler("status", StatusHandler)
}
