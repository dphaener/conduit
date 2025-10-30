package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test models for JSON:API tests - these mimic generated models
type TestUser struct {
	ID   string `jsonapi:"primary,test_users" json:"id"`
	Name string `jsonapi:"attr,name" json:"name"`
}

type TestProduct struct {
	ID    string  `jsonapi:"primary,test_products" json:"id"`
	Name  string  `jsonapi:"attr,name" json:"name"`
	Price float64 `jsonapi:"attr,price" json:"price"`
}

// TestIsJSONAPI verifies the IsJSONAPI function correctly identifies JSON:API requests
func TestIsJSONAPI(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{
			name:   "JSON:API media type",
			accept: "application/vnd.api+json",
			want:   true,
		},
		{
			name:   "Regular JSON",
			accept: "application/json",
			want:   false,
		},
		{
			name:   "Empty accept header",
			accept: "",
			want:   false,
		},
		{
			name:   "Wildcard accept",
			accept: "*/*",
			want:   false,
		},
		{
			name:   "Text HTML",
			accept: "text/html",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", tt.accept)

			got := IsJSONAPI(req)
			if got != tt.want {
				t.Errorf("IsJSONAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRenderJSONAPI verifies basic JSON:API rendering
func TestRenderJSONAPI(t *testing.T) {
	t.Run("sets correct content type", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestUser{
			{ID: "1", Name: "Test User"},
		}

		err := RenderJSONAPI(w, http.StatusOK, payload)
		if err != nil {
			t.Fatalf("RenderJSONAPI() error = %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != JSONAPIMediaType {
			t.Errorf("Content-Type = %v, want %v", contentType, JSONAPIMediaType)
		}
	})

	t.Run("sets correct status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestUser{
			{ID: "1", Name: "Test User"},
		}

		err := RenderJSONAPI(w, http.StatusCreated, payload)
		if err != nil {
			t.Fatalf("RenderJSONAPI() error = %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusCreated)
		}
	})

	t.Run("renders single resource", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := &TestUser{ID: "123", Name: "John Doe"}

		err := RenderJSONAPI(w, http.StatusOK, payload)
		if err != nil {
			t.Fatalf("RenderJSONAPI() error = %v", err)
		}

		// Verify response structure
		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check for data object (not array for single resource)
		data, ok := result["data"]
		if !ok {
			t.Error("Response should contain 'data' field")
		}

		// For single resource, data should be an object
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			t.Error("For single resource, data should be an object, not an array")
		}

		if dataMap["type"] != "test_users" {
			t.Errorf("type = %v, want 'test_users'", dataMap["type"])
		}

		if dataMap["id"] != "123" {
			t.Errorf("id = %v, want '123'", dataMap["id"])
		}
	})

	t.Run("renders collection", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestProduct{
			{ID: "1", Name: "Product 1", Price: 10.99},
			{ID: "2", Name: "Product 2", Price: 20.50},
		}

		err := RenderJSONAPI(w, http.StatusOK, payload)
		if err != nil {
			t.Fatalf("RenderJSONAPI() error = %v", err)
		}

		// Verify response structure
		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check for data array
		data, ok := result["data"]
		if !ok {
			t.Error("Response should contain 'data' field")
		}

		// For collection, data should be an array
		dataArray, ok := data.([]interface{})
		if !ok {
			t.Error("For collection, data should be an array")
		}

		if len(dataArray) != 2 {
			t.Errorf("Expected 2 items in collection, got %d", len(dataArray))
		}
	})
}

// TestRenderJSONAPIWithMeta verifies JSON:API rendering with metadata and links
func TestRenderJSONAPIWithMeta(t *testing.T) {
	t.Run("includes meta and links", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestUser{
			{ID: "1", Name: "Test User"},
		}

		meta := map[string]interface{}{
			"page":     1,
			"per_page": 10,
			"total":    25,
		}

		links := BuildPaginationLinks("/api/resources", 1, 10, 25)

		err := RenderJSONAPIWithMeta(w, http.StatusOK, payload, meta, links)
		if err != nil {
			t.Fatalf("RenderJSONAPIWithMeta() error = %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != JSONAPIMediaType {
			t.Errorf("Content-Type = %v, want %v", contentType, JSONAPIMediaType)
		}

		// Verify response structure
		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check for meta
		if _, ok := result["meta"]; !ok {
			t.Error("Response should contain 'meta' field")
		}

		// Check for links
		if _, ok := result["links"]; !ok {
			t.Error("Response should contain 'links' field")
		}

		// Verify meta values
		metaMap, ok := result["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("meta should be a map")
		}

		if metaMap["page"] != float64(1) {
			t.Errorf("meta.page = %v, want 1", metaMap["page"])
		}

		if metaMap["per_page"] != float64(10) {
			t.Errorf("meta.per_page = %v, want 10", metaMap["per_page"])
		}

		if metaMap["total"] != float64(25) {
			t.Errorf("meta.total = %v, want 25", metaMap["total"])
		}
	})

	t.Run("works with nil meta", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestUser{
			{ID: "1", Name: "Test User"},
		}

		links := BuildPaginationLinks("/api/resources", 1, 10, 25)

		err := RenderJSONAPIWithMeta(w, http.StatusOK, payload, nil, links)
		if err != nil {
			t.Fatalf("RenderJSONAPIWithMeta() with nil meta error = %v", err)
		}
	})

	t.Run("works with nil links", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []*TestUser{
			{ID: "1", Name: "Test User"},
		}

		meta := map[string]interface{}{
			"page": 1,
		}

		err := RenderJSONAPIWithMeta(w, http.StatusOK, payload, meta, nil)
		if err != nil {
			t.Fatalf("RenderJSONAPIWithMeta() with nil links error = %v", err)
		}
	})
}

// TestBuildPaginationLinks verifies pagination link generation
func TestBuildPaginationLinks(t *testing.T) {
	t.Run("first page", func(t *testing.T) {
		links := BuildPaginationLinks("/api/resources", 1, 10, 25)

		if links.Self != "/api/resources?page[limit]=10&page[offset]=0" {
			t.Errorf("Self link = %v, want /api/resources?page[limit]=10&page[offset]=0", links.Self)
		}

		if links.First != "/api/resources?page[limit]=10&page[offset]=0" {
			t.Errorf("First link = %v, want /api/resources?page[limit]=10&page[offset]=0", links.First)
		}

		if links.Last != "/api/resources?page[limit]=10&page[offset]=20" {
			t.Errorf("Last link = %v, want /api/resources?page[limit]=10&page[offset]=20", links.Last)
		}

		if links.Prev != "" {
			t.Errorf("Prev link should be empty on first page, got %v", links.Prev)
		}

		if links.Next != "/api/resources?page[limit]=10&page[offset]=10" {
			t.Errorf("Next link = %v, want /api/resources?page[limit]=10&page[offset]=10", links.Next)
		}
	})

	t.Run("middle page", func(t *testing.T) {
		links := BuildPaginationLinks("/api/resources", 2, 10, 25)

		if links.Self != "/api/resources?page[limit]=10&page[offset]=10" {
			t.Errorf("Self link = %v, want /api/resources?page[limit]=10&page[offset]=10", links.Self)
		}

		if links.Prev != "/api/resources?page[limit]=10&page[offset]=0" {
			t.Errorf("Prev link = %v, want /api/resources?page[limit]=10&page[offset]=0", links.Prev)
		}

		if links.Next != "/api/resources?page[limit]=10&page[offset]=20" {
			t.Errorf("Next link = %v, want /api/resources?page[limit]=10&page[offset]=20", links.Next)
		}
	})

	t.Run("last page", func(t *testing.T) {
		links := BuildPaginationLinks("/api/resources", 3, 10, 25)

		if links.Self != "/api/resources?page[limit]=10&page[offset]=20" {
			t.Errorf("Self link = %v, want /api/resources?page[limit]=10&page[offset]=20", links.Self)
		}

		if links.Prev != "/api/resources?page[limit]=10&page[offset]=10" {
			t.Errorf("Prev link = %v, want /api/resources?page[limit]=10&page[offset]=10", links.Prev)
		}

		if links.Next != "" {
			t.Errorf("Next link should be empty on last page, got %v", links.Next)
		}
	})

	t.Run("single page", func(t *testing.T) {
		links := BuildPaginationLinks("/api/resources", 1, 10, 5)

		if links.Prev != "" {
			t.Errorf("Prev link should be empty when only one page, got %v", links.Prev)
		}

		if links.Next != "" {
			t.Errorf("Next link should be empty when only one page, got %v", links.Next)
		}
	})

	t.Run("handles zero total", func(t *testing.T) {
		links := BuildPaginationLinks("/api/resources", 1, 10, 0)

		// Should still generate valid links even with zero results
		if links.Self == "" {
			t.Error("Self link should not be empty")
		}

		if links.First == "" {
			t.Error("First link should not be empty")
		}

		if links.Last == "" {
			t.Error("Last link should not be empty")
		}
	})

	t.Run("calculates correct offset for different page sizes", func(t *testing.T) {
		// Page 3 with page size 20
		links := BuildPaginationLinks("/api/resources", 3, 20, 100)

		// Page 3 means offset should be (3-1)*20 = 40
		if links.Self != "/api/resources?page[limit]=20&page[offset]=40" {
			t.Errorf("Self link = %v, want /api/resources?page[limit]=20&page[offset]=40", links.Self)
		}
	})
}

// TestBuildPageURL verifies the internal buildPageURL function
func TestBuildPageURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		page    int
		perPage int
		want    string
	}{
		{
			name:    "page 1",
			baseURL: "/api/users",
			page:    1,
			perPage: 10,
			want:    "/api/users?page[limit]=10&page[offset]=0",
		},
		{
			name:    "page 2",
			baseURL: "/api/users",
			page:    2,
			perPage: 10,
			want:    "/api/users?page[limit]=10&page[offset]=10",
		},
		{
			name:    "page 5 with large page size",
			baseURL: "/api/posts",
			page:    5,
			perPage: 50,
			want:    "/api/posts?page[limit]=50&page[offset]=200",
		},
		{
			name:    "base URL with trailing slash",
			baseURL: "/api/comments/",
			page:    1,
			perPage: 25,
			want:    "/api/comments/?page[limit]=25&page[offset]=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPageURL(tt.baseURL, tt.page, tt.perPage)
			if got != tt.want {
				t.Errorf("buildPageURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
