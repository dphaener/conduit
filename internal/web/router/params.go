package router

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ParamExtractor provides utilities for extracting and converting parameters
type ParamExtractor struct {
	req *http.Request
}

// NewParamExtractor creates a new parameter extractor for the given request
func NewParamExtractor(req *http.Request) *ParamExtractor {
	return &ParamExtractor{req: req}
}

// PathParam extracts a path parameter by name
func (p *ParamExtractor) PathParam(name string) string {
	return chi.URLParam(p.req, name)
}

// PathParamUUID extracts a path parameter and converts it to UUID
func (p *ParamExtractor) PathParamUUID(name string) (uuid.UUID, error) {
	value := chi.URLParam(p.req, name)
	if value == "" {
		return uuid.Nil, fmt.Errorf("missing path parameter: %s", name)
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID for parameter %s: %w", name, err)
	}

	return id, nil
}

// PathParamInt extracts a path parameter and converts it to int
func (p *ParamExtractor) PathParamInt(name string) (int, error) {
	value := chi.URLParam(p.req, name)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter: %s", name)
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for parameter %s: %w", name, err)
	}

	return i, nil
}

// PathParamInt64 extracts a path parameter and converts it to int64
func (p *ParamExtractor) PathParamInt64(name string) (int64, error) {
	value := chi.URLParam(p.req, name)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter: %s", name)
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 for parameter %s: %w", name, err)
	}

	return i, nil
}

// QueryParam extracts a query parameter by name
func (p *ParamExtractor) QueryParam(name string) string {
	return p.req.URL.Query().Get(name)
}

// QueryParamWithDefault extracts a query parameter with a default value
func (p *ParamExtractor) QueryParamWithDefault(name, defaultValue string) string {
	value := p.req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryParamInt extracts a query parameter and converts it to int
func (p *ParamExtractor) QueryParamInt(name string, defaultValue int) int {
	value := p.req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return i
}

// QueryParamInt64 extracts a query parameter and converts it to int64
func (p *ParamExtractor) QueryParamInt64(name string, defaultValue int64) int64 {
	value := p.req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return i
}

// QueryParamBool extracts a query parameter and converts it to bool
func (p *ParamExtractor) QueryParamBool(name string, defaultValue bool) bool {
	value := p.req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return b
}

// QueryParamArray extracts a query parameter as an array
func (p *ParamExtractor) QueryParamArray(name string) []string {
	return p.req.URL.Query()[name]
}

// HeaderParam extracts a header parameter by name
func (p *ParamExtractor) HeaderParam(name string) string {
	return p.req.Header.Get(name)
}

// HeaderParamWithDefault extracts a header parameter with a default value
func (p *ParamExtractor) HeaderParamWithDefault(name, defaultValue string) string {
	value := p.req.Header.Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// PaginationParams extracts common pagination parameters
type PaginationParams struct {
	Page    int
	PerPage int
	Offset  int
}

// ExtractPagination extracts pagination parameters from the request
func (p *ParamExtractor) ExtractPagination(defaultPerPage, maxPerPage int) PaginationParams {
	page := p.QueryParamInt("page", 1)
	if page < 1 {
		page = 1
	}

	perPage := p.QueryParamInt("per_page", defaultPerPage)
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	offset := (page - 1) * perPage

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Offset:  offset,
	}
}

// SortParams represents sorting parameters
type SortParams struct {
	Field string
	Order string // "asc" or "desc"
}

// ExtractSort extracts sorting parameters from the request
func (p *ParamExtractor) ExtractSort(defaultField, defaultOrder string, allowedFields []string) SortParams {
	field := p.QueryParamWithDefault("sort", defaultField)
	order := p.QueryParamWithDefault("order", defaultOrder)

	// Validate field is in allowed list
	if len(allowedFields) > 0 {
		allowed := false
		for _, f := range allowedFields {
			if f == field {
				allowed = true
				break
			}
		}
		if !allowed {
			field = defaultField
		}
	}

	// Validate order
	if order != "asc" && order != "desc" {
		order = defaultOrder
	}

	return SortParams{
		Field: field,
		Order: order,
	}
}

// FilterParams represents filtering parameters
type FilterParams map[string]interface{}

// ExtractFilters extracts filter parameters from query string
func (p *ParamExtractor) ExtractFilters(allowedFilters []string) FilterParams {
	filters := make(FilterParams)
	query := p.req.URL.Query()

	for _, key := range allowedFilters {
		if value := query.Get(key); value != "" {
			filters[key] = value
		}
	}

	return filters
}

// Helper functions that can be used without creating an extractor

// GetPathParam is a convenience function to extract a path parameter
func GetPathParam(req *http.Request, name string) string {
	return chi.URLParam(req, name)
}

// GetPathParamUUID is a convenience function to extract a UUID path parameter
func GetPathParamUUID(req *http.Request, name string) (uuid.UUID, error) {
	value := chi.URLParam(req, name)
	if value == "" {
		return uuid.Nil, fmt.Errorf("missing path parameter: %s", name)
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID for parameter %s: %w", name, err)
	}

	return id, nil
}

// GetQueryParam is a convenience function to extract a query parameter
func GetQueryParam(req *http.Request, name string, defaultValue string) string {
	value := req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetQueryParamInt is a convenience function to extract an integer query parameter
func GetQueryParamInt(req *http.Request, name string, defaultValue int) int {
	value := req.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return i
}
