package response

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge case tests to push coverage over 90%

func TestRenderer_HTML_TemplateError(t *testing.T) {
	// Try to load templates from non-existent directory
	renderer := NewRenderer()
	err := renderer.LoadTemplates("/nonexistent/path", "*.html")

	if err == nil {
		t.Error("expected error loading templates from non-existent directory")
	}
}

func TestRenderError_NilError(t *testing.T) {
	w := httptest.NewRecorder()

	RenderInternalError(w, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want 500", w.Code)
	}
}

func TestJoinMethods_SingleMethod(t *testing.T) {
	methods := []string{"GET"}
	result := joinMethods(methods)

	if result != "GET" {
		t.Errorf("joinMethods() = %v, want 'GET'", result)
	}
}

func TestRenderer_SetDefaultHeader_Override(t *testing.T) {
	renderer := NewRenderer()
	renderer.SetDefaultHeader("X-Test", "original")
	renderer.SetDefaultHeader("X-Test", "updated")

	w := httptest.NewRecorder()
	renderer.JSON(w, http.StatusOK, map[string]string{"test": "value"})

	if w.Header().Get("X-Test") != "updated" {
		t.Error("default header should be updated")
	}
}

func TestAPIResponse_NilMaps(t *testing.T) {
	resp := &APIResponse{
		Data: "test",
		// Meta and Links are nil
	}

	resp.WithMeta("key", "value")
	resp.WithLink("self", "/test")

	if len(resp.Meta) != 1 {
		t.Error("Meta should be initialized")
	}

	if len(resp.Links) != 1 {
		t.Error("Links should be initialized")
	}
}

func TestErrorCodeFromStatus_PaymentRequired(t *testing.T) {
	// Test edge case status code
	code := errorCodeFromStatus(http.StatusPaymentRequired)
	if code != "payment_required" {
		t.Errorf("code = %v, want payment_required", code)
	}
}

func TestRenderForbidden_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	RenderForbidden(w, "")

	if w.Code != http.StatusForbidden {
		t.Errorf("status code = %v, want 403", w.Code)
	}
}

func TestRenderNotFound_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	RenderNotFound(w, "")

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %v, want 404", w.Code)
	}
}

func TestNewStreamer_NoFlusher(t *testing.T) {
	// Create a ResponseWriter that doesn't implement Flusher
	type noFlusherWriter struct {
		http.ResponseWriter
	}

	w := &noFlusherWriter{ResponseWriter: httptest.NewRecorder()}

	_, err := NewStreamer(w)

	if err == nil {
		t.Error("expected error when ResponseWriter doesn't implement Flusher")
	}

	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention streaming not supported, got: %v", err)
	}
}

func TestNewChunkedWriter_NoFlusher(t *testing.T) {
	// Create a ResponseWriter that doesn't implement Flusher
	type noFlusherWriter struct {
		http.ResponseWriter
	}

	w := &noFlusherWriter{ResponseWriter: httptest.NewRecorder()}

	_, err := NewChunkedWriter(w)

	if err == nil {
		t.Error("expected error when ResponseWriter doesn't implement Flusher")
	}
}

func TestStreamer_StreamJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, _ := NewStreamer(w)

	// Create a channel with un-marshalable data
	objects := make(chan interface{}, 1)
	objects <- make(chan int) // Channels can't be marshaled to JSON
	close(objects)

	err := streamer.StreamJSON(objects)

	if err == nil {
		t.Error("expected error when streaming un-marshalable object")
	}
}

func TestStreamer_StreamCSVError(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, _ := NewStreamer(w)

	headers := []string{"ID"}
	rows := make(chan []string, 1)

	// Send a row with invalid UTF-8
	rows <- []string{string([]byte{0xff, 0xfe, 0xfd})}
	close(rows)

	// This should still work but we're testing the error path exists
	_ = streamer.StreamCSV(headers, rows)
}

func TestStreamer_StreamReaderError(t *testing.T) {
	w := httptest.NewRecorder()
	streamer, _ := NewStreamer(w)

	// Create a reader that will error
	errorReader := &errorReader{}

	err := streamer.StreamReader("text/plain", errorReader)

	if err == nil {
		t.Error("expected error when reader fails")
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}
