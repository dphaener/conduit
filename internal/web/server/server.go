package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Server represents an optimized HTTP server with production-ready configuration
type Server struct {
	httpServer *http.Server
	config     *Config
	listener   net.Listener
}

// Config holds server configuration
type Config struct {
	// Address is the server listen address (e.g., ":8080")
	Address string

	// Handler is the HTTP handler for the server
	Handler http.Handler

	// TLS configuration
	TLSConfig *TLSConfig

	// Timeouts
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration

	// Connection limits
	MaxHeaderBytes int

	// Database configuration for connection pooling
	Database *DatabaseConfig

	// HTTP/2 settings
	EnableHTTP2 bool
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	// CertFile is the path to the TLS certificate
	CertFile string

	// KeyFile is the path to the TLS private key
	KeyFile string

	// MinVersion is the minimum TLS version (default: TLS 1.2)
	MinVersion uint16

	// Custom tls.Config (optional)
	Config *tls.Config
}

// DatabaseConfig holds database connection pool configuration
type DatabaseConfig struct {
	// DB is the database connection
	DB *sql.DB

	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections in the pool
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns a production-ready server configuration
func DefaultConfig(handler http.Handler) *Config {
	return &Config{
		Address:           ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
		EnableHTTP2:       true,
	}
}

// DefaultDatabaseConfig returns optimized database connection pool settings
func DefaultDatabaseConfig(db *sql.DB) *DatabaseConfig {
	return &DatabaseConfig{
		DB:              db,
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// New creates a new optimized server instance
func New(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	if config.Handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// Configure database connection pool if provided
	if config.Database != nil {
		if err := configureDatabasePool(config.Database); err != nil {
			return nil, fmt.Errorf("failed to configure database pool: %w", err)
		}
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              config.Address,
		Handler:           config.Handler,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}

	// Configure TLS and HTTP/2 if enabled
	if config.TLSConfig != nil {
		tlsConfig := buildTLSConfig(config.TLSConfig, config.EnableHTTP2)
		httpServer.TLSConfig = tlsConfig
	}

	return &Server{
		httpServer: httpServer,
		config:     config,
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Create listener
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Start server
	if s.config.TLSConfig != nil {
		// Use TLS listener
		tlsListener := tls.NewListener(listener, s.httpServer.TLSConfig)
		return s.httpServer.Serve(tlsListener)
	}

	return s.httpServer.Serve(listener)
}

// ListenAndServe starts the server (convenience method)
func (s *Server) ListenAndServe() error {
	if s.config.TLSConfig != nil {
		return s.httpServer.ListenAndServeTLS(
			s.config.TLSConfig.CertFile,
			s.config.TLSConfig.KeyFile,
		)
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close immediately closes the server
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// Addr returns the server's network address
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.Address
}

// configureDatabasePool configures the database connection pool
func configureDatabasePool(config *DatabaseConfig) error {
	if config.DB == nil {
		return fmt.Errorf("database connection cannot be nil")
	}

	// Set connection pool parameters
	config.DB.SetMaxOpenConns(config.MaxOpenConns)
	config.DB.SetMaxIdleConns(config.MaxIdleConns)
	config.DB.SetConnMaxLifetime(config.ConnMaxLifetime)
	config.DB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := config.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// buildTLSConfig builds a TLS configuration with HTTP/2 support
func buildTLSConfig(tlsConfig *TLSConfig, enableHTTP2 bool) *tls.Config {
	// Use custom config if provided
	if tlsConfig.Config != nil {
		config := tlsConfig.Config.Clone()
		if enableHTTP2 {
			config.NextProtos = []string{"h2", "http/1.1"}
		}
		return config
	}

	// Build default config
	config := &tls.Config{
		MinVersion: tlsConfig.MinVersion,
	}

	// Set default minimum TLS version if not specified
	if config.MinVersion == 0 {
		config.MinVersion = tls.VersionTLS12
	}

	// Enable HTTP/2
	if enableHTTP2 {
		config.NextProtos = []string{"h2", "http/1.1"}
	}

	return config
}
