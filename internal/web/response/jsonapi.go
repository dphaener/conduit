package response

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/DataDog/jsonapi"
	"github.com/conduit-lang/conduit/internal/orm/validation"
)

const (
	// JSONAPIMediaType is the official JSON:API media type
	JSONAPIMediaType = "application/vnd.api+json"
)

// IsJSONAPI checks if the request accepts JSON:API format
func IsJSONAPI(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	// Parse media type to handle parameters like charset
	mediaType, _, err := mime.ParseMediaType(accept)
	if err != nil {
		// Fall back to simple check if parsing fails
		return strings.Contains(accept, JSONAPIMediaType)
	}

	return mediaType == JSONAPIMediaType
}

// RenderJSONAPI marshals a single resource or collection using DataDog/jsonapi library
func RenderJSONAPI(w http.ResponseWriter, status int, payload interface{}, opts ...jsonapi.MarshalOption) error {
	// Marshal FIRST, before touching the response
	// This avoids partial writes if marshaling fails
	data, err := jsonapi.Marshal(payload, opts...)
	if err != nil {
		return err // Don't write anything if marshaling fails
	}

	// Only write to response if marshaling succeeded
	w.Header().Set("Content-Type", JSONAPIMediaType)
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}

// RenderJSONAPIWithMeta marshals with pagination metadata
func RenderJSONAPIWithMeta(w http.ResponseWriter, status int, payload interface{}, meta map[string]interface{}, links *jsonapi.Link) error {
	opts := []jsonapi.MarshalOption{}
	if meta != nil {
		opts = append(opts, jsonapi.MarshalMeta(meta))
	}
	if links != nil {
		opts = append(opts, jsonapi.MarshalLinks(links))
	}

	// Marshal FIRST before writing response
	data, err := jsonapi.Marshal(payload, opts...)
	if err != nil {
		return err // Don't write anything if marshaling fails
	}

	// Only write to response if marshaling succeeded
	w.Header().Set("Content-Type", JSONAPIMediaType)
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}

// BuildPaginationLinks creates pagination links for JSON:API responses
func BuildPaginationLinks(baseURL string, page, perPage, total int) *jsonapi.Link {
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	self := buildPageURL(baseURL, page, perPage)
	first := buildPageURL(baseURL, 1, perPage)
	last := buildPageURL(baseURL, totalPages, perPage)

	links := &jsonapi.Link{
		Self:  self,
		First: first,
		Last:  last,
	}

	if page > 1 {
		prev := buildPageURL(baseURL, page-1, perPage)
		links.Prev = prev
	}

	if page < totalPages {
		next := buildPageURL(baseURL, page+1, perPage)
		links.Next = next
	}

	return links
}

func buildPageURL(baseURL string, page, perPage int) string {
	offset := (page - 1) * perPage

	// Parse the base URL to handle existing query parameters
	u, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to simple concatenation if parse fails
		return fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", baseURL, perPage, offset)
	}

	q := u.Query()
	q.Set("page[limit]", strconv.Itoa(perPage))
	q.Set("page[offset]", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	return u.String()
}

// escapeJSONPointer escapes special characters per RFC 6901
func escapeJSONPointer(token string) string {
	// Order matters: escape ~ before /
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}

// TransformValidationErrors converts Conduit validation errors to JSON:API format
func TransformValidationErrors(err *validation.ValidationErrors) []*jsonapi.Error {
	var errors []*jsonapi.Error

	for field, messages := range err.Fields {
		for _, msg := range messages {
			status := http.StatusUnprocessableEntity
			errors = append(errors, &jsonapi.Error{
				Status: &status,
				Code:   "validation_error",
				Title:  "Validation Failed",
				Detail: msg,
				Source: &jsonapi.ErrorSource{
					Pointer: fmt.Sprintf("/data/attributes/%s", escapeJSONPointer(field)),
				},
			})
		}
	}

	return errors
}

// RenderJSONAPIError renders a single JSON:API error
func RenderJSONAPIError(w http.ResponseWriter, statusCode int, err error) {
	// Check if it's a validation error
	if validationErr, ok := err.(*validation.ValidationErrors); ok {
		RenderJSONAPIErrors(w, http.StatusUnprocessableEntity, TransformValidationErrors(validationErr))
		return
	}

	// Single error
	errors := []*jsonapi.Error{{
		Status: &statusCode,
		Code:   errorCodeFromStatus(statusCode),
		Title:  http.StatusText(statusCode),
		Detail: err.Error(),
	}}

	RenderJSONAPIErrors(w, statusCode, errors)
}

// RenderJSONAPIErrors renders multiple JSON:API errors
func RenderJSONAPIErrors(w http.ResponseWriter, statusCode int, errors []*jsonapi.Error) {
	// Marshal errors BEFORE writing headers (critical pattern from Phase 1)
	data, err := json.Marshal(map[string][]*jsonapi.Error{"errors": errors})
	if err != nil {
		// Fallback if marshaling fails
		w.Header().Set("Content-Type", JSONAPIMediaType)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"status":"500","code":"internal_error","title":"Internal Server Error"}]}`))
		return
	}

	w.Header().Set("Content-Type", JSONAPIMediaType)
	w.WriteHeader(statusCode)
	w.Write(data)
}

// ValidateJSONAPIContentType checks if the Content-Type is application/vnd.api+json
// Returns true if valid, writes error response and returns false if invalid
func ValidateJSONAPIContentType(w http.ResponseWriter, r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")

	// Parse media type to check for parameters
	mediaType, params, err := mime.ParseMediaType(contentType)

	if err != nil || mediaType != JSONAPIMediaType {
		RenderJSONAPIError(w, http.StatusUnsupportedMediaType,
			fmt.Errorf("Content-Type must be application/vnd.api+json"))
		return false
	}

	// JSON:API spec: Content-Type MUST NOT have media type parameters
	if len(params) > 0 {
		RenderJSONAPIError(w, http.StatusUnsupportedMediaType,
			fmt.Errorf("Content-Type must be application/vnd.api+json without media type parameters"))
		return false
	}

	return true
}
