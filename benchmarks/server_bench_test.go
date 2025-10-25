package benchmarks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/conduit-lang/conduit/internal/web/router"
)

// BenchmarkSimpleHandler benchmarks a simple handler with no middleware
func BenchmarkSimpleHandler(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkJSONResponse benchmarks JSON serialization
func BenchmarkJSONResponse(b *testing.B) {
	type Response struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Active  bool   `json:"active"`
		Balance float64 `json:"balance"`
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			ID:      123,
			Name:    "John Doe",
			Email:   "john@example.com",
			Active:  true,
			Balance: 1234.56,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkJSONArrayResponse benchmarks JSON array serialization
func BenchmarkJSONArrayResponse(b *testing.B) {
	type Item struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := make([]Item, 100)
		for i := 0; i < 100; i++ {
			items[i] = Item{ID: i, Title: "Item " + string(rune(i))}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkRouterSimple benchmarks router with a single route
func BenchmarkRouterSimple(b *testing.B) {
	r := router.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkRouterWithParams benchmarks router with path parameters
func BenchmarkRouterWithParams(b *testing.B) {
	r := router.NewRouter()
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkMiddlewareChain benchmarks middleware chain
func BenchmarkMiddlewareChain(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Create middleware chain
	chain := middleware.NewChain()
	chain.Use(middleware.RequestID())
	chain.Use(middleware.Recovery())

	wrappedHandler := chain.Then(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkCompression benchmarks gzip compression
func BenchmarkCompression(b *testing.B) {
	data := bytes.Repeat([]byte("test data "), 1000) // 10KB of compressible data

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	})

	compressed := middleware.Compression()(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		compressed.ServeHTTP(w, req)
	}
}

// BenchmarkPOSTWithBody benchmarks POST request with body parsing
func BenchmarkPOSTWithBody(b *testing.B) {
	type Request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
	})

	body, _ := json.Marshal(Request{Name: "John", Email: "john@example.com"})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkStaticFileServing benchmarks static file serving
func BenchmarkStaticFileServing(b *testing.B) {
	// Create a simple static file handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write([]byte("<!DOCTYPE html><html><body>Test</body></html>"))
	})

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkKeepAliveConnections benchmarks keep-alive connections
func BenchmarkKeepAliveConnections(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "keep-alive")
		w.Write([]byte("OK"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
