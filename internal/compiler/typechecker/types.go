// Package typechecker implements the type system and nullability checking for the Conduit compiler.
// It provides type inference, type checking, and comprehensive error reporting.
package typechecker

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Common type name constants to avoid string repetition.
const (
	typeFloat = "float"
	typeInt   = "int"
)

// Type represents a type in the Conduit type system.
// All types must explicitly declare nullability (! vs ?).
type Type interface {
	// String returns the human-readable representation of the type
	String() string

	// IsNullable returns true if this type can be nil
	IsNullable() bool

	// IsAssignableFrom checks if a value of type other can be assigned to this type.
	// Enforces nullability rules: nullable cannot assign to required without unwrap/coalesce.
	IsAssignableFrom(other Type) bool

	// Equals checks if two types are exactly equal
	Equals(other Type) bool

	// MakeNullable returns a new Type that is the nullable version of this type
	MakeNullable() Type

	// MakeRequired returns a new Type that is the required version of this type
	MakeRequired() Type
}

// PrimitiveType represents a built-in primitive type (string, int, float, bool, etc.)
type PrimitiveType struct {
	Name     string // "string", "int", "float", "bool", "timestamp", "uuid", etc.
	Nullable bool   // true for ?, false for !
}

// NewPrimitiveType creates a new primitive type with the specified nullability.
func NewPrimitiveType(name string, nullable bool) *PrimitiveType {
	return &PrimitiveType{Name: name, Nullable: nullable}
}

func (p *PrimitiveType) String() string {
	suffix := "!"
	if p.Nullable {
		suffix = "?"
	}
	return p.Name + suffix
}

// IsNullable returns whether the primitive type can be nil.
func (p *PrimitiveType) IsNullable() bool {
	return p.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this primitive type.
func (p *PrimitiveType) IsAssignableFrom(other Type) bool {
	otherPrim, ok := other.(*PrimitiveType)
	if !ok {
		return false
	}

	// Must be compatible primitive types
	if !p.isCompatibleWith(otherPrim.Name) {
		return false
	}

	// Nullability check: nullable cannot assign to required
	if !p.Nullable && otherPrim.Nullable {
		return false
	}

	return true
}

// isCompatibleWith checks if a primitive type name is compatible with this type
func (p *PrimitiveType) isCompatibleWith(otherName string) bool {
	// Exact match
	if p.Name == otherName {
		return true
	}

	// String family - text and string are compatible
	stringFamily := map[string]bool{"string": true, "text": true, "markdown": true}
	if stringFamily[p.Name] && stringFamily[otherName] {
		return true
	}

	// Numeric family - int can assign to float
	if p.Name == typeFloat && otherName == typeInt {
		return true
	}

	return false
}

// Equals checks if two primitive types are exactly equal.
func (p *PrimitiveType) Equals(other Type) bool {
	otherPrim, ok := other.(*PrimitiveType)
	if !ok {
		return false
	}
	return p.Name == otherPrim.Name && p.Nullable == otherPrim.Nullable
}

// MakeNullable returns a new primitive type that is the nullable version of this type.
func (p *PrimitiveType) MakeNullable() Type {
	return &PrimitiveType{Name: p.Name, Nullable: true}
}

// MakeRequired returns a new primitive type that is the required version of this type.
func (p *PrimitiveType) MakeRequired() Type {
	return &PrimitiveType{Name: p.Name, Nullable: false}
}

// ArrayType represents an array type (array<T>)
type ArrayType struct {
	ElementType Type
	Nullable    bool
}

// NewArrayType creates a new array type with the specified element type and nullability.
func NewArrayType(elementType Type, nullable bool) *ArrayType {
	return &ArrayType{ElementType: elementType, Nullable: nullable}
}

func (a *ArrayType) String() string {
	suffix := "!"
	if a.Nullable {
		suffix = "?"
	}
	return fmt.Sprintf("array<%s>%s", a.ElementType.String(), suffix)
}

// IsNullable returns whether the array type can be nil.
func (a *ArrayType) IsNullable() bool {
	return a.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this array type.
func (a *ArrayType) IsAssignableFrom(other Type) bool {
	otherArray, ok := other.(*ArrayType)
	if !ok {
		return false
	}

	// Element types must be compatible
	if !a.ElementType.IsAssignableFrom(otherArray.ElementType) {
		return false
	}

	// Nullability check
	if !a.Nullable && otherArray.Nullable {
		return false
	}

	return true
}

// Equals checks if two array types are exactly equal.
func (a *ArrayType) Equals(other Type) bool {
	otherArray, ok := other.(*ArrayType)
	if !ok {
		return false
	}
	return a.ElementType.Equals(otherArray.ElementType) && a.Nullable == otherArray.Nullable
}

// MakeNullable returns a new array type that is the nullable version of this type.
func (a *ArrayType) MakeNullable() Type {
	return &ArrayType{ElementType: a.ElementType, Nullable: true}
}

// MakeRequired returns a new array type that is the required version of this type.
func (a *ArrayType) MakeRequired() Type {
	return &ArrayType{ElementType: a.ElementType, Nullable: false}
}

// HashType represents a hash/map type (hash<K, V>)
type HashType struct {
	KeyType   Type
	ValueType Type
	Nullable  bool
}

// NewHashType creates a new hash type with the specified key type, value type, and nullability.
func NewHashType(keyType, valueType Type, nullable bool) *HashType {
	return &HashType{KeyType: keyType, ValueType: valueType, Nullable: nullable}
}

func (h *HashType) String() string {
	suffix := "!"
	if h.Nullable {
		suffix = "?"
	}
	return fmt.Sprintf("hash<%s, %s>%s", h.KeyType.String(), h.ValueType.String(), suffix)
}

// IsNullable returns whether the hash type can be nil.
func (h *HashType) IsNullable() bool {
	return h.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this hash type.
func (h *HashType) IsAssignableFrom(other Type) bool {
	otherHash, ok := other.(*HashType)
	if !ok {
		return false
	}

	// Key and value types must be compatible
	if !h.KeyType.IsAssignableFrom(otherHash.KeyType) {
		return false
	}
	if !h.ValueType.IsAssignableFrom(otherHash.ValueType) {
		return false
	}

	// Nullability check
	if !h.Nullable && otherHash.Nullable {
		return false
	}

	return true
}

// Equals checks if two hash types are exactly equal.
func (h *HashType) Equals(other Type) bool {
	otherHash, ok := other.(*HashType)
	if !ok {
		return false
	}
	return h.KeyType.Equals(otherHash.KeyType) &&
		h.ValueType.Equals(otherHash.ValueType) &&
		h.Nullable == otherHash.Nullable
}

// MakeNullable returns a new hash type that is the nullable version of this type.
func (h *HashType) MakeNullable() Type {
	return &HashType{KeyType: h.KeyType, ValueType: h.ValueType, Nullable: true}
}

// MakeRequired returns a new hash type that is the required version of this type.
func (h *HashType) MakeRequired() Type {
	return &HashType{KeyType: h.KeyType, ValueType: h.ValueType, Nullable: false}
}

// StructField represents a field in a struct type
type StructField struct {
	Name string
	Type Type
}

// StructType represents an inline struct type
type StructType struct {
	Fields   []StructField
	Nullable bool
}

// NewStructType creates a new struct type with the specified fields and nullability.
func NewStructType(fields []StructField, nullable bool) *StructType {
	return &StructType{Fields: fields, Nullable: nullable}
}

func (s *StructType) String() string {
	suffix := "!"
	if s.Nullable {
		suffix = "?"
	}

	var fieldStrs []string
	for _, field := range s.Fields {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s: %s", field.Name, field.Type.String()))
	}

	return fmt.Sprintf("{%s}%s", strings.Join(fieldStrs, ", "), suffix)
}

// IsNullable returns whether the struct type can be nil.
func (s *StructType) IsNullable() bool {
	return s.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this struct type.
func (s *StructType) IsAssignableFrom(other Type) bool {
	otherStruct, ok := other.(*StructType)
	if !ok {
		return false
	}

	// Must have same number of fields
	if len(s.Fields) != len(otherStruct.Fields) {
		return false
	}

	// Build a map of other struct's fields for order-independent comparison
	otherFields := make(map[string]Type)
	for _, field := range otherStruct.Fields {
		otherFields[field.Name] = field.Type
	}

	// All fields must match in name and be assignable
	for _, field := range s.Fields {
		otherType, ok := otherFields[field.Name]
		if !ok {
			return false
		}
		if !field.Type.IsAssignableFrom(otherType) {
			return false
		}
	}

	// Nullability check
	if !s.Nullable && otherStruct.Nullable {
		return false
	}

	return true
}

// Equals checks if two struct types are exactly equal.
func (s *StructType) Equals(other Type) bool {
	otherStruct, ok := other.(*StructType)
	if !ok {
		return false
	}

	if len(s.Fields) != len(otherStruct.Fields) {
		return false
	}

	// Build a map of other struct's fields for order-independent comparison
	otherFields := make(map[string]Type)
	for _, field := range otherStruct.Fields {
		otherFields[field.Name] = field.Type
	}

	for _, field := range s.Fields {
		otherType, ok := otherFields[field.Name]
		if !ok || !field.Type.Equals(otherType) {
			return false
		}
	}

	return s.Nullable == otherStruct.Nullable
}

// MakeNullable returns a new struct type that is the nullable version of this type.
func (s *StructType) MakeNullable() Type {
	return &StructType{Fields: s.Fields, Nullable: true}
}

// MakeRequired returns a new struct type that is the required version of this type.
func (s *StructType) MakeRequired() Type {
	return &StructType{Fields: s.Fields, Nullable: false}
}

// GetField looks up a field by name in the struct type.
func (s *StructType) GetField(name string) (Type, bool) {
	for _, field := range s.Fields {
		if field.Name == name {
			return field.Type, true
		}
	}
	return nil, false
}

// EnumType represents an enum type
type EnumType struct {
	Values   []string
	Nullable bool
}

// NewEnumType creates a new enum type with the specified values and nullability.
func NewEnumType(values []string, nullable bool) *EnumType {
	return &EnumType{Values: values, Nullable: nullable}
}

func (e *EnumType) String() string {
	suffix := "!"
	if e.Nullable {
		suffix = "?"
	}

	valueStrs := make([]string, len(e.Values))
	for i, v := range e.Values {
		valueStrs[i] = fmt.Sprintf("%q", v)
	}

	return fmt.Sprintf("enum [%s]%s", strings.Join(valueStrs, ", "), suffix)
}

// IsNullable returns whether the enum type can be nil.
func (e *EnumType) IsNullable() bool {
	return e.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this enum type.
func (e *EnumType) IsAssignableFrom(other Type) bool {
	otherEnum, ok := other.(*EnumType)
	if !ok {
		return false
	}

	// Must have same values
	if len(e.Values) != len(otherEnum.Values) {
		return false
	}

	for i, val := range e.Values {
		if val != otherEnum.Values[i] {
			return false
		}
	}

	// Nullability check
	if !e.Nullable && otherEnum.Nullable {
		return false
	}

	return true
}

// Equals checks if two enum types are exactly equal.
func (e *EnumType) Equals(other Type) bool {
	otherEnum, ok := other.(*EnumType)
	if !ok {
		return false
	}

	if len(e.Values) != len(otherEnum.Values) {
		return false
	}

	for i, val := range e.Values {
		if val != otherEnum.Values[i] {
			return false
		}
	}

	return e.Nullable == otherEnum.Nullable
}

// MakeNullable returns a new enum type that is the nullable version of this type.
func (e *EnumType) MakeNullable() Type {
	return &EnumType{Values: e.Values, Nullable: true}
}

// MakeRequired returns a new enum type that is the required version of this type.
func (e *EnumType) MakeRequired() Type {
	return &EnumType{Values: e.Values, Nullable: false}
}

// ResourceType represents a reference to another resource (relationship)
type ResourceType struct {
	Name     string
	Nullable bool
}

// NewResourceType creates a new resource type with the specified name and nullability.
func NewResourceType(name string, nullable bool) *ResourceType {
	return &ResourceType{Name: name, Nullable: nullable}
}

func (r *ResourceType) String() string {
	suffix := "!"
	if r.Nullable {
		suffix = "?"
	}
	return r.Name + suffix
}

// IsNullable returns whether the resource type can be nil.
func (r *ResourceType) IsNullable() bool {
	return r.Nullable
}

// IsAssignableFrom checks if a value of type other can be assigned to this resource type.
func (r *ResourceType) IsAssignableFrom(other Type) bool {
	otherRes, ok := other.(*ResourceType)
	if !ok {
		return false
	}

	// Must reference the same resource
	if r.Name != otherRes.Name {
		return false
	}

	// Nullability check
	if !r.Nullable && otherRes.Nullable {
		return false
	}

	return true
}

// Equals checks if two resource types are exactly equal.
func (r *ResourceType) Equals(other Type) bool {
	otherRes, ok := other.(*ResourceType)
	if !ok {
		return false
	}
	return r.Name == otherRes.Name && r.Nullable == otherRes.Nullable
}

// MakeNullable returns a new resource type that is the nullable version of this type.
func (r *ResourceType) MakeNullable() Type {
	return &ResourceType{Name: r.Name, Nullable: true}
}

// MakeRequired returns a new resource type that is the required version of this type.
func (r *ResourceType) MakeRequired() Type {
	return &ResourceType{Name: r.Name, Nullable: false}
}

// TypeFromASTNode converts an AST TypeNode to a Type with the specified nullability.
func TypeFromASTNode(node *ast.TypeNode, nullable bool) (Type, error) {
	if node == nil {
		return nil, fmt.Errorf("nil type node")
	}

	switch node.Kind {
	case ast.TypePrimitive:
		return NewPrimitiveType(node.Name, nullable), nil

	case ast.TypeArray:
		if node.ElementType == nil {
			return nil, fmt.Errorf("array type missing element type")
		}
		// Array element types have their own nullability stored in the TypeNode
		elemType, err := TypeFromASTNode(node.ElementType, node.ElementType.Nullable)
		if err != nil {
			return nil, fmt.Errorf("invalid array element type: %w", err)
		}
		return NewArrayType(elemType, nullable), nil

	case ast.TypeHash:
		if node.KeyType == nil || node.ValueType == nil {
			return nil, fmt.Errorf("hash type missing key or value type")
		}
		// Hash key and value types have their own nullability stored in the TypeNode
		keyType, err := TypeFromASTNode(node.KeyType, node.KeyType.Nullable)
		if err != nil {
			return nil, fmt.Errorf("invalid hash key type: %w", err)
		}
		valueType, err := TypeFromASTNode(node.ValueType, node.ValueType.Nullable)
		if err != nil {
			return nil, fmt.Errorf("invalid hash value type: %w", err)
		}
		return NewHashType(keyType, valueType, nullable), nil

	case ast.TypeEnum:
		return NewEnumType(node.EnumValues, nullable), nil

	case ast.TypeStruct:
		fields := make([]StructField, 0, len(node.StructFields))
		for _, fieldNode := range node.StructFields {
			fieldType, err := TypeFromASTNode(fieldNode.Type, fieldNode.Nullable)
			if err != nil {
				return nil, fmt.Errorf("invalid struct field %s: %w", fieldNode.Name, err)
			}
			fields = append(fields, StructField{
				Name: fieldNode.Name,
				Type: fieldType,
			})
		}
		return NewStructType(fields, nullable), nil

	case ast.TypeResource:
		return NewResourceType(node.Name, nullable), nil

	default:
		return nil, fmt.Errorf("unknown type kind: %d", node.Kind)
	}
}
