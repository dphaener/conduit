package stream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := New(rec)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if streamer == nil {
		t.Fatal("Expected streamer, got nil")
	}
}

func TestNew_StreamingNotSupported(t *testing.T) {
	// Create a non-flushable response writer
	type nonFlushable struct {
		http.ResponseWriter
	}

	w := &nonFlushable{ResponseWriter: httptest.NewRecorder()}
	_, err := New(w)

	if err == nil {
		t.Error("Expected error for non-flushable writer")
	}

	expectedMsg := "streaming not supported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestNewJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	_, err := NewJSON(rec)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}

	transferEncoding := rec.Header().Get("Transfer-Encoding")
	if transferEncoding != "chunked" {
		t.Errorf("Expected Transfer-Encoding 'chunked', got %q", transferEncoding)
	}
}

func TestNewText(t *testing.T) {
	rec := httptest.NewRecorder()
	_, err := NewText(rec)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got %q", contentType)
	}

	transferEncoding := rec.Header().Get("Transfer-Encoding")
	if transferEncoding != "chunked" {
		t.Errorf("Expected Transfer-Encoding 'chunked', got %q", transferEncoding)
	}
}

func TestNewSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	_, err := NewSSE(rec)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %q", contentType)
	}

	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got %q", cacheControl)
	}

	connection := rec.Header().Get("Connection")
	if connection != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got %q", connection)
	}

	transferEncoding := rec.Header().Get("Transfer-Encoding")
	if transferEncoding != "chunked" {
		t.Errorf("Expected Transfer-Encoding 'chunked', got %q", transferEncoding)
	}
}

func TestStreamer_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := New(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data := []byte("Hello, World!")
	n, err := streamer.Write(data)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	body := rec.Body.String()
	if body != string(data) {
		t.Errorf("Expected body %q, got %q", string(data), body)
	}
}

func TestStreamer_WriteString(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := New(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	text := "Hello, World!"
	err = streamer.WriteString(text)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := rec.Body.String()
	if body != text {
		t.Errorf("Expected body %q, got %q", text, body)
	}
}

func TestStreamer_WriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data := map[string]string{"message": "Hello, World!"}
	err = streamer.WriteJSON(data)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["message"] != data["message"] {
		t.Errorf("Expected message %q, got %q", data["message"], result["message"])
	}
}

func TestStreamer_WriteJSONArray(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Create channel with test items
	items := make(chan interface{}, 3)
	items <- map[string]int{"id": 1}
	items <- map[string]int{"id": 2}
	items <- map[string]int{"id": 3}
	close(items)

	err = streamer.WriteJSONArray(items)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the result as a JSON array
	var result []map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON array: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	for i, item := range result {
		if item["id"] != i+1 {
			t.Errorf("Expected id %d, got %d", i+1, item["id"])
		}
	}
}

func TestStreamer_WriteJSONArrayContext_Cancellation(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel that will block
	items := make(chan interface{})

	// Cancel the context immediately
	cancel()

	// WriteJSONArrayContext should return context.Canceled
	err = streamer.WriteJSONArrayContext(ctx, items)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestStreamer_WriteJSONArrayContext_Timeout(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Create a channel that will never send data
	items := make(chan interface{})

	// WriteJSONArrayContext should timeout
	err = streamer.WriteJSONArrayContext(ctx, items)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestStreamer_WriteJSONArrayContext_EmptyChannel(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Create an empty channel and close it
	items := make(chan interface{})
	close(items)

	err = streamer.WriteJSONArrayContext(context.Background(), items)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have written an empty array
	body := rec.Body.String()
	if body != "[]" {
		t.Errorf("Expected empty array '[]', got %q", body)
	}
}

func TestStreamer_WriteSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewSSE(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	event := &SSEEvent{
		ID:    "123",
		Event: "message",
		Data:  "Hello, World!",
		Retry: 5000,
	}

	err = streamer.WriteSSE(event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := rec.Body.String()
	expectedLines := []string{
		"id: 123",
		"event: message",
		"retry: 5000",
		"data: Hello, World!",
		"",
	}

	for _, line := range expectedLines {
		if !strings.Contains(body, line) {
			t.Errorf("Expected body to contain %q, got %q", line, body)
		}
	}
}

func TestStreamer_WriteSSE_MinimalEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewSSE(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	event := &SSEEvent{
		Data: "Hello",
	}

	err = streamer.WriteSSE(event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := rec.Body.String()
	expected := "data: Hello\n\n"
	if body != expected {
		t.Errorf("Expected body %q, got %q", expected, body)
	}
}

func TestStreamer_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := New(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Flush should not panic
	streamer.Flush()
}

func TestStreamHandler(t *testing.T) {
	handler := StreamHandler(func(s *Streamer) error {
		return s.WriteString("Hello, World!")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if body != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got %q", body)
	}
}

func TestStreamHandler_Error(t *testing.T) {
	handler := StreamHandler(func(s *Streamer) error {
		s.WriteString("Start")
		return &testError{msg: "test error"}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Error should be logged but not return error response (streaming already started)
	body := rec.Body.String()
	if !strings.Contains(body, "Start") {
		t.Errorf("Expected body to contain 'Start', got %q", body)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestJSONArrayHandler(t *testing.T) {
	handler := JSONArrayHandler(func() <-chan interface{} {
		items := make(chan interface{}, 2)
		items <- map[string]int{"id": 1}
		items <- map[string]int{"id": 2}
		close(items)
		return items
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var result []map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

func TestSSEHandler(t *testing.T) {
	handler := SSEHandler(func() <-chan *SSEEvent {
		events := make(chan *SSEEvent, 2)
		events <- &SSEEvent{Data: "Event 1"}
		events <- &SSEEvent{Data: "Event 2"}
		close(events)
		return events
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "data: Event 1") {
		t.Error("Expected body to contain 'data: Event 1'")
	}
	if !strings.Contains(body, "data: Event 2") {
		t.Error("Expected body to contain 'data: Event 2'")
	}
}

func TestNewChunkedWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	cw, err := NewChunkedWriter(rec)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cw == nil {
		t.Fatal("Expected chunked writer, got nil")
	}

	transferEncoding := rec.Header().Get("Transfer-Encoding")
	if transferEncoding != "chunked" {
		t.Errorf("Expected Transfer-Encoding 'chunked', got %q", transferEncoding)
	}
}

func TestChunkedWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	cw, err := NewChunkedWriter(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data := []byte("Hello, World!")
	n, err := cw.Write(data)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
}

func TestChunkedWriter_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	cw, err := NewChunkedWriter(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = cw.Flush()
	if err != nil {
		t.Fatalf("Expected no error on flush, got %v", err)
	}
}

func TestConcurrentStreamOperations(t *testing.T) {
	rec := httptest.NewRecorder()
	streamer, err := NewJSON(rec)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Create a channel for concurrent writes
	items := make(chan interface{}, 10)

	// Spawn goroutine to write items
	go func() {
		for i := 0; i < 10; i++ {
			items <- map[string]int{"id": i}
		}
		close(items)
	}()

	err = streamer.WriteJSONArray(items)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var result []map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result))
	}
}

func BenchmarkStreamer_WriteJSON(b *testing.B) {
	rec := httptest.NewRecorder()
	streamer, _ := NewJSON(rec)

	data := map[string]string{"message": "Hello, World!"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		streamer.WriteJSON(data)
	}
}

func BenchmarkStreamer_WriteJSONArray(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		streamer, _ := NewJSON(rec)

		items := make(chan interface{}, 100)
		go func() {
			for j := 0; j < 100; j++ {
				items <- map[string]int{"id": j}
			}
			close(items)
		}()

		streamer.WriteJSONArray(items)
	}
}

func BenchmarkStreamer_WriteSSE(b *testing.B) {
	rec := httptest.NewRecorder()
	streamer, _ := NewSSE(rec)

	event := &SSEEvent{
		ID:    "123",
		Event: "message",
		Data:  "Hello, World!",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		streamer.WriteSSE(event)
	}
}
