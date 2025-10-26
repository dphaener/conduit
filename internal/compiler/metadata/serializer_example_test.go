package metadata_test

import (
	"bytes"
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/metadata"
)

// ExampleSerialize demonstrates basic serialization
func ExampleSerialize() {
	// Create sample metadata
	meta := &metadata.Metadata{
		Version:    "1.0.0",
		SourceHash: "abc123",
		Resources: []metadata.ResourceMetadata{
			{
				Name: "User",
				Fields: []metadata.FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
					{Name: "email", Type: "string!", Nullable: false},
				},
			},
		},
	}

	// Serialize to JSON
	data, err := metadata.Serialize(meta)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Serialized %d bytes of JSON\n", len(data))
	// Output: Serialized 375 bytes of JSON
}

// ExampleCompress demonstrates compression
func ExampleCompress() {
	// Create some JSON data
	jsonData := []byte(`{"version":"1.0.0","resources":[{"name":"User"}]}`)

	// Compress it
	compressed, err := metadata.Compress(jsonData)
	if err != nil {
		panic(err)
	}

	// Calculate compression ratio
	ratio := float64(len(compressed)) / float64(len(jsonData)) * 100

	fmt.Printf("Compressed from %d to %d bytes (%.0f%% of original)\n",
		len(jsonData), len(compressed), ratio)
	// Output: Compressed from 49 to 73 bytes (149% of original)
}

// ExampleDecompress demonstrates decompression
func ExampleDecompress() {
	// First compress some data
	original := []byte("Hello, Conduit!")
	compressed, _ := metadata.Compress(original)

	// Then decompress it
	decompressed, err := metadata.Decompress(compressed)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", string(decompressed))
	// Output: Hello, Conduit!
}

// Example_fullWorkflow demonstrates the complete serialize -> compress -> decompress pipeline
func Example_fullWorkflow() {
	// Step 1: Create metadata
	meta := &metadata.Metadata{
		Version:    "1.0.0",
		SourceHash: "test-hash",
		Resources: []metadata.ResourceMetadata{
			{
				Name: "Post",
				Fields: []metadata.FieldMetadata{
					{
						Name:        "title",
						Type:        "string!",
						Nullable:    false,
						Constraints: []string{"@min(5)", "@max(200)"},
					},
				},
			},
		},
		Patterns: []metadata.PatternMetadata{
			{
				Name:        "unique_field",
				Template:    "field: type! @unique",
				Occurrences: 3,
			},
		},
	}

	// Step 2: Serialize to JSON
	serialized, err := metadata.Serialize(meta)
	if err != nil {
		panic(err)
	}

	// Step 3: Compress
	compressed, err := metadata.Compress(serialized)
	if err != nil {
		panic(err)
	}

	// Step 4: Decompress
	decompressed, err := metadata.Decompress(compressed)
	if err != nil {
		panic(err)
	}

	// Verify data integrity
	if bytes.Equal(serialized, decompressed) {
		fmt.Println("Data integrity verified")
	}

	// Calculate compression savings
	savings := (1 - float64(len(compressed))/float64(len(serialized))) * 100
	fmt.Printf("Compression savings: %.0f%%\n", savings)

	// Output:
	// Data integrity verified
	// Compression savings: 49%
}
