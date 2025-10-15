package typechecker

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestTypeEquals tests the Equals method for all type implementations
func TestTypeEquals(t *testing.T) {
	tests := []struct {
		name   string
		type1  Type
		type2  Type
		equals bool
	}{
		{
			name:   "identical primitives",
			type1:  NewPrimitiveType("string", false),
			type2:  NewPrimitiveType("string", false),
			equals: true,
		},
		{
			name:   "different nullability",
			type1:  NewPrimitiveType("string", false),
			type2:  NewPrimitiveType("string", true),
			equals: false,
		},
		{
			name:   "different primitive types",
			type1:  NewPrimitiveType("string", false),
			type2:  NewPrimitiveType("int", false),
			equals: false,
		},
		{
			name:   "identical arrays",
			type1:  NewArrayType(NewPrimitiveType("int", false), false),
			type2:  NewArrayType(NewPrimitiveType("int", false), false),
			equals: true,
		},
		{
			name:   "different array elements",
			type1:  NewArrayType(NewPrimitiveType("int", false), false),
			type2:  NewArrayType(NewPrimitiveType("string", false), false),
			equals: false,
		},
		{
			name:   "identical hashes",
			type1:  NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false),
			type2:  NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false),
			equals: true,
		},
		{
			name:   "different hash keys",
			type1:  NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false),
			type2:  NewHashType(NewPrimitiveType("int", false), NewPrimitiveType("int", false), false),
			equals: false,
		},
		{
			name: "identical structs",
			type1: NewStructType([]StructField{
				{Name: "a", Type: NewPrimitiveType("int", false)},
			}, false),
			type2: NewStructType([]StructField{
				{Name: "a", Type: NewPrimitiveType("int", false)},
			}, false),
			equals: true,
		},
		{
			name: "different struct fields",
			type1: NewStructType([]StructField{
				{Name: "a", Type: NewPrimitiveType("int", false)},
			}, false),
			type2: NewStructType([]StructField{
				{Name: "b", Type: NewPrimitiveType("int", false)},
			}, false),
			equals: false,
		},
		{
			name:   "identical enums",
			type1:  NewEnumType([]string{"a", "b"}, false),
			type2:  NewEnumType([]string{"a", "b"}, false),
			equals: true,
		},
		{
			name:   "different enum values",
			type1:  NewEnumType([]string{"a", "b"}, false),
			type2:  NewEnumType([]string{"c", "d"}, false),
			equals: false,
		},
		{
			name:   "identical resources",
			type1:  NewResourceType("User", false),
			type2:  NewResourceType("User", false),
			equals: true,
		},
		{
			name:   "different resources",
			type1:  NewResourceType("User", false),
			type2:  NewResourceType("Post", false),
			equals: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.type1.Equals(tt.type2)
			if result != tt.equals {
				t.Errorf("Expected Equals to be %v, got %v", tt.equals, result)
			}
		})
	}
}

// TestMakeNullable tests the MakeNullable method
func TestMakeNullable(t *testing.T) {
	types := []Type{
		NewPrimitiveType("string", false),
		NewArrayType(NewPrimitiveType("int", false), false),
		NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false),
		NewStructType([]StructField{{Name: "a", Type: NewPrimitiveType("int", false)}}, false),
		NewEnumType([]string{"a", "b"}, false),
		NewResourceType("User", false),
	}

	for _, typ := range types {
		t.Run(typ.String(), func(t *testing.T) {
			nullable := typ.MakeNullable()
			if !nullable.IsNullable() {
				t.Error("MakeNullable should return a nullable type")
			}
		})
	}
}

// TestMakeRequired tests the MakeRequired method
func TestMakeRequired(t *testing.T) {
	types := []Type{
		NewPrimitiveType("string", true),
		NewArrayType(NewPrimitiveType("int", false), true),
		NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), true),
		NewStructType([]StructField{{Name: "a", Type: NewPrimitiveType("int", false)}}, true),
		NewEnumType([]string{"a", "b"}, true),
		NewResourceType("User", true),
	}

	for _, typ := range types {
		t.Run(typ.String(), func(t *testing.T) {
			required := typ.MakeRequired()
			if required.IsNullable() {
				t.Error("MakeRequired should return a required type")
			}
		})
	}
}

// TestIsNullable tests the IsNullable method
func TestIsNullable(t *testing.T) {
	tests := []struct {
		typ      Type
		nullable bool
	}{
		{NewPrimitiveType("string", true), true},
		{NewPrimitiveType("string", false), false},
		{NewArrayType(NewPrimitiveType("int", false), true), true},
		{NewArrayType(NewPrimitiveType("int", false), false), false},
		{NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), true), true},
		{NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false), false},
		{NewStructType([]StructField{}, true), true},
		{NewStructType([]StructField{}, false), false},
		{NewEnumType([]string{"a"}, true), true},
		{NewEnumType([]string{"a"}, false), false},
		{NewResourceType("User", true), true},
		{NewResourceType("User", false), false},
	}

	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			if tt.typ.IsNullable() != tt.nullable {
				t.Errorf("Expected IsNullable to be %v, got %v", tt.nullable, tt.typ.IsNullable())
			}
		})
	}
}

// TestStructGetField tests the GetField method for StructType
func TestStructGetField(t *testing.T) {
	structType := NewStructType([]StructField{
		{Name: "name", Type: NewPrimitiveType("string", false)},
		{Name: "age", Type: NewPrimitiveType("int", false)},
	}, false)

	// Test existing field
	typ, ok := structType.GetField("name")
	if !ok {
		t.Error("Expected to find field 'name'")
	}
	if typ.String() != "string!" {
		t.Errorf("Expected string!, got %s", typ.String())
	}

	// Test non-existent field
	_, ok = structType.GetField("nonexistent")
	if ok {
		t.Error("Should not find non-existent field")
	}
}

// TestTypeFromASTNodeEdgeCases tests edge cases for AST type conversion
func TestTypeFromASTNodeEdgeCases(t *testing.T) {
	// Test nil node
	_, err := TypeFromASTNode(nil, false)
	if err == nil {
		t.Error("Expected error for nil type node")
	}

	// Test array without element type
	_, err = TypeFromASTNode(&ast.TypeNode{
		Kind: ast.TypeArray,
	}, false)
	if err == nil {
		t.Error("Expected error for array without element type")
	}

	// Test hash without key/value types
	_, err = TypeFromASTNode(&ast.TypeNode{
		Kind: ast.TypeHash,
	}, false)
	if err == nil {
		t.Error("Expected error for hash without key/value types")
	}

	// Test struct with fields
	structNode := &ast.TypeNode{
		Kind: ast.TypeStruct,
		StructFields: []*ast.FieldNode{
			{
				Name:     "test",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
			},
		},
	}
	typ, err := TypeFromASTNode(structNode, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if typ.String() != "{test: string!}!" {
		t.Errorf("Expected {test: string!}!, got %s", typ.String())
	}

	// Test resource type
	resNode := &ast.TypeNode{
		Kind: ast.TypeResource,
		Name: "User",
	}
	typ, err = TypeFromASTNode(resNode, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if typ.String() != "User!" {
		t.Errorf("Expected User!, got %s", typ.String())
	}
}

// TestStringFamily tests that string, text, and markdown are compatible
func TestStringFamily(t *testing.T) {
	stringType := NewPrimitiveType("string", false)
	textType := NewPrimitiveType("text", false)
	markdownType := NewPrimitiveType("markdown", false)

	// All string family types should be assignable to each other
	if !stringType.IsAssignableFrom(textType) {
		t.Error("string! should accept text!")
	}

	if !stringType.IsAssignableFrom(markdownType) {
		t.Error("string! should accept markdown!")
	}

	if !textType.IsAssignableFrom(stringType) {
		t.Error("text! should accept string!")
	}
}

// TestNumericFamily tests that int can assign to float
func TestNumericFamily(t *testing.T) {
	intType := NewPrimitiveType("int", false)
	floatType := NewPrimitiveType("float", false)

	// int should be assignable to float
	if !floatType.IsAssignableFrom(intType) {
		t.Error("float! should accept int!")
	}

	// But not the reverse
	if intType.IsAssignableFrom(floatType) {
		t.Error("int! should NOT accept float!")
	}
}
