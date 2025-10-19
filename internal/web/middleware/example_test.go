package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/conduit-lang/conduit/internal/web/router"
)

// ExampleChain demonstrates basic middleware chain usage
func ExampleChain() {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Create middleware chain
	chain := middleware.NewChain(
		middleware.Recovery(),
		middleware.RequestID(),
	)

	// Wrap handler
	wrapped := chain.Apply(handler)

	// Test the handler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output: 200
}

// ExampleConditional demonstrates conditional middleware application
func ExampleConditional() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply CORS only to /api paths
	conditional := middleware.Conditional(
		middleware.PathPrefix("/api"),
		middleware.CORS(),
	)

	wrapped := conditional(handler)

	// Test /api path
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Access-Control-Allow-Origin"))
	// Output: http://example.com
}

// ExampleRouter_Use demonstrates middleware integration with router
func ExampleRouter_Use() {
	// Create router
	r := router.NewRouter()

	// Add middleware
	r.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logging(),
	)

	// Add routes
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Users list"))
	})

	// Test the route
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output: 200
}

// ExampleCORS demonstrates CORS middleware configuration
func ExampleCORS() {
	config := middleware.CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORSWithConfig(config)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Access-Control-Allow-Credentials"))
	// Output: true
}

// ExampleGetRequestID demonstrates extracting request ID from context
func ExampleGetRequestID() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		fmt.Printf("Request ID length: %d\n", len(requestID))
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.RequestID()(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Output: Request ID length: 36
}
