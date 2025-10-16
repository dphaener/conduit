package metadata

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Extractor extracts introspection metadata from an AST
type Extractor struct {
	version string
	routes  []RouteMetadata
}

// NewExtractor creates a new metadata extractor
func NewExtractor(version string) *Extractor {
	return &Extractor{
		version: version,
		routes:  make([]RouteMetadata, 0),
	}
}

// Extract generates metadata from a program AST
func (e *Extractor) Extract(prog *ast.Program) (*Metadata, error) {
	meta := &Metadata{
		Version:   e.version,
		Resources: make([]ResourceMetadata, 0, len(prog.Resources)),
		Patterns:  make([]PatternMetadata, 0),
		Routes:    make([]RouteMetadata, 0),
	}

	// Extract metadata for each resource
	for _, resource := range prog.Resources {
		resMeta, err := e.extractResource(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to extract metadata for resource %s: %w", resource.Name, err)
		}
		meta.Resources = append(meta.Resources, resMeta)

		// Generate routes for this resource
		e.generateRoutes(resource)
	}

	// Extract patterns from all resources
	meta.Patterns = e.extractPatterns(prog.Resources)

	// Add generated routes
	meta.Routes = e.routes

	return meta, nil
}

// extractResource extracts metadata for a single resource
func (e *Extractor) extractResource(resource *ast.ResourceNode) (ResourceMetadata, error) {
	resMeta := ResourceMetadata{
		Name:          resource.Name,
		Documentation: resource.Documentation,
		Fields:        make([]FieldMetadata, 0, len(resource.Fields)),
		Relationships: make([]RelationshipMetadata, 0, len(resource.Relationships)),
		Hooks:         make([]HookMetadata, 0, len(resource.Hooks)),
		Validations:   make([]ValidationMetadata, 0, len(resource.Validations)),
		Constraints:   make([]ConstraintMetadata, 0, len(resource.Constraints)),
		Scopes:        make([]ScopeMetadata, 0, len(resource.Scopes)),
		Computed:      make([]ComputedMetadata, 0, len(resource.Computed)),
		Operations:    resource.Operations,
		Middleware:    resource.Middleware,
	}

	// Extract fields
	for _, field := range resource.Fields {
		fieldMeta := e.extractField(field)
		resMeta.Fields = append(resMeta.Fields, fieldMeta)
	}

	// Extract relationships
	for _, rel := range resource.Relationships {
		relMeta := e.extractRelationship(rel)
		resMeta.Relationships = append(resMeta.Relationships, relMeta)
	}

	// Extract hooks
	for _, hook := range resource.Hooks {
		hookMeta := e.extractHook(hook)
		resMeta.Hooks = append(resMeta.Hooks, hookMeta)
	}

	// Extract validations
	for _, validation := range resource.Validations {
		valMeta := e.extractValidation(validation)
		resMeta.Validations = append(resMeta.Validations, valMeta)
	}

	// Extract constraints
	for _, constraint := range resource.Constraints {
		constMeta := e.extractConstraint(constraint)
		resMeta.Constraints = append(resMeta.Constraints, constMeta)
	}

	// Extract scopes
	for _, scope := range resource.Scopes {
		scopeMeta := e.extractScope(scope)
		resMeta.Scopes = append(resMeta.Scopes, scopeMeta)
	}

	// Extract computed fields
	for _, computed := range resource.Computed {
		compMeta := e.extractComputed(computed)
		resMeta.Computed = append(resMeta.Computed, compMeta)
	}

	return resMeta, nil
}

// extractField extracts metadata for a field
func (e *Extractor) extractField(field *ast.FieldNode) FieldMetadata {
	fieldMeta := FieldMetadata{
		Name:        field.Name,
		Type:        e.formatType(field.Type),
		Nullable:    field.Nullable,
		Constraints: make([]string, 0),
	}

	// Extract constraints
	for _, constraint := range field.Constraints {
		constraintStr := e.formatConstraint(constraint)
		fieldMeta.Constraints = append(fieldMeta.Constraints, constraintStr)
	}

	// Extract default value if present
	if field.Default != nil {
		fieldMeta.Default = e.formatExpression(field.Default)
	}

	return fieldMeta
}

// extractRelationship extracts metadata for a relationship
func (e *Extractor) extractRelationship(rel *ast.RelationshipNode) RelationshipMetadata {
	return RelationshipMetadata{
		Name:       rel.Name,
		Type:       rel.Type,
		Kind:       e.formatRelationshipKind(rel.Kind),
		ForeignKey: rel.ForeignKey,
		Through:    rel.Through,
		OnDelete:   rel.OnDelete,
		Nullable:   rel.Nullable,
	}
}

// extractHook extracts metadata for a hook
func (e *Extractor) extractHook(hook *ast.HookNode) HookMetadata {
	return HookMetadata{
		Timing:         hook.Timing,
		Event:          hook.Event,
		HasTransaction: hook.IsTransaction,
		HasAsync:       hook.IsAsync,
		Middleware:     hook.Middleware,
	}
}

// extractValidation extracts metadata for a validation
func (e *Extractor) extractValidation(validation *ast.ValidationNode) ValidationMetadata {
	return ValidationMetadata{
		Name:      validation.Name,
		Condition: e.formatExpression(validation.Condition),
		Error:     validation.Error,
	}
}

// extractConstraint extracts metadata for a constraint
func (e *Extractor) extractConstraint(constraint *ast.ConstraintNode) ConstraintMetadata {
	args := make([]string, 0, len(constraint.Arguments))
	for _, arg := range constraint.Arguments {
		args = append(args, e.formatExpression(arg))
	}

	meta := ConstraintMetadata{
		Name:      constraint.Name,
		Arguments: args,
		On:        constraint.On,
		Error:     constraint.Error,
	}

	if constraint.When != nil {
		meta.When = e.formatExpression(constraint.When)
	}

	if constraint.Condition != nil {
		meta.Condition = e.formatExpression(constraint.Condition)
	}

	return meta
}

// extractScope extracts metadata for a scope
func (e *Extractor) extractScope(scope *ast.ScopeNode) ScopeMetadata {
	args := make([]string, 0, len(scope.Arguments))
	for _, arg := range scope.Arguments {
		argStr := arg.Name
		if arg.Type != nil {
			argStr += ": " + e.formatType(arg.Type)
		}
		args = append(args, argStr)
	}

	return ScopeMetadata{
		Name:      scope.Name,
		Arguments: args,
		Condition: e.formatExpression(scope.Condition),
	}
}

// extractComputed extracts metadata for a computed field
func (e *Extractor) extractComputed(computed *ast.ComputedNode) ComputedMetadata {
	return ComputedMetadata{
		Name: computed.Name,
		Type: e.formatType(computed.Type),
		Body: e.formatExpression(computed.Body),
	}
}

// formatType formats a type node as a string
func (e *Extractor) formatType(t *ast.TypeNode) string {
	if t == nil {
		return "unknown"
	}

	var typeStr string

	switch t.Kind {
	case ast.TypePrimitive:
		typeStr = t.Name
	case ast.TypeArray:
		elemType := e.formatType(t.ElementType)
		typeStr = fmt.Sprintf("array<%s>", elemType)
	case ast.TypeHash:
		keyType := e.formatType(t.KeyType)
		valueType := e.formatType(t.ValueType)
		typeStr = fmt.Sprintf("hash<%s,%s>", keyType, valueType)
	case ast.TypeEnum:
		typeStr = fmt.Sprintf("enum[%s]", strings.Join(t.EnumValues, "|"))
	case ast.TypeResource:
		typeStr = t.Name
	default:
		typeStr = t.Name
	}

	// Add nullability marker
	if t.Nullable {
		typeStr += "?"
	} else {
		typeStr += "!"
	}

	return typeStr
}

// formatRelationshipKind formats a relationship kind
func (e *Extractor) formatRelationshipKind(kind ast.RelationshipKind) string {
	switch kind {
	case ast.RelationshipBelongsTo:
		return "belongs_to"
	case ast.RelationshipHasMany:
		return "has_many"
	case ast.RelationshipHasManyThrough:
		return "has_many_through"
	case ast.RelationshipHasOne:
		return "has_one"
	default:
		return "unknown"
	}
}

// formatConstraint formats a constraint as a string
func (e *Extractor) formatConstraint(constraint *ast.ConstraintNode) string {
	if len(constraint.Arguments) == 0 {
		return constraint.Name
	}

	args := make([]string, 0, len(constraint.Arguments))
	for _, arg := range constraint.Arguments {
		args = append(args, e.formatExpression(arg))
	}

	return fmt.Sprintf("%s(%s)", constraint.Name, strings.Join(args, ", "))
}

// formatExpression formats an expression node as a string
// NOTE: Currently returns a placeholder "<expression>" for all expressions.
// Full expression tree serialization is deferred to keep the initial implementation focused.
// Expression serialization requires recursive tree walking with careful handling of all
// expression types (binary, unary, call, field access, etc.)
func (e *Extractor) formatExpression(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	// For now, return a placeholder
	// In a full implementation, we would recursively format the expression tree
	return "<expression>"
}

// extractPatterns identifies common patterns across resources
func (e *Extractor) extractPatterns(resources []*ast.ResourceNode) []PatternMetadata {
	patterns := make(map[string]*PatternMetadata)

	// Pattern: Authenticated hooks
	authCount := 0
	for _, resource := range resources {
		for _, hook := range resource.Hooks {
			for _, mw := range hook.Middleware {
				if mw == "auth" {
					authCount++
				}
			}
		}
	}
	if authCount > 0 {
		patterns["authenticated_handler"] = &PatternMetadata{
			Name:        "authenticated_handler",
			Template:    "@after <event>: [auth]",
			Description: "Hooks with authentication middleware",
			Occurrences: authCount,
		}
	}

	// Pattern: Transaction hooks
	txCount := 0
	for _, resource := range resources {
		for _, hook := range resource.Hooks {
			if hook.IsTransaction {
				txCount++
			}
		}
	}
	if txCount > 0 {
		patterns["transactional_hook"] = &PatternMetadata{
			Name:        "transactional_hook",
			Template:    "@after <event> @transaction { ... }",
			Description: "Hooks that run within a database transaction",
			Occurrences: txCount,
		}
	}

	// Pattern: Async operations
	asyncCount := 0
	for _, resource := range resources {
		for _, hook := range resource.Hooks {
			if hook.IsAsync {
				asyncCount++
			}
		}
	}
	if asyncCount > 0 {
		patterns["async_operation"] = &PatternMetadata{
			Name:        "async_operation",
			Template:    "@async { ... }",
			Description: "Asynchronous operations in hooks",
			Occurrences: asyncCount,
		}
	}

	// Pattern: Unique constraints
	uniqueCount := 0
	for _, resource := range resources {
		for _, field := range resource.Fields {
			for _, constraint := range field.Constraints {
				if constraint.Name == "unique" {
					uniqueCount++
				}
			}
		}
	}
	if uniqueCount > 0 {
		patterns["unique_field"] = &PatternMetadata{
			Name:        "unique_field",
			Template:    "field: type! @unique",
			Description: "Fields with unique constraints",
			Occurrences: uniqueCount,
		}
	}

	// Convert map to sorted slice
	result := make([]PatternMetadata, 0, len(patterns))
	for _, p := range patterns {
		result = append(result, *p)
	}

	// Sort by name for consistency
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// generateRoutes generates REST API routes for a resource
func (e *Extractor) generateRoutes(resource *ast.ResourceNode) {
	resourcePath := strings.ToLower(resource.Name) + "s"

	// Standard REST routes
	routes := []RouteMetadata{
		{
			Method:      "GET",
			Path:        "/" + resourcePath,
			Handler:     "Index",
			Resource:    resource.Name,
			Middleware:  resource.Middleware,
			Description: fmt.Sprintf("List all %s", resourcePath),
		},
		{
			Method:      "GET",
			Path:        "/" + resourcePath + "/:id",
			Handler:     "Show",
			Resource:    resource.Name,
			Middleware:  resource.Middleware,
			Description: fmt.Sprintf("Get a single %s by ID", resource.Name),
		},
		{
			Method:      "POST",
			Path:        "/" + resourcePath,
			Handler:     "Create",
			Resource:    resource.Name,
			Middleware:  resource.Middleware,
			Description: fmt.Sprintf("Create a new %s", resource.Name),
		},
		{
			Method:      "PUT",
			Path:        "/" + resourcePath + "/:id",
			Handler:     "Update",
			Resource:    resource.Name,
			Middleware:  resource.Middleware,
			Description: fmt.Sprintf("Update an existing %s", resource.Name),
		},
		{
			Method:      "DELETE",
			Path:        "/" + resourcePath + "/:id",
			Handler:     "Delete",
			Resource:    resource.Name,
			Middleware:  resource.Middleware,
			Description: fmt.Sprintf("Delete a %s", resource.Name),
		},
	}

	e.routes = append(e.routes, routes...)
}
