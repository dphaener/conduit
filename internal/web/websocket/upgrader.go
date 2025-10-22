package websocket

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Config holds WebSocket configuration
type Config struct {
	// Buffer sizes
	ReadBufferSize  int
	WriteBufferSize int

	// Origin check function
	CheckOrigin func(r *http.Request) bool

	// Authentication token extraction
	TokenExtractor func(r *http.Request) string

	// Enable compression
	EnableCompression bool
}

// DefaultConfig returns default WebSocket configuration
func DefaultConfig() *Config {
	return &Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins in development
			// In production, implement proper origin checking
			return true
		},
		TokenExtractor: func(r *http.Request) string {
			// Try to get token from query parameter
			token := r.URL.Query().Get("token")
			if token != "" {
				return token
			}

			// Try to get token from Authorization header
			return r.Header.Get("Authorization")
		},
		EnableCompression: false,
	}
}

// Upgrader upgrades HTTP connections to WebSocket
type Upgrader struct {
	config   *Config
	upgrader *websocket.Upgrader
	hub      *Hub
}

// NewUpgrader creates a new Upgrader
func NewUpgrader(config *Config, hub *Hub) *Upgrader {
	if config == nil {
		config = DefaultConfig()
	}

	upgrader := &websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		CheckOrigin:       config.CheckOrigin,
		EnableCompression: config.EnableCompression,
	}

	return &Upgrader{
		config:   config,
		upgrader: upgrader,
		hub:      hub,
	}
}

// ServeHTTP handles WebSocket upgrade requests
func (u *Upgrader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, err := u.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Generate client ID
	clientID := uuid.New().String()

	// Create client
	client := NewClient(clientID, conn, u.hub)

	// Authenticate if auth handler is set
	if u.hub.authHandler != nil {
		token := u.config.TokenExtractor(r)
		if token != "" {
			userID, err := u.hub.authHandler(r.Context(), token)
			if err != nil {
				log.Printf("Authentication failed for client %s: %v", clientID, err)
				conn.Close()
				return
			}
			client.UserID = userID
		}
	}

	// Register client with hub
	u.hub.register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Printf("WebSocket connection established: %s", clientID)
}

// Handler returns an http.HandlerFunc for WebSocket upgrade
func (u *Upgrader) Handler() http.HandlerFunc {
	return u.ServeHTTP
}

// Middleware creates a middleware that upgrades to WebSocket on specific paths
func (u *Upgrader) Middleware(path string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == path {
				u.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Server wraps Hub and Upgrader for convenient WebSocket server setup
type Server struct {
	Hub      *Hub
	Upgrader *Upgrader
	Config   *Config
}

// NewServer creates a new WebSocket server
func NewServer(ctx context.Context, config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	hub := NewHub(ctx)
	upgrader := NewUpgrader(config, hub)

	// Register default handlers
	RegisterDefaultHandlers(hub)

	return &Server{
		Hub:      hub,
		Upgrader: upgrader,
		Config:   config,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() {
	go s.Hub.Run()
}

// Shutdown gracefully shuts down the WebSocket server
func (s *Server) Shutdown() {
	s.Hub.Shutdown()
}

// Handler returns the HTTP handler for WebSocket upgrade
func (s *Server) Handler() http.HandlerFunc {
	return s.Upgrader.Handler()
}
