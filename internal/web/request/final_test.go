package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Final tests to push coverage over 90%

func TestFormToMap_JsonMarshalError(t *testing.T) {
	// Test the JSON marshal/unmarshal path for struct targets
	type ComplexStruct struct {
		Name    string
		Age     int
		Active  bool
		Price   float64
		Missing string
	}

	values := map[string][]string{
		"Name":   {"Test"},
		"Age":    {"25"},
		"Active": {"true"},
		"Price":  {"99.99"},
	}

	var target ComplexStruct
	err := formToMap(values, &target)

	if err != nil {
		t.Errorf("formToMap() error = %v", err)
	}

	if target.Name != "Test" {
		t.Errorf("Name = %v, want Test", target.Name)
	}

	if target.Age != 25 {
		t.Errorf("Age = %v, want 25", target.Age)
	}

	if !target.Active {
		t.Error("Active should be true")
	}
}

func TestFileUploader_GenerateFilenames(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.GenerateNames = true
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "original.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadedFile, err := uploader.GetFile(req, "file")

	if err != nil {
		t.Errorf("GetFile() error = %v", err)
	}

	// Generated filename should be different from original
	if !strings.Contains(uploadedFile.Path, "original") {
		t.Errorf("generated path should contain base name, got: %v", uploadedFile.Path)
	}
}

func TestFileUploader_NoGenerateFilenames(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.GenerateNames = false // Use original filename
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "keep-name.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadedFile, err := uploader.GetFile(req, "file")

	if err != nil {
		t.Errorf("GetFile() error = %v", err)
	}

	// Filename should be preserved
	if !strings.Contains(uploadedFile.Path, "keep-name.txt") {
		t.Errorf("path should contain original filename, got: %v", uploadedFile.Path)
	}
}

func TestFileUploader_ContentTypeValidation(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.AllowedTypes = []string{"image/"}
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create file with image mime type
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.jpg"`}
	h["Content-Type"] = []string{"image/jpeg"}

	part, _ := writer.CreatePart(h)
	// Write actual JPEG magic bytes so content detection works
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := uploader.GetFile(req, "file")

	if err != nil {
		t.Errorf("GetFile() should accept image/jpeg with prefix match, error: %v", err)
	}
}

func TestParser_MultipleFormValuesInStruct(t *testing.T) {
	type Person struct {
		Name  string   `json:"name"`
		Tags  []string `json:"tags"`
		Email string   `json:"email"`
	}

	values := map[string][]string{
		"name":  {"John"},
		"tags":  {"go", "web", "api"},
		"email": {"john@example.com"},
	}

	var target Person
	err := formToMap(values, &target)

	if err != nil {
		t.Errorf("formToMap() error = %v", err)
	}

	if target.Name != "John" {
		t.Errorf("Name = %v, want John", target.Name)
	}

	if len(target.Tags) != 3 {
		t.Errorf("expected 3 tags, got %v", len(target.Tags))
	}
}
