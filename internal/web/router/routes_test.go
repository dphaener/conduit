package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceDefinition(t *testing.T) {
	def := NewResourceDefinition("Post")

	assert.Equal(t, "Post", def.Name)
	assert.Equal(t, "Posts", def.PluralName)
	assert.Equal(t, "/posts", def.BasePath)
	assert.Equal(t, "id", def.IDParamName)
	assert.Equal(t, "uuid", def.IDType)
	assert.Len(t, def.Operations, 6) // All CRUD operations enabled by default
	assert.NotNil(t, def.Middleware)
}

func TestResourceHandlersValidate(t *testing.T) {
	handlers := ResourceHandlers{
		List:   func(w http.ResponseWriter, r *http.Request) {},
		Create: func(w http.ResponseWriter, r *http.Request) {},
		Show:   func(w http.ResponseWriter, r *http.Request) {},
		Update: func(w http.ResponseWriter, r *http.Request) {},
		Patch:  func(w http.ResponseWriter, r *http.Request) {},
		Delete: func(w http.ResponseWriter, r *http.Request) {},
	}

	// Should pass with all handlers
	err := handlers.Validate([]CRUDOperation{OpList, OpCreate, OpShow, OpUpdate, OpPatch, OpDelete})
	assert.NoError(t, err)

	// Should fail with missing handler
	incompleteHandlers := ResourceHandlers{
		List: func(w http.ResponseWriter, r *http.Request) {},
	}
	err = incompleteHandlers.Validate([]CRUDOperation{OpList, OpCreate})
	assert.Error(t, err)
}

func TestResourceHandlersGetHandler(t *testing.T) {
	listHandler := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("list")) }
	createHandler := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("create")) }

	handlers := ResourceHandlers{
		List:   listHandler,
		Create: createHandler,
	}

	assert.NotNil(t, handlers.GetHandler(OpList))
	assert.NotNil(t, handlers.GetHandler(OpCreate))
	assert.Nil(t, handlers.GetHandler(OpShow))
}

func TestRegisterResource(t *testing.T) {
	router := NewRouter()

	listCalled := false
	createCalled := false
	showCalled := false
	updateCalled := false
	patchCalled := false
	deleteCalled := false

	handlers := ResourceHandlers{
		List:   func(w http.ResponseWriter, r *http.Request) { listCalled = true; w.WriteHeader(http.StatusOK) },
		Create: func(w http.ResponseWriter, r *http.Request) { createCalled = true; w.WriteHeader(http.StatusCreated) },
		Show:   func(w http.ResponseWriter, r *http.Request) { showCalled = true; w.WriteHeader(http.StatusOK) },
		Update: func(w http.ResponseWriter, r *http.Request) { updateCalled = true; w.WriteHeader(http.StatusOK) },
		Patch:  func(w http.ResponseWriter, r *http.Request) { patchCalled = true; w.WriteHeader(http.StatusOK) },
		Delete: func(w http.ResponseWriter, r *http.Request) { deleteCalled = true; w.WriteHeader(http.StatusNoContent) },
	}

	def := NewResourceDefinition("Post")
	err := router.RegisterResource(def, handlers)
	require.NoError(t, err)

	// Test List
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, listCalled)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test Create
	req = httptest.NewRequest(http.MethodPost, "/posts", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, createCalled)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Test Show
	req = httptest.NewRequest(http.MethodGet, "/posts/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, showCalled)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test Update
	req = httptest.NewRequest(http.MethodPut, "/posts/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, updateCalled)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test Patch
	req = httptest.NewRequest(http.MethodPatch, "/posts/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, patchCalled)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test Delete
	req = httptest.NewRequest(http.MethodDelete, "/posts/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, deleteCalled)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRegisterResourcePartialOperations(t *testing.T) {
	router := NewRouter()

	handlers := ResourceHandlers{
		List: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
		Show: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
	}

	def := NewResourceDefinition("Post")
	def.Operations = []CRUDOperation{OpList, OpShow} // Only enable list and show

	err := router.RegisterResource(def, handlers)
	require.NoError(t, err)

	// List should work
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Show should work
	req = httptest.NewRequest(http.MethodGet, "/posts/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Create should not be registered
	req = httptest.NewRequest(http.MethodPost, "/posts", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRegisterResourceCustomPaths(t *testing.T) {
	router := NewRouter()

	handlers := ResourceHandlers{
		List: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
	}

	def := NewResourceDefinition("Article")
	def.BasePath = "/api/v1/articles"
	def.Operations = []CRUDOperation{OpList}

	err := router.RegisterResource(def, handlers)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterNestedResource(t *testing.T) {
	router := NewRouter()

	var capturedPostID, capturedCommentID string
	handlers := ResourceHandlers{
		List: func(w http.ResponseWriter, r *http.Request) {
			capturedPostID = GetPathParam(r, "id")
			w.WriteHeader(http.StatusOK)
		},
		Show: func(w http.ResponseWriter, r *http.Request) {
			capturedPostID = GetPathParam(r, "id")
			capturedCommentID = GetPathParam(r, "comment_id")
			w.WriteHeader(http.StatusOK)
		},
	}

	parent := NewResourceDefinition("Post")
	child := NewResourceDefinition("Comment")
	child.IDParamName = "comment_id" // Use different param name to avoid conflicts
	child.Operations = []CRUDOperation{OpList, OpShow}

	err := router.RegisterNestedResource(parent, child, handlers)
	require.NoError(t, err)

	// Test nested list
	req := httptest.NewRequest(http.MethodGet, "/posts/123/comments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "123", capturedPostID)

	// Test nested show
	req = httptest.NewRequest(http.MethodGet, "/posts/123/comments/456", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "456", capturedCommentID)
}

func TestRouteListFormatting(t *testing.T) {
	router := NewRouter()

	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.list")
	router.Post("/posts", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.create")
	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.show")

	list := router.RouteList()

	assert.Contains(t, list, "GET")
	assert.Contains(t, list, "POST")
	assert.Contains(t, list, "/posts")
	assert.Contains(t, list, "/posts/{id}")
	assert.Contains(t, list, "posts.list")
	assert.Contains(t, list, "posts.create")
	assert.Contains(t, list, "posts.show")
}

func TestRouteListJSON(t *testing.T) {
	router := NewRouter()

	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {})
	router.Post("/posts", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.RouteListJSON()

	assert.Len(t, routes, 2)
	assert.Equal(t, "/posts", routes[0].Pattern)
	assert.Equal(t, http.MethodGet, routes[0].Method)
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"post", "posts"},
		{"comment", "comments"},
		{"person", "people"},
		{"child", "children"},
		{"man", "men"},
		{"woman", "women"},
		{"tooth", "teeth"},
		{"mouse", "mice"},
		{"category", "categories"},
		{"box", "boxes"},
		{"buzz", "buzzes"},
		{"leaf", "leaves"},
		{"knife", "knives"},
		{"life", "lives"},
		{"day", "days"},
		{"key", "keys"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := pluralize(tt.singular)
			assert.Equal(t, tt.plural, result)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Post", "post"},
		{"PostComment", "post_comment"},
		{"HTTPRequest", "h_t_t_p_request"},
		{"UserProfile", "user_profile"},
		{"APIKey", "a_p_i_key"},
		{"singleword", "singleword"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVowel(t *testing.T) {
	vowels := []byte{'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U'}
	consonants := []byte{'b', 'c', 'd', 'f', 'g', 'B', 'C', 'D', 'F', 'G'}

	for _, v := range vowels {
		assert.True(t, isVowel(v), "Expected %c to be a vowel", v)
	}

	for _, c := range consonants {
		assert.False(t, isVowel(c), "Expected %c to not be a vowel", c)
	}
}

func TestRegisterResourceWithMetadata(t *testing.T) {
	router := NewRouter()

	handlers := ResourceHandlers{
		List:   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
		Create: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
	}

	def := NewResourceDefinition("Post")
	def.Operations = []CRUDOperation{OpList, OpCreate}

	err := router.RegisterResource(def, handlers)
	require.NoError(t, err)

	// Verify routes have resource metadata
	routes := router.GetRoutes()
	assert.Len(t, routes, 2)

	// Check that route names are generated correctly
	listRoute, err := router.GetRoute("Post.list")
	require.NoError(t, err)
	assert.Equal(t, "Post", listRoute.ResourceName)
	assert.Equal(t, OpList, listRoute.Operation)

	createRoute, err := router.GetRoute("Post.create")
	require.NoError(t, err)
	assert.Equal(t, "Post", createRoute.ResourceName)
	assert.Equal(t, OpCreate, createRoute.Operation)
}

func TestRegisterResourceInvalidHandlers(t *testing.T) {
	router := NewRouter()

	// Missing handlers
	handlers := ResourceHandlers{
		List: func(w http.ResponseWriter, r *http.Request) {},
	}

	def := NewResourceDefinition("Post")
	def.Operations = []CRUDOperation{OpList, OpCreate} // Requires Create handler

	err := router.RegisterResource(def, handlers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid handlers")
}

func TestURLGeneration(t *testing.T) {
	router := NewRouter()

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {}).Named("posts.show")
	router.Get("/posts/{post_id}/comments/{comment_id}", func(w http.ResponseWriter, r *http.Request) {}).Named("comments.show")

	// Test simple URL generation
	url, err := router.URL("posts.show", map[string]string{"id": "123"})
	require.NoError(t, err)
	assert.Equal(t, "/posts/123", url)

	// Test nested URL generation
	url, err = router.URL("comments.show", map[string]string{
		"post_id":    "123",
		"comment_id": "456",
	})
	require.NoError(t, err)
	assert.Equal(t, "/posts/123/comments/456", url)

	// Test missing parameters
	_, err = router.URL("posts.show", map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing parameter values")

	// Test non-existent route
	_, err = router.URL("nonexistent.route", map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route not found")
}
