package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Additional tests to increase coverage

func TestParser_ParseQuery(t *testing.T) {
	parser := NewParser()
	req := httptest.NewRequest(http.MethodGet, "/?name=John&age=30", nil)

	target := &map[string]interface{}{}
	err := parser.ParseQuery(req, target)

	if err != nil {
		t.Errorf("ParseQuery() error = %v", err)
	}

	m := *target
	if m["name"] != "John" {
		t.Errorf("name = %v, want John", m["name"])
	}
}

func TestGetQueryParamInt64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?id=12345678901234", nil)

	id := GetQueryParamInt64(req, "id", 0)
	if id != 12345678901234 {
		t.Errorf("GetQueryParamInt64() = %v, want 12345678901234", id)
	}

	missing := GetQueryParamInt64(req, "missing", 999)
	if missing != 999 {
		t.Errorf("GetQueryParamInt64(missing) = %v, want 999", missing)
	}

	// Test invalid value
	req2 := httptest.NewRequest(http.MethodGet, "/?invalid=abc", nil)
	invalid := GetQueryParamInt64(req2, "invalid", 100)
	if invalid != 100 {
		t.Errorf("GetQueryParamInt64(invalid) = %v, want 100", invalid)
	}
}

func TestFormToMap_MapStringString(t *testing.T) {
	values := map[string][]string{
		"key1": {"value1"},
		"key2": {"value2"},
	}

	target := &map[string]string{}
	err := formToMap(values, target)

	if err != nil {
		t.Errorf("formToMap() error = %v", err)
	}

	m := *target
	if m["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", m["key1"])
	}
}

func TestMultipartToMap_NoMultipartForm(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	target := &map[string]interface{}{}

	err := multipartToMap(req, target)
	if err == nil {
		t.Error("expected error when multipart form not parsed")
	}
}

func TestMultipartToMap_UnsupportedTarget(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("key", "value")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ParseMultipartForm(1024)

	target := &map[string]string{} // Not map[string]interface{}
	err := multipartToMap(req, target)

	if err == nil {
		t.Error("expected error for unsupported target type")
	}
}

func TestParser_ParseJSON_TypeMismatch(t *testing.T) {
	parser := NewParser()

	// Try to parse object into string slice
	body := `{"name":"John"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	target := &[]string{}
	err := parser.ParseJSON(w, req, target)

	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestParser_ParseForm_InvalidForm(t *testing.T) {
	parser := NewParser()

	// Create a request with an invalid body that can't be parsed as form
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("%invalid%"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	target := &map[string]interface{}{}
	err := parser.ParseForm(w, req, target)

	if err == nil {
		t.Error("expected error for invalid form data")
	}
}

// inferParameterType is tested indirectly through parameter extraction in router tests
