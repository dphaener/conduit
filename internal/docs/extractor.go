package docs

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Extractor extracts documentation from AST nodes
type Extractor struct {
	exampleGen *ExampleGenerator
}

// NewExtractor creates a new documentation extractor
func NewExtractor() *Extractor {
	return &Extractor{
		exampleGen: NewExampleGenerator(),
	}
}

// Extract extracts documentation from a parsed program
func (e *Extractor) Extract(program *ast.Program, projectName, projectVersion, projectDescription string) *Documentation {
	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:        projectName,
			Version:     projectVersion,
			Description: projectDescription,
		},
		Resources: make([]*ResourceDoc, 0, len(program.Resources)),
	}

	for _, resource := range program.Resources {
		resourceDoc := e.extractResource(resource)
		doc.Resources = append(doc.Resources, resourceDoc)
	}

	return doc
}

// extractResource extracts documentation from a resource node
func (e *Extractor) extractResource(resource *ast.ResourceNode) *ResourceDoc {
	doc := &ResourceDoc{
		Name:          resource.Name,
		Documentation: cleanDocumentation(resource.Documentation),
		Fields:        make([]*FieldDoc, 0, len(resource.Fields)),
		Relationships: make([]*RelationshipDoc, 0, len(resource.Relationships)),
		Endpoints:     e.generateEndpoints(resource),
		Hooks:         make([]*HookDoc, 0, len(resource.Hooks)),
		Validations:   make([]*ValidationDoc, 0, len(resource.Validations)),
		Constraints:   make([]*ConstraintDoc, 0, len(resource.Constraints)),
	}

	// Extract fields
	for _, field := range resource.Fields {
		fieldDoc := e.extractField(field)
		doc.Fields = append(doc.Fields, fieldDoc)
	}

	// Extract relationships
	for _, rel := range resource.Relationships {
		relDoc := e.extractRelationship(rel)
		doc.Relationships = append(doc.Relationships, relDoc)
	}

	// Extract hooks
	for _, hook := range resource.Hooks {
		hookDoc := e.extractHook(hook)
		doc.Hooks = append(doc.Hooks, hookDoc)
	}

	// Extract validations
	for _, validation := range resource.Validations {
		validationDoc := e.extractValidation(validation)
		doc.Validations = append(doc.Validations, validationDoc)
	}

	// Extract constraints
	for _, constraint := range resource.Constraints {
		constraintDoc := e.extractConstraint(constraint)
		doc.Constraints = append(doc.Constraints, constraintDoc)
	}

	return doc
}

// extractField extracts documentation from a field node
func (e *Extractor) extractField(field *ast.FieldNode) *FieldDoc {
	typeStr := e.formatType(field.Type)

	// Collect constraints
	constraints := make([]string, 0, len(field.Constraints))
	for _, constraint := range field.Constraints {
		constraints = append(constraints, e.formatConstraint(constraint))
	}

	// Generate example value
	example := e.exampleGen.GenerateForType(field.Type)

	// Extract default value
	var defaultValue string
	if field.Default != nil {
		defaultValue = e.formatExpression(field.Default)
	}

	return &FieldDoc{
		Name:        field.Name,
		Type:        typeStr,
		Description: "", // No inline field documentation in current spec
		Required:    !field.Nullable,
		Constraints: constraints,
		Default:     defaultValue,
		Example:     example,
	}
}

// extractRelationship extracts documentation from a relationship node
func (e *Extractor) extractRelationship(rel *ast.RelationshipNode) *RelationshipDoc {
	kind := "belongs_to" // Default
	if rel.Kind == ast.RelationshipHasMany {
		kind = "has_many"
	}
	if rel.Kind == ast.RelationshipHasManyThrough {
		kind = "has_many_through"
	}

	foreignKey := rel.ForeignKey
	if foreignKey == "" {
		foreignKey = fmt.Sprintf("%s_id", strings.ToLower(rel.Type))
	}

	return &RelationshipDoc{
		Name:        rel.Name,
		Type:        rel.Type,
		Kind:        kind,
		ForeignKey:  foreignKey,
		Description: "", // No inline relationship documentation in current spec
	}
}

// extractHook extracts documentation from a hook node
func (e *Extractor) extractHook(hook *ast.HookNode) *HookDoc {
	return &HookDoc{
		Timing:        hook.Timing,
		Event:         hook.Event,
		Description:   fmt.Sprintf("%s %s hook", hook.Timing, hook.Event),
		IsAsync:       hook.IsAsync,
		IsTransaction: hook.IsTransaction,
	}
}

// extractValidation extracts documentation from a validation node
func (e *Extractor) extractValidation(validation *ast.ValidationNode) *ValidationDoc {
	return &ValidationDoc{
		Name:         validation.Name,
		Description:  fmt.Sprintf("Validation: %s", validation.Name),
		ErrorMessage: validation.Error,
	}
}

// extractConstraint extracts documentation from a constraint node
func (e *Extractor) extractConstraint(constraint *ast.ConstraintNode) *ConstraintDoc {
	args := make([]string, 0, len(constraint.Arguments))
	for _, arg := range constraint.Arguments {
		args = append(args, e.formatExpression(arg))
	}

	return &ConstraintDoc{
		Name:        constraint.Name,
		Description: fmt.Sprintf("Constraint: %s", constraint.Name),
		Arguments:   args,
		On:          constraint.On,
	}
}

// generateEndpoints generates REST API endpoints for a resource
func (e *Extractor) generateEndpoints(resource *ast.ResourceNode) []*EndpointDoc {
	endpoints := make([]*EndpointDoc, 0)
	resourceName := strings.ToLower(resource.Name)
	resourcePath := "/" + pluralize(resourceName)

	// List endpoint - GET /resources
	endpoints = append(endpoints, &EndpointDoc{
		Method:      "GET",
		Path:        resourcePath,
		Summary:     fmt.Sprintf("List all %s", pluralize(resource.Name)),
		Description: fmt.Sprintf("Retrieve a paginated list of %s", pluralize(resource.Name)),
		Parameters: []*ParameterDoc{
			{Name: "page", In: "query", Type: "integer", Required: false, Description: "Page number", Example: 1},
			{Name: "limit", In: "query", Type: "integer", Required: false, Description: "Items per page", Example: 20},
		},
		Responses: map[int]*ResponseDoc{
			200: {
				StatusCode:  200,
				Description: "Success",
				ContentType: "application/json",
				Schema:      e.createArraySchema(resource),
				Example:     e.createArrayExample(resource),
			},
		},
		Middleware: resource.Middleware,
	})

	// Get endpoint - GET /resources/:id
	endpoints = append(endpoints, &EndpointDoc{
		Method:      "GET",
		Path:        resourcePath + "/:id",
		Summary:     fmt.Sprintf("Get a %s by ID", resource.Name),
		Description: fmt.Sprintf("Retrieve a single %s by its unique identifier", resource.Name),
		Parameters: []*ParameterDoc{
			{Name: "id", In: "path", Type: "string", Required: true, Description: "Resource ID", Example: "uuid"},
		},
		Responses: map[int]*ResponseDoc{
			200: {
				StatusCode:  200,
				Description: "Success",
				ContentType: "application/json",
				Schema:      e.createObjectSchema(resource),
				Example:     e.createObjectExample(resource),
			},
			404: {
				StatusCode:  404,
				Description: "Not found",
				ContentType: "application/json",
			},
		},
		Middleware: resource.Middleware,
	})

	// Create endpoint - POST /resources
	endpoints = append(endpoints, &EndpointDoc{
		Method:      "POST",
		Path:        resourcePath,
		Summary:     fmt.Sprintf("Create a new %s", resource.Name),
		Description: fmt.Sprintf("Create a new %s with the provided data", resource.Name),
		RequestBody: &RequestBodyDoc{
			Description: fmt.Sprintf("%s data", resource.Name),
			Required:    true,
			ContentType: "application/json",
			Schema:      e.createObjectSchema(resource),
			Example:     e.createObjectExample(resource),
		},
		Responses: map[int]*ResponseDoc{
			201: {
				StatusCode:  201,
				Description: "Created",
				ContentType: "application/json",
				Schema:      e.createObjectSchema(resource),
				Example:     e.createObjectExample(resource),
			},
			400: {
				StatusCode:  400,
				Description: "Bad request",
				ContentType: "application/json",
			},
		},
		Middleware: resource.Middleware,
	})

	// Update endpoint - PUT /resources/:id
	endpoints = append(endpoints, &EndpointDoc{
		Method:      "PUT",
		Path:        resourcePath + "/:id",
		Summary:     fmt.Sprintf("Update a %s", resource.Name),
		Description: fmt.Sprintf("Update an existing %s with the provided data", resource.Name),
		Parameters: []*ParameterDoc{
			{Name: "id", In: "path", Type: "string", Required: true, Description: "Resource ID", Example: "uuid"},
		},
		RequestBody: &RequestBodyDoc{
			Description: fmt.Sprintf("Updated %s data", resource.Name),
			Required:    true,
			ContentType: "application/json",
			Schema:      e.createObjectSchema(resource),
			Example:     e.createObjectExample(resource),
		},
		Responses: map[int]*ResponseDoc{
			200: {
				StatusCode:  200,
				Description: "Success",
				ContentType: "application/json",
				Schema:      e.createObjectSchema(resource),
				Example:     e.createObjectExample(resource),
			},
			404: {
				StatusCode:  404,
				Description: "Not found",
				ContentType: "application/json",
			},
		},
		Middleware: resource.Middleware,
	})

	// Delete endpoint - DELETE /resources/:id
	endpoints = append(endpoints, &EndpointDoc{
		Method:      "DELETE",
		Path:        resourcePath + "/:id",
		Summary:     fmt.Sprintf("Delete a %s", resource.Name),
		Description: fmt.Sprintf("Delete a %s by its unique identifier", resource.Name),
		Parameters: []*ParameterDoc{
			{Name: "id", In: "path", Type: "string", Required: true, Description: "Resource ID", Example: "uuid"},
		},
		Responses: map[int]*ResponseDoc{
			204: {
				StatusCode:  204,
				Description: "No content",
			},
			404: {
				StatusCode:  404,
				Description: "Not found",
				ContentType: "application/json",
			},
		},
		Middleware: resource.Middleware,
	})

	return endpoints
}

// createObjectSchema creates a JSON schema for a resource
func (e *Extractor) createObjectSchema(resource *ast.ResourceNode) *SchemaDoc {
	schema := &SchemaDoc{
		Type:       "object",
		Properties: make(map[string]*PropertyDoc),
		Required:   make([]string, 0),
	}

	for _, field := range resource.Fields {
		propType := e.schemaTypeForFieldType(field.Type)
		format := e.schemaFormatForFieldType(field.Type)

		schema.Properties[field.Name] = &PropertyDoc{
			Type:        propType,
			Description: fmt.Sprintf("%s field", field.Name),
			Format:      format,
			Example:     e.exampleGen.GenerateForType(field.Type),
		}

		if !field.Nullable {
			schema.Required = append(schema.Required, field.Name)
		}
	}

	return schema
}

// createArraySchema creates a JSON schema for an array of resources
func (e *Extractor) createArraySchema(resource *ast.ResourceNode) *SchemaDoc {
	return &SchemaDoc{
		Type:  "array",
		Items: e.createObjectSchema(resource),
	}
}

// createObjectExample creates an example object for a resource
func (e *Extractor) createObjectExample(resource *ast.ResourceNode) map[string]interface{} {
	example := make(map[string]interface{})

	for _, field := range resource.Fields {
		example[field.Name] = e.exampleGen.GenerateForType(field.Type)
	}

	return example
}

// createArrayExample creates an example array for a resource
func (e *Extractor) createArrayExample(resource *ast.ResourceNode) []map[string]interface{} {
	return []map[string]interface{}{
		e.createObjectExample(resource),
	}
}

// Helper functions

func (e *Extractor) formatType(typeNode *ast.TypeNode) string {
	if typeNode == nil {
		return "unknown"
	}

	suffix := "!"
	if typeNode.Nullable {
		suffix = "?"
	}

	switch typeNode.Kind {
	case ast.TypePrimitive:
		return typeNode.Name + suffix
	case ast.TypeArray:
		return fmt.Sprintf("array<%s>%s", e.formatType(typeNode.ElementType), suffix)
	case ast.TypeHash:
		return fmt.Sprintf("hash<%s,%s>%s", e.formatType(typeNode.KeyType), e.formatType(typeNode.ValueType), suffix)
	case ast.TypeEnum:
		return fmt.Sprintf("enum%s", suffix)
	case ast.TypeResource:
		return typeNode.Name + suffix
	default:
		return "unknown" + suffix
	}
}

func (e *Extractor) formatConstraint(constraint *ast.ConstraintNode) string {
	if len(constraint.Arguments) == 0 {
		return fmt.Sprintf("@%s", constraint.Name)
	}

	args := make([]string, len(constraint.Arguments))
	for i, arg := range constraint.Arguments {
		args[i] = e.formatExpression(arg)
	}

	return fmt.Sprintf("@%s(%s)", constraint.Name, strings.Join(args, ", "))
}

func (e *Extractor) formatExpression(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch ex := expr.(type) {
	case *ast.LiteralExpr:
		return fmt.Sprintf("%v", ex.Value)
	case *ast.IdentifierExpr:
		return ex.Name
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", e.formatExpression(ex.Left), ex.Operator, e.formatExpression(ex.Right))
	default:
		return "expression"
	}
}

func (e *Extractor) schemaTypeForFieldType(typeNode *ast.TypeNode) string {
	if typeNode == nil {
		return "string"
	}

	switch typeNode.Kind {
	case ast.TypeArray:
		return "array"
	case ast.TypeHash:
		return "object"
	case ast.TypePrimitive:
		switch typeNode.Name {
		case "int", "integer":
			return "integer"
		case "float", "decimal":
			return "number"
		case "bool", "boolean":
			return "boolean"
		default:
			return "string"
		}
	default:
		return "string"
	}
}

func (e *Extractor) schemaFormatForFieldType(typeNode *ast.TypeNode) string {
	if typeNode == nil || typeNode.Kind != ast.TypePrimitive {
		return ""
	}

	switch typeNode.Name {
	case "uuid":
		return "uuid"
	case "date":
		return "date"
	case "datetime", "timestamp":
		return "date-time"
	case "email":
		return "email"
	case "url":
		return "uri"
	default:
		return ""
	}
}

func cleanDocumentation(doc string) string {
	// Remove leading /// and trim whitespace
	lines := strings.Split(doc, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "///")
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, " ")
}

func pluralize(word string) string {
	// Simple pluralization - can be enhanced later
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
	   strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") {
		return word + "es"
	}
	if strings.HasSuffix(word, "y") && len(word) > 1 {
		// Check if the letter before 'y' is a consonant
		beforeY := rune(word[len(word)-2])
		if !isVowel(beforeY) {
			return word[:len(word)-1] + "ies"
		}
	}
	return word + "s"
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	}
	return false
}
