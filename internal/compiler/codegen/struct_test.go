package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestToJSONAPIType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"BlogPost", "blog_posts"},
		{"Post", "posts"},
		// Note: Simple pluralization just adds 's' (irregular plurals are a TODO)
		{"ProductCategory", "product_categorys"},
		{"Comment", "comments"},
		{"Article", "articles"},
		{"UserProfile", "user_profiles"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			g := NewGenerator()
			result := g.toJSONAPIType(tt.input)
			if result != tt.expected {
				t.Errorf("toJSONAPIType(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateStructTags_PrimaryKey(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expected     string
	}{
		{"User resource", "User", "`jsonapi:\"primary,users\" db:\"id\" json:\"id\"`"},
		{"BlogPost resource", "BlogPost", "`jsonapi:\"primary,blog_posts\" db:\"id\" json:\"id\"`"},
		{"Post resource", "Post", "`jsonapi:\"primary,posts\" db:\"id\" json:\"id\"`"},
		// Note: Simple pluralization just adds 's' (irregular plurals are a TODO)
		{"ProductCategory resource", "ProductCategory", "`jsonapi:\"primary,product_categorys\" db:\"id\" json:\"id\"`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			jsonapiType := g.toJSONAPIType(tt.resourceName)

			// Primary key tags are generated in generateStruct, not generateStructTags
			// So we'll test the format directly
			result := "`jsonapi:\"primary," + jsonapiType + "\" db:\"id\" json:\"id\"`"

			if result != tt.expected {
				t.Errorf("Primary key tag = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGenerateStructTags_AttributeFields(t *testing.T) {
	tests := []struct {
		name        string
		field       *ast.FieldNode
		expected    string
	}{
		{
			name: "string field",
			field: &ast.FieldNode{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"attr,username\" db:\"username\" json:\"username\"`",
		},
		{
			name: "int field",
			field: &ast.FieldNode{
				Name: "age",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"attr,age\" db:\"age\" json:\"age\"`",
		},
		{
			name: "nullable string field",
			field: &ast.FieldNode{
				Name: "bio",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "text",
					Nullable: true,
				},
				Nullable: true,
			},
			expected: "`jsonapi:\"attr,bio\" db:\"bio\" json:\"bio,omitempty\"`",
		},
		{
			name: "snake_case field name",
			field: &ast.FieldNode{
				Name: "email_address",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"attr,email_address\" db:\"email_address\" json:\"email_address\"`",
		},
		{
			name: "camelCase field name",
			field: &ast.FieldNode{
				Name: "firstName",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"attr,firstname\" db:\"firstname\" json:\"firstName\"`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			result := g.generateStructTags(tt.field, "User")

			if result != tt.expected {
				t.Errorf("generateStructTags() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGenerateStructTags_RelationshipFields(t *testing.T) {
	tests := []struct {
		name     string
		field    *ast.FieldNode
		expected string
	}{
		{
			name: "single relationship",
			field: &ast.FieldNode{
				Name: "author",
				Type: &ast.TypeNode{
					Kind:     ast.TypeResource,
					Name:     "User",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"relation,author\" db:\"author\" json:\"author\"`",
		},
		{
			name: "array relationship",
			field: &ast.FieldNode{
				Name: "comments",
				Type: &ast.TypeNode{
					Kind:     ast.TypeArray,
					Name:     "Comment",
					Nullable: false,
				},
				Nullable: false,
			},
			expected: "`jsonapi:\"relation,comments\" db:\"comments\" json:\"comments\"`",
		},
		{
			name: "nullable relationship",
			field: &ast.FieldNode{
				Name: "parent_post",
				Type: &ast.TypeNode{
					Kind:     ast.TypeResource,
					Name:     "Post",
					Nullable: true,
				},
				Nullable: true,
			},
			expected: "`jsonapi:\"relation,parent_post\" db:\"parent_post\" json:\"parent_post,omitempty\"`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			result := g.generateStructTags(tt.field, "Post")

			if result != tt.expected {
				t.Errorf("generateStructTags() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGenerateStruct_FullStructWithJSONAPITags(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "BlogPost",
		Fields: []*ast.FieldNode{
			{
				Name: "title",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "content",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "text",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "author",
				Type: &ast.TypeNode{
					Kind:     ast.TypeResource,
					Name:     "User",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify JSON:API primary tag
	if !strings.Contains(code, "jsonapi:\"primary,blog_posts\"") {
		t.Error("Generated code should contain JSON:API primary tag with correct type")
	}

	// Verify JSON:API attribute tags
	if !strings.Contains(code, "jsonapi:\"attr,title\"") {
		t.Error("Generated code should contain JSON:API attr tag for title")
	}

	if !strings.Contains(code, "jsonapi:\"attr,content\"") {
		t.Error("Generated code should contain JSON:API attr tag for content")
	}

	// Verify JSON:API relationship tag
	if !strings.Contains(code, "jsonapi:\"relation,author\"") {
		t.Error("Generated code should contain JSON:API relation tag for author")
	}

	// Verify all three tag types are present for a field
	// The tags should be in the format: jsonapi, db, json
	if !strings.Contains(code, "db:\"title\"") {
		t.Error("Generated code should contain db tag for title")
	}

	if !strings.Contains(code, "json:\"title\"") {
		t.Error("Generated code should contain json tag for title")
	}
}

func TestGenerateStruct_TagOrdering(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "email",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify tag ordering: jsonapi, db, json
	// The tags should appear in this order in the struct
	emailTagIndex := strings.Index(code, "`jsonapi:\"attr,email\" db:\"email\" json:\"email\"`")
	if emailTagIndex == -1 {
		t.Error("Generated code should contain properly ordered struct tags (jsonapi, db, json)")
	}
}

func TestGenerateStruct_ImplicitIDField(t *testing.T) {
	// Test that when no ID field is explicitly defined, one is generated
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name: "title",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify implicit ID field is generated
	if !strings.Contains(code, "ID") {
		t.Error("Generated code should contain implicit ID field")
	}

	// Verify ID field has correct type
	if !strings.Contains(code, "int64") {
		t.Error("Generated code should use int64 for ID field")
	}

	// Verify ID field has correct JSON:API primary tag
	if !strings.Contains(code, "jsonapi:\"primary,posts\"") {
		t.Error("Generated code should contain JSON:API primary tag for implicit ID")
	}
}

func TestGenerateStruct_ExplicitIDField(t *testing.T) {
	// Test that when an ID field is explicitly defined, it's used
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "id",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "uuid",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify explicit ID field is present
	if !strings.Contains(code, "ID") {
		t.Error("Generated code should contain ID field")
	}

	// Verify ID field uses uuid.UUID type
	if !strings.Contains(code, "uuid.UUID") {
		t.Error("Generated code should use uuid.UUID for explicit ID field")
	}

	// Verify ID field has correct JSON:API primary tag
	if !strings.Contains(code, "jsonapi:\"primary,users\"") {
		t.Error("Generated code should contain JSON:API primary tag for explicit ID field")
	}
}

func TestGenerateStruct_MultipleResourceTypes(t *testing.T) {
	// Test that different resource types get different JSON:API type names
	resources := []struct {
		name         string
		expectedType string
	}{
		{"User", "users"},
		{"Post", "posts"},
		{"Comment", "comments"},
		{"BlogPost", "blog_posts"},
		// Note: Simple pluralization just adds 's' (irregular plurals are a TODO)
		{"ProductCategory", "product_categorys"},
	}

	for _, r := range resources {
		t.Run(r.name, func(t *testing.T) {
			resource := &ast.ResourceNode{
				Name: r.name,
				Fields: []*ast.FieldNode{
					{
						Name: "name",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "string",
							Nullable: false,
						},
						Nullable: false,
					},
				},
			}

			gen := NewGenerator()
			code, err := gen.GenerateResource(resource)
			if err != nil {
				t.Fatalf("GenerateResource failed for %s: %v", r.name, err)
			}

			// Verify JSON:API type in primary tag
			expectedPrimaryTag := "jsonapi:\"primary," + r.expectedType + "\""
			if !strings.Contains(code, expectedPrimaryTag) {
				t.Errorf("Generated code for %s should contain JSON:API primary tag: %s", r.name, expectedPrimaryTag)
			}
		})
	}
}
