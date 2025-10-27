// Package metadata provides structures for capturing introspection metadata
// about Conduit resources, patterns, and dependencies.
package metadata

import "time"

// Metadata is the top-level container for all introspection metadata.
// It captures complete information about compiled resources, routes,
// patterns, and dependencies for use by LLMs and developer tooling.
type Metadata struct {
	Version      string             `json:"version"`      // Schema version for evolution
	Generated    time.Time          `json:"generated"`    // Timestamp of metadata generation
	SourceHash   string             `json:"source_hash"`  // Hash of source files for cache invalidation
	Resources    []ResourceMetadata `json:"resources"`    // All resource definitions
	Routes       []RouteMetadata    `json:"routes"`       // Auto-generated HTTP routes
	Patterns     []PatternMetadata  `json:"patterns"`     // Discovered usage patterns
	Dependencies DependencyGraph    `json:"dependencies"` // Resource dependency graph
}

// ResourceMetadata captures complete information about a single Conduit resource.
type ResourceMetadata struct {
	Name           string                  `json:"name"`                      // Resource name (e.g., "Post", "User")
	Documentation  string                  `json:"documentation,omitempty"`   // Extracted doc comments
	FilePath       string                  `json:"file_path"`                 // Source file location
	Fields         []FieldMetadata         `json:"fields"`                    // All field definitions
	Relationships  []RelationshipMetadata  `json:"relationships"`             // All relationship definitions
	Hooks          []HookMetadata          `json:"hooks"`                     // All lifecycle hooks
	Validations    []ValidationMetadata    `json:"validations"`               // Field-level validations
	Constraints    []ConstraintMetadata    `json:"constraints"`               // Resource-level constraints
	Middleware     map[string][]string     `json:"middleware,omitempty"`      // Middleware per operation
	Scopes         []ScopeMetadata         `json:"scopes,omitempty"`          // Query scopes
	ComputedFields []ComputedFieldMetadata `json:"computed_fields,omitempty"` // Computed fields
}

// FieldMetadata captures metadata about a single field in a resource.
type FieldMetadata struct {
	Name          string   `json:"name"`                    // Field name
	Type          string   `json:"type"`                    // Field type (e.g., "string", "uuid", "integer")
	Nullable      bool     `json:"nullable"`                // Whether field accepts null values (type?)
	Required      bool     `json:"required"`                // Whether field is required (type!)
	DefaultValue  string   `json:"default_value,omitempty"` // Default value if specified
	Constraints   []string `json:"constraints,omitempty"`   // Applied constraints (e.g., "@min(5)", "@max(200)")
	Documentation string   `json:"documentation,omitempty"` // Field-level doc comments
	Tags          []string `json:"tags,omitempty"`          // Additional metadata tags
}

// RelationshipMetadata captures metadata about relationships between resources.
type RelationshipMetadata struct {
	Name           string `json:"name"`                    // Relationship field name
	Type           string `json:"type"`                    // "belongs_to", "has_many", "has_many_through"
	TargetResource string `json:"target_resource"`         // Target resource name
	ForeignKey     string `json:"foreign_key,omitempty"`   // Foreign key column name
	ThroughTable   string `json:"through_table,omitempty"` // Join table for has_many_through
	OnDelete       string `json:"on_delete,omitempty"`     // Delete behavior (cascade, restrict, set_null)
	OnUpdate       string `json:"on_update,omitempty"`     // Update behavior
}

// HookMetadata captures metadata about lifecycle hooks.
type HookMetadata struct {
	Type        string `json:"type"`                  // Hook type (e.g., "before_create", "after_update")
	Transaction bool   `json:"transaction"`           // Whether hook runs in transaction
	Async       bool   `json:"async"`                 // Whether hook runs asynchronously
	SourceCode  string `json:"source_code,omitempty"` // Hook implementation source
	LineNumber  int    `json:"line_number"`           // Source file line number
}

// ValidationMetadata captures field-level validation rules.
type ValidationMetadata struct {
	Field      string `json:"field"`             // Field name
	Type       string `json:"type"`              // Validation type (e.g., "min", "max", "pattern")
	Value      string `json:"value,omitempty"`   // Validation parameter
	Message    string `json:"message,omitempty"` // Custom error message
	LineNumber int    `json:"line_number"`       // Source line number
}

// ConstraintMetadata captures resource-level constraints.
type ConstraintMetadata struct {
	Name       string   `json:"name"`           // Constraint name
	Operations []string `json:"operations"`     // Operations to validate (create, update, delete)
	Condition  string   `json:"condition"`      // Constraint condition expression
	When       string   `json:"when,omitempty"` // Optional precondition
	Error      string   `json:"error"`          // Error message
	LineNumber int      `json:"line_number"`    // Source line number
}

// ScopeMetadata captures query scope definitions.
type ScopeMetadata struct {
	Name       string   `json:"name"`                 // Scope name
	Query      string   `json:"query"`                // Scope query expression
	Parameters []string `json:"parameters,omitempty"` // Scope parameters
	LineNumber int      `json:"line_number"`          // Source line number
}

// ComputedFieldMetadata captures computed field definitions.
type ComputedFieldMetadata struct {
	Name       string `json:"name"`        // Computed field name
	Type       string `json:"type"`        // Return type
	Expression string `json:"expression"`  // Computation expression
	LineNumber int    `json:"line_number"` // Source line number
}

// RouteMetadata captures information about auto-generated HTTP routes.
type RouteMetadata struct {
	Method       string   `json:"method"`                  // HTTP method (GET, POST, PUT, DELETE)
	Path         string   `json:"path"`                    // URL path pattern
	Handler      string   `json:"handler"`                 // Handler function name
	Resource     string   `json:"resource"`                // Associated resource name
	Operation    string   `json:"operation"`               // CRUD operation (list, show, create, update, delete)
	Middleware   []string `json:"middleware,omitempty"`    // Applied middleware
	RequestBody  string   `json:"request_body,omitempty"`  // Expected request body type
	ResponseBody string   `json:"response_body,omitempty"` // Response body type
}

// PatternMetadata captures discovered usage patterns for LLM learning.
type PatternMetadata struct {
	ID          string           `json:"id"`          // Unique pattern identifier
	Name        string           `json:"name"`        // Pattern name
	Category    string           `json:"category"`    // Pattern category (hook, validation, etc.)
	Description string           `json:"description"` // Human-readable description
	Template    string           `json:"template"`    // Code template for pattern
	Examples    []PatternExample `json:"examples"`    // Real usage examples from codebase
	Frequency   int              `json:"frequency"`   // Number of times pattern appears
	Confidence  float64          `json:"confidence"`  // Confidence score (0.0-1.0)
}

// PatternExample captures a single example of a pattern in use.
type PatternExample struct {
	Resource   string `json:"resource"`              // Resource where pattern is used
	FilePath   string `json:"file_path"`             // Source file path
	LineNumber int    `json:"line_number,omitempty"` // Line number in source
	Code       string `json:"code"`                  // Example code snippet
}

// DependencyGraph captures the dependency relationships between resources.
type DependencyGraph struct {
	Nodes map[string]*DependencyNode `json:"nodes"` // All nodes indexed by ID
	Edges []DependencyEdge           `json:"edges"` // All dependency edges

	// Pre-computed adjacency lists for fast traversal (not serialized)
	outgoingEdges map[string][]DependencyEdge `json:"-"` // from -> []edges
	incomingEdges map[string][]DependencyEdge `json:"-"` // to -> []edges
}

// DependencyNode represents a single node in the dependency graph.
type DependencyNode struct {
	ID       string `json:"id"`        // Unique node identifier
	Type     string `json:"type"`      // Node type (resource, function, middleware)
	Name     string `json:"name"`      // Node name
	FilePath string `json:"file_path"` // Source file location
}

// DependencyEdge represents a dependency relationship between two nodes.
type DependencyEdge struct {
	From         string `json:"from"`         // Source node ID
	To           string `json:"to"`           // Target node ID
	Relationship string `json:"relationship"` // Relationship type (uses, calls, belongs_to)
	Weight       int    `json:"weight"`       // Relationship weight/importance
}
