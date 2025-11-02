package build

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestSchemaExtractor_ExtractSchemas(t *testing.T) {
	tests := []struct {
		name        string
		resources   []*ast.ResourceNode
		wantSchemas int
		wantErr     bool
	}{
		{
			name: "single resource with fields",
			resources: []*ast.ResourceNode{
				{
					Name: "User",
					Fields: []*ast.FieldNode{
						{
							Name: "name",
							Type: &ast.TypeNode{
								Kind: ast.TypePrimitive,
								Name: "string",
							},
							Nullable: false,
						},
						{
							Name: "email",
							Type: &ast.TypeNode{
								Kind: ast.TypePrimitive,
								Name: "string",
							},
							Nullable: false,
						},
					},
				},
			},
			wantSchemas: 1,
			wantErr:     false,
		},
		{
			name: "multiple resources",
			resources: []*ast.ResourceNode{
				{
					Name: "User",
					Fields: []*ast.FieldNode{
						{
							Name: "name",
							Type: &ast.TypeNode{
								Kind: ast.TypePrimitive,
								Name: "string",
							},
							Nullable: false,
						},
					},
				},
				{
					Name: "Post",
					Fields: []*ast.FieldNode{
						{
							Name: "title",
							Type: &ast.TypeNode{
								Kind: ast.TypePrimitive,
								Name: "string",
							},
							Nullable: false,
						},
					},
				},
			},
			wantSchemas: 2,
			wantErr:     false,
		},
		{
			name: "duplicate resource names",
			resources: []*ast.ResourceNode{
				{Name: "User"},
				{Name: "User"},
			},
			wantSchemas: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create compiled files from resources
			compiled := []*CompiledFile{
				{
					Path: "test.cdt",
					Hash: "test-hash",
					Program: &ast.Program{
						Resources: tt.resources,
					},
				},
			}

			extractor := NewSchemaExtractor()
			schemas, err := extractor.ExtractSchemas(compiled)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSchemas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(schemas) != tt.wantSchemas {
				t.Errorf("ExtractSchemas() got %d schemas, want %d", len(schemas), tt.wantSchemas)
			}

			// Verify schema contents
			if !tt.wantErr && len(tt.resources) > 0 {
				for _, res := range tt.resources {
					s, ok := schemas[res.Name]
					if !ok {
						t.Errorf("Expected schema for resource %s not found", res.Name)
						continue
					}

					if s.Name != res.Name {
						t.Errorf("Schema name = %s, want %s", s.Name, res.Name)
					}

					if len(s.Fields) != len(res.Fields) {
						t.Errorf("Schema fields count = %d, want %d", len(s.Fields), len(res.Fields))
					}
				}
			}
		})
	}
}

func TestSchemaExtractor_ExtractSchemasFromProgram(t *testing.T) {
	extractor := NewSchemaExtractor()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name: "id",
						Type: &ast.TypeNode{
							Kind: ast.TypePrimitive,
							Name: "uuid",
						},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "primary"},
						},
					},
					{
						Name: "name",
						Type: &ast.TypeNode{
							Kind: ast.TypePrimitive,
							Name: "string",
						},
						Nullable: false,
					},
				},
			},
		},
	}

	schemas, err := extractor.ExtractSchemasFromProgram(program, "test.cdt")
	if err != nil {
		t.Fatalf("ExtractSchemasFromProgram() error = %v", err)
	}

	if len(schemas) != 1 {
		t.Fatalf("Expected 1 schema, got %d", len(schemas))
	}

	userSchema, ok := schemas["User"]
	if !ok {
		t.Fatal("User schema not found")
	}

	if userSchema.Name != "User" {
		t.Errorf("Schema name = %s, want User", userSchema.Name)
	}

	if len(userSchema.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(userSchema.Fields))
	}

	// Verify field types
	if idField, ok := userSchema.Fields["id"]; ok {
		if idField.Type.BaseType != schema.TypeUUID {
			t.Errorf("id field type = %v, want UUID", idField.Type.BaseType)
		}
		if idField.Type.Nullable {
			t.Error("id field should not be nullable")
		}
	} else {
		t.Error("id field not found")
	}

	if nameField, ok := userSchema.Fields["name"]; ok {
		if nameField.Type.BaseType != schema.TypeString {
			t.Errorf("name field type = %v, want String", nameField.Type.BaseType)
		}
	} else {
		t.Error("name field not found")
	}
}
