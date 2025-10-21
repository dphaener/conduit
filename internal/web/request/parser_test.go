package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParser_ParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		target      interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid JSON object",
			body:    `{"name":"John","age":30}`,
			target:  &map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			body:    `[1,2,3]`,
			target:  &[]int{},
			wantErr: false,
		},
		{
			name:        "empty body",
			body:        "",
			target:      &map[string]interface{}{},
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid JSON",
			body:        `{invalid}`,
			target:      &map[string]interface{}{},
			wantErr:     true,
			errContains: "invalid JSON",
		},
		{
			name:        "unknown field with strict parsing",
			body:        `{"name":"John","unknown_field":"value"}`,
			target:      &struct{ Name string }{},
			wantErr:     true,
			errContains: "unknown field",
		},
		{
			name:        "multiple JSON objects",
			body:        `{"a":1}{"b":2}`,
			target:      &map[string]interface{}{},
			wantErr:     true,
			errContains: "multiple JSON objects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			err := parser.ParseJSON(w, req, tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("ParseJSON() error = %v, should contain %v", err, tt.errContains)
			}
		})
	}
}

func TestParser_ParseForm(t *testing.T) {
	tests := []struct {
		name        string
		formData    url.Values
		target      interface{}
		wantErr     bool
		checkResult func(t *testing.T, target interface{})
	}{
		{
			name: "simple form data to map",
			formData: url.Values{
				"name":  []string{"John"},
				"email": []string{"john@example.com"},
			},
			target:  &map[string]interface{}{},
			wantErr: false,
			checkResult: func(t *testing.T, target interface{}) {
				m := *target.(*map[string]interface{})
				if m["name"] != "John" {
					t.Errorf("expected name=John, got %v", m["name"])
				}
			},
		},
		{
			name: "numeric values",
			formData: url.Values{
				"age":   []string{"30"},
				"price": []string{"19.99"},
			},
			target:  &map[string]string{},
			wantErr: false,
			checkResult: func(t *testing.T, target interface{}) {
				m := *target.(*map[string]string)
				if m["age"] != "30" {
					t.Errorf("expected age=30, got %v", m["age"])
				}
			},
		},
		{
			name: "boolean values",
			formData: url.Values{
				"active": []string{"true"},
			},
			target:  &map[string]string{},
			wantErr: false,
			checkResult: func(t *testing.T, target interface{}) {
				m := *target.(*map[string]string)
				if m["active"] != "true" {
					t.Errorf("expected active=true, got %v", m["active"])
				}
			},
		},
		{
			name: "multiple values",
			formData: url.Values{
				"tags": []string{"go", "web", "api"},
			},
			target:  &map[string]interface{}{},
			wantErr: false,
			checkResult: func(t *testing.T, target interface{}) {
				m := *target.(*map[string]interface{})
				tags, ok := m["tags"].([]string)
				if !ok || len(tags) != 3 {
					t.Errorf("expected 3 tags, got %v", m["tags"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			w := httptest.NewRecorder()
			body := tt.formData.Encode()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			err := parser.ParseForm(w, req, tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseForm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, tt.target)
			}
		})
	}
}

func TestParser_ParseMultipart(t *testing.T) {
	parser := NewParser()
	w := httptest.NewRecorder()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	writer.WriteField("name", "John")
	writer.WriteField("age", "30")

	// Add file (we'll test file handling separately in upload_test.go)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	target := &map[string]interface{}{}
	err := parser.ParseMultipart(w, req, target)

	if err != nil {
		t.Errorf("ParseMultipart() error = %v", err)
		return
	}

	m := *target
	if m["name"] != "John" {
		t.Errorf("expected name=John, got %v", m["name"])
	}
}

func TestParser_Parse_ContentTypeDetection(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     bool
	}{
		{
			name:        "JSON content type",
			contentType: "application/json",
			body:        `{"test":"value"}`,
			wantErr:     false,
		},
		{
			name:        "JSON with charset",
			contentType: "application/json; charset=utf-8",
			body:        `{"test":"value"}`,
			wantErr:     false,
		},
		{
			name:        "form urlencoded",
			contentType: "application/x-www-form-urlencoded",
			body:        "test=value",
			wantErr:     false,
		},
		{
			name:        "unsupported content type",
			contentType: "application/xml",
			body:        "<test>value</test>",
			wantErr:     true,
		},
		{
			name:        "no content type defaults to JSON",
			contentType: "",
			body:        `{"test":"value"}`,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			target := &map[string]interface{}{}
			err := parser.Parse(w, req, target)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_MaxBodySize(t *testing.T) {
	parser := NewParserWithMaxSize(100) // 100 bytes max
	w := httptest.NewRecorder()

	// Create a body larger than max size
	largeBody := strings.Repeat("a", 200)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	target := &map[string]interface{}{}
	err := parser.ParseJSON(w, req, target)

	if err == nil {
		t.Error("expected error for body exceeding max size")
	}
}

func TestGetParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.SetPathValue("id", "123")

	result := GetParam(req, "id")
	if result != "123" {
		t.Errorf("GetParam() = %v, want %v", result, "123")
	}
}

func TestGetQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?name=John&age=30", nil)

	name := GetQueryParam(req, "name")
	if name != "John" {
		t.Errorf("GetQueryParam(name) = %v, want John", name)
	}

	missing := GetQueryParam(req, "missing")
	if missing != "" {
		t.Errorf("GetQueryParam(missing) = %v, want empty string", missing)
	}
}

func TestGetQueryParamInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=5&invalid=abc", nil)

	page := GetQueryParamInt(req, "page", 1)
	if page != 5 {
		t.Errorf("GetQueryParamInt(page) = %v, want 5", page)
	}

	invalid := GetQueryParamInt(req, "invalid", 1)
	if invalid != 1 {
		t.Errorf("GetQueryParamInt(invalid) = %v, want default 1", invalid)
	}

	missing := GetQueryParamInt(req, "missing", 10)
	if missing != 10 {
		t.Errorf("GetQueryParamInt(missing) = %v, want default 10", missing)
	}
}

func TestGetQueryParamBool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?active=true&invalid=xyz", nil)

	active := GetQueryParamBool(req, "active", false)
	if !active {
		t.Error("GetQueryParamBool(active) should be true")
	}

	invalid := GetQueryParamBool(req, "invalid", true)
	if !invalid {
		t.Error("GetQueryParamBool(invalid) should return default true")
	}

	missing := GetQueryParamBool(req, "missing", false)
	if missing {
		t.Error("GetQueryParamBool(missing) should return default false")
	}
}

func TestGetHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Header", "test-value")

	value := GetHeader(req, "X-Custom-Header")
	if value != "test-value" {
		t.Errorf("GetHeader() = %v, want test-value", value)
	}

	missing := GetHeader(req, "Missing-Header")
	if missing != "" {
		t.Errorf("GetHeader(missing) = %v, want empty string", missing)
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"true", true},
		{"false", false},
		{"123", int64(123)},
		{"45.67", 45.67},
		{"hello", "hello"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseValue(tt.input)
			if result != tt.expected {
				t.Errorf("parseValue(%v) = %v (%T), want %v (%T)",
					tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParser_ParseJSON(b *testing.B) {
	parser := NewParser()
	jsonBody := `{"name":"John","email":"john@example.com","age":30,"active":true}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		target := &map[string]interface{}{}
		parser.ParseJSON(w, req, target)
	}
}

func BenchmarkParser_ParseForm(b *testing.B) {
	parser := NewParser()
	formData := "name=John&email=john@example.com&age=30"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		target := &map[string]interface{}{}
		parser.ParseForm(w, req, target)
	}
}
