package response

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderer_JSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantErr    bool
	}{
		{
			name:       "simple object",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantErr:    false,
		},
		{
			name:       "array",
			statusCode: http.StatusOK,
			data:       []int{1, 2, 3},
			wantErr:    false,
		},
		{
			name:       "nil data",
			statusCode: http.StatusOK,
			data:       nil,
			wantErr:    false,
		},
		{
			name:       "created status",
			statusCode: http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRenderer()
			w := httptest.NewRecorder()

			err := renderer.JSON(w, tt.statusCode, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if w.Code != tt.statusCode {
				t.Errorf("status code = %v, want %v", w.Code, tt.statusCode)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("content type = %v, want application/json", contentType)
			}

			// Verify valid JSON
			if tt.data != nil {
				var result interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Errorf("response is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestRenderer_JSONWithHeaders(t *testing.T) {
	renderer := NewRenderer()
	w := httptest.NewRecorder()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"X-Request-ID":    "123",
	}

	data := map[string]string{"status": "ok"}
	err := renderer.JSONWithHeaders(w, http.StatusOK, data, headers)

	if err != nil {
		t.Errorf("JSONWithHeaders() error = %v", err)
	}

	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("custom header not set")
	}

	if w.Header().Get("X-Request-ID") != "123" {
		t.Error("request ID header not set")
	}
}

func TestRenderer_JSONPrettyPrint(t *testing.T) {
	renderer := NewRendererWithPrettyPrint()
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	err := renderer.JSON(w, http.StatusOK, data)

	if err != nil {
		t.Errorf("JSON() error = %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "\n") || !strings.Contains(body, "  ") {
		t.Error("JSON should be pretty-printed with indentation")
	}
}

func TestRenderer_HTML(t *testing.T) {
	// Create a simple template
	tmpl := template.Must(template.New("test").Parse("<h1>{{.Title}}</h1>"))

	renderer := NewRenderer()
	renderer.SetTemplates(tmpl)

	w := httptest.NewRecorder()
	data := map[string]string{"Title": "Hello World"}

	err := renderer.HTML(w, http.StatusOK, "test", data)

	if err != nil {
		t.Errorf("HTML() error = %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("content type = %v, want text/html", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Hello World") {
		t.Errorf("body = %v, should contain 'Hello World'", body)
	}
}

func TestRenderer_HTML_NoTemplates(t *testing.T) {
	renderer := NewRenderer()
	w := httptest.NewRecorder()

	err := renderer.HTML(w, http.StatusOK, "test", nil)

	if err == nil {
		t.Error("HTML() should error when no templates loaded")
	}

	if !strings.Contains(err.Error(), "no templates") {
		t.Errorf("error should mention missing templates, got: %v", err)
	}
}

func TestRenderer_Text(t *testing.T) {
	renderer := NewRenderer()
	w := httptest.NewRecorder()

	text := "Hello, World!"
	err := renderer.Text(w, http.StatusOK, text)

	if err != nil {
		t.Errorf("Text() error = %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("content type = %v, want text/plain", contentType)
	}

	if w.Body.String() != text {
		t.Errorf("body = %v, want %v", w.Body.String(), text)
	}
}

func TestRenderer_NoContent(t *testing.T) {
	renderer := NewRenderer()
	w := httptest.NewRecorder()

	err := renderer.NoContent(w)

	if err != nil {
		t.Errorf("NoContent() error = %v", err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusNoContent)
	}

	if w.Body.Len() != 0 {
		t.Error("body should be empty for NoContent")
	}
}

func TestRenderer_Redirect(t *testing.T) {
	renderer := NewRenderer()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/old", nil)

	renderer.Redirect(w, req, "/new", http.StatusMovedPermanently)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusMovedPermanently)
	}

	location := w.Header().Get("Location")
	if location != "/new" {
		t.Errorf("Location header = %v, want /new", location)
	}
}

func TestRenderer_Negotiate(t *testing.T) {
	tests := []struct {
		name        string
		accept      string
		expectType  string
		expectError bool
	}{
		{
			name:       "JSON accept header",
			accept:     "application/json",
			expectType: "application/json",
		},
		{
			name:       "wildcard accept header",
			accept:     "*/*",
			expectType: "application/json",
		},
		{
			name:       "plain text accept header",
			accept:     "text/plain",
			expectType: "text/plain",
		},
		{
			name:       "no accept header defaults to JSON",
			accept:     "",
			expectType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRenderer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			data := map[string]string{"status": "ok"}
			err := renderer.Negotiate(w, req, http.StatusOK, data)

			if err != nil && !tt.expectError {
				t.Errorf("Negotiate() error = %v", err)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectType) {
				t.Errorf("content type = %v, want %v", contentType, tt.expectType)
			}
		})
	}
}

func TestRenderer_SetDefaultHeader(t *testing.T) {
	renderer := NewRenderer()
	renderer.SetDefaultHeader("X-API-Version", "1.0")

	w := httptest.NewRecorder()
	renderer.JSON(w, http.StatusOK, map[string]string{"test": "value"})

	if w.Header().Get("X-API-Version") != "1.0" {
		t.Error("default header not set")
	}
}

func TestAPIResponse(t *testing.T) {
	resp := NewAPIResponse(map[string]string{"name": "John"})

	resp.WithMeta("total", 100)
	resp.WithMeta("page", 1)
	resp.WithLink("self", "/api/users")
	resp.WithLink("next", "/api/users?page=2")

	if resp.Meta["total"] != 100 {
		t.Error("meta total not set")
	}

	if resp.Links["self"] != "/api/users" {
		t.Error("link not set")
	}
}

func TestAPIResponse_WithPagination(t *testing.T) {
	resp := NewAPIResponse([]int{1, 2, 3})
	resp.WithPagination(2, 10, 25)

	if resp.Meta["page"] != 2 {
		t.Errorf("page = %v, want 2", resp.Meta["page"])
	}

	if resp.Meta["per_page"] != 10 {
		t.Errorf("per_page = %v, want 10", resp.Meta["per_page"])
	}

	if resp.Meta["total"] != 25 {
		t.Errorf("total = %v, want 25", resp.Meta["total"])
	}

	if resp.Meta["total_pages"] != 3 {
		t.Errorf("total_pages = %v, want 3", resp.Meta["total_pages"])
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("RenderJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := RenderJSON(w, http.StatusOK, map[string]string{"test": "value"})
		if err != nil {
			t.Errorf("RenderJSON() error = %v", err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("status code = %v, want 200", w.Code)
		}
	})

	t.Run("RenderText", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := RenderText(w, http.StatusOK, "test")
		if err != nil {
			t.Errorf("RenderText() error = %v", err)
		}
		if w.Body.String() != "test" {
			t.Errorf("body = %v, want 'test'", w.Body.String())
		}
	})

	t.Run("RenderNoContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := RenderNoContent(w)
		if err != nil {
			t.Errorf("RenderNoContent() error = %v", err)
		}
		if w.Code != http.StatusNoContent {
			t.Errorf("status code = %v, want 204", w.Code)
		}
	})
}

// Benchmark tests
func BenchmarkRenderer_JSON(b *testing.B) {
	renderer := NewRenderer()
	data := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
		"age":   30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		renderer.JSON(w, http.StatusOK, data)
	}
}

func BenchmarkRenderer_JSONPretty(b *testing.B) {
	renderer := NewRendererWithPrettyPrint()
	data := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
		"age":   30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		renderer.JSON(w, http.StatusOK, data)
	}
}
