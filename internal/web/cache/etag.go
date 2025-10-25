package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// GenerateETag generates an ETag for the given content
func GenerateETag(content []byte) string {
	hash := sha256.Sum256(content)
	// Truncate to 16 bytes for shorter ETags (still 128-bit security)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:16]))
}

// GenerateWeakETag generates a weak ETag for the given content
func GenerateWeakETag(content []byte) string {
	hash := sha256.Sum256(content)
	// Truncate to 16 bytes for shorter ETags (still 128-bit security)
	return fmt.Sprintf(`W/"%s"`, hex.EncodeToString(hash[:16]))
}

// GenerateLastModified formats a time as an HTTP Last-Modified header value
func GenerateLastModified(t time.Time) string {
	return t.UTC().Format(http.TimeFormat)
}

// ParseIfNoneMatch parses the If-None-Match header value
func ParseIfNoneMatch(header string) []string {
	if header == "" {
		return nil
	}

	// Simple implementation - split by comma
	// In production, you might want more robust parsing
	var etags []string
	if header == "*" {
		return []string{"*"}
	}

	// Split and trim
	for i := 0; i < len(header); {
		// Skip whitespace
		for i < len(header) && (header[i] == ' ' || header[i] == ',') {
			i++
		}
		if i >= len(header) {
			break
		}

		// Check for weak ETag
		weak := false
		if i+2 < len(header) && header[i:i+2] == "W/" {
			weak = true
			i += 2
		}

		// Find quoted string
		if i < len(header) && header[i] == '"' {
			start := i
			i++
			for i < len(header) && header[i] != '"' {
				i++
			}
			if i < len(header) {
				i++
				etag := header[start:i]
				if weak {
					etag = "W/" + etag
				}
				etags = append(etags, etag)
			}
		}
	}

	return etags
}

// ParseIfModifiedSince parses the If-Modified-Since header value
func ParseIfModifiedSince(header string) (time.Time, error) {
	if header == "" {
		return time.Time{}, fmt.Errorf("empty header")
	}
	return http.ParseTime(header)
}

// MatchesETag checks if the given ETag matches any of the provided ETags
func MatchesETag(etag string, etags []string) bool {
	if len(etags) == 0 {
		return false
	}

	// * matches everything
	if len(etags) == 1 && etags[0] == "*" {
		return true
	}

	// Check for exact match (both strong and weak ETags)
	for _, e := range etags {
		if e == etag {
			return true
		}

		// Weak comparison: strip W/ prefix from both and compare
		cleanE := e
		cleanEtag := etag
		if len(cleanE) > 2 && cleanE[:2] == "W/" {
			cleanE = cleanE[2:]
		}
		if len(cleanEtag) > 2 && cleanEtag[:2] == "W/" {
			cleanEtag = cleanEtag[2:]
		}
		if cleanE == cleanEtag {
			return true
		}
	}

	return false
}

// CheckConditionalRequest checks if a conditional request should return 304 Not Modified
func CheckConditionalRequest(w http.ResponseWriter, r *http.Request, etag string, lastModified time.Time) bool {
	// Check If-None-Match first (takes precedence)
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" {
		etags := ParseIfNoneMatch(ifNoneMatch)
		if MatchesETag(etag, etags) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
		// If If-None-Match is present but doesn't match, don't check If-Modified-Since
		return false
	}

	// Check If-Modified-Since
	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" && !lastModified.IsZero() {
		ifModifiedSinceTime, err := ParseIfModifiedSince(ifModifiedSince)
		if err == nil {
			// Truncate to second precision for comparison
			if !lastModified.Truncate(time.Second).After(ifModifiedSinceTime) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

// SetCacheHeaders sets appropriate cache headers on the response
func SetCacheHeaders(w http.ResponseWriter, etag string, lastModified time.Time, cacheControl string) {
	if etag != "" {
		w.Header().Set("ETag", etag)
	}
	if !lastModified.IsZero() {
		w.Header().Set("Last-Modified", GenerateLastModified(lastModified))
	}
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
}
