package router

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	assert.NotNil(t, router)
	assert.NotNil(t, router.mux)
	assert.NotNil(t, router.routes)
	assert.NotNil(t, router.groups)
	assert.NotNil(t, router.registeredRoutes)
}

func TestRouterHTTPMethods(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		pattern string
		setup   func(*Router, http.HandlerFunc) *Route
	}{
		{
			name:    "GET route",
			method:  http.MethodGet,
			pattern: "/test",
			setup:   func(r *Router, h http.HandlerFunc) *Route { return r.Get("/test", h) },
		},
		{
			name:    "POST route",
			method:  http.MethodPost,
			pattern: "/test",
			setup:   func(r *Router, h http.HandlerFunc) *Route { return r.Post("/test", h) },
		},
		{
			name:    "PUT route",
			method:  http.MethodPut,
			pattern: "/test",
			setup:   func(r *Router, h http.HandlerFunc) *Route { return r.Put("/test", h) },
		},
		{
			name:    "PATCH route",
			method:  http.MethodPatch,
			pattern: "/test",
			setup:   func(r *Router, h http.HandlerFunc) *Route { return r.Patch("/test", h) },
		},
		{
			name:    "DELETE route",
			method:  http.MethodDelete,
			pattern: "/test",
			setup:   func(r *Router, h http.HandlerFunc) *Route { return r.Delete("/test", h) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			called := false
			handler := func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}

			route := tt.setup(router, handler)

			// Verify route was registered
			assert.NotNil(t, route)
			assert.Equal(t, tt.pattern, route.Pattern)
			assert.Equal(t, tt.method, route.Method)

			// Test the route
			req := httptest.NewRequest(tt.method, tt.pattern, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.True(t, called, "handler should have been called")
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "success", w.Body.String())
		})
	}
}

func TestRouterPathParameters(t *testing.T) {
	router := NewRouter()

	var capturedID string
	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetPathParam(r, "id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "123", capturedID)
}

func TestRouterMultiplePathParameters(t *testing.T) {
	router := NewRouter()

	var capturedPostID, capturedCommentID string
	router.Get("/posts/{post_id}/comments/{comment_id}", func(w http.ResponseWriter, r *http.Request) {
		capturedPostID = GetPathParam(r, "post_id")
		capturedCommentID = GetPathParam(r, "comment_id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/456/comments/789", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "456", capturedPostID)
	assert.Equal(t, "789", capturedCommentID)
}

func TestRouterNamedRoutes(t *testing.T) {
	router := NewRouter()

	route := router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Named("posts.show")

	assert.Equal(t, "posts.show", route.Name)

	// Test GetRoute
	found, err := router.GetRoute("posts.show")
	require.NoError(t, err)
	assert.Equal(t, "/posts/{id}", found.Pattern)
	assert.Equal(t, http.MethodGet, found.Method)

	// Test non-existent route
	_, err = router.GetRoute("posts.invalid")
	assert.Error(t, err)
}

func TestRouterResourceMetadata(t *testing.T) {
	router := NewRouter()

	route := router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).WithResource("Post", OpShow)

	assert.Equal(t, "Post", route.ResourceName)
	assert.Equal(t, OpShow, route.Operation)
}

func TestRouterGroup(t *testing.T) {
	router := NewRouter()

	router.Group("/api", func(r chi.Router) {
		r.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("api posts"))
		})
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("api users"))
		})
	})

	// Test /api/posts
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "api posts", w.Body.String())

	// Test /api/users
	req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "api users", w.Body.String())
}

func TestRouterGetRoutes(t *testing.T) {
	router := NewRouter()

	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {})
	router.Post("/posts", func(w http.ResponseWriter, r *http.Request) {})
	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.GetRoutes()
	assert.Len(t, routes, 3)

	// Verify each route has proper metadata
	for _, route := range routes {
		assert.NotEmpty(t, route.Pattern)
		assert.NotEmpty(t, route.Method)
	}
}

func TestRouterNotFound(t *testing.T) {
	router := NewRouter()

	customNotFound := false
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		customNotFound = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom not found"))
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.True(t, customNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "custom not found", w.Body.String())
}

func TestRouterMethodNotAllowed(t *testing.T) {
	router := NewRouter()

	customMethodNotAllowed := false
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		customMethodNotAllowed = true
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("method not allowed"))
	})

	// Register a GET route
	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Try to POST to it (method not allowed)
	req := httptest.NewRequest(http.MethodPost, "/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.True(t, customMethodNotAllowed)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestExtractParameters(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []RouteParameter
	}{
		{
			name:     "no parameters",
			pattern:  "/posts",
			expected: []RouteParameter{},
		},
		{
			name:    "single parameter",
			pattern: "/posts/{id}",
			expected: []RouteParameter{
				{Name: "id", Type: "uuid", Required: true, Source: PathParam},
			},
		},
		{
			name:    "multiple parameters",
			pattern: "/posts/{post_id}/comments/{comment_id}",
			expected: []RouteParameter{
				{Name: "post_id", Type: "uuid", Required: true, Source: PathParam},
				{Name: "comment_id", Type: "uuid", Required: true, Source: PathParam},
			},
		},
		{
			name:    "non-id parameter",
			pattern: "/posts/{slug}",
			expected: []RouteParameter{
				{Name: "slug", Type: "string", Required: true, Source: PathParam},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := extractParameters(tt.pattern)
			assert.Equal(t, len(tt.expected), len(params))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.Name, params[i].Name)
				assert.Equal(t, expected.Type, params[i].Type)
				assert.Equal(t, expected.Required, params[i].Required)
				assert.Equal(t, expected.Source, params[i].Source)
			}
		})
	}
}

func TestInferParameterType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"id", "uuid"},
		{"post_id", "uuid"},
		{"userId", "uuid"},
		{"page", "int"},
		{"limit", "int"},
		{"offset", "int"},
		{"count", "int"},
		{"slug", "string"},
		{"name", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferParameterType(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCRUDOperationString(t *testing.T) {
	tests := []struct {
		op       CRUDOperation
		expected string
	}{
		{OpList, "list"},
		{OpCreate, "create"},
		{OpShow, "show"},
		{OpUpdate, "update"},
		{OpPatch, "patch"},
		{OpDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.op.String())
		})
	}
}

func TestParameterSourceString(t *testing.T) {
	tests := []struct {
		source   ParameterSource
		expected string
	}{
		{PathParam, "path"},
		{QueryParam, "query"},
		{HeaderParam, "header"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}

func TestRouterServeHTTP(t *testing.T) {
	router := NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	// Create a test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Make a request
	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "healthy", string(body))
}

func TestRouterRouteList(t *testing.T) {
	router := NewRouter()

	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.list")
	router.Post("/posts", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.create")
	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.show")

	list := router.RouteList()
	assert.NotEmpty(t, list)
	assert.Contains(t, list, "GET")
	assert.Contains(t, list, "POST")
	assert.Contains(t, list, "/posts")
	assert.Contains(t, list, "/posts/{id}")
}

func TestRouterURL(t *testing.T) {
	router := NewRouter()

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.show")
	router.Get("/posts/{post_id}/comments/{id}", func(w http.ResponseWriter, r *http.Request) {}).Named("comments.show")

	tests := []struct {
		name     string
		params   map[string]string
		expected string
		hasError bool
	}{
		{
			name:     "posts.show",
			params:   map[string]string{"id": "123"},
			expected: "/posts/123",
			hasError: false,
		},
		{
			name:     "comments.show",
			params:   map[string]string{"post_id": "456", "id": "789"},
			expected: "/posts/456/comments/789",
			hasError: false,
		},
		{
			name:     "posts.show",
			params:   map[string]string{}, // Missing id
			expected: "",
			hasError: true,
		},
		{
			name:     "nonexistent.route",
			params:   map[string]string{},
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%v", tt.name, tt.params), func(t *testing.T) {
			url, err := router.URL(tt.name, tt.params)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, url)
			}
		})
	}
}
