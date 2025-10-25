package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MarkdownGenerator generates Markdown documentation
type MarkdownGenerator struct {
	config *Config
}

// NewMarkdownGenerator creates a new Markdown generator
func NewMarkdownGenerator(config *Config) *MarkdownGenerator {
	return &MarkdownGenerator{
		config: config,
	}
}

// Generate generates Markdown documentation
func (g *MarkdownGenerator) Generate(doc *Documentation) error {
	outputDir := filepath.Join(g.config.OutputDir, "markdown")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate index
	if err := g.generateIndex(doc, outputDir); err != nil {
		return err
	}

	// Generate resource documentation
	for _, resource := range doc.Resources {
		if err := g.generateResourceDoc(resource, outputDir); err != nil {
			return err
		}
	}

	return nil
}

// generateIndex generates the index/README file
func (g *MarkdownGenerator) generateIndex(doc *Documentation, outputDir string) error {
	var buf strings.Builder

	// Header
	buf.WriteString(fmt.Sprintf("# %s API Documentation\n\n", doc.ProjectInfo.Name))

	if doc.ProjectInfo.Description != "" {
		buf.WriteString(fmt.Sprintf("%s\n\n", doc.ProjectInfo.Description))
	}

	buf.WriteString(fmt.Sprintf("**Version:** v%s\n\n", doc.ProjectInfo.Version))

	// Table of contents
	buf.WriteString("## Resources\n\n")
	for _, resource := range doc.Resources {
		buf.WriteString(fmt.Sprintf("- [%s](%s.md)\n", resource.Name, strings.ToLower(resource.Name)))
	}
	buf.WriteString("\n")

	// Quick start
	buf.WriteString("## Quick Start\n\n")
	buf.WriteString("This API documentation provides comprehensive information about all available resources and endpoints.\n\n")
	buf.WriteString("### Base URL\n\n")
	if g.config.BaseURL != "" {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", g.config.BaseURL))
	} else {
		buf.WriteString("```\nhttp://localhost:3000\n```\n\n")
	}

	// Write to file
	outputPath := filepath.Join(outputDir, "README.md")
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

// generateResourceDoc generates documentation for a single resource
func (g *MarkdownGenerator) generateResourceDoc(resource *ResourceDoc, outputDir string) error {
	var buf strings.Builder

	// Header
	buf.WriteString(fmt.Sprintf("# %s\n\n", resource.Name))

	if resource.Documentation != "" {
		buf.WriteString(fmt.Sprintf("> %s\n\n", resource.Documentation))
	}

	// Table of contents
	buf.WriteString("## Table of Contents\n\n")
	buf.WriteString("- [Fields](#fields)\n")
	if len(resource.Relationships) > 0 {
		buf.WriteString("- [Relationships](#relationships)\n")
	}
	buf.WriteString("- [Endpoints](#endpoints)\n")
	if len(resource.Hooks) > 0 {
		buf.WriteString("- [Hooks](#hooks)\n")
	}
	if len(resource.Validations) > 0 {
		buf.WriteString("- [Validations](#validations)\n")
	}
	if len(resource.Constraints) > 0 {
		buf.WriteString("- [Constraints](#constraints)\n")
	}
	buf.WriteString("\n")

	// Fields
	buf.WriteString("## Fields\n\n")
	if len(resource.Fields) == 0 {
		buf.WriteString("No fields defined.\n\n")
	} else {
		buf.WriteString("| Name | Type | Required | Constraints | Description |\n")
		buf.WriteString("|------|------|----------|-------------|-------------|\n")
		for _, field := range resource.Fields {
			required := "No"
			if field.Required {
				required = "Yes"
			}
			constraints := strings.Join(field.Constraints, ", ")
			if constraints == "" {
				constraints = "-"
			}
			description := field.Description
			if description == "" {
				description = "-"
			}
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |\n",
				field.Name, field.Type, required, constraints, description))
		}
		buf.WriteString("\n")

		// Add examples section
		buf.WriteString("### Example\n\n")
		buf.WriteString("```json\n")
		example := g.createResourceExample(resource)
		exampleJSON, _ := json.MarshalIndent(example, "", "  ")
		buf.WriteString(string(exampleJSON))
		buf.WriteString("\n```\n\n")
	}

	// Relationships
	if len(resource.Relationships) > 0 {
		buf.WriteString("## Relationships\n\n")
		for _, rel := range resource.Relationships {
			buf.WriteString(fmt.Sprintf("### %s\n\n", rel.Name))
			buf.WriteString(fmt.Sprintf("- **Type:** `%s`\n", rel.Type))
			buf.WriteString(fmt.Sprintf("- **Kind:** `%s`\n", rel.Kind))
			buf.WriteString(fmt.Sprintf("- **Foreign Key:** `%s`\n", rel.ForeignKey))
			if rel.Description != "" {
				buf.WriteString(fmt.Sprintf("- **Description:** %s\n", rel.Description))
			}
			buf.WriteString("\n")
		}
	}

	// Endpoints
	buf.WriteString("## Endpoints\n\n")
	if len(resource.Endpoints) == 0 {
		buf.WriteString("No endpoints defined.\n\n")
	} else {
		for _, endpoint := range resource.Endpoints {
			g.writeEndpoint(&buf, endpoint)
		}
	}

	// Hooks
	if len(resource.Hooks) > 0 {
		buf.WriteString("## Hooks\n\n")
		buf.WriteString("| Timing | Event | Async | Transaction |\n")
		buf.WriteString("|--------|-------|-------|-------------|\n")
		for _, hook := range resource.Hooks {
			async := "No"
			if hook.IsAsync {
				async = "Yes"
			}
			transaction := "No"
			if hook.IsTransaction {
				transaction = "Yes"
			}
			buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				hook.Timing, hook.Event, async, transaction))
		}
		buf.WriteString("\n")
	}

	// Validations
	if len(resource.Validations) > 0 {
		buf.WriteString("## Validations\n\n")
		for _, validation := range resource.Validations {
			buf.WriteString(fmt.Sprintf("### %s\n\n", validation.Name))
			if validation.Description != "" {
				buf.WriteString(fmt.Sprintf("%s\n\n", validation.Description))
			}
			buf.WriteString(fmt.Sprintf("**Error Message:** %s\n\n", validation.ErrorMessage))
		}
	}

	// Constraints
	if len(resource.Constraints) > 0 {
		buf.WriteString("## Constraints\n\n")
		for _, constraint := range resource.Constraints {
			buf.WriteString(fmt.Sprintf("### %s\n\n", constraint.Name))
			if constraint.Description != "" {
				buf.WriteString(fmt.Sprintf("%s\n\n", constraint.Description))
			}
			if len(constraint.Arguments) > 0 {
				buf.WriteString(fmt.Sprintf("**Arguments:** %s\n\n", strings.Join(constraint.Arguments, ", ")))
			}
			if len(constraint.On) > 0 {
				buf.WriteString(fmt.Sprintf("**Applies to:** %s\n\n", strings.Join(constraint.On, ", ")))
			}
		}
	}

	// Write to file
	outputPath := filepath.Join(outputDir, strings.ToLower(resource.Name)+".md")
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

// writeEndpoint writes a single endpoint to the buffer
func (g *MarkdownGenerator) writeEndpoint(buf *strings.Builder, endpoint *EndpointDoc) {
	buf.WriteString(fmt.Sprintf("### %s\n\n", endpoint.Summary))

	// Method and path
	buf.WriteString("```http\n")
	buf.WriteString(fmt.Sprintf("%s %s\n", endpoint.Method, endpoint.Path))
	buf.WriteString("```\n\n")

	if endpoint.Description != "" {
		buf.WriteString(fmt.Sprintf("%s\n\n", endpoint.Description))
	}

	// Parameters
	if len(endpoint.Parameters) > 0 {
		buf.WriteString("**Parameters:**\n\n")
		buf.WriteString("| Name | In | Type | Required | Description |\n")
		buf.WriteString("|------|-------|------|----------|-------------|\n")
		for _, param := range endpoint.Parameters {
			required := "No"
			if param.Required {
				required = "Yes"
			}
			buf.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %s | %s |\n",
				param.Name, param.In, param.Type, required, param.Description))
		}
		buf.WriteString("\n")
	}

	// Request body
	if endpoint.RequestBody != nil {
		buf.WriteString("**Request Body:**\n\n")
		buf.WriteString("```json\n")
		if endpoint.RequestBody.Example != nil {
			exampleJSON, _ := json.MarshalIndent(endpoint.RequestBody.Example, "", "  ")
			buf.WriteString(string(exampleJSON))
		}
		buf.WriteString("\n```\n\n")
	}

	// Responses
	buf.WriteString("**Responses:**\n\n")
	for statusCode, response := range endpoint.Responses {
		buf.WriteString(fmt.Sprintf("- **%d %s**\n", statusCode, response.Description))
		if response.Example != nil {
			buf.WriteString("\n```json\n")
			exampleJSON, _ := json.MarshalIndent(response.Example, "", "  ")
			buf.WriteString(string(exampleJSON))
			buf.WriteString("\n```\n")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
}

// createResourceExample creates an example object for a resource
func (g *MarkdownGenerator) createResourceExample(resource *ResourceDoc) map[string]interface{} {
	example := make(map[string]interface{})
	for _, field := range resource.Fields {
		if field.Example != nil {
			example[field.Name] = field.Example
		}
	}
	return example
}
