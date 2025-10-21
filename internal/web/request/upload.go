package request

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadConfig configures file upload handling
type UploadConfig struct {
	MaxFileSize   int64    // Maximum size per file (in bytes)
	MaxTotalSize  int64    // Maximum total size for all files
	AllowedTypes  []string // Allowed MIME types (empty = allow all)
	AllowedExts   []string // Allowed file extensions (empty = allow all)
	UploadDir     string   // Directory to store uploaded files
	GenerateNames bool     // Auto-generate unique filenames
}

// DefaultUploadConfig returns default upload configuration
func DefaultUploadConfig() *UploadConfig {
	return &UploadConfig{
		MaxFileSize:   10 << 20, // 10MB per file
		MaxTotalSize:  50 << 20, // 50MB total
		AllowedTypes:  nil,      // Allow all types
		AllowedExts:   nil,      // Allow all extensions
		UploadDir:     "/tmp/uploads",
		GenerateNames: true,
	}
}

// UploadedFile represents a file uploaded in a request
type UploadedFile struct {
	Filename    string // Original filename
	Size        int64  // File size in bytes
	ContentType string // MIME type
	Path        string // Path where file was saved (if saved)
	Header      multipart.FileHeader
}

// FileUploader handles file uploads
type FileUploader struct {
	config *UploadConfig
}

// NewFileUploader creates a new file uploader with config
func NewFileUploader(config *UploadConfig) *FileUploader {
	return &FileUploader{
		config: config,
	}
}

// GetFile retrieves a single uploaded file from the request
func (u *FileUploader) GetFile(r *http.Request, fieldName string) (*UploadedFile, error) {
	// Parse multipart form if not already parsed
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(u.config.MaxTotalSize); err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	}

	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, fmt.Errorf("no file uploaded for field %s", fieldName)
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	defer file.Close()

	// Validate file
	if err := u.validateFile(header); err != nil {
		return nil, err
	}

	// Create uploaded file
	uploadedFile := &UploadedFile{
		Filename:    header.Filename,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
		Header:      *header,
	}

	// Save file if upload directory is configured
	if u.config.UploadDir != "" {
		path, err := u.saveFile(file, header)
		if err != nil {
			return nil, fmt.Errorf("failed to save file: %w", err)
		}
		uploadedFile.Path = path
	}

	return uploadedFile, nil
}

// GetFiles retrieves multiple uploaded files from the request
func (u *FileUploader) GetFiles(r *http.Request, fieldName string) ([]*UploadedFile, error) {
	// Parse multipart form if not already parsed
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(u.config.MaxTotalSize); err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	}

	headers, ok := r.MultipartForm.File[fieldName]
	if !ok {
		return nil, fmt.Errorf("no files uploaded for field %s", fieldName)
	}

	var uploadedFiles []*UploadedFile
	var totalSize int64

	for _, header := range headers {
		// Check total size limit
		totalSize += header.Size
		if totalSize > u.config.MaxTotalSize {
			return nil, fmt.Errorf("total file size exceeds maximum of %d bytes", u.config.MaxTotalSize)
		}

		// Validate file
		if err := u.validateFile(header); err != nil {
			return nil, fmt.Errorf("file %s: %w", header.Filename, err)
		}

		// Open file
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", header.Filename, err)
		}
		defer file.Close()

		// Create uploaded file
		uploadedFile := &UploadedFile{
			Filename:    header.Filename,
			Size:        header.Size,
			ContentType: header.Header.Get("Content-Type"),
			Header:      *header,
		}

		// Save file if upload directory is configured
		if u.config.UploadDir != "" {
			path, err := u.saveFile(file, header)
			if err != nil {
				return nil, fmt.Errorf("failed to save file %s: %w", header.Filename, err)
			}
			uploadedFile.Path = path
		}

		uploadedFiles = append(uploadedFiles, uploadedFile)
	}

	return uploadedFiles, nil
}

// validateFile validates a file against configured constraints
func (u *FileUploader) validateFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > u.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum of %d bytes", header.Size, u.config.MaxFileSize)
	}

	if header.Size == 0 {
		return fmt.Errorf("file is empty")
	}

	// Check file extension if restrictions are configured
	if len(u.config.AllowedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := false
		for _, allowedExt := range u.config.AllowedExts {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s not allowed", ext)
		}
	}

	// Check MIME type if restrictions are configured
	if len(u.config.AllowedTypes) > 0 {
		// Open file to detect actual content type
		file, err := header.Open()
		if err != nil {
			return fmt.Errorf("failed to open file for validation: %w", err)
		}
		defer file.Close()

		// Read first 512 bytes for content type detection
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file for validation: %w", err)
		}

		// Detect actual content type from file content
		actualType := http.DetectContentType(buffer[:n])

		// Check if detected type is allowed
		if !isTypeAllowed(actualType, u.config.AllowedTypes) {
			return fmt.Errorf("file content type %s not allowed", actualType)
		}
	}

	return nil
}

// isTypeAllowed checks if a content type is in the allowed list
func isTypeAllowed(contentType string, allowedTypes []string) bool {
	for _, allowed := range allowedTypes {
		// Allow exact matches or prefix matches (e.g., "image/" matches "image/jpeg")
		if contentType == allowed || strings.HasPrefix(contentType, allowed) {
			return true
		}
	}
	return false
}

// saveFile saves an uploaded file to disk
func (u *FileUploader) saveFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Ensure upload directory exists
	if err := os.MkdirAll(u.config.UploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate filename
	filename := header.Filename
	if u.config.GenerateNames {
		filename = generateUniqueFilename(header.Filename)
	}

	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)

	// Create destination path
	destPath := filepath.Join(u.config.UploadDir, filename)

	// Create destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	// Copy file content
	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath) // Clean up on error
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return destPath, nil
}

// generateUniqueFilename generates a unique filename based on the original
func generateUniqueFilename(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(filepath.Base(original), ext)

	// Use timestamp with nanosecond precision + random bytes for true uniqueness
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%s_%d_%s%s", base, timestamp, randomHex, ext)
}

// HasFile checks if a file field exists in the request
func HasFile(r *http.Request, fieldName string) bool {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return false
		}
	}
	_, ok := r.MultipartForm.File[fieldName]
	return ok
}

// GetFileFromRequest is a convenience function to get a single file
func GetFileFromRequest(r *http.Request, fieldName string, maxSize int64) (*UploadedFile, error) {
	config := &UploadConfig{
		MaxFileSize:  maxSize,
		MaxTotalSize: maxSize,
	}
	uploader := NewFileUploader(config)
	return uploader.GetFile(r, fieldName)
}
