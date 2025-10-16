// Package cache provides incremental compilation and build caching functionality.
// It implements file content hashing, AST caching, dependency tracking, and cache invalidation.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// FileHasher computes content hashes for cache keys
type FileHasher struct{}

// NewFileHasher creates a new file hasher
func NewFileHasher() *FileHasher {
	return &FileHasher{}
}

// HashFile computes a SHA-256 hash of the file contents
func (fh *FileHasher) HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashContent computes a SHA-256 hash of the given content
func (fh *FileHasher) HashContent(content []byte) string {
	hasher := sha256.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

// HashString computes a SHA-256 hash of the given string
func (fh *FileHasher) HashString(content string) string {
	return fh.HashContent([]byte(content))
}
