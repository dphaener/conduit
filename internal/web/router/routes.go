package router

import (
	"fmt"
	"net/http"
	"strings"
)

// ResourceDefinition represents a resource that can be registered with the router
type ResourceDefinition struct {
	Name        string                     // Resource name (e.g., "Post")
	PluralName  string                     // Plural name (e.g., "posts")
	BasePath    string                     // Base path (e.g., "/api/posts")
	IDParamName string                     // ID parameter name (default: "id")
	IDType      string                     // ID type (uuid, int, string)
	Operations  []CRUDOperation            // Enabled operations
	Middleware  map[CRUDOperation][]string // Middleware per operation
}

// NewResourceDefinition creates a new resource definition with defaults
func NewResourceDefinition(name string) *ResourceDefinition {
	pluralName := pluralize(name)
	return &ResourceDefinition{
		Name:        name,
		PluralName:  pluralName,
		BasePath:    "/" + toSnakeCase(pluralName),
		IDParamName: "id",
		IDType:      "uuid",
		Operations:  []CRUDOperation{OpList, OpCreate, OpShow, OpUpdate, OpPatch, OpDelete},
		Middleware:  make(map[CRUDOperation][]string),
	}
}

// RegisterResource registers REST routes for a resource
func (r *Router) RegisterResource(def *ResourceDefinition, handlers ResourceHandlers) error {
	// Validate handlers
	if err := handlers.Validate(def.Operations); err != nil {
		return fmt.Errorf("invalid handlers: %w", err)
	}

	// Register routes for each enabled operation
	for _, op := range def.Operations {
		if err := r.registerResourceOperation(def, op, handlers); err != nil {
			return fmt.Errorf("failed to register operation %s: %w", op, err)
		}
	}

	return nil
}

// ResourceHandlers contains handlers for resource operations
type ResourceHandlers struct {
	List   http.HandlerFunc
	Create http.HandlerFunc
	Show   http.HandlerFunc
	Update http.HandlerFunc
	Patch  http.HandlerFunc
	Delete http.HandlerFunc
}

// Validate checks that all required handlers are present
func (h *ResourceHandlers) Validate(operations []CRUDOperation) error {
	for _, op := range operations {
		handler := h.GetHandler(op)
		if handler == nil {
			return fmt.Errorf("missing handler for operation: %s", op)
		}
	}
	return nil
}

// GetHandler returns the handler for the given operation
func (h *ResourceHandlers) GetHandler(op CRUDOperation) http.HandlerFunc {
	switch op {
	case OpList:
		return h.List
	case OpCreate:
		return h.Create
	case OpShow:
		return h.Show
	case OpUpdate:
		return h.Update
	case OpPatch:
		return h.Patch
	case OpDelete:
		return h.Delete
	default:
		return nil
	}
}

// registerResourceOperation registers a single resource operation
func (r *Router) registerResourceOperation(def *ResourceDefinition, op CRUDOperation, handlers ResourceHandlers) error {
	handler := handlers.GetHandler(op)
	if handler == nil {
		return fmt.Errorf("missing handler for operation: %s", op)
	}

	var route *Route
	switch op {
	case OpList:
		route = r.Get(def.BasePath, handler)
	case OpCreate:
		route = r.Post(def.BasePath, handler)
	case OpShow:
		pattern := fmt.Sprintf("%s/{%s}", def.BasePath, def.IDParamName)
		route = r.Get(pattern, handler)
	case OpUpdate:
		pattern := fmt.Sprintf("%s/{%s}", def.BasePath, def.IDParamName)
		route = r.Put(pattern, handler)
	case OpPatch:
		pattern := fmt.Sprintf("%s/{%s}", def.BasePath, def.IDParamName)
		route = r.Patch(pattern, handler)
	case OpDelete:
		pattern := fmt.Sprintf("%s/{%s}", def.BasePath, def.IDParamName)
		route = r.Delete(pattern, handler)
	default:
		return fmt.Errorf("unknown operation: %s", op)
	}

	// Set resource metadata
	route.WithResource(def.Name, op)

	// Set route name
	routeName := fmt.Sprintf("%s.%s", def.Name, op.String())
	route.Named(routeName)

	// Store middleware names
	if middlewares, ok := def.Middleware[op]; ok {
		route.Middleware = middlewares
	}

	return nil
}

// RegisterNestedResource registers nested resource routes
func (r *Router) RegisterNestedResource(parent, child *ResourceDefinition, handlers ResourceHandlers) error {
	// Create nested path pattern using lowercase plural name
	childPath := strings.ToLower(child.PluralName)
	nestedPath := fmt.Sprintf("%s/{%s}/%s", parent.BasePath, parent.IDParamName, childPath)

	// Create a new definition for the nested resource
	nestedDef := &ResourceDefinition{
		Name:        child.Name,
		PluralName:  child.PluralName,
		BasePath:    nestedPath,
		IDParamName: child.IDParamName,
		IDType:      child.IDType,
		Operations:  child.Operations,
		Middleware:  child.Middleware,
	}

	return r.RegisterResource(nestedDef, handlers)
}

// RouteList returns a formatted list of all routes
func (r *Router) RouteList() string {
	var sb strings.Builder
	sb.WriteString("Registered Routes:\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("%-8s %-40s %-20s\n", "METHOD", "PATTERN", "NAME"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, info := range r.registeredRoutes {
		name := ""
		// Find the route to get its name
		for _, route := range r.routes {
			if route.Pattern == info.Pattern && route.Method == info.Method {
				name = route.Name
				break
			}
		}
		sb.WriteString(fmt.Sprintf("%-8s %-40s %-20s\n", info.Method, info.Pattern, name))
	}

	return sb.String()
}

// RouteListJSON returns route information as a structured format
func (r *Router) RouteListJSON() []RouteInfo {
	// Create a copy to avoid exposing internal state
	routes := make([]RouteInfo, len(r.registeredRoutes))
	for i, route := range r.registeredRoutes {
		routes[i] = *route
	}
	return routes
}

// URL generates a URL for a named route with parameters
func (r *Router) URL(name string, params map[string]string) (string, error) {
	route, err := r.GetRoute(name)
	if err != nil {
		return "", err
	}

	url := route.Pattern
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		url = strings.ReplaceAll(url, placeholder, value)
	}

	// Check if all placeholders were replaced
	if strings.Contains(url, "{") && strings.Contains(url, "}") {
		return "", fmt.Errorf("missing parameter values for route: %s", name)
	}

	return url, nil
}

// Helper functions

// pluralize returns the plural form of a word (simple implementation)
func pluralize(word string) string {
	if word == "" {
		return word
	}

	// Handle common special cases
	specialCases := map[string]string{
		"person": "people",
		"child":  "children",
		"man":    "men",
		"woman":  "women",
		"tooth":  "teeth",
		"foot":   "feet",
		"mouse":  "mice",
		"goose":  "geese",
	}

	if plural, ok := specialCases[strings.ToLower(word)]; ok {
		return plural
	}

	// Simple rules
	lastChar := word[len(word)-1]
	switch {
	case strings.HasSuffix(word, "y"):
		if len(word) > 1 && !isVowel(word[len(word)-2]) {
			return word[:len(word)-1] + "ies"
		}
		return word + "s"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh"):
		return word + "es"
	case lastChar == 'f':
		return word[:len(word)-1] + "ves"
	case strings.HasSuffix(word, "fe"):
		return word[:len(word)-2] + "ves"
	default:
		return word + "s"
	}
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isVowel checks if a byte is a vowel
func isVowel(b byte) bool {
	vowels := "aeiouAEIOU"
	return strings.ContainsRune(vowels, rune(b))
}
