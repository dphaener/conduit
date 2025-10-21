package request

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Integration tests that cover real-world scenarios

func TestFullRequestLifecycle_JSON(t *testing.T) {
	parser := NewParser()

	// Simulate a full JSON request
	reqData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	bodyBytes, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := parser.Parse(w, req, &result)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result["name"] != "John Doe" {
		t.Errorf("name = %v, want 'John Doe'", result["name"])
	}
}

func TestFullRequestLifecycle_Form(t *testing.T) {
	parser := NewParser()

	formData := "name=Jane+Doe&email=jane%40example.com&subscribe=true"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result map[string]interface{}
	err := parser.Parse(w, req, &result)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result["email"] != "jane@example.com" {
		t.Errorf("email = %v, want 'jane@example.com'", result["email"])
	}
}

func TestFullRequestLifecycle_Multipart(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	uploader := NewFileUploader(config)

	// Create multipart request with both fields and files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	writer.WriteField("title", "My Document")
	writer.WriteField("description", "Test document")

	// Add file
	part, _ := writer.CreateFormFile("file", "document.txt")
	part.Write([]byte("File content here"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Get file
	uploadedFile, err := uploader.GetFile(req, "file")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}

	if uploadedFile.Filename != "document.txt" {
		t.Errorf("filename = %v, want 'document.txt'", uploadedFile.Filename)
	}

	// Also parse form fields
	parser := NewParser()
	w := httptest.NewRecorder()
	formData := &map[string]interface{}{}
	if err := parser.ParseMultipart(w, req, formData); err != nil {
		t.Fatalf("ParseMultipart() error = %v", err)
	}

	m := *formData
	if m["title"] != "My Document" {
		t.Errorf("title = %v, want 'My Document'", m["title"])
	}
}

func TestRequestHelpers_Integration(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/posts/123?page=2&limit=20&active=true", nil)
	req.SetPathValue("id", "123")
	req.Header.Set("Authorization", "Bearer token123")

	// Test all helper functions together
	id := GetParam(req, "id")
	if id != "123" {
		t.Errorf("id = %v, want '123'", id)
	}

	page := GetQueryParamInt(req, "page", 1)
	if page != 2 {
		t.Errorf("page = %v, want 2", page)
	}

	limit := GetQueryParamInt(req, "limit", 10)
	if limit != 20 {
		t.Errorf("limit = %v, want 20", limit)
	}

	active := GetQueryParamBool(req, "active", false)
	if !active {
		t.Error("active should be true")
	}

	auth := GetHeader(req, "Authorization")
	if auth != "Bearer token123" {
		t.Errorf("Authorization = %v, want 'Bearer token123'", auth)
	}
}
