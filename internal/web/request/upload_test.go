package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFileUploader_GetFile(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir() // Use temp dir for tests
	uploader := NewFileUploader(config)

	// Create multipart form with file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("document", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("test file content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadedFile, err := uploader.GetFile(req, "document")
	if err != nil {
		t.Errorf("GetFile() error = %v", err)
		return
	}

	if uploadedFile.Filename != "test.txt" {
		t.Errorf("filename = %v, want test.txt", uploadedFile.Filename)
	}

	if uploadedFile.Size != 17 {
		t.Errorf("size = %v, want 17", uploadedFile.Size)
	}

	// Verify file was saved
	if uploadedFile.Path == "" {
		t.Error("file path is empty")
	}

	if _, err := os.Stat(uploadedFile.Path); os.IsNotExist(err) {
		t.Error("file was not saved to disk")
	}
}

func TestFileUploader_GetFile_Missing(t *testing.T) {
	config := DefaultUploadConfig()
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := uploader.GetFile(req, "document")
	if err == nil {
		t.Error("expected error for missing file")
	}

	if !strings.Contains(err.Error(), "no file uploaded") {
		t.Errorf("error should mention missing file, got: %v", err)
	}
}

func TestFileUploader_GetFiles_Multiple(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	uploader := NewFileUploader(config)

	// Create multipart form with multiple files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add first file
	part1, _ := writer.CreateFormFile("documents", "file1.txt")
	part1.Write([]byte("content 1"))

	// Add second file
	part2, _ := writer.CreateFormFile("documents", "file2.txt")
	part2.Write([]byte("content 2"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	files, err := uploader.GetFiles(req, "documents")
	if err != nil {
		t.Errorf("GetFiles() error = %v", err)
		return
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %v", len(files))
	}

	if files[0].Filename != "file1.txt" {
		t.Errorf("first file name = %v, want file1.txt", files[0].Filename)
	}

	if files[1].Filename != "file2.txt" {
		t.Errorf("second file name = %v, want file2.txt", files[1].Filename)
	}
}

func TestFileUploader_ValidateFile_Size(t *testing.T) {
	config := DefaultUploadConfig()
	config.MaxFileSize = 10 // 10 bytes max
	uploader := NewFileUploader(config)

	// Create file larger than max
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "large.txt")
	part.Write([]byte(strings.Repeat("a", 100))) // 100 bytes
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := uploader.GetFile(req, "file")
	if err == nil {
		t.Error("expected error for file exceeding max size")
	}

	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestFileUploader_ValidateFile_Extension(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.AllowedExts = []string{".txt", ".pdf"}
	uploader := NewFileUploader(config)

	tests := []struct {
		filename string
		wantErr  bool
	}{
		{"document.txt", false},
		{"document.pdf", false},
		{"document.exe", true},
		{"document.sh", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", tt.filename)
			part.Write([]byte("content"))
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			_, err := uploader.GetFile(req, "file")
			if (err != nil) != tt.wantErr {
				t.Errorf("extension validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileUploader_ValidateFile_ContentType(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.AllowedTypes = []string{"image/jpeg", "image/png", "image/"}

	tests := []struct {
		name        string
		fileContent []byte
		filename    string
		wantErr     bool
	}{
		{
			name:        "valid JPEG image",
			fileContent: []byte{0xFF, 0xD8, 0xFF, 0xE0}, // JPEG magic bytes
			filename:    "image.jpg",
			wantErr:     false,
		},
		{
			name:        "valid PNG image",
			fileContent: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG magic bytes
			filename:    "image.png",
			wantErr:     false,
		},
		{
			name:        "text file with .jpg extension (should fail)",
			fileContent: []byte("This is plain text, not an image"),
			filename:    "fake.jpg",
			wantErr:     true,
		},
		{
			name:        "PDF file (not allowed)",
			fileContent: []byte("%PDF-1.4"),
			filename:    "document.pdf",
			wantErr:     true,
		},
	}

	uploader := NewFileUploader(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", tt.filename)
			part.Write(tt.fileContent)
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			_, err := uploader.GetFile(req, "file")
			if (err != nil) != tt.wantErr {
				t.Errorf("content type validation error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "content type") {
				t.Errorf("error should mention content type, got: %v", err)
			}
		})
	}
}

func TestIsTypeAllowed(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		allowedTypes []string
		want         bool
	}{
		{
			name:         "exact match",
			contentType:  "image/jpeg",
			allowedTypes: []string{"image/jpeg", "image/png"},
			want:         true,
		},
		{
			name:         "prefix match",
			contentType:  "image/jpeg",
			allowedTypes: []string{"image/"},
			want:         true,
		},
		{
			name:         "not allowed",
			contentType:  "application/pdf",
			allowedTypes: []string{"image/jpeg", "image/png"},
			want:         false,
		},
		{
			name:         "empty allowed list",
			contentType:  "image/jpeg",
			allowedTypes: []string{},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTypeAllowed(tt.contentType, tt.allowedTypes)
			if got != tt.want {
				t.Errorf("isTypeAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileUploader_ValidateFile_EmptyFile(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "empty.txt")
	part.Write([]byte("")) // Empty file
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := uploader.GetFile(req, "file")
	if err == nil {
		t.Error("expected error for empty file")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty file, got: %v", err)
	}
}

func TestFileUploader_TotalSizeLimit(t *testing.T) {
	config := DefaultUploadConfig()
	config.UploadDir = t.TempDir()
	config.MaxFileSize = 100
	config.MaxTotalSize = 150 // Total limit smaller than 2 files
	uploader := NewFileUploader(config)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add two files that individually fit but together exceed total
	part1, _ := writer.CreateFormFile("files", "file1.txt")
	part1.Write([]byte(strings.Repeat("a", 90)))

	part2, _ := writer.CreateFormFile("files", "file2.txt")
	part2.Write([]byte(strings.Repeat("b", 90)))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := uploader.GetFiles(req, "files")
	if err == nil {
		t.Error("expected error for total size exceeding limit")
	}

	if !strings.Contains(err.Error(), "total file size") {
		t.Errorf("error should mention total size, got: %v", err)
	}
}

func TestGenerateUniqueFilename(t *testing.T) {
	filename1 := generateUniqueFilename("test.txt")

	// Should contain base name and extension
	if !strings.Contains(filename1, "test") {
		t.Error("generated filename should contain base name")
	}

	if !strings.HasSuffix(filename1, ".txt") {
		t.Error("generated filename should preserve extension")
	}

	// Should contain separator
	if !strings.Contains(filename1, "_") {
		t.Error("generated filename should contain separator")
	}
}

func TestGenerateUniqueFilename_Uniqueness(t *testing.T) {
	// Generate multiple filenames and ensure they're all unique
	filenames := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		filename := generateUniqueFilename("test.txt")
		if filenames[filename] {
			t.Errorf("duplicate filename generated: %s", filename)
		}
		filenames[filename] = true
	}

	if len(filenames) != count {
		t.Errorf("expected %d unique filenames, got %d", count, len(filenames))
	}
}

func TestGenerateUniqueFilename_Concurrent(t *testing.T) {
	// Test concurrent filename generation to ensure no collisions
	filenames := make(chan string, 100)
	done := make(chan bool)

	// Spawn 10 goroutines each generating 10 filenames
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				filenames <- generateUniqueFilename("concurrent.txt")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(filenames)

	// Check for duplicates
	seen := make(map[string]bool)
	count := 0
	for filename := range filenames {
		if seen[filename] {
			t.Errorf("duplicate filename in concurrent test: %s", filename)
		}
		seen[filename] = true
		count++
	}

	if count != 100 {
		t.Errorf("expected 100 filenames, got %d", count)
	}
}

func TestHasFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("document", "test.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if !HasFile(req, "document") {
		t.Error("HasFile() should return true for existing file field")
	}

	if HasFile(req, "missing") {
		t.Error("HasFile() should return false for missing file field")
	}
}

func TestGetFileFromRequest(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadedFile, err := GetFileFromRequest(req, "file", 1024)
	if err != nil {
		t.Errorf("GetFileFromRequest() error = %v", err)
		return
	}

	if uploadedFile.Filename != "test.txt" {
		t.Errorf("filename = %v, want test.txt", uploadedFile.Filename)
	}
}

// Benchmark tests
func BenchmarkFileUploader_GetFile(b *testing.B) {
	config := DefaultUploadConfig()
	config.UploadDir = b.TempDir()
	uploader := NewFileUploader(config)

	content := []byte(strings.Repeat("a", 1024)) // 1KB file

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write(content)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		uploader.GetFile(req, "file")
	}
}
