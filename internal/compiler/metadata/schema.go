// Package metadata provides introspection data structures and generation
// for runtime queries about the application schema and patterns.
package metadata

import "encoding/json"

// Metadata represents the complete introspection metadata for a Conduit application
type Metadata struct {
	Version    string             `json:"version"`
	SourceHash string             `json:"source_hash"` // Hash of all source files for change detection
	Resources  []ResourceMetadata `json:"resources"`
	Patterns   []PatternMetadata  `json:"patterns"`
	Routes     []RouteMetadata    `json:"routes"`
}

// ResourceMetadata describes a resource and its components
type ResourceMetadata struct {
	Name          string                 `json:"name"`
	Documentation string                 `json:"documentation,omitempty"`
	FilePath      string                 `json:"file_path,omitempty"`      // Source file path
	Line          int                    `json:"line,omitempty"`           // Line number in source
	Fields        []FieldMetadata        `json:"fields"`
	Relationships []RelationshipMetadata `json:"relationships,omitempty"`
	Hooks         []HookMetadata         `json:"hooks,omitempty"`
	Validations   []ValidationMetadata   `json:"validations,omitempty"`
	Constraints   []ConstraintMetadata   `json:"constraints,omitempty"`
	Scopes        []ScopeMetadata        `json:"scopes,omitempty"`
	Computed      []ComputedMetadata     `json:"computed,omitempty"`
	Operations    []string               `json:"operations,omitempty"`
	Middleware    []string               `json:"middleware,omitempty"`
}

// FieldMetadata describes a field in a resource
type FieldMetadata struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Nullable    bool     `json:"nullable"`
	Constraints []string `json:"constraints,omitempty"`
	Default     string   `json:"default,omitempty"`
}

// RelationshipMetadata describes a relationship between resources
type RelationshipMetadata struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Kind       string `json:"kind"` // belongs_to, has_many, has_one, has_many_through
	ForeignKey string `json:"foreign_key,omitempty"`
	Through    string `json:"through,omitempty"`
	OnDelete   string `json:"on_delete,omitempty"`
	Nullable   bool   `json:"nullable"`
}

// HookMetadata describes a lifecycle hook
type HookMetadata struct {
	Timing         string   `json:"timing"`          // before, after
	Event          string   `json:"event"`           // create, update, delete, save
	HasTransaction bool     `json:"has_transaction"` // @transaction annotation
	HasAsync       bool     `json:"has_async"`       // @async annotation
	SourceCode     string   `json:"source_code,omitempty"` // Hook body as source code
	Line           int      `json:"line,omitempty"`  // Line number in source
	Middleware     []string `json:"middleware,omitempty"`
}

// ValidationMetadata describes a validation rule
type ValidationMetadata struct {
	Name      string `json:"name"`
	Condition string `json:"condition"` // Expression as string
	Error     string `json:"error"`
}

// ConstraintMetadata describes a constraint
type ConstraintMetadata struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments,omitempty"`
	On        []string `json:"on,omitempty"`        // create, update
	When      string   `json:"when,omitempty"`      // Condition as string
	Condition string   `json:"condition,omitempty"` // Constraint condition
	Error     string   `json:"error,omitempty"`
}

// ScopeMetadata describes a named scope
type ScopeMetadata struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments,omitempty"`
	Condition string   `json:"condition"` // Expression as string
}

// ComputedMetadata describes a computed field
type ComputedMetadata struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Body string `json:"body"` // Expression as string
}

// PatternMetadata describes a common pattern found in the codebase
type PatternMetadata struct {
	Name        string `json:"name"`
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
	Occurrences int    `json:"occurrences"`
}

// RouteMetadata describes an API route
type RouteMetadata struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Handler     string   `json:"handler"`
	Resource    string   `json:"resource"`
	Middleware  []string `json:"middleware,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ToJSON converts metadata to JSON string
func (m *Metadata) ToJSON() (string, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses metadata from JSON string
func FromJSON(data string) (*Metadata, error) {
	var m Metadata
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, err
	}
	return &m, nil
}
