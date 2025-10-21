package request

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge case tests to push coverage over 90%

func TestFormToMap_StructTarget(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	values := map[string][]string{
		"name":  {"John"},
		"age":   {"30"},
		"email": {"john@example.com"},
	}

	var target TestStruct
	err := formToMap(values, &target)

	if err != nil {
		t.Errorf("formToMap() error = %v", err)
	}

	if target.Name != "John" {
		t.Errorf("Name = %v, want John", target.Name)
	}

	if target.Age != 30 {
		t.Errorf("Age = %v, want 30", target.Age)
	}
}

func TestParser_ParseJSON_LargeBody(t *testing.T) {
	parser := NewParserWithMaxSize(1024 * 1024) // 1MB

	// Create a large JSON array
	arr := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		arr[i] = map[string]interface{}{
			"id":   i,
			"name": strings.Repeat("test", 10),
			"data": strings.Repeat("x", 100),
		}
	}

	bodyBytes, _ := json.Marshal(arr)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	var result []map[string]interface{}
	err := parser.ParseJSON(w, req, &result)

	if err != nil {
		t.Errorf("ParseJSON() error = %v", err)
	}

	if len(result) != 100 {
		t.Errorf("expected 100 items, got %v", len(result))
	}
}

func TestMultipartToMap_WithMultipleValues(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add multiple values for same field
	writer.WriteField("tags", "go")
	writer.WriteField("tags", "web")
	writer.WriteField("tags", "api")
	writer.WriteField("name", "Test")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ParseMultipartForm(1024)

	target := &map[string]interface{}{}
	err := multipartToMap(req, target)

	if err != nil {
		t.Errorf("multipartToMap() error = %v", err)
	}

	m := *target
	if tags, ok := m["tags"].([]string); !ok || len(tags) != 3 {
		t.Errorf("tags should be array of 3 strings, got %v", m["tags"])
	}
}

func TestFileUploader_SaveWithoutUploadDir(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = "" // No upload directory
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadedFile, err := uploader.GetFile(req, "file")

	if err != nil {
		t.Errorf("GetFile() error = %v", err)
	}

	// File should not be saved when UploadDir is empty
	if uploadedFile.Path != "" {
		t.Error("file path should be empty when UploadDir not configured")
	}
}
