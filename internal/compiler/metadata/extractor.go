package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Extractor extracts introspection metadata from an AST
type Extractor struct {
	version  string
	filePath string // Current source file being processed
	routes   []RouteMetadata
}

// NewExtractor creates a new metadata extractor
func NewExtractor(version string) *Extractor {
	return &Extractor{
		version: version,
		routes:  make([]RouteMetadata, 0),
	}
}

// SetFilePath sets the current source file path for location tracking
func (e *Extractor) SetFilePath(path string) {
	e.filePath = path
}

// Extract generates metadata from a program AST.
//
// This function employs graceful error handling: if errors occur while processing
// individual resources, it continues extracting metadata from remaining resources
// and returns partial metadata along with the first error encountered. This ensures
// that incremental compilation can still benefit from successfully extracted resources
// even when some resources contain errors.
//
// Returns:
//   - *Metadata: Complete or partial metadata (never nil)
//   - error: The first error encountered during extraction, or nil if all succeeded
func (e *Extractor) Extract(prog *ast.Program) (*Metadata, error) {
	meta := &Metadata{
		Version:   e.version,
		Resources: make([]ResourceMetadata, 0, len(prog.Resources)),
		Patterns:  make([]PatternMetadata, 0),
		Routes:    make([]RouteMetadata, 0),
	}

	// Collect resource metadata and track errors
	var extractionErrors []error

	// Extract metadata for each resource
	for _, resource := range prog.Resources {
		resMeta, err := e.extractResource(resource)
		if err != nil {
			// Graceful error handling: log error but continue with other resources
			extractionErrors = append(extractionErrors,
				fmt.Errorf("resource %s: %w", resource.Name, err))
			continue
		}
		meta.Resources = append(meta.Resources, resMeta)

		// Generate routes for this resource
		e.generateRoutes(resource)
	}

	// Extract patterns from all resources
	meta.Patterns = e.extractPatterns(prog.Resources)

	// Add generated routes
	meta.Routes = e.routes

	// Compute source hash for change detection
	meta.SourceHash = e.computeSourceHash(prog)

	// Return first error if any occurred, but still return partial metadata
	if len(extractionErrors) > 0 {
		return meta, extractionErrors[0]
	}

	return meta, nil
}

// computeSourceHash computes a hash of the entire AST for change detection
func (e *Extractor) computeSourceHash(prog *ast.Program) string {
	h := sha256.New()

	// Hash resource names and locations in deterministic order
	for _, resource := range prog.Resources {
		h.Write([]byte(resource.Name))
		h.Write([]byte(fmt.Sprintf("%d:%d", resource.Loc.Line, resource.Loc.Column)))

		// Hash field signatures
		for _, field := range resource.Fields {
			h.Write([]byte(field.Name))
			if field.Type != nil {
				h.Write([]byte(e.formatType(field.Type)))
			}
			// Hash field constraints
			for _, constraint := range field.Constraints {
				h.Write([]byte(constraint.Name))
			}
			// Hash default values
			if field.Default != nil {
				h.Write([]byte(e.formatExpression(field.Default)))
			}
		}

		// Hash relationship signatures
		for _, rel := range resource.Relationships {
			h.Write([]byte(rel.Name))
			h.Write([]byte(rel.Type))
			h.Write([]byte(e.formatRelationshipKind(rel.Kind)))
			h.Write([]byte(rel.ForeignKey))
			h.Write([]byte(rel.OnDelete))
		}

		// Hash hook signatures
		for _, hook := range resource.Hooks {
			h.Write([]byte(hook.Timing))
			h.Write([]byte(hook.Event))
			h.Write([]byte(fmt.Sprintf("%t", hook.IsTransaction)))
			h.Write([]byte(fmt.Sprintf("%t", hook.IsAsync)))
			// Hash middleware
			for _, mw := range hook.Middleware {
				h.Write([]byte(mw))
			}
		}

		// Hash validations
		for _, validation := range resource.Validations {
			h.Write([]byte(validation.Name))
			if validation.Condition != nil {
				h.Write([]byte(e.formatExpression(validation.Condition)))
			}
			h.Write([]byte(validation.Error))
		}

		// Hash constraints
		for _, constraint := range resource.Constraints {
			h.Write([]byte(constraint.Name))
			if constraint.When != nil {
				h.Write([]byte(e.formatExpression(constraint.When)))
			}
			if constraint.Condition != nil {
				h.Write([]byte(e.formatExpression(constraint.Condition)))
			}
			h.Write([]byte(constraint.Error))
			// Hash constraint events
			for _, event := range constraint.On {
				h.Write([]byte(event))
			}
		}

		// Hash scopes
		for _, scope := range resource.Scopes {
			h.Write([]byte(scope.Name))
			if scope.Condition != nil {
				h.Write([]byte(e.formatExpression(scope.Condition)))
			}
			// Hash scope arguments
			for _, arg := range scope.Arguments {
				h.Write([]byte(arg.Name))
				if arg.Type != nil {
					h.Write([]byte(e.formatType(arg.Type)))
				}
			}
		}

		// Hash computed fields
		for _, computed := range resource.Computed {
			h.Write([]byte(computed.Name))
			if computed.Type != nil {
				h.Write([]byte(e.formatType(computed.Type)))
			}
			if computed.Body != nil {
				h.Write([]byte(e.formatExpression(computed.Body)))
			}
		}

		// Hash operations
		for _, op := range resource.Operations {
			h.Write([]byte(op))
		}

		// Hash middleware
		for _, mw := range resource.Middleware {
			h.Write([]byte(mw))
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// extractResource extracts metadata for a single resource
func (e *Extractor) extractResource(resource *ast.ResourceNode) (ResourceMetadata, error) {
	resMeta := ResourceMetadata{
		Name:          resource.Name,
		Documentation: resource.Documentation,
		FilePath:      e.filePath,
		Line:          resource.Loc.Line,
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
	// Format hook body as source code
	sourceCode := e.formatHookBody(hook.Body)

	return HookMetadata{
		Timing:         hook.Timing,
		Event:          hook.Event,
		HasTransaction: hook.IsTransaction,
		HasAsync:       hook.IsAsync,
		SourceCode:     sourceCode,
		Line:           hook.Loc.Line,
		Middleware:     hook.Middleware,
	}
}

// formatHookBody formats hook body statements as source code
func (e *Extractor) formatHookBody(stmts []ast.StmtNode) string {
	if len(stmts) == 0 {
		return ""
	}

	lines := make([]string, 0, len(stmts))
	for _, stmt := range stmts {
		lines = append(lines, e.formatStatement(stmt))
	}

	return strings.Join(lines, "\n")
}

// formatStatement formats a statement node as source code
func (e *Extractor) formatStatement(stmt ast.StmtNode) string {
	if stmt == nil {
		return ""
	}

	switch node := stmt.(type) {
	case *ast.ExprStmt:
		return e.formatExpression(node.Expr)

	case *ast.AssignmentStmt:
		target := e.formatExpression(node.Target)
		value := e.formatExpression(node.Value)
		return fmt.Sprintf("%s = %s", target, value)

	case *ast.LetStmt:
		typeStr := ""
		if node.Type != nil {
			typeStr = ": " + e.formatType(node.Type)
		}
		value := e.formatExpression(node.Value)
		return fmt.Sprintf("let %s%s = %s", node.Name, typeStr, value)

	case *ast.ReturnStmt:
		if node.Value != nil {
			return fmt.Sprintf("return %s", e.formatExpression(node.Value))
		}
		return "return"

	case *ast.IfStmt:
		condition := e.formatExpression(node.Condition)
		return fmt.Sprintf("if %s { ... }", condition)

	case *ast.BlockStmt:
		if node.IsAsync {
			return "@async { ... }"
		}
		return "{ ... }"

	case *ast.RescueStmt:
		return fmt.Sprintf("rescue %s { ... }", node.ErrorVar)

	case *ast.MatchStmt:
		value := e.formatExpression(node.Value)
		return fmt.Sprintf("match %s { ... }", value)

	default:
		return "<statement>"
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
	case ast.TypeStruct:
		// Format struct fields as {field1: type1, field2: type2, ...}
		if len(t.StructFields) == 0 {
			typeStr = "struct{}"
		} else {
			fieldStrs := make([]string, 0, len(t.StructFields))
			for _, field := range t.StructFields {
				fieldType := e.formatType(field.Type)
				fieldStrs = append(fieldStrs, fmt.Sprintf("%s: %s", field.Name, fieldType))
			}
			typeStr = fmt.Sprintf("struct{%s}", strings.Join(fieldStrs, ", "))
		}
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
// Recursively serializes the complete expression tree to preserve source code
func (e *Extractor) formatExpression(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch node := expr.(type) {
	case *ast.LiteralExpr:
		return e.formatLiteral(node.Value)

	case *ast.IdentifierExpr:
		return node.Name

	case *ast.SelfExpr:
		return "self"

	case *ast.BinaryExpr:
		// Defensive nil checks for critical fields
		if node.Left == nil || node.Right == nil {
			return "<invalid-binary-expr>"
		}
		left := e.formatExpression(node.Left)
		right := e.formatExpression(node.Right)
		return fmt.Sprintf("%s %s %s", left, node.Operator, right)

	case *ast.UnaryExpr:
		if node.Operand == nil {
			return "<invalid-unary-expr>"
		}
		operand := e.formatExpression(node.Operand)
		return fmt.Sprintf("%s%s", node.Operator, operand)

	case *ast.LogicalExpr:
		// Defensive nil checks for critical fields
		if node.Left == nil || node.Right == nil {
			return "<invalid-logical-expr>"
		}
		left := e.formatExpression(node.Left)
		right := e.formatExpression(node.Right)
		return fmt.Sprintf("%s %s %s", left, node.Operator, right)

	case *ast.CallExpr:
		var funcName string
		if node.Namespace != "" {
			funcName = fmt.Sprintf("%s.%s", node.Namespace, node.Function)
		} else {
			funcName = node.Function
		}

		args := make([]string, 0, len(node.Arguments))
		for _, arg := range node.Arguments {
			args = append(args, e.formatExpression(arg))
		}

		return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))

	case *ast.FieldAccessExpr:
		object := e.formatExpression(node.Object)
		return fmt.Sprintf("%s.%s", object, node.Field)

	case *ast.SafeNavigationExpr:
		object := e.formatExpression(node.Object)
		return fmt.Sprintf("%s?.%s", object, node.Field)

	case *ast.ArrayLiteralExpr:
		elements := make([]string, 0, len(node.Elements))
		for _, elem := range node.Elements {
			elements = append(elements, e.formatExpression(elem))
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))

	case *ast.HashLiteralExpr:
		pairs := make([]string, 0, len(node.Pairs))
		for _, pair := range node.Pairs {
			key := e.formatExpression(pair.Key)
			value := e.formatExpression(pair.Value)
			pairs = append(pairs, fmt.Sprintf("%s: %s", key, value))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))

	case *ast.IndexExpr:
		object := e.formatExpression(node.Object)
		index := e.formatExpression(node.Index)
		return fmt.Sprintf("%s[%s]", object, index)

	case *ast.NullCoalesceExpr:
		left := e.formatExpression(node.Left)
		right := e.formatExpression(node.Right)
		return fmt.Sprintf("%s ?? %s", left, right)

	case *ast.ParenExpr:
		inner := e.formatExpression(node.Expr)
		return fmt.Sprintf("(%s)", inner)

	case *ast.InterpolatedStringExpr:
		parts := make([]string, 0, len(node.Parts))
		for _, part := range node.Parts {
			parts = append(parts, e.formatExpression(part))
		}
		return fmt.Sprintf("\"%s\"", strings.Join(parts, ""))

	case *ast.RangeExpr:
		start := e.formatExpression(node.Start)
		end := e.formatExpression(node.End)
		if node.Exclusive {
			return fmt.Sprintf("%s...%s", start, end)
		}
		return fmt.Sprintf("%s..%s", start, end)

	case *ast.LambdaExpr:
		params := make([]string, 0, len(node.Parameters))
		for _, param := range node.Parameters {
			paramStr := param.Name
			if param.Type != nil {
				paramStr += ": " + e.formatType(param.Type)
			}
			params = append(params, paramStr)
		}
		return fmt.Sprintf("|%s| { ... }", strings.Join(params, ", "))

	default:
		// Fallback for unknown expression types
		return "<expression>"
	}
}

// formatLiteral formats a literal value
func (e *Extractor) formatLiteral(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
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

// generateRoutes generates REST API routes for a resource according to these rules:
//
// Standard Operations:
//   - list:   GET    /resources
//   - get:    GET    /resources/:id
//   - create: POST   /resources
//   - update: PUT    /resources/:id
//   - delete: DELETE /resources/:id
//
// Operation Filtering:
//   - If resource.Operations is empty, all 5 standard operations are generated
//   - If resource.Operations is set, only specified operations are generated
//   - Unknown operation names are silently ignored (they don't match any standard route)
//
// Middleware:
//   - Resource-level middleware (resource.Middleware) is applied to all routes
//   - Per-operation middleware (@on <op>: [mw]) is not yet supported
//     TODO(CON-56): Implement per-operation middleware extraction when AST supports it
//     This requires adding OperationMiddleware map[string][]string to ResourceNode
//
// Nested Routes:
//   - Has-many relationships generate: GET /parents/:id/children
//   - Handler format uses relationship name: Parent.relationshipName.list
//     This allows the web framework to distinguish between different relationships
//     to the same resource type (e.g., Post.comments vs Post.replies)
//
// Pluralization:
//   - Resource names are pluralized for paths using simple English rules
//   - Handles common cases: post→posts, category→categories, person→people
//   - See toPlural() for full pluralization logic and limitations
func (e *Extractor) generateRoutes(resource *ast.ResourceNode) {
	resourcePath := e.toPlural(strings.ToLower(resource.Name))

	// Map of operation to route configuration
	type routeConfig struct {
		method      string
		path        string
		handler     string
		operation   string
		description string
	}

	standardRoutes := map[string]routeConfig{
		"list": {
			method:      "GET",
			path:        "/" + resourcePath,
			handler:     resource.Name + ".list",
			operation:   "list",
			description: fmt.Sprintf("List all %s", resourcePath),
		},
		"get": {
			method:      "GET",
			path:        "/" + resourcePath + "/:id",
			handler:     resource.Name + ".get",
			operation:   "get",
			description: fmt.Sprintf("Get a single %s by ID", resource.Name),
		},
		"create": {
			method:      "POST",
			path:        "/" + resourcePath,
			handler:     resource.Name + ".create",
			operation:   "create",
			description: fmt.Sprintf("Create a new %s", resource.Name),
		},
		"update": {
			method:      "PUT",
			path:        "/" + resourcePath + "/:id",
			handler:     resource.Name + ".update",
			operation:   "update",
			description: fmt.Sprintf("Update an existing %s", resource.Name),
		},
		"delete": {
			method:      "DELETE",
			path:        "/" + resourcePath + "/:id",
			handler:     resource.Name + ".delete",
			operation:   "delete",
			description: fmt.Sprintf("Delete a %s", resource.Name),
		},
	}

	// Determine which operations to generate routes for
	allowedOps := make(map[string]bool)
	if len(resource.Operations) > 0 {
		// If @operations is specified, only generate routes for those operations
		for _, op := range resource.Operations {
			allowedOps[op] = true
		}
	} else {
		// If no @operations restriction, generate all standard routes
		for op := range standardRoutes {
			allowedOps[op] = true
		}
	}

	// Generate standard REST routes
	for opName, config := range standardRoutes {
		if !allowedOps[opName] {
			continue
		}

		route := RouteMetadata{
			Method:      config.method,
			Path:        config.path,
			Handler:     config.handler,
			Resource:    resource.Name,
			Operation:   config.operation,
			Middleware:  resource.Middleware,
			Description: config.description,
		}
		e.routes = append(e.routes, route)
	}

	// Generate nested resource routes for has_many relationships
	for _, rel := range resource.Relationships {
		if rel.Kind == ast.RelationshipHasMany {
			e.generateNestedRoutes(resource, rel)
		}
	}
}

// generateNestedRoutes generates nested routes for has_many relationships.
// Example: GET /posts/:post_id/comments
func (e *Extractor) generateNestedRoutes(parent *ast.ResourceNode, rel *ast.RelationshipNode) {
	parentPath := e.toPlural(strings.ToLower(parent.Name))
	childPath := e.toPlural(strings.ToLower(rel.Type))

	// Nested list route: GET /parents/:parent_id/children
	route := RouteMetadata{
		Method:      "GET",
		Path:        fmt.Sprintf("/%s/:id/%s", parentPath, childPath),
		Handler:     fmt.Sprintf("%s.%s.list", parent.Name, rel.Name),
		Resource:    parent.Name,
		Operation:   fmt.Sprintf("list_%s", rel.Name),
		Middleware:  parent.Middleware,
		Description: fmt.Sprintf("List all %s for a %s", childPath, parent.Name),
	}
	e.routes = append(e.routes, route)
}

// toPlural converts a singular resource name to plural form.
//
// This is a simple implementation that handles common English pluralization rules:
//   - Irregular plurals: person→people, child→children, man→men, woman→women
//   - Words ending in consonant+y: category→categories, story→stories
//   - Words ending in vowel+y: day→days, boy→boys
//   - Words ending in s/ss/sh/ch/x/z: class→classes, box→boxes, church→churches
//   - Default: add 's' (post→posts, user→users)
//
// Known Limitations:
//   - Does not handle all irregular plurals (e.g., tooth→teeth, mouse→mice)
//   - May not work correctly for words from other languages
//   - For production use with user-defined resource names, consider using a
//     comprehensive pluralization library like github.com/gertd/go-pluralize
//
// The implementation is intentionally simple to avoid external dependencies
// while handling the most common cases in typical API resource names.
func (e *Extractor) toPlural(singular string) string {
	if singular == "" {
		return ""
	}

	// Handle common irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"child":  "children",
		"man":    "men",
		"woman":  "women",
	}

	if plural, ok := irregulars[singular]; ok {
		return plural
	}

	// Handle words ending in 'y' preceded by a consonant
	if len(singular) >= 2 && singular[len(singular)-1] == 'y' {
		prevChar := singular[len(singular)-2]
		if !isVowel(prevChar) {
			return singular[:len(singular)-1] + "ies"
		}
	}

	// Handle words ending in 's', 'ss', 'sh', 'ch', 'x', 'z'
	if strings.HasSuffix(singular, "s") ||
		strings.HasSuffix(singular, "ss") ||
		strings.HasSuffix(singular, "sh") ||
		strings.HasSuffix(singular, "ch") ||
		strings.HasSuffix(singular, "x") ||
		strings.HasSuffix(singular, "z") {
		return singular + "es"
	}

	// Default: just add 's'
	return singular + "s"
}

// isVowel returns true if the character is a vowel
func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}
