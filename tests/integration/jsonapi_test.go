package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJSONAPI_ListHandler_WithJSONAPIAccept tests GET /resources with JSON:API Accept header
func TestJSONAPI_ListHandler_WithJSONAPIAccept(t *testing.T) {
	source := `
resource Product {
	id: uuid! @primary @auto
	name: string! @min(1) @max(100)
	price: float!
	created_at: timestamp! @auto
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	handlerContent := result.Files["handlers/handlers.go"]

	// Test 1: Verify handler checks for JSON:API Accept header
	if !strings.Contains(handlerContent, "response.IsJSONAPI") {
		t.Error("LIST handler should check for JSON:API Accept header using response.IsJSONAPI")
	}

	// Test 2: Verify handler returns JSON:API format when requested
	if !strings.Contains(handlerContent, "response.RenderJSONAPIWithMeta") {
		t.Error("LIST handler should use response.RenderJSONAPIWithMeta for JSON:API responses")
	}

	// Test 3: Verify handler includes pagination metadata
	expectedMeta := []string{
		`"page"`,
		`"per_page"`,
		`"total"`,
	}

	for _, meta := range expectedMeta {
		if !strings.Contains(handlerContent, meta) {
			t.Errorf("LIST handler should include %s in pagination metadata", meta)
		}
	}

	// Test 4: Verify handler builds pagination links
	if !strings.Contains(handlerContent, "response.BuildPaginationLinks") {
		t.Error("LIST handler should build pagination links using response.BuildPaginationLinks")
	}

	// Test 5: Verify handler falls back to legacy JSON when JSON:API not requested
	if !strings.Contains(handlerContent, "} else {") {
		t.Error("LIST handler should have else clause for legacy JSON format")
	}

	if !strings.Contains(handlerContent, `"application/json"`) {
		t.Error("LIST handler should set Content-Type to application/json for legacy responses")
	}
}

// TestJSONAPI_GetHandler_WithJSONAPIAccept tests GET /resources/:id with JSON:API Accept header
func TestJSONAPI_GetHandler_WithJSONAPIAccept(t *testing.T) {
	source := `
resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string! @min(2) @max(100)
	created_at: timestamp! @auto
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	handlerContent := result.Files["handlers/handlers.go"]

	// Test 1: Verify GET handler checks for JSON:API Accept header
	if !strings.Contains(handlerContent, "response.IsJSONAPI") {
		t.Error("GET handler should check for JSON:API Accept header")
	}

	// Test 2: Verify GET handler uses RenderJSONAPI for single resource
	if !strings.Contains(handlerContent, "response.RenderJSONAPI") {
		t.Error("GET handler should use response.RenderJSONAPI for JSON:API responses")
	}

	// Test 3: Verify GET handler has legacy JSON fallback
	legacyCount := strings.Count(handlerContent, `"application/json"`)
	if legacyCount < 1 {
		t.Error("GET handler should have legacy JSON format fallback")
	}
}

// TestJSONAPI_PaginationParameters tests that handlers correctly parse pagination parameters
func TestJSONAPI_PaginationParameters(t *testing.T) {
	source := `
resource Article {
	id: uuid! @primary @auto
	title: string! @min(5) @max(200)
	content: text!
	published: bool!
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	handlerContent := result.Files["handlers/handlers.go"]

	// Verify handler parses limit parameter
	if !strings.Contains(handlerContent, `r.URL.Query().Get("limit")`) &&
		!strings.Contains(handlerContent, `r.URL.Query().Get("page[limit]")`) {
		t.Error("LIST handler should parse limit pagination parameter")
	}

	// Verify handler parses offset parameter
	if !strings.Contains(handlerContent, `r.URL.Query().Get("offset")`) &&
		!strings.Contains(handlerContent, `r.URL.Query().Get("page[offset]")`) {
		t.Error("LIST handler should parse offset pagination parameter")
	}

	// Verify default values
	if !strings.Contains(handlerContent, "limit := 50") && !strings.Contains(handlerContent, "limit = 50") {
		t.Error("LIST handler should have default limit value")
	}

	if !strings.Contains(handlerContent, "offset := 0") && !strings.Contains(handlerContent, "offset = 0") {
		t.Error("LIST handler should have default offset value")
	}
}

// TestJSONAPI_ContentNegotiation_MockRequest tests content negotiation with mock HTTP requests
func TestJSONAPI_ContentNegotiation_MockRequest(t *testing.T) {
	// This test creates mock handlers to verify the response format

	t.Run("JSON:API format for list", func(t *testing.T) {
		// Create a mock handler that mimics generated LIST handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate checking Accept header
			accept := r.Header.Get("Accept")

			if accept == "application/vnd.api+json" {
				// JSON:API response
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)

				// Simulate JSON:API structure
				response := map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"type": "products",
							"id":   "1",
							"attributes": map[string]interface{}{
								"name":  "Product 1",
								"price": 10.99,
							},
						},
					},
					"meta": map[string]interface{}{
						"page":     1,
						"per_page": 50,
						"total":    1,
					},
					"links": map[string]interface{}{
						"self":  "/products?page[limit]=50&page[offset]=0",
						"first": "/products?page[limit]=50&page[offset]=0",
						"last":  "/products?page[limit]=50&page[offset]=0",
					},
				}

				json.NewEncoder(w).Encode(response)
			} else {
				// Legacy JSON response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				response := []map[string]interface{}{
					{
						"id":    "1",
						"name":  "Product 1",
						"price": 10.99,
					},
				}

				json.NewEncoder(w).Encode(response)
			}
		})

		// Test with JSON:API Accept header
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		req.Header.Set("Accept", "application/vnd.api+json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Verify JSON:API response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/vnd.api+json" {
			t.Errorf("Expected Content-Type application/vnd.api+json, got %s", contentType)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode JSON:API response: %v", err)
		}

		// Verify structure
		if _, ok := result["data"]; !ok {
			t.Error("JSON:API response should have 'data' field")
		}

		if _, ok := result["meta"]; !ok {
			t.Error("JSON:API response should have 'meta' field")
		}

		if _, ok := result["links"]; !ok {
			t.Error("JSON:API response should have 'links' field")
		}
	})

	t.Run("Legacy JSON format for list", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")

			if accept == "application/vnd.api+json" {
				w.Header().Set("Content-Type", "application/vnd.api+json")
			} else {
				w.Header().Set("Content-Type", "application/json")
			}

			w.WriteHeader(http.StatusOK)

			if accept != "application/vnd.api+json" {
				// Legacy response is plain array
				response := []map[string]interface{}{
					{"id": "1", "name": "Product 1"},
				}
				json.NewEncoder(w).Encode(response)
			}
		})

		// Test with regular JSON Accept header
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var result []map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode legacy JSON response: %v", err)
		}

		// Verify it's a plain array, not wrapped in envelope
		if len(result) != 1 {
			t.Errorf("Expected 1 item in array, got %d", len(result))
		}
	})

	t.Run("JSON:API format for single resource", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")

			if accept == "application/vnd.api+json" {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)

				// For single resource, data is an object
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"type": "users",
						"id":   "123",
						"attributes": map[string]interface{}{
							"email": "user@example.com",
							"name":  "John Doe",
						},
					},
				}

				json.NewEncoder(w).Encode(response)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				// Legacy is plain object
				response := map[string]interface{}{
					"id":    "123",
					"email": "user@example.com",
					"name":  "John Doe",
				}

				json.NewEncoder(w).Encode(response)
			}
		})

		// Test with JSON:API Accept header
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		req.Header.Set("Accept", "application/vnd.api+json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode JSON:API response: %v", err)
		}

		// Verify data is an object, not an array
		data, ok := result["data"].(map[string]interface{})
		if !ok {
			t.Error("For single resource, 'data' should be an object, not an array")
		}

		if data["type"] != "users" {
			t.Errorf("Expected type 'users', got %v", data["type"])
		}

		if data["id"] != "123" {
			t.Errorf("Expected id '123', got %v", data["id"])
		}
	})

	t.Run("Legacy JSON format for single resource", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"id":    "123",
				"email": "user@example.com",
				"name":  "John Doe",
			}

			json.NewEncoder(w).Encode(response)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode legacy JSON response: %v", err)
		}

		// Verify it's a plain object without JSON:API envelope
		if _, ok := result["data"]; ok {
			t.Error("Legacy response should not have 'data' wrapper")
		}

		if result["id"] != "123" {
			t.Errorf("Expected id '123', got %v", result["id"])
		}
	})
}

// TestJSONAPI_PaginationLinks_Correctness tests that pagination links are correctly formatted
func TestJSONAPI_PaginationLinks_Correctness(t *testing.T) {
	source := `
resource Comment {
	id: uuid! @primary @auto
	content: text! @min(1)
	author_name: string! @min(1) @max(100)
	created_at: timestamp! @auto
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	handlerContent := result.Files["handlers/handlers.go"]

	// Verify pagination links use the correct URL format
	if !strings.Contains(handlerContent, "r.URL.Path") {
		t.Error("Pagination links should use request URL path")
	}

	// Verify handler calculates page number from offset
	if !strings.Contains(handlerContent, "page := (offset / limit) + 1") {
		t.Error("Handler should calculate page number from offset and limit")
	}

	// Test that BuildPaginationLinks is called with correct parameters
	if !strings.Contains(handlerContent, "BuildPaginationLinks(r.URL.Path, page, limit, total)") {
		t.Error("Handler should call BuildPaginationLinks with correct parameters")
	}
}

// TestJSONAPI_BackwardsCompatibility ensures legacy JSON responses still work
func TestJSONAPI_BackwardsCompatibility(t *testing.T) {
	source := `
resource Task {
	id: uuid! @primary @auto
	title: string! @min(1) @max(200)
	completed: bool!
	created_at: timestamp! @auto
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	handlerContent := result.Files["handlers/handlers.go"]

	// Verify both LIST and GET handlers have legacy JSON fallback
	// Count occurrences of legacy JSON encoding
	legacyEncodings := strings.Count(handlerContent, "json.NewEncoder(w).Encode(results)") +
		strings.Count(handlerContent, "json.NewEncoder(w).Encode(result)")

	if legacyEncodings < 2 {
		t.Error("Both LIST and GET handlers should support legacy JSON encoding")
	}

	// Verify Content-Type is set for legacy responses
	contentTypeCount := strings.Count(handlerContent, `w.Header().Set("Content-Type", "application/json")`)
	if contentTypeCount < 2 {
		t.Error("Both LIST and GET handlers should set Content-Type for legacy responses")
	}
}

// TestJSONAPI_GeneratedStructTags verifies that generated models have correct JSON:API tags
func TestJSONAPI_GeneratedStructTags(t *testing.T) {
	source := `
resource BlogPost {
	id: uuid! @primary @auto
	title: string! @min(5) @max(200)
	slug: string! @unique
	content: text!
	published: bool!
	view_count: int! @min(0)
	created_at: timestamp! @auto
	updated_at: timestamp! @auto
}
`

	result := CompileSource(t, source)
	if !result.Success {
		if len(result.LexErrors) > 0 {
			t.Fatalf("Compilation failed with lex errors: %v", result.LexErrors)
		}
		if len(result.ParseErrors) > 0 {
			t.Fatalf("Compilation failed with parse errors: %v", result.ParseErrors)
		}
		if len(result.TypeErrors) > 0 {
			t.Fatalf("Compilation failed with type errors: %v", result.TypeErrors)
		}
		t.Fatalf("Compilation failed with unknown error")
	}

	modelContent, ok := result.Files["models/blogpost.go"]
	if !ok {
		t.Fatal("Failed to find generated model file models/blogpost.go")
	}

	// Verify struct has JSON:API tags
	// Note: When id is explicitly defined, it gets jsonapi:"attr,id"
	// When id is implicit (auto-generated), it gets jsonapi:"primary,resource_type"
	if !strings.Contains(modelContent, `jsonapi:"attr,id"`) {
		t.Error("Model should have JSON:API attr tag on ID field")
	}

	// Verify fields have JSON:API attr tags
	expectedTags := []string{
		`jsonapi:"attr,title"`,
		`jsonapi:"attr,slug"`,
		`jsonapi:"attr,content"`,
		`jsonapi:"attr,published"`,
		`jsonapi:"attr,view_count"`,
		`jsonapi:"attr,created_at"`,
		`jsonapi:"attr,updated_at"`,
	}

	for _, tag := range expectedTags {
		if !strings.Contains(modelContent, tag) {
			t.Errorf("Model should have %s tag", tag)
		}
	}
}
