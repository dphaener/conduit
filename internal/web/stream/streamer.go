package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Streamer provides utilities for streaming HTTP responses
type Streamer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	writer  io.Writer
	encoder *json.Encoder
}

// New creates a new response streamer
func New(w http.ResponseWriter) (*Streamer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &Streamer{
		w:       w,
		flusher: flusher,
		writer:  w,
	}, nil
}

// NewJSON creates a JSON response streamer
func NewJSON(w http.ResponseWriter) (*Streamer, error) {
	s, err := New(w)
	if err != nil {
		return nil, err
	}

	s.w.Header().Set("Content-Type", "application/json")
	s.w.Header().Set("Transfer-Encoding", "chunked")
	s.encoder = json.NewEncoder(s.writer)

	return s, nil
}

// NewText creates a text response streamer
func NewText(w http.ResponseWriter) (*Streamer, error) {
	s, err := New(w)
	if err != nil {
		return nil, err
	}

	s.w.Header().Set("Content-Type", "text/plain")
	s.w.Header().Set("Transfer-Encoding", "chunked")

	return s, nil
}

// NewSSE creates a Server-Sent Events streamer
func NewSSE(w http.ResponseWriter) (*Streamer, error) {
	s, err := New(w)
	if err != nil {
		return nil, err
	}

	s.w.Header().Set("Content-Type", "text/event-stream")
	s.w.Header().Set("Cache-Control", "no-cache")
	s.w.Header().Set("Connection", "keep-alive")
	s.w.Header().Set("Transfer-Encoding", "chunked")

	return s, nil
}

// Write writes bytes to the stream and flushes
func (s *Streamer) Write(data []byte) (int, error) {
	n, err := s.writer.Write(data)
	if err != nil {
		return n, err
	}
	s.flusher.Flush()
	return n, nil
}

// WriteString writes a string to the stream and flushes
func (s *Streamer) WriteString(str string) error {
	_, err := s.Write([]byte(str))
	return err
}

// WriteJSON writes a JSON object to the stream and flushes
func (s *Streamer) WriteJSON(v interface{}) error {
	if s.encoder == nil {
		s.encoder = json.NewEncoder(s.writer)
	}

	if err := s.encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// WriteJSONArray streams a JSON array one element at a time
func (s *Streamer) WriteJSONArray(items <-chan interface{}) error {
	return s.WriteJSONArrayContext(context.Background(), items)
}

// WriteJSONArrayContext streams a JSON array one element at a time with context cancellation
func (s *Streamer) WriteJSONArrayContext(ctx context.Context, items <-chan interface{}) error {
	// Write opening bracket
	if _, err := s.Write([]byte("[")); err != nil {
		return err
	}

	first := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-items:
			if !ok {
				// Channel closed, write closing bracket
				if _, err := s.Write([]byte("]")); err != nil {
					return err
				}
				return nil
			}

			// Write comma separator (except for first item)
			if !first {
				if _, err := s.Write([]byte(",")); err != nil {
					return err
				}
			}
			first = false

			// Write item
			if err := s.WriteJSON(item); err != nil {
				return err
			}
		}
	}
}

// WriteSSE writes a Server-Sent Event
func (s *Streamer) WriteSSE(event *SSEEvent) error {
	if event.ID != "" {
		if err := s.WriteString(fmt.Sprintf("id: %s\n", event.ID)); err != nil {
			return err
		}
	}

	if event.Event != "" {
		if err := s.WriteString(fmt.Sprintf("event: %s\n", event.Event)); err != nil {
			return err
		}
	}

	if event.Retry > 0 {
		if err := s.WriteString(fmt.Sprintf("retry: %d\n", event.Retry)); err != nil {
			return err
		}
	}

	if event.Data != "" {
		if err := s.WriteString(fmt.Sprintf("data: %s\n", event.Data)); err != nil {
			return err
		}
	}

	// End of event (double newline)
	return s.WriteString("\n")
}

// Flush manually flushes the stream
func (s *Streamer) Flush() {
	s.flusher.Flush()
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// StreamHandler wraps a handler function that uses streaming
func StreamHandler(handler func(*Streamer) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streamer, err := New(w)
		if err != nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		if err := handler(streamer); err != nil {
			// Can't send error response after streaming has started
			// Log the error instead
			fmt.Printf("Streaming error: %v\n", err)
		}
	}
}

// JSONArrayHandler creates a handler that streams a JSON array
func JSONArrayHandler(producer func() <-chan interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streamer, err := NewJSON(w)
		if err != nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		items := producer()
		if err := streamer.WriteJSONArray(items); err != nil {
			fmt.Printf("JSON streaming error: %v\n", err)
		}
	}
}

// SSEHandler creates a handler for Server-Sent Events
func SSEHandler(producer func() <-chan *SSEEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streamer, err := NewSSE(w)
		if err != nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		events := producer()
		for event := range events {
			if err := streamer.WriteSSE(event); err != nil {
				fmt.Printf("SSE streaming error: %v\n", err)
				return
			}
		}
	}
}

// ChunkedWriter wraps a response writer with chunked encoding
type ChunkedWriter struct {
	w       *bufio.Writer
	flusher http.Flusher
}

// NewChunkedWriter creates a new chunked response writer
func NewChunkedWriter(w http.ResponseWriter) (*ChunkedWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Transfer-Encoding", "chunked")

	return &ChunkedWriter{
		w:       bufio.NewWriter(w),
		flusher: flusher,
	}, nil
}

// Write writes data in chunks
func (cw *ChunkedWriter) Write(data []byte) (int, error) {
	n, err := cw.w.Write(data)
	if err != nil {
		return n, err
	}

	if err := cw.w.Flush(); err != nil {
		return n, err
	}

	cw.flusher.Flush()
	return n, nil
}

// Flush flushes the buffered data
func (cw *ChunkedWriter) Flush() error {
	if err := cw.w.Flush(); err != nil {
		return err
	}
	cw.flusher.Flush()
	return nil
}
