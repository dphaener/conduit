package metadata

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Serialize converts metadata to JSON format.
// The output is deterministic - same input will always produce the same output.
// This is important for caching and change detection.
func Serialize(metadata *Metadata) ([]byte, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	// Use MarshalIndent for readable JSON output (debugging-friendly)
	// The indentation is consistent, making the output deterministic
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return data, nil
}

// Compress compresses data using gzip compression.
// Target compression ratio: 60-70% size reduction for typical metadata.
// Uses best compression level for optimal size reduction.
func Compress(data []byte) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	if len(data) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer

	// Use gzip.BestCompression for maximum size reduction
	// This is acceptable since compression happens at build time
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Write the data
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close() // Ignore close error when write failed
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	// Close the writer to flush any remaining data
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses gzip-compressed data.
// Target decompression time: <10ms for typical metadata.
func Decompress(data []byte) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	if len(data) == 0 {
		return []byte{}, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		_ = reader.Close() // Ignore close error - we already have the data
	}()

	// Read all decompressed data
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return decompressed, nil
}

// WriteToFile writes uncompressed JSON metadata to a file for debugging.
// This writes to build/app.meta.json by default.
func WriteToFile(metadata *Metadata, outputPath string) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// Serialize metadata to JSON
	data, err := Serialize(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write metadata to %s: %w", outputPath, err)
	}

	return nil
}

// WriteCompressedToFile writes compressed metadata to a file.
// This can be used for distributing compressed metadata.
func WriteCompressedToFile(metadata *Metadata, outputPath string) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// Serialize and compress
	data, err := Serialize(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	compressed, err := Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress metadata: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write compressed data to file
	if err := os.WriteFile(outputPath, compressed, 0o644); err != nil {
		return fmt.Errorf("failed to write compressed metadata to %s: %w", outputPath, err)
	}

	return nil
}
