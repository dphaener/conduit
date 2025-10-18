package migrate

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestGenerator_GenerateMigration_AddResource(t *testing.T) {
	gen := NewGenerator()

	oldSchemas := map[string]*schema.ResourceSchema{}
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:      "User",
			TableName: "users",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
					Annotations: []schema.Annotation{
						{Name: "primary"},
					},
				},
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeEmail,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	if migration == nil {
		t.Fatal("Expected migration, got nil")
	}

	// Verify migration properties
	if migration.Breaking {
		t.Error("Adding a resource should not be breaking")
	}

	if migration.DataLoss {
		t.Error("Adding a resource should not cause data loss")
	}

	// Verify up SQL contains CREATE TABLE
	if !strings.Contains(migration.Up, "CREATE TABLE") {
		t.Error("Up SQL should contain CREATE TABLE")
	}

	if !strings.Contains(migration.Up, "users") {
		t.Error("Up SQL should reference 'users' table")
	}

	// Verify down SQL contains DROP TABLE
	if !strings.Contains(migration.Down, "DROP TABLE") {
		t.Error("Down SQL should contain DROP TABLE")
	}
}

func TestGenerator_GenerateMigration_AddField(t *testing.T) {
	gen := NewGenerator()

	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:      "User",
			TableName: "users",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:      "User",
			TableName: "users",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeEmail,
						Nullable: true,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	if migration == nil {
		t.Fatal("Expected migration, got nil")
	}

	// Verify up SQL contains ALTER TABLE ADD COLUMN
	if !strings.Contains(migration.Up, "ALTER TABLE") {
		t.Error("Up SQL should contain ALTER TABLE")
	}

	if !strings.Contains(migration.Up, "ADD COLUMN") {
		t.Error("Up SQL should contain ADD COLUMN")
	}

	if !strings.Contains(migration.Up, "email") {
		t.Error("Up SQL should reference 'email' column")
	}

	// Verify down SQL contains DROP COLUMN
	if !strings.Contains(migration.Down, "DROP COLUMN") {
		t.Error("Down SQL should contain DROP COLUMN")
	}
}

func TestGenerator_GenerateMigration_DropField(t *testing.T) {
	gen := NewGenerator()

	emailField := &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}

	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:      "User",
			TableName: "users",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
				"email": emailField,
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:      "User",
			TableName: "users",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	if !migration.Breaking {
		t.Error("Dropping a field should be breaking")
	}

	if !migration.DataLoss {
		t.Error("Dropping a field should cause data loss")
	}

	// Verify SQL contains DROP COLUMN
	if !strings.Contains(migration.Up, "DROP COLUMN") {
		t.Error("Up SQL should contain DROP COLUMN")
	}

	// Verify down migration adds the column back
	if !strings.Contains(migration.Down, "ADD COLUMN") {
		t.Error("Down SQL should contain ADD COLUMN to restore the field")
	}
}

func TestGenerator_GenerateMigration_NoChanges(t *testing.T) {
	gen := NewGenerator()

	schemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:          "User",
			TableName:     "users",
			Fields:        map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	migration, err := gen.GenerateMigration(schemas, schemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	if migration != nil {
		t.Error("Expected nil migration for no changes")
	}
}

func TestGenerator_GenerateAddRelationship(t *testing.T) {
	gen := NewGenerator()

	oldSchemas := map[string]*schema.ResourceSchema{
		"Post": {
			Name:          "Post",
			TableName:     "posts",
			Fields:        map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"Post": {
			Name:      "Post",
			TableName: "posts",
			Fields:    map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{
				"author": {
					Type:           schema.RelationshipBelongsTo,
					FieldName:      "author",
					TargetResource: "User",
					ForeignKey:     "author_id",
					OnDelete:       schema.CascadeRestrict,
					OnUpdate:       schema.CascadeCascade,
				},
			},
		},
	}

	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	// Verify up SQL contains ADD CONSTRAINT
	if !strings.Contains(migration.Up, "ADD CONSTRAINT") {
		t.Error("Up SQL should contain ADD CONSTRAINT for foreign key")
	}

	if !strings.Contains(migration.Up, "FOREIGN KEY") {
		t.Error("Up SQL should contain FOREIGN KEY")
	}

	if !strings.Contains(migration.Up, "REFERENCES") {
		t.Error("Up SQL should contain REFERENCES")
	}

	if !strings.Contains(migration.Up, "ON DELETE RESTRICT") {
		t.Error("Up SQL should contain ON DELETE RESTRICT")
	}

	if !strings.Contains(migration.Up, "ON UPDATE CASCADE") {
		t.Error("Up SQL should contain ON UPDATE CASCADE")
	}

	// Verify down SQL contains DROP CONSTRAINT
	if !strings.Contains(migration.Down, "DROP CONSTRAINT") {
		t.Error("Down SQL should contain DROP CONSTRAINT")
	}
}

func TestGenerator_SQLComments(t *testing.T) {
	gen := NewGenerator()

	oldSchemas := map[string]*schema.ResourceSchema{}
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:          "User",
			TableName:     "users",
			Fields:        map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	// Verify comments are present
	if !strings.Contains(migration.Up, "-- Auto-generated migration") {
		t.Error("Up SQL should contain comment header")
	}

	if !strings.Contains(migration.Up, "-- Generated at:") {
		t.Error("Up SQL should contain timestamp comment")
	}

	if !strings.Contains(migration.Down, "-- Rollback migration") {
		t.Error("Down SQL should contain rollback comment")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserAccount", "user_account"},
		{"HTTPServer", "http_server"},
		{"APIKey", "api_key"},
		{"BlogPost", "blog_post"},
		{"ID", "id"},
		{"userId", "user_id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
