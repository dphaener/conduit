package codegen

import (
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
	g.writeLine("return nil, fmt.Errorf(\"failed to parse metadata: %%w\", err)")
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
	g.writeLine("return nil, fmt.Errorf(\"resource not found: %%s\", name)")
	g.indent--
	g.writeLine("}")

	return g.buf.String(), nil
}

// escapeBackticks escapes backticks in a string for use in Go raw string literals
func escapeBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "` + \"`\" + `")
}
