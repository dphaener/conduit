package response

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/DataDog/jsonapi"
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
