package typechecker

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestNullableArrayElements tests nullable array element types
func TestNullableArrayElements(t *testing.T) {
	// Test array<string?>!
	arrayNode := &ast.TypeNode{
		Kind:     ast.TypeArray,
		Name:     "array",
		Nullable: false, // The array itself is required
		ElementType: &ast.TypeNode{
			Kind:     ast.TypePrimitive,
			Name:     "string",
			Nullable: true, // But elements are nullable
		},
	}

	arrayType, err := TypeFromASTNode(arrayNode, arrayNode.Nullable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check array is not nullable
	if arrayType.IsNullable() {
		t.Error("Array should be required (!)")
	}

	// Check element type is nullable
	arr, ok := arrayType.(*ArrayType)
	if !ok {
		t.Fatal("Expected ArrayType")
	}
	if !arr.ElementType.IsNullable() {
		t.Error("Array elements should be nullable (?)")
	}

	// Expected string: array<string?>!
	expected := "array<string?>!"
	if arrayType.String() != expected {
		t.Errorf("Expected %s, got %s", expected, arrayType.String())
	}
}

// TestNullableHashKeysAndValues tests nullable hash key and value types
func TestNullableHashKeysAndValues(t *testing.T) {
	// Test hash<string!, int?>!
	hashNode := &ast.TypeNode{
		Kind:     ast.TypeHash,
		Name:     "hash",
		Nullable: false, // The hash itself is required
		KeyType: &ast.TypeNode{
			Kind:     ast.TypePrimitive,
			Name:     "string",
			Nullable: false, // Keys are required
		},
		ValueType: &ast.TypeNode{
			Kind:     ast.TypePrimitive,
			Name:     "int",
			Nullable: true, // Values are nullable
		},
	}

	hashType, err := TypeFromASTNode(hashNode, hashNode.Nullable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check hash is not nullable
	if hashType.IsNullable() {
		t.Error("Hash should be required (!)")
	}

	// Check key and value types
	hash, ok := hashType.(*HashType)
	if !ok {
		t.Fatal("Expected HashType")
	}
	if hash.KeyType.IsNullable() {
		t.Error("Hash keys should be required (!)")
	}
	if !hash.ValueType.IsNullable() {
		t.Error("Hash values should be nullable (?)")
	}

	// Expected string: hash<string!, int?>!
	expected := "hash<string!, int?>!"
	if hashType.String() != expected {
		t.Errorf("Expected %s, got %s", expected, hashType.String())
	}
}

// TestNestedNullableTypes tests deeply nested nullable types
func TestNestedNullableTypes(t *testing.T) {
	// Test array<array<string?>!>!
	nestedArrayNode := &ast.TypeNode{
		Kind:     ast.TypeArray,
		Name:     "array",
		Nullable: false, // Outer array is required
		ElementType: &ast.TypeNode{
			Kind:     ast.TypeArray,
			Name:     "array",
			Nullable: false, // Inner array is required
			ElementType: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "string",
				Nullable: true, // But strings are nullable
			},
		},
	}

	nestedType, err := TypeFromASTNode(nestedArrayNode, nestedArrayNode.Nullable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Expected string: array<array<string?>!>!
	expected := "array<array<string?>!>!"
	if nestedType.String() != expected {
		t.Errorf("Expected %s, got %s", expected, nestedType.String())
	}
}

// TestHashWithNullableKeys tests hash with nullable keys
func TestHashWithNullableKeys(t *testing.T) {
	// Test hash<string?, int!>?
	hashNode := &ast.TypeNode{
		Kind:     ast.TypeHash,
		Name:     "hash",
		Nullable: true, // The hash itself is nullable
		KeyType: &ast.TypeNode{
			Kind:     ast.TypePrimitive,
			Name:     "string",
			Nullable: true, // Keys are nullable
		},
		ValueType: &ast.TypeNode{
			Kind:     ast.TypePrimitive,
			Name:     "int",
			Nullable: false, // Values are required
		},
	}

	hashType, err := TypeFromASTNode(hashNode, hashNode.Nullable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Expected string: hash<string?, int!>?
	expected := "hash<string?, int!>?"
	if hashType.String() != expected {
		t.Errorf("Expected %s, got %s", expected, hashType.String())
	}
}
