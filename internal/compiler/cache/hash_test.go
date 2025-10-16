package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileHasher_HashContent(t *testing.T) {
	hasher := NewFileHasher()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "empty content",
			content:  []byte(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple content",
			content:  []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "conduit resource",
			content:  []byte("resource User { name: string! }"),
			expected: "8f4c5c9c6e7d5e7b6a2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasher.HashContent(tt.content)
			if len(result) != 64 {
				t.Errorf("HashContent() returned hash of length %d, expected 64", len(result))
			}
			// SHA-256 should be deterministic
			result2 := hasher.HashContent(tt.content)
			if result != result2 {
				t.Errorf("HashContent() not deterministic: %s != %s", result, result2)
			}
		})
	}
}

func TestFileHasher_HashString(t *testing.T) {
	hasher := NewFileHasher()

	content := "resource Post { title: string! }"
	hash1 := hasher.HashString(content)
	hash2 := hasher.HashString(content)

	if hash1 != hash2 {
		t.Errorf("HashString() not deterministic: %s != %s", hash1, hash2)
	}

	if len(hash1) != 64 {
		t.Errorf("HashString() returned hash of length %d, expected 64", len(hash1))
	}

	// Different content should produce different hash
	differentContent := "resource Post { title: string? }"
	hash3 := hasher.HashString(differentContent)
	if hash1 == hash3 {
		t.Errorf("HashString() returned same hash for different content")
	}
}

func TestFileHasher_HashFile(t *testing.T) {
	hasher := NewFileHasher()

	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.cdt")

	content := `resource User {
  name: string!
  email: string!
}`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Hash the file
	hash1, err := hasher.HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile() error: %v", err)
	}

	if len(hash1) != 64 {
		t.Errorf("HashFile() returned hash of length %d, expected 64", len(hash1))
	}

	// Hash again - should be the same
	hash2, err := hasher.HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile() error: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("HashFile() not deterministic: %s != %s", hash1, hash2)
	}

	// Modify file
	modifiedContent := content + "\n  age: int!"
	if err := os.WriteFile(tmpFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify temp file: %v", err)
	}

	hash3, err := hasher.HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile() error: %v", err)
	}

	if hash1 == hash3 {
		t.Errorf("HashFile() returned same hash after modification")
	}
}

func TestFileHasher_HashFile_NotFound(t *testing.T) {
	hasher := NewFileHasher()

	_, err := hasher.HashFile("/nonexistent/file.cdt")
	if err == nil {
		t.Errorf("HashFile() should return error for non-existent file")
	}
}

func TestFileHasher_Consistency(t *testing.T) {
	hasher := NewFileHasher()

	content := "resource Post { title: string! }"

	// Hash via different methods should be consistent
	hashContent := hasher.HashContent([]byte(content))
	hashString := hasher.HashString(content)

	if hashContent != hashString {
		t.Errorf("HashContent() and HashString() produced different hashes for same content")
	}

	// Hash via file should also match
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.cdt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	hashFile, err := hasher.HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile() error: %v", err)
	}

	if hashContent != hashFile {
		t.Errorf("HashContent() and HashFile() produced different hashes for same content")
	}
}
