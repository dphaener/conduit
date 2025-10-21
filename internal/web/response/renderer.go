package response

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

// RendererConfig configures the renderer
type RendererConfig struct {
	PrettyPrint bool
	AllowedDirs []string // Allowed directories for file serving
}

// Renderer handles rendering of HTTP responses
type Renderer struct {
	prettyPrint    bool
	templates      *template.Template
	defaultHeaders map[string]string
	allowedDirs    []string
}

// NewRenderer creates a new response renderer
func NewRenderer() *Renderer {
	return &Renderer{
		prettyPrint:    false,
		defaultHeaders: make(map[string]string),
		allowedDirs:    []string{},
	}
}

// NewRendererWithPrettyPrint creates a renderer with pretty-printed JSON
func NewRendererWithPrettyPrint() *Renderer {
	return &Renderer{
		prettyPrint:    true,
		defaultHeaders: make(map[string]string),
		allowedDirs:    []string{},
	}
}

// NewRendererWithConfig creates a renderer with custom configuration
func NewRendererWithConfig(config *RendererConfig) *Renderer {
	return &Renderer{
		prettyPrint:    config.PrettyPrint,
		defaultHeaders: make(map[string]string),
		allowedDirs:    config.AllowedDirs,
	}
}

// SetAllowedDirs sets the allowed directories for file serving
func (r *Renderer) SetAllowedDirs(dirs []string) {
	r.allowedDirs = dirs
}

// SetTemplates sets the template for HTML rendering
func (r *Renderer) SetTemplates(templates *template.Template) {
	r.templates = templates
}

// LoadTemplates loads HTML templates from a directory
func (r *Renderer) LoadTemplates(dir string, pattern string) error {
	tmpl, err := template.ParseGlob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}
	r.templates = tmpl
	return nil
}

// SetDefaultHeader sets a default header for all responses
func (r *Renderer) SetDefaultHeader(key, value string) {
	r.defaultHeaders[key] = value
}

// JSON renders a JSON response
func (r *Renderer) JSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	return r.JSONWithHeaders(w, statusCode, data, nil)
}

// JSONWithHeaders renders a JSON response with custom headers
func (r *Renderer) JSONWithHeaders(w http.ResponseWriter, statusCode int, data interface{}, headers map[string]string) error {
	// Set default headers
	for key, value := range r.defaultHeaders {
		w.Header().Set(key, value)
	}

	// Set custom headers
	for key, value := range headers {
		w.Header().Set(key, value)
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Write status code
	w.WriteHeader(statusCode)

	// Encode and write JSON
	encoder := json.NewEncoder(w)
	if r.prettyPrint {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// HTML renders an HTML response using templates
func (r *Renderer) HTML(w http.ResponseWriter, statusCode int, templateName string, data interface{}) error {
	return r.HTMLWithHeaders(w, statusCode, templateName, data, nil)
}

// HTMLWithHeaders renders an HTML response with custom headers
func (r *Renderer) HTMLWithHeaders(w http.ResponseWriter, statusCode int, templateName string, data interface{}, headers map[string]string) error {
	if r.templates == nil {
		return fmt.Errorf("no templates loaded")
	}

	// Set default headers
	for key, value := range r.defaultHeaders {
		w.Header().Set(key, value)
	}

	// Set custom headers
	for key, value := range headers {
		w.Header().Set(key, value)
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Write status code
	w.WriteHeader(statusCode)

	// Execute template
	if err := r.templates.ExecuteTemplate(w, templateName, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// Text renders a plain text response
func (r *Renderer) Text(w http.ResponseWriter, statusCode int, text string) error {
	return r.TextWithHeaders(w, statusCode, text, nil)
}

// TextWithHeaders renders a plain text response with custom headers
func (r *Renderer) TextWithHeaders(w http.ResponseWriter, statusCode int, text string, headers map[string]string) error {
	// Set default headers
	for key, value := range r.defaultHeaders {
		w.Header().Set(key, value)
	}

	// Set custom headers
	for key, value := range headers {
		w.Header().Set(key, value)
	}

	// Set content type
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Write status code
	w.WriteHeader(statusCode)

	// Write text
	_, err := w.Write([]byte(text))
	return err
}

// File serves a file for download
func (r *Renderer) File(w http.ResponseWriter, req *http.Request, filePath string, filename string) error {
	// Validate and clean the file path
	validatedPath, err := r.validateFilePath(filePath)
	if err != nil {
		return err
	}

	// Set content disposition header
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	// Set default headers
	for key, value := range r.defaultHeaders {
		w.Header().Set(key, value)
	}

	// Serve file
	http.ServeFile(w, req, validatedPath)
	return nil
}

// validateFilePath validates and cleans a file path to prevent directory traversal
func (r *Renderer) validateFilePath(filePath string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(filePath)

	// Resolve symlinks to prevent bypass
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// If no allowed directories configured, reject all file access
	if len(r.allowedDirs) == 0 {
		return "", fmt.Errorf("file serving not configured: no allowed directories set")
	}

	// Check if the resolved path is within one of the allowed directories
	allowed := false
	for _, allowedDir := range r.allowedDirs {
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

// Redirect sends a redirect response
func (r *Renderer) Redirect(w http.ResponseWriter, req *http.Request, url string, statusCode int) {
	http.Redirect(w, req, url, statusCode)
}

// NoContent sends a 204 No Content response
func (r *Renderer) NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Negotiate performs content negotiation based on Accept header
func (r *Renderer) Negotiate(w http.ResponseWriter, req *http.Request, statusCode int, data interface{}) error {
	accept := req.Header.Get("Accept")

	// Parse Accept header and determine best format
	if accept == "" || strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*") {
		return r.JSON(w, statusCode, data)
	}

	if strings.Contains(accept, "text/html") {
		// For HTML, we need a template name - use "index" as default
		return r.HTML(w, statusCode, "index", data)
	}

	if strings.Contains(accept, "text/plain") {
		// Convert data to string for plain text
		text := fmt.Sprintf("%v", data)
		return r.Text(w, statusCode, text)
	}

	// Default to JSON
	return r.JSON(w, statusCode, data)
}

// APIResponse represents a standard API response structure
type APIResponse struct {
	Data  interface{}            `json:"data,omitempty"`
	Meta  map[string]interface{} `json:"meta,omitempty"`
	Links map[string]string      `json:"links,omitempty"`
}

// NewAPIResponse creates a new API response
func NewAPIResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Data:  data,
		Meta:  make(map[string]interface{}),
		Links: make(map[string]string),
	}
}

// WithMeta adds metadata to the response
func (r *APIResponse) WithMeta(key string, value interface{}) *APIResponse {
	if r.Meta == nil {
		r.Meta = make(map[string]interface{})
	}
	r.Meta[key] = value
	return r
}

// WithLink adds a link to the response
func (r *APIResponse) WithLink(rel, href string) *APIResponse {
	if r.Links == nil {
		r.Links = make(map[string]string)
	}
	r.Links[rel] = href
	return r
}

// WithPagination adds pagination metadata
func (r *APIResponse) WithPagination(page, perPage, total int) *APIResponse {
	if r.Meta == nil {
		r.Meta = make(map[string]interface{})
	}
	r.Meta["page"] = page
	r.Meta["per_page"] = perPage
	r.Meta["total"] = total
	r.Meta["total_pages"] = (total + perPage - 1) / perPage
	return r
}

// RenderJSON is a convenience function to render JSON
func RenderJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	renderer := NewRenderer()
	return renderer.JSON(w, statusCode, data)
}

// RenderJSONPretty is a convenience function to render pretty-printed JSON
func RenderJSONPretty(w http.ResponseWriter, statusCode int, data interface{}) error {
	renderer := NewRendererWithPrettyPrint()
	return renderer.JSON(w, statusCode, data)
}

// RenderText is a convenience function to render plain text
func RenderText(w http.ResponseWriter, statusCode int, text string) error {
	renderer := NewRenderer()
	return renderer.Text(w, statusCode, text)
}

// RenderNoContent is a convenience function to send 204 No Content
func RenderNoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}
