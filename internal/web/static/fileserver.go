package static

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileServerConfig holds configuration for the static file server
type FileServerConfig struct {
	// Root is the root directory to serve files from
	Root string

	// Prefix is the URL prefix to strip (e.g., "/static")
	Prefix string

	// MaxAge is the cache duration in seconds (default: 1 year)
	MaxAge int

	// EnableETag enables ETag header generation
	EnableETag bool

	// EnableGzip enables gzip compression for text files
	EnableGzip bool

	// IndexFile is the default file to serve for directories (default: "index.html")
	IndexFile string

	// NotFoundHandler is called when a file is not found
	NotFoundHandler http.HandlerFunc

	// ETags is a cache of file ETags
	etagCache *sync.Map
}

// DefaultFileServerConfig returns default static file server configuration
func DefaultFileServerConfig(root string) *FileServerConfig {
	return &FileServerConfig{
		Root:       root,
		Prefix:     "/static",
		MaxAge:     31536000, // 1 year
		EnableETag: true,
		EnableGzip: true,
		IndexFile:  "index.html",
		etagCache:  &sync.Map{},
	}
}

// FileServer creates an optimized static file server
func FileServer(config *FileServerConfig) http.Handler {
	if config.etagCache == nil {
		config.etagCache = &sync.Map{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET and HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Clean the URL path
		urlPath := r.URL.Path
		if config.Prefix != "" {
			urlPath = strings.TrimPrefix(urlPath, config.Prefix)
		}
		urlPath = path.Clean(urlPath)

		// Prevent directory traversal - check cleaned path first
		cleanPath := filepath.Clean(urlPath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Build file path
		filePath := filepath.Join(config.Root, cleanPath)

		// Verify resolved path doesn't escape the root directory
		absRoot, err := filepath.Abs(config.Root)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(absFile, absRoot) {
			http.Error(w, "Invalid path", http.StatusForbidden)
			return
		}

		// Check if file exists
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				if config.NotFoundHandler != nil {
					config.NotFoundHandler(w, r)
				} else {
					http.NotFound(w, r)
				}
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Handle directories
		if info.IsDir() {
			if config.IndexFile != "" {
				indexPath := filepath.Join(filePath, config.IndexFile)
				indexInfo, err := os.Stat(indexPath)
				if err == nil && !indexInfo.IsDir() {
					filePath = indexPath
					info = indexInfo
				} else {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			} else {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Set cache headers
		setCacheHeaders(w, config.MaxAge)

		// Set content type
		contentType := detectContentType(filePath)
		w.Header().Set("Content-Type", contentType)

		// Handle ETag
		if config.EnableETag {
			etag, err := getETag(filePath, info, config.etagCache)
			if err == nil {
				w.Header().Set("ETag", etag)

				// Check If-None-Match header
				if match := r.Header.Get("If-None-Match"); match != "" {
					if match == etag {
						w.WriteHeader(http.StatusNotModified)
						return
					}
				}
			}
		}

		// Handle Last-Modified
		lastModified := info.ModTime().UTC().Format(http.TimeFormat)
		w.Header().Set("Last-Modified", lastModified)

		// Check If-Modified-Since header
		if ims := r.Header.Get("If-Modified-Since"); ims != "" {
			if t, err := time.Parse(http.TimeFormat, ims); err == nil {
				if !info.ModTime().After(t) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		// Serve the file
		http.ServeFile(w, r, filePath)
	})
}

// NewFileServer creates a static file server with default configuration
func NewFileServer(root, prefix string) http.Handler {
	config := DefaultFileServerConfig(root)
	config.Prefix = prefix
	return FileServer(config)
}

// setCacheHeaders sets cache control headers
func setCacheHeaders(w http.ResponseWriter, maxAge int) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
}

// detectContentType detects the content type from file extension
func detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Common content types
	contentTypes := map[string]string{
		".html": "text/html; charset=utf-8",
		".htm":  "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript; charset=utf-8",
		".json": "application/json; charset=utf-8",
		".xml":  "application/xml; charset=utf-8",
		".txt":  "text/plain; charset=utf-8",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".ico":  "image/x-icon",
		".woff": "font/woff",
		".woff2": "font/woff2",
		".ttf":  "font/ttf",
		".eot":  "application/vnd.ms-fontobject",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}

// getETag generates or retrieves an ETag for a file
func getETag(filePath string, info os.FileInfo, cache *sync.Map) (string, error) {
	// Create cache key from path and modification time
	cacheKey := fmt.Sprintf("%s:%d", filePath, info.ModTime().Unix())

	// Check cache
	if etag, ok := cache.Load(cacheKey); ok {
		return etag.(string), nil
	}

	// Use file size and modification time for ETag (fast and sufficient)
	// W/ prefix indicates a weak ETag which is appropriate for static files
	etag := fmt.Sprintf(`"W/%x-%x"`, info.Size(), info.ModTime().Unix())

	// Store in cache
	cache.Store(cacheKey, etag)

	return etag, nil
}
