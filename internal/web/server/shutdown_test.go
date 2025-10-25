package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

// mockLogger captures log messages for testing
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *mockLogger) Printf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Store the raw format for easier testing
	l.messages = append(l.messages, format)
}

func (l *mockLogger) Contains(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.messages {
		if msg == substr || msg == substr+"\n" {
			return true
		}
	}
	return false
}

// Helper function to create a test server with a dummy handler
func createTestServer(addr string) *Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server, _ := New(&Config{Address: addr, Handler: handler})
	return server
}

func TestDefaultShutdownConfig(t *testing.T) {
	config := DefaultShutdownConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if len(config.Signals) != 2 {
		t.Errorf("Expected 2 signals, got %d", len(config.Signals))
	}

	expectedSignals := map[os.Signal]bool{
		syscall.SIGINT:  true,
		syscall.SIGTERM: true,
	}

	for _, sig := range config.Signals {
		if !expectedSignals[sig] {
			t.Errorf("Unexpected signal: %v", sig)
		}
	}

	if config.Logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestNewGracefulShutdown_WithConfig(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}

	config := &ShutdownConfig{
		Timeout: 10 * time.Second,
		Signals: []os.Signal{syscall.SIGTERM},
		Logger:  logger,
	}

	gs := NewGracefulShutdown(server, config)

	if gs.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", gs.timeout)
	}

	if len(gs.signals) != 1 {
		t.Errorf("Expected 1 signal, got %d", len(gs.signals))
	}

	if gs.logger != logger {
		t.Error("Expected custom logger to be set")
	}
}

func TestNewGracefulShutdown_NilConfig(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	if gs.timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", gs.timeout)
	}

	if len(gs.signals) != 2 {
		t.Errorf("Expected 2 default signals, got %d", len(gs.signals))
	}

	if gs.logger == nil {
		t.Error("Expected default logger to be set")
	}
}

func TestNewGracefulShutdown_NilLogger(t *testing.T) {
	server := createTestServer(":0")
	config := &ShutdownConfig{
		Timeout: 10 * time.Second,
		Logger:  nil, // Explicitly nil
	}

	gs := NewGracefulShutdown(server, config)

	if gs.logger == nil {
		t.Error("Expected default logger when nil provided")
	}
}

func TestNewGracefulShutdown_EmptySignals(t *testing.T) {
	server := createTestServer(":0")
	config := &ShutdownConfig{
		Timeout: 10 * time.Second,
		Signals: []os.Signal{}, // Empty slice
	}

	gs := NewGracefulShutdown(server, config)

	if len(gs.signals) != 2 {
		t.Errorf("Expected 2 default signals when empty provided, got %d", len(gs.signals))
	}
}

func TestRegisterHook(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	hookCalled := false
	hook := func(ctx context.Context) error {
		hookCalled = true
		return nil
	}

	gs.RegisterHook(hook)

	if len(gs.shutdownHooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(gs.shutdownHooks))
	}

	// Test that hook is called during shutdown
	ctx := context.Background()
	gs.shutdownHooks[0](ctx)

	if !hookCalled {
		t.Error("Expected hook to be called")
	}
}

func TestRegisterHook_Multiple(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	hook1 := func(ctx context.Context) error { return nil }
	hook2 := func(ctx context.Context) error { return nil }
	hook3 := func(ctx context.Context) error { return nil }

	gs.RegisterHook(hook1)
	gs.RegisterHook(hook2)
	gs.RegisterHook(hook3)

	if len(gs.shutdownHooks) != 3 {
		t.Errorf("Expected 3 hooks, got %d", len(gs.shutdownHooks))
	}
}

func TestShutdown_HookExecution(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}
	config := &ShutdownConfig{
		Timeout: 5 * time.Second,
		Logger:  logger,
	}
	gs := NewGracefulShutdown(server, config)

	var executionOrder []int
	var mu sync.Mutex

	// Register hooks in order
	for i := 1; i <= 3; i++ {
		index := i
		gs.RegisterHook(func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, index)
			mu.Unlock()
			return nil
		})
	}

	// Shutdown the server
	err := gs.Shutdown()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify hooks were executed in order
	mu.Lock()
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 hooks executed, got %d", len(executionOrder))
	}
	for i, val := range executionOrder {
		if val != i+1 {
			t.Errorf("Expected hook %d to execute at position %d, got %d", i+1, i, val)
		}
	}
	mu.Unlock()
}

func TestShutdown_HookFailure(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}
	config := &ShutdownConfig{
		Timeout: 5 * time.Second,
		Logger:  logger,
	}
	gs := NewGracefulShutdown(server, config)

	hook1Called := false
	hook2Called := false
	hook3Called := false

	// First hook succeeds
	gs.RegisterHook(func(ctx context.Context) error {
		hook1Called = true
		return nil
	})

	// Second hook fails
	gs.RegisterHook(func(ctx context.Context) error {
		hook2Called = true
		return errors.New("hook 2 failed")
	})

	// Third hook should still execute
	gs.RegisterHook(func(ctx context.Context) error {
		hook3Called = true
		return nil
	})

	err := gs.Shutdown()

	// Shutdown itself should succeed (hook errors are logged but not returned)
	if err != nil {
		t.Errorf("Expected no error from shutdown, got %v", err)
	}

	if !hook1Called {
		t.Error("Expected hook 1 to be called")
	}
	if !hook2Called {
		t.Error("Expected hook 2 to be called")
	}
	if !hook3Called {
		t.Error("Expected hook 3 to be called even after hook 2 failed")
	}

	// Check that error was logged
	if !logger.Contains("Shutdown hook %d failed: %v") {
		t.Error("Expected hook failure to be logged")
	}
}

func TestShutdown_Timeout(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}
	config := &ShutdownConfig{
		Timeout: 100 * time.Millisecond, // Short timeout
		Logger:  logger,
	}
	gs := NewGracefulShutdown(server, config)

	// Register a hook that takes longer than timeout
	gs.RegisterHook(func(ctx context.Context) error {
		select {
		case <-time.After(500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	start := time.Now()
	gs.Shutdown()
	duration := time.Since(start)

	// Should complete near the timeout (not wait for full hook duration)
	if duration > 300*time.Millisecond {
		t.Errorf("Shutdown took too long: %v (expected around 100ms)", duration)
	}
}

func TestShutdown_Idempotency(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}
	config := &ShutdownConfig{
		Timeout: 5 * time.Second,
		Logger:  logger,
	}
	gs := NewGracefulShutdown(server, config)

	hookCallCount := 0
	var mu sync.Mutex

	gs.RegisterHook(func(ctx context.Context) error {
		mu.Lock()
		hookCallCount++
		mu.Unlock()
		return nil
	})

	// Call Shutdown multiple times
	err1 := gs.Shutdown()
	err2 := gs.Shutdown()
	err3 := gs.Shutdown()

	if err1 != nil {
		t.Errorf("First shutdown error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second shutdown error: %v", err2)
	}
	if err3 != nil {
		t.Errorf("Third shutdown error: %v", err3)
	}

	// Hook should only be called once despite multiple Shutdown calls
	mu.Lock()
	if hookCallCount != 1 {
		t.Errorf("Expected hook to be called once, got %d times", hookCallCount)
	}
	mu.Unlock()
}

func TestShutdown_ConcurrentCalls(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	hookCallCount := 0
	var mu sync.Mutex

	gs.RegisterHook(func(ctx context.Context) error {
		mu.Lock()
		hookCallCount++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Call Shutdown concurrently from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gs.Shutdown()
		}()
	}

	wg.Wait()

	// Hook should only be called once despite concurrent calls
	mu.Lock()
	if hookCallCount != 1 {
		t.Errorf("Expected hook to be called once, got %d times", hookCallCount)
	}
	mu.Unlock()
}

func TestWait(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	// Start a goroutine that shuts down after a delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		gs.Shutdown()
	}()

	// Wait should block until shutdown completes
	start := time.Now()
	err := gs.Wait()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error from Wait, got %v", err)
	}

	// Should have waited at least 100ms
	if duration < 100*time.Millisecond {
		t.Errorf("Wait returned too quickly: %v", duration)
	}
}

func TestWait_AfterShutdown(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	// Shutdown first
	gs.Shutdown()

	// Wait should return immediately
	start := time.Now()
	err := gs.Wait()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error from Wait, got %v", err)
	}

	// Should return immediately
	if duration > 10*time.Millisecond {
		t.Errorf("Wait took too long after shutdown: %v", duration)
	}
}

func TestStartWithGracefulShutdown(t *testing.T) {
	// This test is difficult to fully test without signal handling
	// We'll just verify the function exists and doesn't panic with basic config

	server := createTestServer("127.0.0.1:0") // Use localhost to get an actual port
	config := &ShutdownConfig{
		Timeout: 1 * time.Second,
	}

	// We can't easily test the full Start() flow without sending signals,
	// but we can verify the setup doesn't panic
	gs := NewGracefulShutdown(server, config)
	if gs == nil {
		t.Error("Expected graceful shutdown instance")
	}
}

func TestShutdown_LogMessages(t *testing.T) {
	server := createTestServer(":0")
	logger := &mockLogger{}
	config := &ShutdownConfig{
		Timeout: 5 * time.Second,
		Logger:  logger,
	}
	gs := NewGracefulShutdown(server, config)

	gs.RegisterHook(func(ctx context.Context) error {
		return nil
	})

	gs.Shutdown()

	// Check expected log messages
	if !logger.Contains("Initiating graceful shutdown (timeout: %v)") {
		t.Error("Expected shutdown initiation message")
	}

	if !logger.Contains("Running %d shutdown hooks...") {
		t.Error("Expected hooks running message")
	}

	if !logger.Contains("Shutting down HTTP server...") {
		t.Error("Expected server shutdown message")
	}

	if !logger.Contains("Server shutdown completed successfully") {
		t.Error("Expected shutdown success message")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := &defaultLogger{}

	// Should not panic
	logger.Printf("Test message: %s", "value")
}

func TestGracefulShutdown_NilServer(t *testing.T) {
	// Test that we handle nil server gracefully (should panic or error appropriately)
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil server
			t.Log("Correctly panicked with nil server")
		}
	}()

	gs := NewGracefulShutdown(nil, nil)
	if gs.server == nil {
		// If we get here, the function accepted nil
		// Shutdown should fail
		err := gs.Shutdown()
		if err == nil {
			t.Error("Expected error when shutting down nil server")
		}
	}
}

func TestRegisterHook_ThreadSafety(t *testing.T) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	var wg sync.WaitGroup
	hookCount := 100

	// Register hooks concurrently
	for i := 0; i < hookCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gs.RegisterHook(func(ctx context.Context) error {
				return nil
			})
		}()
	}

	wg.Wait()

	// All hooks should be registered
	if len(gs.shutdownHooks) != hookCount {
		t.Errorf("Expected %d hooks, got %d", hookCount, len(gs.shutdownHooks))
	}
}

func BenchmarkShutdown_NoHooks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server := createTestServer(":0")
		gs := NewGracefulShutdown(server, nil)
		gs.Shutdown()
	}
}

func BenchmarkShutdown_WithHooks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server := createTestServer(":0")
		gs := NewGracefulShutdown(server, nil)

		for j := 0; j < 10; j++ {
			gs.RegisterHook(func(ctx context.Context) error {
				return nil
			})
		}

		gs.Shutdown()
	}
}

func BenchmarkRegisterHook(b *testing.B) {
	server := createTestServer(":0")
	gs := NewGracefulShutdown(server, nil)

	hook := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gs.RegisterHook(hook)
	}
}
