package response

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamer_StreamJSON(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, err := NewStreamer(w)
	if err != nil {
		t.Fatalf("NewStreamer() error = %v", err)
	}

	// Create channel with test data
	objects := make(chan interface{}, 3)
	objects <- map[string]string{"id": "1", "name": "Alice"}
	objects <- map[string]string{"id": "2", "name": "Bob"}
	objects <- map[string]string{"id": "3", "name": "Charlie"}
	close(objects)

	err = streamer.StreamJSON(objects)
	if err != nil {
		t.Errorf("StreamJSON() error = %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/x-ndjson" {
		t.Errorf("content type = %v, want application/x-ndjson", contentType)
	}

	// Each line should be a valid JSON object
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %v", len(lines))
	}
}

func TestStreamer_StreamCSV(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, err := NewStreamer(w)
	if err != nil {
		t.Fatalf("NewStreamer() error = %v", err)
	}

	headers := []string{"ID", "Name", "Email"}
	rows := make(chan []string, 2)
	rows <- []string{"1", "Alice", "alice@example.com"}
	rows <- []string{"2", "Bob", "bob@example.com"}
	close(rows)

	err = streamer.StreamCSV(headers, rows)
	if err != nil {
		t.Errorf("StreamCSV() error = %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("content type = %v, want text/csv", contentType)
	}

	body := w.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("expected 3 lines, got %v", len(lines))
	}

	if !strings.Contains(lines[0], "ID,Name,Email") {
		t.Errorf("header line = %v, want 'ID,Name,Email'", lines[0])
	}
}

func TestStreamer_StreamText(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, err := NewStreamer(w)
	if err != nil {
		t.Fatalf("NewStreamer() error = %v", err)
	}

	lines := make(chan string, 3)
	lines <- "Line 1"
	lines <- "Line 2"
	lines <- "Line 3"
	close(lines)

	err = streamer.StreamText(lines)
	if err != nil {
		t.Errorf("StreamText() error = %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("content type = %v, want text/plain", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Line 1\n") {
		t.Error("body should contain 'Line 1'")
	}
	if !strings.Contains(body, "Line 2\n") {
		t.Error("body should contain 'Line 2'")
	}
}

func TestStreamer_StreamReader(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, err := NewStreamer(w)
	if err != nil {
		t.Fatalf("NewStreamer() error = %v", err)
	}

	reader := strings.NewReader("test content")
	err = streamer.StreamReader("text/plain", reader)
	if err != nil {
		t.Errorf("StreamReader() error = %v", err)
	}

	if w.Body.String() != "test content" {
		t.Errorf("body = %v, want 'test content'", w.Body.String())
	}
}

func TestStreamer_StreamSSE(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, err := NewStreamer(w)
	if err != nil {
		t.Fatalf("NewStreamer() error = %v", err)
	}

	events := make(chan SSEEvent, 2)
	events <- NewSSEEvent("Event 1").WithID("1").WithEvent("message")
	events <- NewSSEEvent("Event 2").WithID("2").WithRetry(1000)
	close(events)

	err = streamer.StreamSSE(events)
	if err != nil {
		t.Errorf("StreamSSE() error = %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("content type = %v, want text/event-stream", contentType)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("cache control = %v, want no-cache", cacheControl)
	}

	body := w.Body.String()
	if !strings.Contains(body, "id: 1") {
		t.Error("body should contain event ID")
	}
	if !strings.Contains(body, "event: message") {
		t.Error("body should contain event type")
	}
	if !strings.Contains(body, "data: Event 1") {
		t.Error("body should contain event data")
	}
}

func TestSSEEvent(t *testing.T) {
	event := NewSSEEvent("test data")
	event = event.WithID("123")
	event = event.WithEvent("update")
	event = event.WithRetry(5000)

	if event.ID != "123" {
		t.Errorf("ID = %v, want 123", event.ID)
	}
	if event.Event != "update" {
		t.Errorf("Event = %v, want update", event.Event)
	}
	if event.Data != "test data" {
		t.Errorf("Data = %v, want 'test data'", event.Data)
	}
	if event.Retry != 5000 {
		t.Errorf("Retry = %v, want 5000", event.Retry)
	}
}

func TestStreamJSONArray(t *testing.T) {
	w := httptest.NewRecorder()

	objects := make(chan interface{}, 3)
	objects <- map[string]string{"id": "1"}
	objects <- map[string]string{"id": "2"}
	objects <- map[string]string{"id": "3"}
	close(objects)

	err := StreamJSONArray(w, objects)
	if err != nil {
		t.Errorf("StreamJSONArray() error = %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("content type = %v, want application/json", contentType)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "[") {
		t.Error("body should start with '['")
	}
	if !strings.HasSuffix(strings.TrimSpace(body), "]") {
		t.Error("body should end with ']'")
	}
}

func TestChunkedWriter(t *testing.T) {
	w := httptest.NewRecorder()
	cw, err := NewChunkedWriter(w)
	if err != nil {
		t.Fatalf("NewChunkedWriter() error = %v", err)
	}

	n, err := cw.Write([]byte("chunk 1"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 7 {
		t.Errorf("wrote %v bytes, want 7", n)
	}

	n, err = cw.WriteString("chunk 2")
	if err != nil {
		t.Errorf("WriteString() error = %v", err)
	}
	if n != 7 {
		t.Errorf("wrote %v bytes, want 7", n)
	}

	body := w.Body.String()
	if body != "chunk 1chunk 2" {
		t.Errorf("body = %v, want 'chunk 1chunk 2'", body)
	}
}

// Benchmark tests
func BenchmarkStreamer_StreamJSON(b *testing.B) {
	w := httptest.NewRecorder()
	streamer, _ := NewStreamer(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objects := make(chan interface{}, 100)
		for j := 0; j < 100; j++ {
			objects <- map[string]int{"id": j}
		}
		close(objects)

		streamer.StreamJSON(objects)
		w.Body.Reset()
	}
}

func BenchmarkStreamer_StreamCSV(b *testing.B) {
	w := httptest.NewRecorder()
	streamer, _ := NewStreamer(w)

	headers := []string{"ID", "Name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := make(chan []string, 100)
		for j := 0; j < 100; j++ {
			rows <- []string{"1", "Test"}
		}
		close(rows)

		streamer.StreamCSV(headers, rows)
		w.Body.Reset()
	}
}
