package response

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// Streamer handles streaming responses
type Streamer struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewStreamer creates a new response streamer
func NewStreamer(w http.ResponseWriter) (*Streamer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &Streamer{
		writer:  w,
		flusher: flusher,
	}, nil
}

// StreamJSON streams JSON objects one by one (JSON Lines format)
func (s *Streamer) StreamJSON(objects <-chan interface{}) error {
	s.writer.Header().Set("Content-Type", "application/x-ndjson")
	s.writer.Header().Set("X-Content-Type-Options", "nosniff")
	s.writer.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(s.writer)

	for obj := range objects {
		if err := encoder.Encode(obj); err != nil {
			return fmt.Errorf("failed to encode object: %w", err)
		}
		s.flusher.Flush()
	}

	return nil
}

// StreamCSV streams CSV rows
func (s *Streamer) StreamCSV(headers []string, rows <-chan []string) error {
	s.writer.Header().Set("Content-Type", "text/csv")
	s.writer.Header().Set("X-Content-Type-Options", "nosniff")
	s.writer.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(s.writer)

	// Write headers
	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
		writer.Flush()
		s.flusher.Flush()
	}

	// Write rows
	for row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
		writer.Flush()
		s.flusher.Flush()
	}

	return nil
}

// StreamText streams text line by line
func (s *Streamer) StreamText(lines <-chan string) error {
	s.writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	s.writer.Header().Set("X-Content-Type-Options", "nosniff")
	s.writer.WriteHeader(http.StatusOK)

	writer := bufio.NewWriter(s.writer)

	for line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
		writer.Flush()
		s.flusher.Flush()
	}

	return nil
}

// StreamReader streams content from an io.Reader
func (s *Streamer) StreamReader(contentType string, reader io.Reader) error {
	s.writer.Header().Set("Content-Type", contentType)
	s.writer.Header().Set("X-Content-Type-Options", "nosniff")
	s.writer.WriteHeader(http.StatusOK)

	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if _, writeErr := s.writer.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			s.flusher.Flush()
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read: %w", err)
		}
	}

	return nil
}

// StreamSSE streams Server-Sent Events
func (s *Streamer) StreamSSE(events <-chan SSEEvent) error {
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")
	s.writer.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	s.writer.WriteHeader(http.StatusOK)

	writer := bufio.NewWriter(s.writer)

	for event := range events {
		// Write event ID if present
		if event.ID != "" {
			fmt.Fprintf(writer, "id: %s\n", event.ID)
		}

		// Write event type if present
		if event.Event != "" {
			fmt.Fprintf(writer, "event: %s\n", event.Event)
		}

		// Write retry if present
		if event.Retry > 0 {
			fmt.Fprintf(writer, "retry: %d\n", event.Retry)
		}

		// Write data
		fmt.Fprintf(writer, "data: %s\n\n", event.Data)

		writer.Flush()
		s.flusher.Flush()
	}

	return nil
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string // Event ID
	Event string // Event type
	Data  string // Event data
	Retry int    // Retry timeout in milliseconds
}

// NewSSEEvent creates a new SSE event
func NewSSEEvent(data string) SSEEvent {
	return SSEEvent{
		Data: data,
	}
}

// WithID sets the event ID
func (e SSEEvent) WithID(id string) SSEEvent {
	e.ID = id
	return e
}

// WithEvent sets the event type
func (e SSEEvent) WithEvent(event string) SSEEvent {
	e.Event = event
	return e
}

// WithRetry sets the retry timeout
func (e SSEEvent) WithRetry(retry int) SSEEvent {
	e.Retry = retry
	return e
}

// StreamJSONArray streams a JSON array incrementally
func StreamJSONArray(w http.ResponseWriter, objects <-chan interface{}) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Write opening bracket
	w.Write([]byte("["))
	flusher.Flush()

	encoder := json.NewEncoder(w)
	first := true

	for obj := range objects {
		if !first {
			w.Write([]byte(","))
		}
		first = false

		if err := encoder.Encode(obj); err != nil {
			return fmt.Errorf("failed to encode object: %w", err)
		}
		flusher.Flush()
	}

	// Write closing bracket
	w.Write([]byte("]"))
	flusher.Flush()

	return nil
}

// StreamFile streams a file with proper headers
func StreamFile(w http.ResponseWriter, req *http.Request, filePath string, filename string, contentType string, allowedDirs []string) error {
	// Validate and clean the file path
	validatedPath, err := validateFilePath(filePath, allowedDirs)
	if err != nil {
		return err
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Use http.ServeFile for efficient file streaming
	http.ServeFile(w, req, validatedPath)
	return nil
}

// validateFilePath validates and cleans a file path to prevent directory traversal
func validateFilePath(filePath string, allowedDirs []string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(filePath)

	// Resolve symlinks to prevent bypass
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// If no allowed directories configured, reject all file access
	if len(allowedDirs) == 0 {
		return "", fmt.Errorf("file serving not configured: no allowed directories set")
	}

	// Check if the resolved path is within one of the allowed directories
	allowed := false
	for _, allowedDir := range allowedDirs {
		// Clean and resolve the allowed directory
		cleanAllowedDir := filepath.Clean(allowedDir)
		resolvedAllowedDir, err := filepath.EvalSymlinks(cleanAllowedDir)
		if err != nil {
			// Skip directories that don't exist or can't be resolved
			continue
		}

		// Check if the file is within this allowed directory
		if strings.HasPrefix(resolvedPath, resolvedAllowedDir+string(filepath.Separator)) ||
			resolvedPath == resolvedAllowedDir {
			allowed = true
			break
		}
	}

	if !allowed {
		return "", fmt.Errorf("file path not in allowed directories")
	}

	return resolvedPath, nil
}

// ChunkedWriter wraps an http.ResponseWriter for chunked transfer encoding
type ChunkedWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewChunkedWriter creates a new chunked writer
func NewChunkedWriter(w http.ResponseWriter) (*ChunkedWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &ChunkedWriter{
		writer:  w,
		flusher: flusher,
	}, nil
}

// Write writes data and immediately flushes
func (cw *ChunkedWriter) Write(data []byte) (int, error) {
	n, err := cw.writer.Write(data)
	if err != nil {
		return n, err
	}
	cw.flusher.Flush()
	return n, nil
}

// WriteString writes a string and immediately flushes
func (cw *ChunkedWriter) WriteString(s string) (int, error) {
	return cw.Write([]byte(s))
}

// Flush manually flushes the writer
func (cw *ChunkedWriter) Flush() {
	cw.flusher.Flush()
}
