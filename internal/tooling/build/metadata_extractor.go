package build

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/runtime/metadata"
)

// MetadataExtractor extracts introspection metadata from compiled AST nodes.
type MetadataExtractor struct {
	// Track file paths for each resource
	resourceFiles map[string]string
}

// NewMetadataExtractor creates a new metadata extractor.
func NewMetadataExtractor() *MetadataExtractor {
	return &MetadataExtractor{
		resourceFiles: make(map[string]string),
	}
}

// Extract generates metadata from compiled files.
func (e *MetadataExtractor) Extract(compiled []*CompiledFile) (*metadata.Metadata, error) {
	// Collect all resources
	var allResources []*ast.ResourceNode
	for _, cf := range compiled {
		for _, res := range cf.Program.Resources {
			e.resourceFiles[res.Name] = cf.Path
			allResources = append(allResources, res)
		}
	}

	// Sort resources by name for consistent output
	sort.Slice(allResources, func(i, j int) bool {
		return allResources[i].Name < allResources[j].Name
	})

	// Extract metadata components
	resources := e.extractResources(allResources)
	routes := e.extractRoutes(allResources)
	patterns := e.extractPatterns(allResources)
	dependencyGraph := e.extractDependencyGraph(allResources)

	// Compute source hash for cache invalidation
	sourceHash := e.computeSourceHash(compiled)

	meta := &metadata.Metadata{
		Version:      "1.0",
		Generated:    time.Now(),
		SourceHash:   sourceHash,
		Resources:    resources,
		Routes:       routes,
		Patterns:     patterns,
		Dependencies: dependencyGraph,
	}

	return meta, nil
}

// extractResources extracts resource metadata from AST nodes.
func (e *MetadataExtractor) extractResources(resources []*ast.ResourceNode) []metadata.ResourceMetadata {
	result := make([]metadata.ResourceMetadata, 0, len(resources))

	for _, res := range resources {
		resMeta := metadata.ResourceMetadata{
			Name:          res.Name,
			Documentation: res.Documentation,
			FilePath:      e.resourceFiles[res.Name],
			Fields:        e.extractFields(res.Fields),
			Relationships: e.extractRelationships(res.Relationships),
			Hooks:         e.extractHooks(res.Hooks),
			Validations:   e.extractValidations(res.Validations),
			Constraints:   e.extractConstraints(res.Constraints),
			Middleware:    e.extractMiddleware(res),
			Scopes:        e.extractScopes(res.Scopes),
			ComputedFields: e.extractComputedFields(res.Computed),
		}

		result = append(result, resMeta)
	}

	return result
}

// extractFields extracts field metadata from AST field nodes.
func (e *MetadataExtractor) extractFields(fields []*ast.FieldNode) []metadata.FieldMetadata {
	result := make([]metadata.FieldMetadata, 0, len(fields))

	for _, field := range fields {
		fieldMeta := metadata.FieldMetadata{
			Name:     field.Name,
			Type:     e.formatType(field.Type),
			Nullable: field.Nullable,
			Required: !field.Nullable && field.Default == nil,
		}

		// Extract default value
		if field.Default != nil {
			fieldMeta.DefaultValue = e.formatExpr(field.Default)
		}

		// Extract constraints
		if len(field.Constraints) > 0 {
			constraints := make([]string, 0, len(field.Constraints))
			for _, c := range field.Constraints {
				constraints = append(constraints, e.formatConstraintName(c))
			}
			fieldMeta.Constraints = constraints
		}

		result = append(result, fieldMeta)
	}

	return result
}

// extractRelationships extracts relationship metadata from AST relationship nodes.
func (e *MetadataExtractor) extractRelationships(relationships []*ast.RelationshipNode) []metadata.RelationshipMetadata {
	result := make([]metadata.RelationshipMetadata, 0, len(relationships))

	for _, rel := range relationships {
		relMeta := metadata.RelationshipMetadata{
			Name:           rel.Name,
			Type:           e.formatRelationshipKind(rel.Kind),
			TargetResource: rel.Type,
			ForeignKey:     rel.ForeignKey,
			ThroughTable:   rel.Through,
			OnDelete:       rel.OnDelete,
		}

		result = append(result, relMeta)
	}

	return result
}

// extractHooks extracts hook metadata from AST hook nodes.
func (e *MetadataExtractor) extractHooks(hooks []*ast.HookNode) []metadata.HookMetadata {
	result := make([]metadata.HookMetadata, 0, len(hooks))

	for _, hook := range hooks {
		hookType := hook.Timing + "_" + hook.Event

		hookMeta := metadata.HookMetadata{
			Type:        hookType,
			Transaction: hook.IsTransaction,
			Async:       hook.IsAsync,
			LineNumber:  hook.Loc.Line,
		}

		// Include source code for verbose introspection
		if len(hook.Body) > 0 {
			hookMeta.SourceCode = e.formatHookBody(hook.Body)
		}

		result = append(result, hookMeta)
	}

	return result
}

// extractValidations extracts validation metadata from AST validation nodes.
func (e *MetadataExtractor) extractValidations(validations []*ast.ValidationNode) []metadata.ValidationMetadata {
	result := make([]metadata.ValidationMetadata, 0, len(validations))

	for _, val := range validations {
		valMeta := metadata.ValidationMetadata{
			Field:      val.Name,
			Type:       "custom",
			Message:    val.Error,
			LineNumber: val.Loc.Line,
		}

		// Try to extract validation type from condition
		if val.Condition != nil {
			valMeta.Type = e.inferValidationType(val.Condition)
			valMeta.Value = e.formatExpr(val.Condition)
		}

		result = append(result, valMeta)
	}

	return result
}

// extractConstraints extracts constraint metadata from AST constraint nodes.
func (e *MetadataExtractor) extractConstraints(constraints []*ast.ConstraintNode) []metadata.ConstraintMetadata {
	result := make([]metadata.ConstraintMetadata, 0, len(constraints))

	for _, constraint := range constraints {
		conMeta := metadata.ConstraintMetadata{
			Name:       constraint.Name,
			Operations: constraint.On,
			Condition:  e.formatExpr(constraint.Condition),
			Error:      constraint.Error,
			LineNumber: constraint.Loc.Line,
		}

		if constraint.When != nil {
			conMeta.When = e.formatExpr(constraint.When)
		}

		// Default to all operations if not specified
		if len(conMeta.Operations) == 0 {
			conMeta.Operations = []string{"create", "update"}
		}

		result = append(result, conMeta)
	}

	return result
}

// extractMiddleware extracts middleware configuration from a resource.
func (e *MetadataExtractor) extractMiddleware(res *ast.ResourceNode) map[string][]string {
	middleware := make(map[string][]string)

	// Resource-level middleware applies to all operations
	if len(res.Middleware) > 0 {
		for _, op := range []string{"list", "show", "create", "update", "delete"} {
			middleware[op] = res.Middleware
		}
	}

	// TODO: Extract operation-specific middleware when AST supports it

	return middleware
}

// extractScopes extracts scope metadata from AST scope nodes.
func (e *MetadataExtractor) extractScopes(scopes []*ast.ScopeNode) []metadata.ScopeMetadata {
	result := make([]metadata.ScopeMetadata, 0, len(scopes))

	for _, scope := range scopes {
		scopeMeta := metadata.ScopeMetadata{
			Name:       scope.Name,
			Query:      e.formatExpr(scope.Condition),
			LineNumber: scope.Loc.Line,
		}

		// Extract parameter names
		if len(scope.Arguments) > 0 {
			params := make([]string, 0, len(scope.Arguments))
			for _, arg := range scope.Arguments {
				params = append(params, arg.Name)
			}
			scopeMeta.Parameters = params
		}

		result = append(result, scopeMeta)
	}

	return result
}

// extractComputedFields extracts computed field metadata from AST computed nodes.
func (e *MetadataExtractor) extractComputedFields(computed []*ast.ComputedNode) []metadata.ComputedFieldMetadata {
	result := make([]metadata.ComputedFieldMetadata, 0, len(computed))

	for _, comp := range computed {
		compMeta := metadata.ComputedFieldMetadata{
			Name:       comp.Name,
			Type:       e.formatType(comp.Type),
			Expression: e.formatExpr(comp.Body),
			LineNumber: comp.Loc.Line,
		}

		result = append(result, compMeta)
	}

	return result
}

// extractRoutes generates route metadata for standard CRUD operations.
func (e *MetadataExtractor) extractRoutes(resources []*ast.ResourceNode) []metadata.RouteMetadata {
	routes := make([]metadata.RouteMetadata, 0)

	for _, res := range resources {
		resourceName := res.Name
		resourcePath := e.toSnakeCase(resourceName)

		// Determine which operations are allowed
		allowedOps := map[string]bool{
			"list":   true,
			"show":   true,
			"create": true,
			"update": true,
			"delete": true,
		}

		// If Operations is specified, restrict to those
		if len(res.Operations) > 0 {
			// Reset to false and only enable specified operations
			for op := range allowedOps {
				allowedOps[op] = false
			}
			for _, op := range res.Operations {
				allowedOps[op] = true
			}
		}

		// LIST: GET /resources
		if allowedOps["list"] {
			routes = append(routes, metadata.RouteMetadata{
				Method:       "GET",
				Path:         "/" + resourcePath,
				Handler:      "List" + resourceName,
				Resource:     resourceName,
				Operation:    "list",
				Middleware:   e.getOperationMiddleware(res, "list"),
				ResponseBody: "[]" + resourceName,
			})
		}

		// SHOW: GET /resources/:id
		if allowedOps["show"] {
			routes = append(routes, metadata.RouteMetadata{
				Method:       "GET",
				Path:         "/" + resourcePath + "/:id",
				Handler:      "Show" + resourceName,
				Resource:     resourceName,
				Operation:    "show",
				Middleware:   e.getOperationMiddleware(res, "show"),
				ResponseBody: resourceName,
			})
		}

		// CREATE: POST /resources
		if allowedOps["create"] {
			routes = append(routes, metadata.RouteMetadata{
				Method:       "POST",
				Path:         "/" + resourcePath,
				Handler:      "Create" + resourceName,
				Resource:     resourceName,
				Operation:    "create",
				Middleware:   e.getOperationMiddleware(res, "create"),
				RequestBody:  resourceName + "Input",
				ResponseBody: resourceName,
			})
		}

		// UPDATE: PUT /resources/:id
		if allowedOps["update"] {
			routes = append(routes, metadata.RouteMetadata{
				Method:       "PUT",
				Path:         "/" + resourcePath + "/:id",
				Handler:      "Update" + resourceName,
				Resource:     resourceName,
				Operation:    "update",
				Middleware:   e.getOperationMiddleware(res, "update"),
				RequestBody:  resourceName + "Input",
				ResponseBody: resourceName,
			})
		}

		// DELETE: DELETE /resources/:id
		if allowedOps["delete"] {
			routes = append(routes, metadata.RouteMetadata{
				Method:       "DELETE",
				Path:         "/" + resourcePath + "/:id",
				Handler:      "Delete" + resourceName,
				Resource:     resourceName,
				Operation:    "delete",
				Middleware:   e.getOperationMiddleware(res, "delete"),
			})
		}
	}

	return routes
}

// getOperationMiddleware returns middleware for a specific operation.
func (e *MetadataExtractor) getOperationMiddleware(res *ast.ResourceNode, operation string) []string {
	// For now, return resource-level middleware
	// TODO: Add support for operation-specific middleware
	return res.Middleware
}

// extractPatterns discovers common patterns in the codebase.
func (e *MetadataExtractor) extractPatterns(resources []*ast.ResourceNode) []metadata.PatternMetadata {
	patterns := make([]metadata.PatternMetadata, 0)

	// Pattern discovery categories
	hookPatterns := e.discoverHookPatterns(resources)
	validationPatterns := e.discoverValidationPatterns(resources)
	middlewarePatterns := e.discoverMiddlewarePatterns(resources)

	patterns = append(patterns, hookPatterns...)
	patterns = append(patterns, validationPatterns...)
	patterns = append(patterns, middlewarePatterns...)

	return patterns
}

// discoverHookPatterns discovers common hook patterns.
func (e *MetadataExtractor) discoverHookPatterns(resources []*ast.ResourceNode) []metadata.PatternMetadata {
	hookFrequency := make(map[string][]metadata.PatternExample)

	for _, res := range resources {
		for _, hook := range res.Hooks {
			hookType := hook.Timing + "_" + hook.Event

			// Create pattern key with modifiers
			patternKey := hookType
			if hook.IsAsync {
				patternKey += "_async"
			}
			if hook.IsTransaction {
				patternKey += "_transaction"
			}

			example := metadata.PatternExample{
				Resource:   res.Name,
				FilePath:   e.resourceFiles[res.Name],
				LineNumber: hook.Loc.Line,
				Code:       e.formatHookBody(hook.Body),
			}

			hookFrequency[patternKey] = append(hookFrequency[patternKey], example)
		}
	}

	// Convert to pattern metadata
	patterns := make([]metadata.PatternMetadata, 0)
	for patternKey, examples := range hookFrequency {
		pattern := metadata.PatternMetadata{
			ID:          "hook_" + patternKey,
			Name:        e.formatPatternName(patternKey),
			Category:    "hook",
			Description: e.generateHookDescription(patternKey),
			Template:    e.generateHookTemplate(patternKey),
			Examples:    examples,
			Frequency:   len(examples),
			Confidence:  e.calculateConfidence(len(examples), len(resources)),
		}
		patterns = append(patterns, pattern)
	}

	return patterns
}

// discoverValidationPatterns discovers common validation patterns.
func (e *MetadataExtractor) discoverValidationPatterns(resources []*ast.ResourceNode) []metadata.PatternMetadata {
	// Simplified implementation for MVP
	return []metadata.PatternMetadata{}
}

// discoverMiddlewarePatterns discovers common middleware patterns.
func (e *MetadataExtractor) discoverMiddlewarePatterns(resources []*ast.ResourceNode) []metadata.PatternMetadata {
	middlewareFrequency := make(map[string][]metadata.PatternExample)

	for _, res := range resources {
		for _, mw := range res.Middleware {
			example := metadata.PatternExample{
				Resource:   res.Name,
				FilePath:   e.resourceFiles[res.Name],
				LineNumber: res.Loc.Line,
			}

			middlewareFrequency[mw] = append(middlewareFrequency[mw], example)
		}
	}

	// Convert to pattern metadata
	patterns := make([]metadata.PatternMetadata, 0)
	for mw, examples := range middlewareFrequency {
		pattern := metadata.PatternMetadata{
			ID:          "middleware_" + mw,
			Name:        mw + " middleware",
			Category:    "middleware",
			Description: "Common middleware: " + mw,
			Template:    "@middleware(" + mw + ")",
			Examples:    examples,
			Frequency:   len(examples),
			Confidence:  e.calculateConfidence(len(examples), len(resources)),
		}
		patterns = append(patterns, pattern)
	}

	return patterns
}

// extractDependencyGraph builds the dependency graph from resources.
func (e *MetadataExtractor) extractDependencyGraph(resources []*ast.ResourceNode) metadata.DependencyGraph {
	graph := metadata.DependencyGraph{
		Nodes: make(map[string]*metadata.DependencyNode),
		Edges: []metadata.DependencyEdge{},
	}

	// Create nodes for all resources
	for _, res := range resources {
		nodeID := "resource:" + res.Name
		graph.Nodes[nodeID] = &metadata.DependencyNode{
			ID:       nodeID,
			Type:     "resource",
			Name:     res.Name,
			FilePath: e.resourceFiles[res.Name],
		}
	}

	// Create edges for relationships
	for _, res := range resources {
		fromID := "resource:" + res.Name

		for _, rel := range res.Relationships {
			toID := "resource:" + rel.Type

			edge := metadata.DependencyEdge{
				From:         fromID,
				To:           toID,
				Relationship: e.formatRelationshipKind(rel.Kind),
				Weight:       1,
			}

			graph.Edges = append(graph.Edges, edge)
		}
	}

	return graph
}

// Helper methods

func (e *MetadataExtractor) formatType(t *ast.TypeNode) string {
	if t == nil {
		return "unknown"
	}

	base := t.Name
	if t.Nullable {
		return base + "?"
	}
	return base + "!"
}

func (e *MetadataExtractor) formatExpr(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}
	// Simplified expression formatting
	return fmt.Sprintf("%v", expr)
}

func (e *MetadataExtractor) formatHookBody(body []ast.StmtNode) string {
	if len(body) == 0 {
		return ""
	}
	// Simplified hook body formatting
	return fmt.Sprintf("// %d statements", len(body))
}

func (e *MetadataExtractor) formatConstraintName(c *ast.ConstraintNode) string {
	if len(c.Arguments) == 0 {
		return "@" + c.Name
	}
	return fmt.Sprintf("@%s(%v)", c.Name, c.Arguments[0])
}

func (e *MetadataExtractor) formatRelationshipKind(kind ast.RelationshipKind) string {
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

func (e *MetadataExtractor) inferValidationType(expr ast.ExprNode) string {
	// Simplified validation type inference
	return "custom"
}

func (e *MetadataExtractor) formatPatternName(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func (e *MetadataExtractor) generateHookDescription(patternKey string) string {
	parts := strings.Split(patternKey, "_")
	timing := parts[0]
	event := parts[1]

	desc := fmt.Sprintf("Lifecycle hook that runs %s %s operation", timing, event)

	if strings.Contains(patternKey, "async") {
		desc += " (asynchronous)"
	}
	if strings.Contains(patternKey, "transaction") {
		desc += " (in transaction)"
	}

	return desc
}

func (e *MetadataExtractor) generateHookTemplate(patternKey string) string {
	parts := strings.Split(patternKey, "_")
	timing := parts[0]
	event := parts[1]

	template := fmt.Sprintf("@%s_%s", timing, event)

	if strings.Contains(patternKey, "async") {
		template += " @async"
	}
	if strings.Contains(patternKey, "transaction") {
		template += " @transaction"
	}

	template += " {\n  // hook logic\n}"

	return template
}

func (e *MetadataExtractor) calculateConfidence(frequency, totalResources int) float64 {
	if totalResources == 0 {
		return 0.0
	}
	// Simple confidence calculation based on frequency
	coverage := float64(frequency) / float64(totalResources)
	if coverage >= 0.5 {
		return 1.0
	} else if coverage >= 0.25 {
		return 0.8
	} else if coverage >= 0.1 {
		return 0.6
	}
	return 0.4
}

func (e *MetadataExtractor) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func (e *MetadataExtractor) computeSourceHash(compiled []*CompiledFile) string {
	hasher := sha256.New()

	// Sort files by path for deterministic hash
	sortedFiles := make([]*CompiledFile, len(compiled))
	copy(sortedFiles, compiled)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	for _, cf := range sortedFiles {
		hasher.Write([]byte(cf.Path))
		hasher.Write([]byte(cf.Hash))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
