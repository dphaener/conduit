package codegen

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/metadata"
)

// GenerateMetadata generates introspection metadata JSON for a program
func (g *Generator) GenerateMetadata(prog *ast.Program) (string, error) {
	// Use version 1.0.0 as default
	extractor := metadata.NewExtractor("1.0.0")

	meta, err := extractor.Extract(prog)
	if err != nil {
		return "", fmt.Errorf("metadata extraction failed: %w", err)
	}

	jsonStr, err := meta.ToJSON()
	if err != nil {
		return "", fmt.Errorf("metadata JSON generation failed: %w", err)
	}

	return jsonStr, nil
}

// GenerateMetadataAccessor generates Go code that embeds and exposes metadata
func (g *Generator) GenerateMetadataAccessor(metadataJSON string) (string, error) {
	g.reset()

	// Package declaration
	g.writeLine("// Package introspection provides runtime access to application metadata")
	g.writeLine("package introspection")
	g.writeLine("")

	// Imports
	g.writeLine("import (")
	g.indent++
	g.writeLine("%q", "encoding/json")
	g.writeLine("%q", "fmt")
	g.indent--
	g.writeLine(")")
	g.writeLine("")

	// Embedded metadata constant
	g.writeLine("// Metadata contains the complete introspection metadata as JSON")
	g.writeLine("const Metadata = `%s`", escapeBackticks(metadataJSON))
	g.writeLine("")

	// GetMetadata function
	g.writeLine("// GetMetadata parses and returns the introspection metadata")
	g.writeLine("func GetMetadata() (map[string]interface{}, error) {")
	g.indent++
	g.writeLine("var meta map[string]interface{}")
	g.writeLine("if err := json.Unmarshal([]byte(Metadata), &meta); err != nil {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"failed to parse metadata: %w\", err)")
	g.indent--
	g.writeLine("}")
	g.writeLine("return meta, nil")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// QueryResources function
	g.writeLine("// QueryResources returns information about all resources")
	g.writeLine("func QueryResources() ([]interface{}, error) {")
	g.indent++
	g.writeLine("meta, err := GetMetadata()")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("resources, ok := meta[\"resources\"].([]interface{})")
	g.writeLine("if !ok {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"invalid metadata: resources not found\")")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return resources, nil")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// QueryPatterns function
	g.writeLine("// QueryPatterns returns information about common patterns")
	g.writeLine("func QueryPatterns() ([]interface{}, error) {")
	g.indent++
	g.writeLine("meta, err := GetMetadata()")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("patterns, ok := meta[\"patterns\"].([]interface{})")
	g.writeLine("if !ok {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"invalid metadata: patterns not found\")")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return patterns, nil")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// QueryRoutes function
	g.writeLine("// QueryRoutes returns information about all API routes")
	g.writeLine("func QueryRoutes() ([]interface{}, error) {")
	g.indent++
	g.writeLine("meta, err := GetMetadata()")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("routes, ok := meta[\"routes\"].([]interface{})")
	g.writeLine("if !ok {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"invalid metadata: routes not found\")")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return routes, nil")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// FindResource function
	g.writeLine("// FindResource finds a resource by name")
	g.writeLine("func FindResource(name string) (map[string]interface{}, error) {")
	g.indent++
	g.writeLine("resources, err := QueryResources()")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("for _, r := range resources {")
	g.indent++
	g.writeLine("resource, ok := r.(map[string]interface{})")
	g.writeLine("if !ok {")
	g.indent++
	g.writeLine("continue")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("if resName, ok := resource[\"name\"].(string); ok && resName == name {")
	g.indent++
	g.writeLine("return resource, nil")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return nil, fmt.Errorf(\"resource not found: %s\", name)")
	g.indent--
	g.writeLine("}")

	return g.buf.String(), nil
}

// escapeBackticks escapes backticks in a string for use in Go raw string literals
func escapeBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "` + \"`\" + `")
}

// EmbedMetadata generates Go code that embeds compressed metadata as a byte array.
// The generated code includes an init() function that decompresses and registers
// the metadata at application startup.
func (g *Generator) EmbedMetadata(metadataJSON string) (string, error) {
	// Compress the metadata
	compressed, err := compressMetadata([]byte(metadataJSON))
	if err != nil {
		return "", fmt.Errorf("failed to compress metadata: %w", err)
	}

	// Convert to Go byte array literal
	byteArray := bytesToGoArray(compressed)

	g.reset()

	// Package declaration
	g.writeLine("// Package metadata provides embedded introspection metadata")
	g.writeLine("package metadata")
	g.writeLine("")

	// Imports
	g.writeLine("import (")
	g.indent++
	g.writeLine("%q", "bytes")
	g.writeLine("%q", "compress/gzip")
	g.writeLine("%q", "fmt")
	g.writeLine("%q", "io")
	g.writeLine("")
	g.writeLine("%q", "github.com/conduit-lang/conduit/runtime/metadata")
	g.indent--
	g.writeLine(")")
	g.writeLine("")

	// Embedded metadata as byte array
	g.writeLine("// embeddedMetadata contains compressed introspection metadata")
	g.writeLine("// Generated at compile time - DO NOT EDIT")
	g.writeLine("var embeddedMetadata = []byte{")
	g.indent++
	g.writeLine("%s,", byteArray)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// init function
	g.writeLine("// init decompresses and registers metadata at application startup")
	g.writeLine("func init() {")
	g.indent++
	g.writeLine("// Decompress metadata")
	g.writeLine("decompressed, err := decompressMetadata(embeddedMetadata)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("panic(fmt.Sprintf(\"Failed to decompress embedded metadata: %v\", err))")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// Register with runtime")
	g.writeLine("if err := metadata.RegisterMetadata(decompressed); err != nil {")
	g.indent++
	g.writeLine("panic(fmt.Sprintf(\"Failed to register metadata: %v\", err))")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// decompressMetadata helper function
	g.writeLine("// decompressMetadata decompresses gzip-compressed metadata")
	g.writeLine("func decompressMetadata(data []byte) ([]byte, error) {")
	g.indent++
	g.writeLine("reader, err := gzip.NewReader(bytes.NewReader(data))")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("defer reader.Close()")
	g.writeLine("")
	g.writeLine("// Protect against decompression bombs with 10MB limit")
	g.writeLine("const maxMetadataSize = 10 * 1024 * 1024 // 10MB max")
	g.writeLine("decompressed, err := io.ReadAll(io.LimitReader(reader, maxMetadataSize))")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, err")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("return decompressed, nil")
	g.indent--
	g.writeLine("}")

	return g.buf.String(), nil
}

// compressMetadata compresses data using gzip with best compression
func compressMetadata(data []byte) ([]byte, error) {
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

// bytesToGoArray converts a byte slice to a Go byte array literal
func bytesToGoArray(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var buf strings.Builder

	// Write bytes in groups of 16 per line for readability
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			buf.WriteString(",\n\t")
		} else if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("0x%02x", b))
	}

	return buf.String()
}
