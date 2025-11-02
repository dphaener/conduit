package parser

import "github.com/conduit-lang/conduit/compiler/lexer"

// SourceLocation represents a location in source code
type SourceLocation struct {
	File   string
	Line   int
	Column int
}

// Program is the root node of the AST
type Program struct {
	Resources []*ResourceNode
	Location  SourceLocation
}

// ResourceNode represents a resource definition
type ResourceNode struct {
	Name           string
	Documentation  string // /// comments above resource
	LeadingComment string // # comments above resource (before resource keyword)
	Fields         []*FieldNode
	Relationships  []*RelationshipNode
	Hooks          []*HookNode
	Constraints    []*CustomConstraintNode
	Location       SourceLocation
}

// FieldNode represents a field definition
type FieldNode struct {
	Name          string
	Type          TypeNode
	Nullable      bool // ! vs ?
	Constraints   []*ConstraintNode
	LeadingComment  string // Comment on line(s) before this field
	TrailingComment string // Comment at end of field line
	Location      SourceLocation
}

// TypeKind represents the kind of type
type TypeKind int

const (
	TypeKindPrimitive TypeKind = iota
	TypeKindArray
	TypeKindHash
	TypeKindEnum
	TypeKindStruct
	TypeKindResource
)

// TypeNode represents a type
type TypeNode struct {
	Kind         TypeKind
	Name         string      // For primitives and resources
	ElementType  *TypeNode   // For array<T>
	KeyType      *TypeNode   // For hash<K,V>
	ValueType    *TypeNode   // For hash<K,V>
	EnumValues   []string    // For enum types
	StructFields []*FieldNode // For inline structs
	Location     SourceLocation
}

// RelationshipNode represents a belongs-to relationship
type RelationshipNode struct {
	Name            string
	TargetType      string // Resource name
	Nullable        bool   // ! vs ?
	ForeignKey      string // Optional metadata
	OnDelete        string // restrict, cascade, set_null, no_action
	OnUpdate        string // cascade, restrict, etc.
	LeadingComment  string // Comment on line(s) before this relationship
	TrailingComment string // Comment at end of relationship line
	Location        SourceLocation
}

// ConstraintNode represents a field constraint annotation
type ConstraintNode struct {
	Name      string                 // min, max, unique, pattern, etc.
	Arguments []interface{}          // Arguments to the constraint
	Location  SourceLocation
}

// HookNode represents a resource-level hook (@before or @after)
type HookNode struct {
	Type     string // "before" or "after"
	Trigger  string // e.g., "create", "update", "delete"
	Body     string // Raw body content captured as string
	Location SourceLocation
}

// CustomConstraintNode represents a resource-level custom constraint
type CustomConstraintNode struct {
	Name     string // The constraint name
	Body     string // Raw body content captured as string
	Location SourceLocation
}

// ConstraintKind represents the type of constraint
type ConstraintKind int

const (
	ConstraintKindMin ConstraintKind = iota
	ConstraintKindMax
	ConstraintKindUnique
	ConstraintKindPrimary
	ConstraintKindAuto
	ConstraintKindAutoUpdate
	ConstraintKindDefault
	ConstraintKindPattern
	ConstraintKindRequired
)

// Helper methods for AST nodes

// NewProgram creates a new Program node
func NewProgram(resources []*ResourceNode, loc SourceLocation) *Program {
	return &Program{
		Resources: resources,
		Location:  loc,
	}
}

// NewResourceNode creates a new ResourceNode
func NewResourceNode(name string, doc string, loc SourceLocation) *ResourceNode {
	return &ResourceNode{
		Name:          name,
		Documentation: doc,
		Fields:        []*FieldNode{},
		Relationships: []*RelationshipNode{},
		Hooks:         []*HookNode{},
		Constraints:   []*CustomConstraintNode{},
		Location:      loc,
	}
}

// NewFieldNode creates a new FieldNode
func NewFieldNode(name string, typ TypeNode, nullable bool, loc SourceLocation) *FieldNode {
	return &FieldNode{
		Name:        name,
		Type:        typ,
		Nullable:    nullable,
		Constraints: []*ConstraintNode{},
		Location:    loc,
	}
}

// NewPrimitiveType creates a primitive type node
func NewPrimitiveType(name string, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:     TypeKindPrimitive,
		Name:     name,
		Location: loc,
	}
}

// NewArrayType creates an array type node
func NewArrayType(elementType TypeNode, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:        TypeKindArray,
		ElementType: &elementType,
		Location:    loc,
	}
}

// NewHashType creates a hash type node
func NewHashType(keyType, valueType TypeNode, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:      TypeKindHash,
		KeyType:   &keyType,
		ValueType: &valueType,
		Location:  loc,
	}
}

// NewEnumType creates an enum type node
func NewEnumType(values []string, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:       TypeKindEnum,
		EnumValues: values,
		Location:   loc,
	}
}

// NewStructType creates an inline struct type node
func NewStructType(fields []*FieldNode, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:         TypeKindStruct,
		StructFields: fields,
		Location:     loc,
	}
}

// NewResourceType creates a resource reference type node
func NewResourceType(name string, loc SourceLocation) TypeNode {
	return TypeNode{
		Kind:     TypeKindResource,
		Name:     name,
		Location: loc,
	}
}

// NewRelationshipNode creates a new RelationshipNode
func NewRelationshipNode(name, targetType string, nullable bool, loc SourceLocation) *RelationshipNode {
	return &RelationshipNode{
		Name:       name,
		TargetType: targetType,
		Nullable:   nullable,
		Location:   loc,
	}
}

// NewConstraintNode creates a new ConstraintNode
func NewConstraintNode(name string, args []interface{}, loc SourceLocation) *ConstraintNode {
	return &ConstraintNode{
		Name:      name,
		Arguments: args,
		Location:  loc,
	}
}

// NewHookNode creates a new HookNode
func NewHookNode(hookType, trigger, body string, loc SourceLocation) *HookNode {
	return &HookNode{
		Type:     hookType,
		Trigger:  trigger,
		Body:     body,
		Location: loc,
	}
}

// NewCustomConstraintNode creates a new CustomConstraintNode
func NewCustomConstraintNode(name, body string, loc SourceLocation) *CustomConstraintNode {
	return &CustomConstraintNode{
		Name:     name,
		Body:     body,
		Location: loc,
	}
}

// AddField adds a field to the resource
func (r *ResourceNode) AddField(field *FieldNode) {
	r.Fields = append(r.Fields, field)
}

// AddRelationship adds a relationship to the resource
func (r *ResourceNode) AddRelationship(rel *RelationshipNode) {
	r.Relationships = append(r.Relationships, rel)
}

// AddHook adds a hook to the resource
func (r *ResourceNode) AddHook(hook *HookNode) {
	r.Hooks = append(r.Hooks, hook)
}

// AddCustomConstraint adds a custom constraint to the resource
func (r *ResourceNode) AddCustomConstraint(constraint *CustomConstraintNode) {
	r.Constraints = append(r.Constraints, constraint)
}

// AddConstraint adds a constraint to the field
func (f *FieldNode) AddConstraint(constraint *ConstraintNode) {
	f.Constraints = append(f.Constraints, constraint)
}

// IsPrimitive returns true if the type is a primitive
func (t TypeNode) IsPrimitive() bool {
	return t.Kind == TypeKindPrimitive
}

// IsArray returns true if the type is an array
func (t TypeNode) IsArray() bool {
	return t.Kind == TypeKindArray
}

// IsHash returns true if the type is a hash
func (t TypeNode) IsHash() bool {
	return t.Kind == TypeKindHash
}

// IsEnum returns true if the type is an enum
func (t TypeNode) IsEnum() bool {
	return t.Kind == TypeKindEnum
}

// IsStruct returns true if the type is an inline struct
func (t TypeNode) IsStruct() bool {
	return t.Kind == TypeKindStruct
}

// IsResource returns true if the type is a resource reference
func (t TypeNode) IsResource() bool {
	return t.Kind == TypeKindResource
}

// String returns a string representation of the type
func (t TypeNode) String() string {
	switch t.Kind {
	case TypeKindPrimitive:
		return t.Name
	case TypeKindArray:
		return "array<" + t.ElementType.String() + ">"
	case TypeKindHash:
		return "hash<" + t.KeyType.String() + ", " + t.ValueType.String() + ">"
	case TypeKindEnum:
		return "enum"
	case TypeKindStruct:
		return "struct"
	case TypeKindResource:
		return t.Name
	default:
		return "unknown"
	}
}

// TokenToLocation converts a token to a SourceLocation
func TokenToLocation(token lexer.Token) SourceLocation {
	return SourceLocation{
		File:   token.File,
		Line:   token.Line,
		Column: token.Column,
	}
}
