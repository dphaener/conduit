package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/go-chi/chi/v5"
)

// Router manages HTTP routing using chi framework
type Router struct {
	mux    chi.Router
	routes map[string]*Route
	groups map[string]*RouteGroup

	// Middleware chain
	chain *middleware.Chain

	// For introspection and debugging
	registeredRoutes []*RouteInfo
}

// Route represents a single registered route
type Route struct {
	Pattern    string           // /posts/{id}
	Method     string           // GET, POST, etc.
	Handler    http.HandlerFunc // Handler function
	Name       string           // Named route for URL generation
	Middleware []string         // Middleware names applied to this route

	// Resource metadata (if auto-generated)
	ResourceName string
	Operation    CRUDOperation
}

// RouteGroup represents a group of routes with a common prefix
type RouteGroup struct {
	Prefix     string
	Routes     []*Route
	Middleware []string
}

// RouteInfo provides metadata about a route for introspection
type RouteInfo struct {
	Pattern      string
	Method       string
	Name         string
	ResourceName string
	Operation    string
	Middleware   []string
	Parameters   []RouteParameter
}

// RouteParameter describes a parameter in a route
type RouteParameter struct {
	Name     string
	Type     string // uuid, int, string
	Required bool
	Source   ParameterSource // path, query, header
}

// ParameterSource indicates where a parameter comes from
type ParameterSource int

const (
	// PathParam indicates a URL path parameter
	PathParam ParameterSource = iota
	// QueryParam indicates a URL query parameter
	QueryParam
	// HeaderParam indicates an HTTP header parameter
	HeaderParam
)

// String returns the string representation of ParameterSource
func (p ParameterSource) String() string {
	switch p {
	case PathParam:
		return "path"
	case QueryParam:
		return "query"
	case HeaderParam:
		return "header"
	default:
		return "unknown"
	}
}

// CRUDOperation represents a REST operation type
type CRUDOperation int

const (
	// OpList represents the list/index operation (GET /)
	OpList CRUDOperation = iota
	// OpCreate represents the create operation (POST /)
	OpCreate
	// OpShow represents the show/read operation (GET /{id})
	OpShow
	// OpUpdate represents the update operation (PUT /{id})
	OpUpdate
	// OpPatch represents the partial update operation (PATCH /{id})
	OpPatch
	// OpDelete represents the delete operation (DELETE /{id})
	OpDelete
)

// String returns the string representation of CRUDOperation
func (o CRUDOperation) String() string {
	switch o {
	case OpList:
		return "list"
	case OpCreate:
		return "create"
	case OpShow:
		return "show"
	case OpUpdate:
		return "update"
	case OpPatch:
		return "patch"
	case OpDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		mux:              chi.NewRouter(),
		routes:           make(map[string]*Route),
		groups:           make(map[string]*RouteGroup),
		chain:            middleware.NewChain(),
		registeredRoutes: make([]*RouteInfo, 0),
	}
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Use adds middleware to the router's middleware chain
func (r *Router) Use(middlewares ...middleware.Middleware) {
	for _, m := range middlewares {
		r.chain.Use(m)
		// Also register with chi for proper execution
		r.mux.Use(func(next http.Handler) http.Handler {
			return m(next)
		})
	}
}

// Get registers a GET route
func (r *Router) Get(pattern string, handler http.HandlerFunc) *Route {
	return r.addRoute(http.MethodGet, pattern, handler)
}

// Post registers a POST route
func (r *Router) Post(pattern string, handler http.HandlerFunc) *Route {
	return r.addRoute(http.MethodPost, pattern, handler)
}

// Put registers a PUT route
func (r *Router) Put(pattern string, handler http.HandlerFunc) *Route {
	return r.addRoute(http.MethodPut, pattern, handler)
}

// Patch registers a PATCH route
func (r *Router) Patch(pattern string, handler http.HandlerFunc) *Route {
	return r.addRoute(http.MethodPatch, pattern, handler)
}

// Delete registers a DELETE route
func (r *Router) Delete(pattern string, handler http.HandlerFunc) *Route {
	return r.addRoute(http.MethodDelete, pattern, handler)
}

// addRoute registers a route with the given method, pattern, and handler
func (r *Router) addRoute(method, pattern string, handler http.HandlerFunc) *Route {
	route := &Route{
		Pattern: pattern,
		Method:  method,
		Handler: handler,
	}

	// Register with chi
	switch method {
	case http.MethodGet:
		r.mux.Get(pattern, handler)
	case http.MethodPost:
		r.mux.Post(pattern, handler)
	case http.MethodPut:
		r.mux.Put(pattern, handler)
	case http.MethodPatch:
		r.mux.Patch(pattern, handler)
	case http.MethodDelete:
		r.mux.Delete(pattern, handler)
	}

	// Store route
	routeKey := fmt.Sprintf("%s:%s", method, pattern)
	r.routes[routeKey] = route

	// Add to introspection info
	r.registeredRoutes = append(r.registeredRoutes, &RouteInfo{
		Pattern:    pattern,
		Method:     method,
		Parameters: extractParameters(pattern),
	})

	return route
}

// Group creates a route group with a common prefix
func (r *Router) Group(prefix string, fn func(r chi.Router)) {
	r.mux.Route(prefix, fn)
}

// Named sets a name for the route (for URL generation)
func (route *Route) Named(name string) *Route {
	route.Name = name
	return route
}

// WithResource sets resource metadata for the route
func (route *Route) WithResource(resourceName string, operation CRUDOperation) *Route {
	route.ResourceName = resourceName
	route.Operation = operation
	return route
}

// GetRoutes returns all registered routes for introspection
func (r *Router) GetRoutes() []*RouteInfo {
	return r.registeredRoutes
}

// GetRoute returns a route by name
func (r *Router) GetRoute(name string) (*Route, error) {
	for _, route := range r.routes {
		if route.Name == name {
			return route, nil
		}
	}
	return nil, fmt.Errorf("route not found: %s", name)
}

// NotFound sets the handler for 404 Not Found
func (r *Router) NotFound(handler http.HandlerFunc) {
	r.mux.NotFound(handler)
}

// MethodNotAllowed sets the handler for 405 Method Not Allowed
func (r *Router) MethodNotAllowed(handler http.HandlerFunc) {
	r.mux.MethodNotAllowed(handler)
}

// extractParameters extracts parameter definitions from a route pattern
func extractParameters(pattern string) []RouteParameter {
	params := make([]RouteParameter, 0)
	parts := strings.Split(pattern, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			params = append(params, RouteParameter{
				Name:     paramName,
				Type:     inferParameterType(paramName),
				Required: true,
				Source:   PathParam,
			})
		}
	}

	return params
}

// inferParameterType infers the type of a parameter from its name
func inferParameterType(name string) string {
	// Common naming conventions
	if name == "id" || strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "Id") {
		return "uuid"
	}
	if strings.HasPrefix(name, "page") || strings.HasPrefix(name, "limit") ||
		strings.HasPrefix(name, "offset") || strings.HasPrefix(name, "count") {
		return "int"
	}
	return "string"
}
