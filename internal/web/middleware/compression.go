package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// CompressionConfig holds configuration for the compression middleware
type CompressionConfig struct {
	// Level is the gzip compression level (1-9, default 6)
	Level int
	// MinSize is the minimum response size to compress (in bytes)
	MinSize int
	// ExcludedContentTypes is a list of content types to exclude from compression
	ExcludedContentTypes []string
	// ExcludedPaths is a list of paths to exclude from compression
	ExcludedPaths []string
}

// DefaultCompressionConfig returns the default compression configuration
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 1024, // 1KB minimum
		ExcludedContentTypes: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"video/",
			"audio/",
		},
		ExcludedPaths: []string{},
	}
}

// Compression creates a compression middleware with default configuration
func Compression() Middleware {
	return CompressionWithConfig(DefaultCompressionConfig())
}

// CompressionWithConfig creates a compression middleware with custom configuration
func CompressionWithConfig(config CompressionConfig) Middleware {
	// Create a pool of gzip writers for reuse
	gzipPool := &sync.Pool{
		New: func() interface{} {
			writer, _ := gzip.NewWriterLevel(io.Discard, config.Level)
			return writer
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is excluded
			for _, path := range config.ExcludedPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Wrap response writer with gzip writer
			gzw := &gzipResponseWriter{
				ResponseWriter: w,
				gzipPool:       gzipPool,
				config:         config,
			}
			defer gzw.Close()

			// Set encoding header (will be removed if compression not used)
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			next.ServeHTTP(gzw, r)
		})
	}
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter     *gzip.Writer
	gzipPool       *sync.Pool
	config         CompressionConfig
	wroteHeader    bool
	shouldCompress bool
	checkedType    bool
}

// WriteHeader checks content type and decides whether to compress
func (gzw *gzipResponseWriter) WriteHeader(statusCode int) {
	if !gzw.wroteHeader {
		gzw.wroteHeader = true

		// Check content type
		if !gzw.checkedType {
			gzw.checkContentType()
		}

		// Remove Content-Encoding header if not compressing
		if !gzw.shouldCompress {
			gzw.ResponseWriter.Header().Del("Content-Encoding")
			gzw.ResponseWriter.Header().Del("Vary")
		}

		gzw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write compresses data if compression is enabled
func (gzw *gzipResponseWriter) Write(b []byte) (int, error) {
	if !gzw.wroteHeader {
		gzw.WriteHeader(http.StatusOK)
	}

	// Check content type on first write if not already checked
	if !gzw.checkedType {
		gzw.checkContentType()
	}

	// If not compressing, write directly
	if !gzw.shouldCompress {
		return gzw.ResponseWriter.Write(b)
	}

	// Check minimum size
	if len(b) < gzw.config.MinSize && gzw.gzipWriter == nil {
		gzw.shouldCompress = false
		gzw.ResponseWriter.Header().Del("Content-Encoding")
		gzw.ResponseWriter.Header().Del("Vary")
		return gzw.ResponseWriter.Write(b)
	}

	// Initialize gzip writer if needed
	if gzw.gzipWriter == nil {
		gzw.gzipWriter = gzw.gzipPool.Get().(*gzip.Writer)
		gzw.gzipWriter.Reset(gzw.ResponseWriter)
	}

	// Write compressed data
	return gzw.gzipWriter.Write(b)
}

// Close flushes and closes the gzip writer
func (gzw *gzipResponseWriter) Close() error {
	if gzw.gzipWriter != nil {
		err := gzw.gzipWriter.Close()
		gzw.gzipPool.Put(gzw.gzipWriter)
		gzw.gzipWriter = nil
		return err
	}
	return nil
}

// checkContentType determines if the content type should be compressed
func (gzw *gzipResponseWriter) checkContentType() {
	gzw.checkedType = true
	contentType := gzw.ResponseWriter.Header().Get("Content-Type")

	// Default to compress if no content type
	if contentType == "" {
		gzw.shouldCompress = true
		return
	}

	// Check excluded content types
	for _, excluded := range gzw.config.ExcludedContentTypes {
		if strings.HasPrefix(contentType, excluded) {
			gzw.shouldCompress = false
			return
		}
	}

	gzw.shouldCompress = true
}
