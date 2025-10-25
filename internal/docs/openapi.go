package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// OpenAPIGenerator generates OpenAPI 3.0 specifications
type OpenAPIGenerator struct {
	config *Config
}

// NewOpenAPIGenerator creates a new OpenAPI generator
func NewOpenAPIGenerator(config *Config) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		config: config,
	}
}

// Generate generates an OpenAPI 3.0 specification
func (g *OpenAPIGenerator) Generate(doc *Documentation) error {
	spec := g.createSpec(doc)

	// Validate the output directory BEFORE making it absolute
	if containsPathTraversal(g.config.OutputDir) {
		return fmt.Errorf("invalid output directory: path traversal detected")
	}

	// Clean and make the path absolute
	outputDir := filepath.Clean(g.config.OutputDir)
	if !filepath.IsAbs(outputDir) {
		cwd, _ := os.Getwd()
		outputDir = filepath.Join(cwd, outputDir)
	}

	// Ensure output directory exists
	outputPath := filepath.Join(outputDir, "openapi.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec: %w", err)
	}

	return nil
}

// createSpec creates the complete OpenAPI specification
func (g *OpenAPIGenerator) createSpec(doc *Documentation) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       doc.ProjectInfo.Name,
			"version":     doc.ProjectInfo.Version,
			"description": doc.ProjectInfo.Description,
		},
		"servers":    g.createServers(),
		"paths":      g.createPaths(doc.Resources),
		"components": g.createComponents(doc.Resources),
	}

	return spec
}

// createServers creates the servers section
func (g *OpenAPIGenerator) createServers() []map[string]interface{} {
	servers := make([]map[string]interface{}, 0)

	// Add base URL
	if g.config.BaseURL != "" {
		servers = append(servers, map[string]interface{}{
			"url":         g.config.BaseURL,
			"description": "Production server",
		})
	}

	// Add additional server URLs
	for _, serverURL := range g.config.ServerURLs {
		servers = append(servers, map[string]interface{}{
			"url":         serverURL.URL,
			"description": serverURL.Description,
		})
	}

	// Default server if none provided
	if len(servers) == 0 {
		servers = append(servers, map[string]interface{}{
			"url":         "http://localhost:3000",
			"description": "Development server",
		})
	}

	return servers
}

// createPaths creates the paths section
func (g *OpenAPIGenerator) createPaths(resources []*ResourceDoc) map[string]interface{} {
	paths := make(map[string]interface{})

	for _, resource := range resources {
		for _, endpoint := range resource.Endpoints {
			pathItem, ok := paths[endpoint.Path].(map[string]interface{})
			if !ok {
				pathItem = make(map[string]interface{})
				paths[endpoint.Path] = pathItem
			}

			operation := g.createOperation(endpoint, resource.Name)
			pathItem[endpoint.Method] = operation
		}
	}

	return paths
}

// createOperation creates an operation object
func (g *OpenAPIGenerator) createOperation(endpoint *EndpointDoc, resourceName string) map[string]interface{} {
	operation := map[string]interface{}{
		"summary":     endpoint.Summary,
		"description": endpoint.Description,
		"operationId": fmt.Sprintf("%s_%s", endpoint.Method, resourceName),
		"tags":        []string{resourceName},
		"responses":   g.createResponses(endpoint.Responses),
	}

	// Add parameters if any
	if len(endpoint.Parameters) > 0 {
		operation["parameters"] = g.createParameters(endpoint.Parameters)
	}

	// Add request body if any
	if endpoint.RequestBody != nil {
		operation["requestBody"] = g.createRequestBody(endpoint.RequestBody)
	}

	return operation
}

// createParameters creates parameters array
func (g *OpenAPIGenerator) createParameters(params []*ParameterDoc) []map[string]interface{} {
	parameters := make([]map[string]interface{}, 0, len(params))

	for _, param := range params {
		parameter := map[string]interface{}{
			"name":        param.Name,
			"in":          param.In,
			"required":    param.Required,
			"description": param.Description,
			"schema": map[string]interface{}{
				"type": param.Type,
			},
		}

		if param.Example != nil {
			parameter["example"] = param.Example
		}

		parameters = append(parameters, parameter)
	}

	return parameters
}

// createRequestBody creates a request body object
func (g *OpenAPIGenerator) createRequestBody(body *RequestBodyDoc) map[string]interface{} {
	requestBody := map[string]interface{}{
		"description": body.Description,
		"required":    body.Required,
		"content": map[string]interface{}{
			body.ContentType: map[string]interface{}{
				"schema": g.createSchemaObject(body.Schema),
			},
		},
	}

	if body.Example != nil {
		content := requestBody["content"].(map[string]interface{})
		mediaType := content[body.ContentType].(map[string]interface{})
		mediaType["example"] = body.Example
	}

	return requestBody
}

// createResponses creates responses object
func (g *OpenAPIGenerator) createResponses(responses map[int]*ResponseDoc) map[string]interface{} {
	responsesObj := make(map[string]interface{})

	for statusCode, response := range responses {
		statusKey := fmt.Sprintf("%d", statusCode)
		responseObj := map[string]interface{}{
			"description": response.Description,
		}

		if response.ContentType != "" && response.Schema != nil {
			responseObj["content"] = map[string]interface{}{
				response.ContentType: map[string]interface{}{
					"schema": g.createSchemaObject(response.Schema),
				},
			}

			if response.Example != nil {
				content := responseObj["content"].(map[string]interface{})
				mediaType := content[response.ContentType].(map[string]interface{})
				mediaType["example"] = response.Example
			}
		}

		responsesObj[statusKey] = responseObj
	}

	return responsesObj
}

// createSchemaObject creates a schema object
func (g *OpenAPIGenerator) createSchemaObject(schema *SchemaDoc) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{}
	}

	schemaObj := map[string]interface{}{
		"type": schema.Type,
	}

	// Handle array type
	if schema.Type == "array" && schema.Items != nil {
		schemaObj["items"] = g.createSchemaObject(schema.Items)
	}

	// Handle object type
	if schema.Type == "object" && len(schema.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, prop := range schema.Properties {
			properties[name] = g.createPropertyObject(prop)
		}
		schemaObj["properties"] = properties

		if len(schema.Required) > 0 {
			schemaObj["required"] = schema.Required
		}
	}

	return schemaObj
}

// createPropertyObject creates a property object
func (g *OpenAPIGenerator) createPropertyObject(prop *PropertyDoc) map[string]interface{} {
	propObj := map[string]interface{}{
		"type": prop.Type,
	}

	if prop.Description != "" {
		propObj["description"] = prop.Description
	}

	if prop.Format != "" {
		propObj["format"] = prop.Format
	}

	if prop.Example != nil {
		propObj["example"] = prop.Example
	}

	if len(prop.Enum) > 0 {
		propObj["enum"] = prop.Enum
	}

	return propObj
}

// createComponents creates the components section
func (g *OpenAPIGenerator) createComponents(resources []*ResourceDoc) map[string]interface{} {
	schemas := make(map[string]interface{})

	for _, resource := range resources {
		// Create schema for the resource
		properties := make(map[string]interface{})
		required := make([]string, 0)

		for _, field := range resource.Fields {
			properties[field.Name] = map[string]interface{}{
				"type":        g.mapTypeToOpenAPI(field.Type),
				"description": field.Description,
			}

			if field.Example != nil {
				properties[field.Name].(map[string]interface{})["example"] = field.Example
			}

			if field.Required {
				required = append(required, field.Name)
			}
		}

		schema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}

		if len(required) > 0 {
			schema["required"] = required
		}

		if resource.Documentation != "" {
			schema["description"] = resource.Documentation
		}

		schemas[resource.Name] = schema
	}

	return map[string]interface{}{
		"schemas": schemas,
	}
}

// mapTypeToOpenAPI maps Conduit types to OpenAPI types
func (g *OpenAPIGenerator) mapTypeToOpenAPI(conduitType string) string {
	if len(conduitType) == 0 {
		return "string"
	}

	// Remove nullability markers (! or ?)
	baseType := conduitType
	if lastChar := conduitType[len(conduitType)-1]; lastChar == '!' || lastChar == '?' {
		baseType = conduitType[:len(conduitType)-1]
	}

	switch baseType {
	case "int", "integer", "bigint":
		return "integer"
	case "float", "decimal", "money":
		return "number"
	case "bool", "boolean":
		return "boolean"
	default:
		return "string"
	}
}
