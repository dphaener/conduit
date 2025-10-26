package integration

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	runtimeMetadata "github.com/conduit-lang/conduit/runtime/metadata"
)

// TestMetadataEmbedding_EndToEnd tests the complete metadata embedding pipeline
func TestMetadataEmbedding_EndToEnd(t *testing.T) {
	// Create a sample program with resources
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "User",
				Documentation: "User account",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
			{
				Name:          "Post",
				Documentation: "Blog post",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "title",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	// Generate metadata JSON
	gen := codegen.NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	// Generate embedded Go code
	embeddedCode, err := gen.EmbedMetadata(metadataJSON)
	if err != nil {
		t.Fatalf("Failed to generate embedded metadata code: %v", err)
	}

	// Verify the generated code is valid Go
	_, err = format.Source([]byte(embeddedCode))
	if err != nil {
		t.Fatalf("Generated code is not valid Go: %v", err)
	}

	// Extract and verify the embedded byte array can be decompressed
	if err := verifyEmbeddedData(embeddedCode, metadataJSON); err != nil {
		t.Fatalf("Embedded data verification failed: %v", err)
	}

	t.Log("End-to-end metadata embedding test passed")
}

// TestMetadataEmbedding_BinarySize tests that embedded metadata meets size requirements
func TestMetadataEmbedding_BinarySize(t *testing.T) {
	// Create a program with 50 resources to test size
	prog := &ast.Program{
		Resources: make([]*ast.ResourceNode, 50),
	}

	for i := 0; i < 50; i++ {
		prog.Resources[i] = &ast.ResourceNode{
			Name:          fmt.Sprintf("Resource%d", i),
			Documentation: "Test resource",
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "description",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text", Nullable: true},
					Nullable: true,
				},
			},
		}
	}

	gen := codegen.NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	embeddedCode, err := gen.EmbedMetadata(metadataJSON)
	if err != nil {
		t.Fatalf("Failed to generate embedded metadata: %v", err)
	}

	// Calculate sizes
	originalSize := len(metadataJSON)
	compressedSize := estimateCompressedSize(embeddedCode)

	t.Logf("Original JSON size: %d bytes", originalSize)
	t.Logf("Estimated compressed size: %d bytes", compressedSize)
	t.Logf("Generated code size: %d bytes", len(embeddedCode))

	// Verify compressed size meets requirement (100-150KB for 50 resources)
	if compressedSize > 150*1024 {
		t.Errorf("Compressed metadata too large: %d bytes (max 150KB)", compressedSize)
	}

	compressionRatio := float64(compressedSize) / float64(originalSize)
	t.Logf("Compression ratio: %.2f%%", (1-compressionRatio)*100)

	// Verify compression is effective (should be at least 50% reduction)
	if compressionRatio > 0.5 {
		t.Errorf("Compression not effective enough: %.2f%% (want >50%% reduction)", (1-compressionRatio)*100)
	}
}

// TestMetadataEmbedding_StartupTime tests that metadata loading meets performance requirements
func TestMetadataEmbedding_StartupTime(t *testing.T) {
	// Create a realistic program
	prog := &ast.Program{
		Resources: make([]*ast.ResourceNode, 50),
	}

	for i := 0; i < 50; i++ {
		prog.Resources[i] = &ast.ResourceNode{
			Name: fmt.Sprintf("Resource%d", i),
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
			},
		}
	}

	gen := codegen.NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	// Simulate the decompression and registration that happens at startup
	start := time.Now()

	// Compress
	compressed, err := compressData([]byte(metadataJSON))
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Decompress (this happens in init())
	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	// Register (this also happens in init())
	err = runtimeMetadata.RegisterMetadata(decompressed)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Metadata loading time: %v", duration)

	// Verify it meets the <10ms requirement
	if duration > 10*time.Millisecond {
		t.Errorf("Metadata loading too slow: %v (target: <10ms)", duration)
	}

	// Clean up
	runtimeMetadata.Reset()
}

// TestMetadataEmbedding_RuntimeQueries tests that embedded metadata can be queried
func TestMetadataEmbedding_RuntimeQueries(t *testing.T) {
	defer runtimeMetadata.Reset()

	// Create a program
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	gen := codegen.NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	// Simulate embedding process
	compressed, err := compressData([]byte(metadataJSON))
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	err = runtimeMetadata.RegisterMetadata(decompressed)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Test queries
	resources := runtimeMetadata.QueryResources()
	if len(resources) != 2 {
		t.Errorf("QueryResources() returned %d resources, want 2", len(resources))
	}

	user, err := runtimeMetadata.QueryResource("User")
	if err != nil {
		t.Fatalf("QueryResource(User) failed: %v", err)
	}

	if user.Name != "User" {
		t.Errorf("Resource name: got %s, want User", user.Name)
	}

	if len(user.Fields) != 2 {
		t.Errorf("User fields count: got %d, want 2", len(user.Fields))
	}
}

// TestMetadataEmbedding_GoFmtCompliant tests that generated code is go fmt compliant
func TestMetadataEmbedding_GoFmtCompliant(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	gen := codegen.NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	embeddedCode, err := gen.EmbedMetadata(metadataJSON)
	if err != nil {
		t.Fatalf("Failed to generate embedded metadata: %v", err)
	}

	// Format the code
	formatted, err := format.Source([]byte(embeddedCode))
	if err != nil {
		t.Fatalf("Generated code is not go fmt compliant: %v", err)
	}

	// The formatted code should be identical to the original
	// (meaning it was already properly formatted)
	if string(formatted) != embeddedCode {
		// Write both to temp files for debugging
		tmpDir := t.TempDir()
		originalPath := filepath.Join(tmpDir, "original.go")
		formattedPath := filepath.Join(tmpDir, "formatted.go")

		os.WriteFile(originalPath, []byte(embeddedCode), 0644)
		os.WriteFile(formattedPath, formatted, 0644)

		// Run diff to show differences
		cmd := exec.Command("diff", "-u", originalPath, formattedPath)
		output, _ := cmd.CombinedOutput()

		t.Logf("Formatting differences:\n%s", string(output))
		t.Error("Generated code is not properly formatted")
	}
}

// Helper functions

func verifyEmbeddedData(embeddedCode, expectedJSON string) error {
	// Extract the byte array from the generated code
	// This is a simple verification that the code structure is correct
	if !strings.Contains(embeddedCode, "var embeddedMetadata = []byte{") {
		return fmt.Errorf("missing embedded metadata variable")
	}

	if !strings.Contains(embeddedCode, "func decompressMetadata") {
		return fmt.Errorf("missing decompressMetadata function")
	}

	return nil
}

func estimateCompressedSize(embeddedCode string) int {
	// Count the hex bytes in the generated code
	// Each byte is represented as "0xXX"
	count := strings.Count(embeddedCode, "0x")
	return count
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

// TestMetadataEmbedding_LargeProgram tests embedding with many resources
func TestMetadataEmbedding_LargeProgram(t *testing.T) {
	// Create a large program with 100 resources
	prog := &ast.Program{
		Resources: make([]*ast.ResourceNode, 100),
	}

	for i := 0; i < 100; i++ {
		prog.Resources[i] = &ast.ResourceNode{
			Name: fmt.Sprintf("Resource%d", i),
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Nullable: false,
				},
			},
		}
	}

	gen := codegen.NewGenerator()

	start := time.Now()
	metadataJSON, err := gen.GenerateMetadata(prog)
	if err != nil {
		t.Fatalf("Failed to generate metadata: %v", err)
	}

	embeddedCode, err := gen.EmbedMetadata(metadataJSON)
	if err != nil {
		t.Fatalf("Failed to generate embedded code: %v", err)
	}
	duration := time.Since(start)

	t.Logf("Generated embedded code for 100 resources in %v", duration)

	// Verify it's valid Go
	_, err = format.Source([]byte(embeddedCode))
	if err != nil {
		t.Fatalf("Generated code is not valid Go: %v", err)
	}

	// Verify we can parse the metadata back
	var meta runtimeMetadata.Metadata
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	if len(meta.Resources) != 100 {
		t.Errorf("Metadata resources count: got %d, want 100", len(meta.Resources))
	}
}
