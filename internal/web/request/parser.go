package request

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Parser handles parsing of HTTP request bodies
type Parser struct {
	maxBodySize int64 // Maximum size for request bodies (in bytes)
}

// NewParser creates a new request parser with default settings
func NewParser() *Parser {
	return &Parser{
		maxBodySize: 10 << 20, // 10MB default
	}
}

// NewParserWithMaxSize creates a parser with a custom max body size
func NewParserWithMaxSize(maxBytes int64) *Parser {
	return &Parser{
		maxBodySize: maxBytes,
	}
}

// Parse parses the request body into the target based on Content-Type
func (p *Parser) Parse(w http.ResponseWriter, r *http.Request, target interface{}) error {
	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}

	switch {
	case mediaType == "application/json":
		return p.ParseJSON(w, r, target)
	case mediaType == "application/x-www-form-urlencoded":
		return p.ParseForm(w, r, target)
	case strings.HasPrefix(mediaType, "multipart/form-data"):
		return p.ParseMultipart(w, r, target)
	case contentType == "":
		// No content type specified, try JSON as default
		return p.ParseJSON(w, r, target)
	default:
		return fmt.Errorf("unsupported content type: %s", mediaType)
	}
}

// ParseJSON parses a JSON request body
func (p *Parser) ParseJSON(w http.ResponseWriter, r *http.Request, target interface{}) error {
	// Limit body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, p.maxBodySize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing - reject unknown fields

	if err := decoder.Decode(target); err != nil {
		// Provide more helpful error messages
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		if strings.Contains(err.Error(), "unknown field") {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		if strings.Contains(err.Error(), "cannot unmarshal") {
			return fmt.Errorf("invalid JSON format: %w", err)
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check if there's additional data after the JSON object
	if decoder.More() {
		return fmt.Errorf("request body contains multiple JSON objects")
	}

	return nil
}

// ParseForm parses URL-encoded form data
func (p *Parser) ParseForm(w http.ResponseWriter, r *http.Request, target interface{}) error {
	// Limit body size
	r.Body = http.MaxBytesReader(w, r.Body, p.maxBodySize)
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("invalid form data: %w", err)
	}

	// Convert form values to target structure
	return formToMap(r.Form, target)
}

// ParseMultipart parses multipart/form-data (used for file uploads)
func (p *Parser) ParseMultipart(w http.ResponseWriter, r *http.Request, target interface{}) error {
	// Parse multipart form with size limit
	if err := r.ParseMultipartForm(p.maxBodySize); err != nil {
		return fmt.Errorf("invalid multipart form: %w", err)
	}

	// Convert multipart form to target structure
	return multipartToMap(r, target)
}

// ParseQuery parses URL query parameters (no body, so no ResponseWriter needed)
func (p *Parser) ParseQuery(r *http.Request, target interface{}) error {
	return formToMap(r.URL.Query(), target)
}

// formToMap converts url.Values to a map or struct
func formToMap(values url.Values, target interface{}) error {
	// Check if target is a map[string]interface{}
	if m, ok := target.(*map[string]interface{}); ok {
		result := make(map[string]interface{})
		for key, vals := range values {
			if len(vals) == 1 {
				result[key] = vals[0]
			} else {
				result[key] = vals
			}
		}
		*m = result
		return nil
	}

	// Check if target is a map[string]string
	if m, ok := target.(*map[string]string); ok {
		result := make(map[string]string)
		for key, vals := range values {
			if len(vals) > 0 {
				result[key] = vals[0]
			}
		}
		*m = result
		return nil
	}

	// For other types, marshal to JSON and unmarshal to target
	// This is a simple approach that works for most cases
	jsonData := make(map[string]interface{})
	for key, vals := range values {
		if len(vals) == 1 {
			// Try to parse as different types
			jsonData[key] = parseValue(vals[0])
		} else {
			// Multiple values - keep as array
			jsonData[key] = vals
		}
	}

	// Marshal to JSON and unmarshal to target type
	data, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to convert form data: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse form data: %w", err)
	}

	return nil
}

// multipartToMap converts multipart form data to a map or struct
func multipartToMap(form *http.Request, target interface{}) error {
	if form.MultipartForm == nil {
		return fmt.Errorf("no multipart form data")
	}

	// Check if target is a map[string]interface{}
	if m, ok := target.(*map[string]interface{}); ok {
		result := make(map[string]interface{})

		// Add form values
		for key, vals := range form.MultipartForm.Value {
			if len(vals) == 1 {
				result[key] = vals[0]
			} else {
				result[key] = vals
			}
		}

		// Note: Files are handled separately via GetFile/GetFiles methods
		// We don't auto-parse files into the target to avoid memory issues

		*m = result
		return nil
	}

	return fmt.Errorf("multipart parsing only supports map[string]interface{} target")
}

// parseValue attempts to parse a string value as int, bool, or float
func parseValue(s string) interface{} {
	// Try bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Return as string
	return s
}

// GetParam gets a path parameter from the request
func GetParam(r *http.Request, name string) string {
	// chi stores path params in context
	return r.PathValue(name)
}

// GetQueryParam gets a query parameter
func GetQueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// GetQueryParamInt gets a query parameter as integer
func GetQueryParamInt(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

// GetQueryParamInt64 gets a query parameter as int64
func GetQueryParamInt64(r *http.Request, name string, defaultValue int64) int64 {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}
	return i
}

// GetQueryParamBool gets a query parameter as boolean
func GetQueryParamBool(r *http.Request, name string, defaultValue bool) bool {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

// GetHeader gets an HTTP header value
func GetHeader(r *http.Request, name string) string {
	return r.Header.Get(name)
}
