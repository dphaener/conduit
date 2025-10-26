package metadata

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSerialize_RoundTrip tests that serialization is reversible
func TestSerialize_RoundTrip(t *testing.T) {
	// Create sample metadata
	original := &Metadata{
		Version:    "1.0.0",
		SourceHash: "abc123",
		Resources: []ResourceMetadata{
			{
				Name:          "User",
				Documentation: "User resource",
				FilePath:      "/path/to/user.cdt",
				Line:          1,
				Fields: []FieldMetadata{
					{
						Name:        "id",
						Type:        "uuid!",
						Nullable:    false,
						Constraints: []string{"@primary", "@auto"},
					},
					{
						Name:        "email",
						Type:        "string!",
						Nullable:    false,
						Constraints: []string{"@unique"},
					},
					{
						Name:     "bio",
						Type:     "text?",
						Nullable: true,
					},
				},
				Hooks: []HookMetadata{
					{
						Timing:         "before",
						Event:          "create",
						HasTransaction: false,
						HasAsync:       false,
						SourceCode:     "self.email = String.lowercase(self.email)",
						Line:           15,
					},
				},
			},
		},
		Patterns: []PatternMetadata{
			{
				Name:        "unique_field",
				Template:    "field: type! @unique",
				Description: "Fields with unique constraints",
				Occurrences: 5,
			},
		},
		Routes: []RouteMetadata{
			{
				Method:      "GET",
				Path:        "/users",
				Handler:     "Index",
				Resource:    "User",
				Description: "List all users",
			},
		},
	}

	// Serialize
	data, err := Serialize(original)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Verify we got valid JSON
	if !json.Valid(data) {
		t.Fatal("Serialized data is not valid JSON")
	}

	// Deserialize
	var restored Metadata
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify key fields
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", restored.Version, original.Version)
	}

	if restored.SourceHash != original.SourceHash {
		t.Errorf("SourceHash mismatch: got %s, want %s", restored.SourceHash, original.SourceHash)
	}

	if len(restored.Resources) != len(original.Resources) {
		t.Errorf("Resources length mismatch: got %d, want %d", len(restored.Resources), len(original.Resources))
	}

	if len(restored.Resources) > 0 {
		if restored.Resources[0].Name != original.Resources[0].Name {
			t.Errorf("Resource name mismatch: got %s, want %s",
				restored.Resources[0].Name, original.Resources[0].Name)
		}

		if len(restored.Resources[0].Fields) != len(original.Resources[0].Fields) {
			t.Errorf("Fields length mismatch: got %d, want %d",
				len(restored.Resources[0].Fields), len(original.Resources[0].Fields))
		}
	}
}

// TestSerialize_Deterministic tests that serialization produces consistent output
func TestSerialize_Deterministic(t *testing.T) {
	metadata := &Metadata{
		Version:    "1.0.0",
		SourceHash: "test123",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
					{Name: "title", Type: "string!", Nullable: false},
				},
			},
		},
	}

	// Serialize multiple times
	data1, err := Serialize(metadata)
	if err != nil {
		t.Fatalf("First serialization failed: %v", err)
	}

	data2, err := Serialize(metadata)
	if err != nil {
		t.Fatalf("Second serialization failed: %v", err)
	}

	// Results should be identical
	if !bytes.Equal(data1, data2) {
		t.Error("Serialization is not deterministic")
	}
}

// TestSerialize_NilMetadata tests error handling for nil metadata
func TestSerialize_NilMetadata(t *testing.T) {
	_, err := Serialize(nil)
	if err == nil {
		t.Error("Expected error for nil metadata, got nil")
	}

	if !strings.Contains(err.Error(), "metadata cannot be nil") {
		t.Errorf("Expected 'metadata cannot be nil' error, got: %v", err)
	}
}

// TestSerialize_EmptyMetadata tests serialization of empty metadata
func TestSerialize_EmptyMetadata(t *testing.T) {
	metadata := &Metadata{}

	data, err := Serialize(metadata)
	if err != nil {
		t.Fatalf("Failed to serialize empty metadata: %v", err)
	}

	// Should still produce valid JSON
	if !json.Valid(data) {
		t.Error("Serialized empty metadata is not valid JSON")
	}
}

// TestCompress_SmallData tests compression of small data
func TestCompress_SmallData(t *testing.T) {
	original := []byte("Hello, World!")

	compressed, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Verify we got some output
	if len(compressed) == 0 {
		t.Error("Compressed data is empty")
	}

	// Decompress and verify
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("Decompressed data doesn't match original:\ngot:  %s\nwant: %s",
			string(decompressed), string(original))
	}
}

// TestCompress_LargeData tests compression of large data
func TestCompress_LargeData(t *testing.T) {
	// Create large metadata with many resources
	metadata := &Metadata{
		Version:    "1.0.0",
		SourceHash: "hash123",
		Resources:  make([]ResourceMetadata, 50),
	}

	// Populate with realistic data
	for i := 0; i < 50; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name:          "Resource" + string(rune('A'+i)),
			Documentation: "This is a sample resource with documentation",
			FilePath:      "/path/to/resource.cdt",
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!", Nullable: false, Constraints: []string{"@primary", "@auto"}},
				{Name: "name", Type: "string!", Nullable: false, Constraints: []string{"@min(1)", "@max(200)"}},
				{Name: "description", Type: "text?", Nullable: true},
				{Name: "created_at", Type: "datetime!", Nullable: false, Constraints: []string{"@auto"}},
			},
			Hooks: []HookMetadata{
				{
					Timing:         "before",
					Event:          "create",
					HasTransaction: false,
					SourceCode:     "self.name = String.trim(self.name)",
					Line:           20,
				},
			},
		}
	}

	// Serialize
	original, err := Serialize(metadata)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Compress
	compressed, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Calculate compression ratio
	ratio := float64(len(compressed)) / float64(len(original))
	compressionPercent := (1 - ratio) * 100

	t.Logf("Original size: %d bytes", len(original))
	t.Logf("Compressed size: %d bytes", len(compressed))
	t.Logf("Compression ratio: %.2f%%", compressionPercent)

	// Verify compression achieves target of 60-70% reduction
	if compressionPercent < 60 {
		t.Errorf("Compression ratio too low: %.2f%% (target: 60-70%%)", compressionPercent)
	}

	// Decompress and verify
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Error("Decompressed data doesn't match original")
	}
}

// TestCompress_EmptyData tests compression of empty data
func TestCompress_EmptyData(t *testing.T) {
	compressed, err := Compress([]byte{})
	if err != nil {
		t.Fatalf("Compress failed for empty data: %v", err)
	}

	if len(compressed) != 0 {
		t.Error("Expected empty compressed data for empty input")
	}

	// Decompress should also return empty
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed for empty data: %v", err)
	}

	if len(decompressed) != 0 {
		t.Error("Expected empty decompressed data for empty input")
	}
}

// TestCompress_NilData tests error handling for nil data
func TestCompress_NilData(t *testing.T) {
	_, err := Compress(nil)
	if err == nil {
		t.Error("Expected error for nil data, got nil")
	}

	if !strings.Contains(err.Error(), "data cannot be nil") {
		t.Errorf("Expected 'data cannot be nil' error, got: %v", err)
	}
}

// TestDecompress_NilData tests error handling for nil data
func TestDecompress_NilData(t *testing.T) {
	_, err := Decompress(nil)
	if err == nil {
		t.Error("Expected error for nil data, got nil")
	}

	if !strings.Contains(err.Error(), "data cannot be nil") {
		t.Errorf("Expected 'data cannot be nil' error, got: %v", err)
	}
}

// TestDecompress_CorruptedData tests error handling for corrupted data
func TestDecompress_CorruptedData(t *testing.T) {
	// Create invalid gzip data
	corrupted := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}

	_, err := Decompress(corrupted)
	if err == nil {
		t.Error("Expected error for corrupted data, got nil")
	}
}

// TestDecompress_MalformedJSON tests handling of decompressed but invalid JSON
func TestDecompress_MalformedJSON(t *testing.T) {
	// Create valid gzip of invalid JSON
	malformedJSON := []byte(`{"invalid": "json"`)

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	writer.Write(malformedJSON)
	writer.Close()

	// Decompression should succeed
	decompressed, err := Decompress(buf.Bytes())
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	// But JSON unmarshaling should fail
	var metadata Metadata
	err = json.Unmarshal(decompressed, &metadata)
	if err == nil {
		t.Error("Expected JSON unmarshal error for malformed JSON")
	}
}

// TestWriteToFile tests writing metadata to file
func TestWriteToFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "build", "app.meta.json")

	metadata := &Metadata{
		Version:    "1.0.0",
		SourceHash: "test123",
		Resources: []ResourceMetadata{
			{
				Name: "User",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
				},
			},
		},
	}

	// Write to file
	err := WriteToFile(metadata, outputPath)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file does not exist: %s", outputPath)
	}

	// Read and verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify it's valid JSON
	if !json.Valid(data) {
		t.Error("Output file does not contain valid JSON")
	}

	// Verify we can deserialize it
	var restored Metadata
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal output file: %v", err)
	}

	if restored.Version != metadata.Version {
		t.Errorf("Version mismatch: got %s, want %s", restored.Version, metadata.Version)
	}
}

// TestWriteToFile_NilMetadata tests error handling for nil metadata
func TestWriteToFile_NilMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app.meta.json")

	err := WriteToFile(nil, outputPath)
	if err == nil {
		t.Error("Expected error for nil metadata, got nil")
	}
}

// TestWriteToFile_EmptyPath tests error handling for empty path
func TestWriteToFile_EmptyPath(t *testing.T) {
	metadata := &Metadata{Version: "1.0.0"}

	err := WriteToFile(metadata, "")
	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

// TestWriteCompressedToFile tests writing compressed metadata to file
func TestWriteCompressedToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "build", "app.meta.json.gz")

	metadata := &Metadata{
		Version:    "1.0.0",
		SourceHash: "test123",
		Resources: []ResourceMetadata{
			{
				Name: "User",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
					{Name: "email", Type: "string!", Nullable: false},
				},
			},
		},
	}

	// Write compressed file
	err := WriteCompressedToFile(metadata, outputPath)
	if err != nil {
		t.Fatalf("WriteCompressedToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file does not exist: %s", outputPath)
	}

	// Read compressed file
	compressedData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Decompress
	decompressed, err := Decompress(compressedData)
	if err != nil {
		t.Fatalf("Failed to decompress file: %v", err)
	}

	// Verify it's valid JSON
	if !json.Valid(decompressed) {
		t.Error("Decompressed data is not valid JSON")
	}

	// Verify we can deserialize it
	var restored Metadata
	if err := json.Unmarshal(decompressed, &restored); err != nil {
		t.Fatalf("Failed to unmarshal decompressed data: %v", err)
	}

	if restored.Version != metadata.Version {
		t.Errorf("Version mismatch: got %s, want %s", restored.Version, metadata.Version)
	}
}

// TestFullPipeline tests the complete serialize -> compress -> decompress -> deserialize pipeline
func TestFullPipeline(t *testing.T) {
	// Create realistic metadata
	original := &Metadata{
		Version:    "1.0.0",
		SourceHash: "abc123def456",
		Resources: []ResourceMetadata{
			{
				Name:          "Post",
				Documentation: "Blog post resource",
				FilePath:      "/app/post.cdt",
				Line:          1,
				Fields: []FieldMetadata{
					{
						Name:        "id",
						Type:        "uuid!",
						Nullable:    false,
						Constraints: []string{"@primary", "@auto"},
					},
					{
						Name:        "title",
						Type:        "string!",
						Nullable:    false,
						Constraints: []string{"@min(5)", "@max(200)"},
					},
					{
						Name:     "content",
						Type:     "text!",
						Nullable: false,
					},
				},
				Relationships: []RelationshipMetadata{
					{
						Name:       "author",
						Type:       "User",
						Kind:       "belongs_to",
						ForeignKey: "author_id",
						OnDelete:   "restrict",
						Nullable:   false,
					},
				},
				Hooks: []HookMetadata{
					{
						Timing:         "before",
						Event:          "create",
						HasTransaction: false,
						SourceCode:     "self.slug = String.slugify(self.title)",
						Line:           25,
					},
				},
			},
		},
		Patterns: []PatternMetadata{
			{
				Name:        "slugify_pattern",
				Template:    "String.slugify(field)",
				Description: "Generate URL-friendly slugs",
				Occurrences: 3,
			},
		},
		Routes: []RouteMetadata{
			{
				Method:      "GET",
				Path:        "/posts",
				Handler:     "Index",
				Resource:    "Post",
				Description: "List all posts",
			},
			{
				Method:      "POST",
				Path:        "/posts",
				Handler:     "Create",
				Resource:    "Post",
				Description: "Create a new post",
			},
		},
	}

	// Step 1: Serialize
	serialized, err := Serialize(original)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Step 2: Compress
	compressed, err := Compress(serialized)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Step 3: Decompress
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	// Step 4: Deserialize
	var restored Metadata
	if err := json.Unmarshal(decompressed, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all major fields
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", restored.Version, original.Version)
	}

	if restored.SourceHash != original.SourceHash {
		t.Errorf("SourceHash mismatch: got %s, want %s", restored.SourceHash, original.SourceHash)
	}

	if len(restored.Resources) != len(original.Resources) {
		t.Errorf("Resources count mismatch: got %d, want %d", len(restored.Resources), len(original.Resources))
	}

	if len(restored.Patterns) != len(original.Patterns) {
		t.Errorf("Patterns count mismatch: got %d, want %d", len(restored.Patterns), len(original.Patterns))
	}

	if len(restored.Routes) != len(original.Routes) {
		t.Errorf("Routes count mismatch: got %d, want %d", len(restored.Routes), len(original.Routes))
	}

	// Log compression stats
	ratio := float64(len(compressed)) / float64(len(serialized))
	compressionPercent := (1 - ratio) * 100
	t.Logf("Serialized size: %d bytes", len(serialized))
	t.Logf("Compressed size: %d bytes", len(compressed))
	t.Logf("Compression: %.2f%%", compressionPercent)
}

// Benchmark compression with various data sizes
func BenchmarkCompress_Small(b *testing.B) {
	metadata := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User", Fields: []FieldMetadata{{Name: "id", Type: "uuid!"}}},
		},
	}
	data, _ := Serialize(metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compress(data)
	}
}

func BenchmarkCompress_Medium(b *testing.B) {
	metadata := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 10),
	}
	for i := 0; i < 10; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!", Constraints: []string{"@primary"}},
				{Name: "name", Type: "string!", Constraints: []string{"@min(1)", "@max(200)"}},
			},
		}
	}
	data, _ := Serialize(metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compress(data)
	}
}

func BenchmarkCompress_Large(b *testing.B) {
	metadata := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}
	for i := 0; i < 50; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!", Constraints: []string{"@primary", "@auto"}},
				{Name: "name", Type: "string!", Constraints: []string{"@min(1)", "@max(200)"}},
				{Name: "description", Type: "text?"},
				{Name: "created_at", Type: "datetime!", Constraints: []string{"@auto"}},
			},
			Hooks: []HookMetadata{
				{Timing: "before", Event: "create", SourceCode: "self.name = String.trim(self.name)"},
			},
		}
	}
	data, _ := Serialize(metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compress(data)
	}
}

// BenchmarkDecompress tests decompression speed
func BenchmarkDecompress(b *testing.B) {
	// Create typical metadata
	metadata := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}
	for i := 0; i < 50; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!"},
				{Name: "name", Type: "string!"},
			},
		}
	}
	data, _ := Serialize(metadata)
	compressed, _ := Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decompress(compressed)
	}
}

// BenchmarkFullPipeline benchmarks the complete serialize+compress cycle
func BenchmarkFullPipeline(b *testing.B) {
	metadata := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}
	for i := 0; i < 50; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!", Constraints: []string{"@primary"}},
				{Name: "name", Type: "string!"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := Serialize(metadata)
		Compress(data)
	}
}

// TestDecompress_Performance verifies decompression meets <10ms requirement
func TestDecompress_Performance(t *testing.T) {
	// Create typical metadata
	metadata := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}
	for i := 0; i < 50; i++ {
		metadata.Resources[i] = ResourceMetadata{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!"},
				{Name: "name", Type: "string!"},
				{Name: "description", Type: "text?"},
			},
		}
	}

	data, _ := Serialize(metadata)
	compressed, _ := Compress(data)

	// Measure decompression time
	start := time.Now()
	_, err := Decompress(compressed)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	t.Logf("Decompression time: %v", duration)

	// Verify it's under 10ms
	if duration > 10*time.Millisecond {
		t.Errorf("Decompression too slow: %v (target: <10ms)", duration)
	}
}
