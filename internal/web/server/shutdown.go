package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// GracefulShutdown handles graceful server shutdown with cleanup hooks
type GracefulShutdown struct {
	server         *Server
	shutdownHooks  []ShutdownHook
	timeout        time.Duration
	signals        []os.Signal
	logger         Logger
	mu             sync.Mutex
	shutdownOnce   sync.Once
	shutdownChan   chan struct{}
	shutdownError  error
}

// ShutdownHook is a function called during graceful shutdown
type ShutdownHook func(ctx context.Context) error

// Logger is a simple logging interface
type Logger interface {
	Printf(format string, v ...interface{})
}

// defaultLogger uses standard log package
type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// ShutdownConfig holds graceful shutdown configuration
type ShutdownConfig struct {
	// Timeout is the maximum time to wait for shutdown
	Timeout time.Duration

	// Signals to listen for (default: SIGINT, SIGTERM)
	Signals []os.Signal

	// Logger for shutdown messages
	Logger Logger
}

// DefaultShutdownConfig returns default shutdown configuration
func DefaultShutdownConfig() *ShutdownConfig {
	return &ShutdownConfig{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		Logger:  &defaultLogger{},
	}
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(server *Server, config *ShutdownConfig) *GracefulShutdown {
	if config == nil {
		config = DefaultShutdownConfig()
	}

	if config.Logger == nil {
		config.Logger = &defaultLogger{}
	}

	if len(config.Signals) == 0 {
		config.Signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	return &GracefulShutdown{
		server:        server,
		shutdownHooks: make([]ShutdownHook, 0),
		timeout:       config.Timeout,
		signals:       config.Signals,
		logger:        config.Logger,
		shutdownChan:  make(chan struct{}),
	}
}

// RegisterHook registers a shutdown hook to be called during shutdown
func (gs *GracefulShutdown) RegisterHook(hook ShutdownHook) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.shutdownHooks = append(gs.shutdownHooks, hook)
}

// Start starts the server and waits for shutdown signal
func (gs *GracefulShutdown) Start() error {
	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		gs.logger.Printf("Starting server on %s", gs.server.Addr())
		if err := gs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server failed: %w", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, gs.signals...)

	select {
	case <-quit:
		gs.logger.Printf("Shutdown signal received, shutting down gracefully...")
		return gs.Shutdown()
	case err := <-errChan:
		return err
	}
}

// Shutdown performs graceful shutdown
func (gs *GracefulShutdown) Shutdown() error {
	gs.shutdownOnce.Do(func() {
		gs.logger.Printf("Initiating graceful shutdown (timeout: %v)", gs.timeout)

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), gs.timeout)
		defer cancel()

		// Run shutdown hooks
		gs.mu.Lock()
		hooks := make([]ShutdownHook, len(gs.shutdownHooks))
		copy(hooks, gs.shutdownHooks)
		gs.mu.Unlock()

		if len(hooks) > 0 {
			gs.logger.Printf("Running %d shutdown hooks...", len(hooks))
			for i, hook := range hooks {
				if err := hook(ctx); err != nil {
					gs.logger.Printf("Shutdown hook %d failed: %v", i, err)
					// Continue with other hooks
				}
			}
		}

		// Shutdown HTTP server
		gs.logger.Printf("Shutting down HTTP server...")
		if err := gs.server.Shutdown(ctx); err != nil {
			gs.shutdownError = fmt.Errorf("server shutdown error: %w", err)
			gs.logger.Printf("Server shutdown error: %v", err)
		} else {
			gs.logger.Printf("Server shutdown completed successfully")
		}

		close(gs.shutdownChan)
	})

	// Wait for shutdown to complete
	<-gs.shutdownChan
	return gs.shutdownError
}

// Wait blocks until shutdown is complete
func (gs *GracefulShutdown) Wait() error {
	<-gs.shutdownChan
	return gs.shutdownError
}

// StartWithGracefulShutdown is a convenience function to start a server with graceful shutdown
func StartWithGracefulShutdown(server *Server, config *ShutdownConfig) error {
	gs := NewGracefulShutdown(server, config)
	return gs.Start()
}
