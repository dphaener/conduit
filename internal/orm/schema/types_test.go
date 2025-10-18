package schema

import (
	"testing"
)

func TestPrimitiveTypeString(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  PrimitiveType
		expected string
	}{
		{"TypeString", TypeString, "string"},
		{"TypeText", TypeText, "text"},
		{"TypeInt", TypeInt, "int"},
		{"TypeFloat", TypeFloat, "float"},
		{"TypeBool", TypeBool, "bool"},
		{"TypeUUID", TypeUUID, "uuid"},
		{"TypeEmail", TypeEmail, "email"},
		{"TypeURL", TypeURL, "url"},
		{"TypeTimestamp", TypeTimestamp, "timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typeVal.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParsePrimitiveType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  PrimitiveType
		expectErr bool
	}{
		{"valid string", "string", TypeString, false},
		{"valid int", "int", TypeInt, false},
		{"valid uuid", "uuid", TypeUUID, false},
		{"invalid type", "unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePrimitiveType(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestTypeSpecString(t *testing.T) {
	tests := []struct {
		name     string
		typeSpec *TypeSpec
		expected string
	}{
		{
			name: "simple required string",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
			expected: "string!",
		},
		{
			name: "optional int",
			typeSpec: &TypeSpec{
				BaseType: TypeInt,
				Nullable: true,
			},
			expected: "int?",
		},
		{
			name: "array of strings",
			typeSpec: &TypeSpec{
				Nullable: false,
				ArrayElement: &TypeSpec{
					BaseType: TypeString,
					Nullable: false,
				},
			},
			expected: "array<string!>!",
		},
		{
			name: "hash of string to int",
			typeSpec: &TypeSpec{
				Nullable: false,
				HashKey: &TypeSpec{
					BaseType: TypeString,
					Nullable: false,
				},
				HashValue: &TypeSpec{
					BaseType: TypeInt,
					Nullable: false,
				},
			},
			expected: "hash<string!, int!>!",
		},
		{
			name: "enum",
			typeSpec: &TypeSpec{
				BaseType:   TypeEnum,
				Nullable:   false,
				EnumValues: []string{"draft", "published"},
			},
			expected: "enum[draft published]!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typeSpec.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTypeSpecHelperMethods(t *testing.T) {
	t.Run("IsNumeric", func(t *testing.T) {
		numericTypes := []PrimitiveType{TypeInt, TypeBigInt, TypeFloat, TypeDecimal}
		for _, pt := range numericTypes {
			ts := &TypeSpec{BaseType: pt}
			if !ts.IsNumeric() {
				t.Errorf("expected %v to be numeric", pt)
			}
		}

		nonNumeric := &TypeSpec{BaseType: TypeString}
		if nonNumeric.IsNumeric() {
			t.Error("string should not be numeric")
		}
	})

	t.Run("IsText", func(t *testing.T) {
		textTypes := []PrimitiveType{TypeString, TypeText, TypeMarkdown}
		for _, pt := range textTypes {
			ts := &TypeSpec{BaseType: pt}
			if !ts.IsText() {
				t.Errorf("expected %v to be text", pt)
			}
		}

		nonText := &TypeSpec{BaseType: TypeInt}
		if nonText.IsText() {
			t.Error("int should not be text")
		}
	})

	t.Run("IsValidated", func(t *testing.T) {
		validatedTypes := []PrimitiveType{TypeEmail, TypeURL, TypePhone}
		for _, pt := range validatedTypes {
			ts := &TypeSpec{BaseType: pt}
			if !ts.IsValidated() {
				t.Errorf("expected %v to be validated", pt)
			}
		}

		nonValidated := &TypeSpec{BaseType: TypeString}
		if nonValidated.IsValidated() {
			t.Error("string should not be validated")
		}
	})
}

func TestCascadeAction(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			action   CascadeAction
			expected string
		}{
			{CascadeRestrict, "restrict"},
			{CascadeCascade, "cascade"},
			{CascadeSetNull, "set_null"},
			{CascadeNoAction, "no_action"},
		}

		for _, tt := range tests {
			result := tt.action.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		}
	})

	t.Run("Parse", func(t *testing.T) {
		tests := []struct {
			input     string
			expected  CascadeAction
			expectErr bool
		}{
			{"restrict", CascadeRestrict, false},
			{"cascade", CascadeCascade, false},
			{"set_null", CascadeSetNull, false},
			{"invalid", 0, true},
		}

		for _, tt := range tests {
			result, err := ParseCascadeAction(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("for %s: expected %v, got %v", tt.input, tt.expected, result)
				}
			}
		}
	})
}

func TestNewResourceSchema(t *testing.T) {
	schema := NewResourceSchema("Post")

	if schema.Name != "Post" {
		t.Errorf("expected name Post, got %s", schema.Name)
	}

	if schema.TableName != "post" {
		t.Errorf("expected table name post, got %s", schema.TableName)
	}

	if schema.Fields == nil {
		t.Error("fields map should be initialized")
	}

	if schema.Relationships == nil {
		t.Error("relationships map should be initialized")
	}

	if schema.Hooks == nil {
		t.Error("hooks map should be initialized")
	}
}

func TestResourceSchemaHelpers(t *testing.T) {
	schema := NewResourceSchema("Post")

	// Add a field
	schema.Fields["title"] = &Field{
		Name: "title",
		Type: &TypeSpec{
			BaseType: TypeString,
			Nullable: false,
		},
	}

	// Add a primary key
	schema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{
			{Name: "primary"},
		},
	}

	// Add a relationship
	schema.Relationships["author"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	t.Run("HasField", func(t *testing.T) {
		if !schema.HasField("title") {
			t.Error("should have title field")
		}
		if schema.HasField("nonexistent") {
			t.Error("should not have nonexistent field")
		}
	})

	t.Run("HasRelationship", func(t *testing.T) {
		if !schema.HasRelationship("author") {
			t.Error("should have author relationship")
		}
		if schema.HasRelationship("nonexistent") {
			t.Error("should not have nonexistent relationship")
		}
	})

	t.Run("GetPrimaryKey", func(t *testing.T) {
		pk, err := schema.GetPrimaryKey()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if pk.Name != "id" {
			t.Errorf("expected id, got %s", pk.Name)
		}
	})
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Post", "post"},
		{"BlogPost", "blog_post"},
		{"UserAccount", "user_account"},
		{"API", "a_p_i"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
