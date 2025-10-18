// Package schema provides type definitions and validation for Conduit's ORM schema system.
// It defines the core data structures for representing database schemas with explicit nullability,
// type safety, and comprehensive constraint validation.
package schema

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// PrimitiveType represents the built-in primitive types in Conduit
type PrimitiveType int

const (
	// Text types
	TypeString PrimitiveType = iota
	TypeText
	TypeMarkdown

	// Numeric types
	TypeInt
	TypeBigInt
	TypeFloat
	TypeDecimal

	// Boolean
	TypeBool

	// Time types
	TypeTimestamp
	TypeDate
	TypeTime

	// Unique identifiers
	TypeUUID
	TypeULID

	// Validated types
	TypeEmail
	TypeURL
	TypePhone

	// JSON types
	TypeJSON
	TypeJSONB

	// Enum
	TypeEnum
)

// String returns the string representation of the primitive type
func (p PrimitiveType) String() string {
	switch p {
	case TypeString:
		return "string"
	case TypeText:
		return "text"
	case TypeMarkdown:
		return "markdown"
	case TypeInt:
		return "int"
	case TypeBigInt:
		return "bigint"
	case TypeFloat:
		return "float"
	case TypeDecimal:
		return "decimal"
	case TypeBool:
		return "bool"
	case TypeTimestamp:
		return "timestamp"
	case TypeDate:
		return "date"
	case TypeTime:
		return "time"
	case TypeUUID:
		return "uuid"
	case TypeULID:
		return "ulid"
	case TypeEmail:
		return "email"
	case TypeURL:
		return "url"
	case TypePhone:
		return "phone"
	case TypeJSON:
		return "json"
	case TypeJSONB:
		return "jsonb"
	case TypeEnum:
		return "enum"
	default:
		return "unknown"
	}
}

// ParsePrimitiveType converts a string to a PrimitiveType
func ParsePrimitiveType(s string) (PrimitiveType, error) {
	switch s {
	case "string":
		return TypeString, nil
	case "text":
		return TypeText, nil
	case "markdown":
		return TypeMarkdown, nil
	case "int":
		return TypeInt, nil
	case "bigint":
		return TypeBigInt, nil
	case "float":
		return TypeFloat, nil
	case "decimal":
		return TypeDecimal, nil
	case "bool":
		return TypeBool, nil
	case "timestamp":
		return TypeTimestamp, nil
	case "date":
		return TypeDate, nil
	case "time":
		return TypeTime, nil
	case "uuid":
		return TypeUUID, nil
	case "ulid":
		return TypeULID, nil
	case "email":
		return TypeEmail, nil
	case "url":
		return TypeURL, nil
	case "phone":
		return TypePhone, nil
	case "json":
		return TypeJSON, nil
	case "jsonb":
		return TypeJSONB, nil
	case "enum":
		return TypeEnum, nil
	default:
		return 0, fmt.Errorf("unknown primitive type: %s", s)
	}
}

// TypeSpec represents a complete type specification with nullability and constraints
type TypeSpec struct {
	BaseType       PrimitiveType // The base primitive type
	Nullable       bool          // ! = false, ? = true
	NullabilitySet bool          // Track if nullability was explicitly set
	Constraints    []Constraint  // Field-level constraints (@min, @max, etc.)
	Default        interface{}   // Default value

	// Complex types
	ArrayElement *TypeSpec            // For array<T>
	HashKey      *TypeSpec            // For hash<K,V>
	HashValue    *TypeSpec            // For hash<K,V>
	StructFields map[string]*TypeSpec // For inline structs
	EnumValues   []string             // For enum types

	// Type parameters (e.g., string(50), decimal(10,2))
	Length    *int // For string(N)
	Precision *int // For decimal(P,S)
	Scale     *int // For decimal(P,S)
}

// String returns a string representation of the TypeSpec
func (t *TypeSpec) String() string {
	var s string

	switch {
	case t.ArrayElement != nil:
		s = fmt.Sprintf("array<%s>", t.ArrayElement.String())
	case t.HashKey != nil && t.HashValue != nil:
		s = fmt.Sprintf("hash<%s, %s>", t.HashKey.String(), t.HashValue.String())
	case len(t.StructFields) > 0:
		s = "struct"
	case len(t.EnumValues) > 0:
		s = fmt.Sprintf("enum%v", t.EnumValues)
	default:
		s = t.BaseType.String()
		if t.Length != nil {
			s = fmt.Sprintf("%s(%d)", s, *t.Length)
		}
		if t.Precision != nil && t.Scale != nil {
			s = fmt.Sprintf("%s(%d,%d)", s, *t.Precision, *t.Scale)
		}
	}

	if t.Nullable {
		s += "?"
	} else {
		s += "!"
	}

	return s
}

// IsNumeric returns true if the type is a numeric type
func (t *TypeSpec) IsNumeric() bool {
	return t.BaseType == TypeInt ||
		t.BaseType == TypeBigInt ||
		t.BaseType == TypeFloat ||
		t.BaseType == TypeDecimal
}

// IsText returns true if the type is a text type
func (t *TypeSpec) IsText() bool {
	return t.BaseType == TypeString ||
		t.BaseType == TypeText ||
		t.BaseType == TypeMarkdown
}

// IsValidated returns true if the type has built-in validation
func (t *TypeSpec) IsValidated() bool {
	return t.BaseType == TypeEmail ||
		t.BaseType == TypeURL ||
		t.BaseType == TypePhone
}

// ConstraintType represents the type of constraint
type ConstraintType int

const (
	ConstraintMin ConstraintType = iota
	ConstraintMax
	ConstraintPattern
	ConstraintUnique
	ConstraintIndex
	ConstraintPrimary
	ConstraintAuto
	ConstraintAutoUpdate
	ConstraintDefault
)

// String returns the string representation of the constraint type
func (c ConstraintType) String() string {
	switch c {
	case ConstraintMin:
		return "min"
	case ConstraintMax:
		return "max"
	case ConstraintPattern:
		return "pattern"
	case ConstraintUnique:
		return "unique"
	case ConstraintIndex:
		return "index"
	case ConstraintPrimary:
		return "primary"
	case ConstraintAuto:
		return "auto"
	case ConstraintAutoUpdate:
		return "auto_update"
	case ConstraintDefault:
		return "default"
	default:
		return "unknown"
	}
}

// Constraint represents a field constraint
type Constraint struct {
	Type         ConstraintType
	Value        interface{} // Constraint value (e.g., min value, max value, pattern)
	ErrorMessage string      // Custom error message
	Location     ast.SourceLocation
}

// Field represents a field in a resource schema
type Field struct {
	Name        string
	Type        *TypeSpec
	Constraints []Constraint
	Annotations []Annotation

	// For nested structs
	IsNested     bool
	NestedFields map[string]*Field

	// Source location for error reporting
	Location ast.SourceLocation
}

// Annotation represents field annotations like @primary, @auto, @unique
type Annotation struct {
	Name string
	Args []interface{}
}

// RelationType represents the type of relationship
type RelationType int

const (
	RelationshipBelongsTo RelationType = iota
	RelationshipHasMany
	RelationshipHasManyThrough
	RelationshipHasOne
)

// String returns the string representation of the relationship type
func (r RelationType) String() string {
	switch r {
	case RelationshipBelongsTo:
		return "belongs_to"
	case RelationshipHasMany:
		return "has_many"
	case RelationshipHasManyThrough:
		return "has_many_through"
	case RelationshipHasOne:
		return "has_one"
	default:
		return "unknown"
	}
}

// CascadeAction represents cascade actions for foreign keys
type CascadeAction int

const (
	CascadeRestrict CascadeAction = iota
	CascadeCascade
	CascadeSetNull
	CascadeNoAction
)

// String returns the string representation of the cascade action
func (c CascadeAction) String() string {
	switch c {
	case CascadeRestrict:
		return "restrict"
	case CascadeCascade:
		return "cascade"
	case CascadeSetNull:
		return "set_null"
	case CascadeNoAction:
		return "no_action"
	default:
		return "unknown"
	}
}

// ParseCascadeAction converts a string to a CascadeAction
func ParseCascadeAction(s string) (CascadeAction, error) {
	switch s {
	case "restrict":
		return CascadeRestrict, nil
	case "cascade":
		return CascadeCascade, nil
	case "set_null":
		return CascadeSetNull, nil
	case "no_action":
		return CascadeNoAction, nil
	default:
		return 0, fmt.Errorf("unknown cascade action: %s", s)
	}
}

// Relationship represents a relationship between resources
type Relationship struct {
	Type           RelationType
	TargetResource string
	FieldName      string
	Nullable       bool

	// Foreign key configuration
	ForeignKey string
	OnDelete   CascadeAction
	OnUpdate   CascadeAction

	// For has_many
	OrderBy string

	// For has_many_through
	ThroughResource string
	JoinTable       string
	AssociationKey  string

	Location ast.SourceLocation
}

// HookType represents the type of lifecycle hook
type HookType int

const (
	BeforeCreate HookType = iota
	BeforeUpdate
	BeforeDelete
	BeforeSave
	AfterCreate
	AfterUpdate
	AfterDelete
	AfterSave
)

// String returns the string representation of the hook type
func (h HookType) String() string {
	switch h {
	case BeforeCreate:
		return "before_create"
	case BeforeUpdate:
		return "before_update"
	case BeforeDelete:
		return "before_delete"
	case BeforeSave:
		return "before_save"
	case AfterCreate:
		return "after_create"
	case AfterUpdate:
		return "after_update"
	case AfterDelete:
		return "after_delete"
	case AfterSave:
		return "after_save"
	default:
		return "unknown"
	}
}

// Hook represents a lifecycle hook
type Hook struct {
	Type        HookType
	Transaction bool // @transaction annotation
	Async       bool // @async block present
	Body        []ast.StmtNode
	Location    ast.SourceLocation
}

// Validator represents a procedural validation block
type Validator struct {
	Name      string
	Code      []ast.StmtNode
	Location  ast.SourceLocation
}

// ConstraintBlock represents a declarative constraint
type ConstraintBlock struct {
	Name      string
	On        []string // Events this applies to (create, update)
	When      ast.ExprNode
	Condition ast.ExprNode
	Error     string
	Location  ast.SourceLocation
}

// Invariant represents a runtime invariant
type Invariant struct {
	Name      string
	Condition ast.ExprNode
	Error     string
	Location  ast.SourceLocation
}

// Scope represents a named query scope
type Scope struct {
	Name      string
	Arguments []*ScopeArgument
	Where     map[string]interface{}
	OrderBy   string
	Limit     *int
	Offset    *int
	Location  ast.SourceLocation
}

// ScopeArgument represents an argument to a scope
type ScopeArgument struct {
	Name     string
	Type     *TypeSpec
	Default  interface{}
	Location ast.SourceLocation
}

// ComputedField represents a computed field
type ComputedField struct {
	Name     string
	Type     *TypeSpec
	Body     ast.ExprNode
	Location ast.SourceLocation
}

// ResourceSchema represents the complete schema for a resource
type ResourceSchema struct {
	Name          string
	Documentation string
	FilePath      string

	Fields        map[string]*Field
	Relationships map[string]*Relationship

	// Lifecycle hooks
	Hooks map[HookType][]*Hook

	// Validations
	Validators        []*Validator
	ConstraintBlocks  []*ConstraintBlock
	Invariants        []*Invariant

	// Query scopes
	Scopes map[string]*Scope

	// Computed fields
	Computed map[string]*ComputedField

	// Middleware
	Middleware map[string][]string // operation -> middleware list

	// Metadata
	TableName string
	Location  ast.SourceLocation
}

// NewResourceSchema creates a new ResourceSchema
func NewResourceSchema(name string) *ResourceSchema {
	return &ResourceSchema{
		Name:             name,
		Fields:           make(map[string]*Field),
		Relationships:    make(map[string]*Relationship),
		Hooks:            make(map[HookType][]*Hook),
		Validators:       make([]*Validator, 0),
		ConstraintBlocks: make([]*ConstraintBlock, 0),
		Invariants:       make([]*Invariant, 0),
		Scopes:           make(map[string]*Scope),
		Computed:         make(map[string]*ComputedField),
		Middleware:       make(map[string][]string),
		TableName:        toSnakeCase(name),
	}
}

// GetPrimaryKey returns the primary key field
func (r *ResourceSchema) GetPrimaryKey() (*Field, error) {
	for _, field := range r.Fields {
		for _, annotation := range field.Annotations {
			if annotation.Name == "primary" {
				return field, nil
			}
		}
	}
	return nil, fmt.Errorf("resource %s has no primary key", r.Name)
}

// HasField returns true if the resource has a field with the given name
func (r *ResourceSchema) HasField(name string) bool {
	_, exists := r.Fields[name]
	return exists
}

// HasRelationship returns true if the resource has a relationship with the given name
func (r *ResourceSchema) HasRelationship(name string) bool {
	_, exists := r.Relationships[name]
	return exists
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			// Add underscore if:
			// 1. Previous char was lowercase (camelCase boundary)
			// 2. Next char is lowercase and current is uppercase (acronym end: "HTTPServer" -> "http_server")
			// 3. Previous char was uppercase AND current is uppercase (acronym: "API" -> "a_p_i")
			if prev >= 'a' && prev <= 'z' {
				result = append(result, '_')
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result = append(result, '_')
			} else if prev >= 'A' && prev <= 'Z' {
				result = append(result, '_')
			}
		}
		// Convert uppercase to lowercase
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
