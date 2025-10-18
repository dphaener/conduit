package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestDDLGenerator_GenerateCreateTable_Simple(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeUUID,
			Nullable: false,
		},
		Annotations: []schema.Annotation{
			{Name: "primary"},
			{Name: "auto"},
		},
	}
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	// Check for key components (now with quoted identifiers)
	expected := []string{
		`CREATE TABLE IF NOT EXISTS "post"`,
		`"id" UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY`,
		`"title" VARCHAR(255) NOT NULL`,
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("GenerateCreateTable() missing %q\nGot:\n%s", exp, result)
		}
	}
}

func TestDDLGenerator_GenerateCreateTable_AllTypes(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["str"] = &schema.Field{
		Name: "str",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
	}
	resource.Fields["text"] = &schema.Field{
		Name: "text",
		Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true},
	}
	resource.Fields["num"] = &schema.Field{
		Name: "num",
		Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
	}
	resource.Fields["big"] = &schema.Field{
		Name: "big",
		Type: &schema.TypeSpec{BaseType: schema.TypeBigInt, Nullable: false},
	}
	resource.Fields["dec"] = &schema.Field{
		Name: "dec",
		Type: &schema.TypeSpec{
			BaseType:  schema.TypeDecimal,
			Nullable:  false,
			Precision: intPtr(10),
			Scale:     intPtr(2),
		},
	}
	resource.Fields["flag"] = &schema.Field{
		Name: "flag",
		Type: &schema.TypeSpec{BaseType: schema.TypeBool, Nullable: false},
	}
	resource.Fields["ts"] = &schema.Field{
		Name: "ts",
		Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
		Annotations: []schema.Annotation{{Name: "auto"}},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	expected := []string{
		`"id" UUID NOT NULL PRIMARY KEY`,
		`"str" VARCHAR(255) NOT NULL`,
		`"text" TEXT NULL`,
		`"num" INTEGER NOT NULL`,
		`"big" BIGINT NOT NULL`,
		`"dec" NUMERIC(10,2) NOT NULL`,
		`"flag" BOOLEAN NOT NULL`,
		`"ts" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP`,
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("GenerateCreateTable() missing %q\nGot:\n%s", exp, result)
		}
	}
}

func TestDDLGenerator_GenerateCreateTable_WithDefaults(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
			Default:  "draft",
		},
	}
	resource.Fields["count"] = &schema.Field{
		Name: "count",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
			Default:  0,
		},
	}
	resource.Fields["active"] = &schema.Field{
		Name: "active",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeBool,
			Nullable: false,
			Default:  true,
		},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	expected := []string{
		`"status" VARCHAR(255) NOT NULL DEFAULT 'draft'`,
		`"count" INTEGER NOT NULL DEFAULT 0`,
		`"active" BOOLEAN NOT NULL DEFAULT TRUE`,
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("GenerateCreateTable() missing %q\nGot:\n%s", exp, result)
		}
	}
}

func TestDDLGenerator_GenerateEnumType(t *testing.T) {
	gen := NewDDLGenerator()

	result := gen.GenerateEnumType("Post", "status", []string{"draft", "published", "archived"})

	expected := `CREATE TYPE "post_status_enum" AS ENUM ('draft', 'published', 'archived');`

	if result != expected {
		t.Errorf("GenerateEnumType() = %q, want %q", result, expected)
	}
}

func TestDDLGenerator_GenerateEnumType_SQLInjectionPrevention(t *testing.T) {
	gen := NewDDLGenerator()

	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "enum with single quote",
			values:   []string{"it's", "draft"},
			expected: `CREATE TYPE "post_status_enum" AS ENUM ('it''s', 'draft');`,
		},
		{
			name:     "enum with multiple quotes",
			values:   []string{"O'Reilly's", "test"},
			expected: `CREATE TYPE "post_status_enum" AS ENUM ('O''Reilly''s', 'test');`,
		},
		{
			name:     "SQL injection attempt",
			values:   []string{"'; DROP TYPE post_status_enum; --", "normal"},
			expected: `CREATE TYPE "post_status_enum" AS ENUM ('''; DROP TYPE post_status_enum; --', 'normal');`,
		},
		{
			name:     "empty quotes",
			values:   []string{"''", "test"},
			expected: `CREATE TYPE "post_status_enum" AS ENUM ('''''', 'test');`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateEnumType("Post", "status", tt.values)
			if result != tt.expected {
				t.Errorf("GenerateEnumType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDDLGenerator_GenerateEnumTypes(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{
			BaseType:   schema.TypeEnum,
			Nullable:   false,
			EnumValues: []string{"draft", "published"},
		},
	}
	resource.Fields["visibility"] = &schema.Field{
		Name: "visibility",
		Type: &schema.TypeSpec{
			BaseType:   schema.TypeEnum,
			Nullable:   false,
			EnumValues: []string{"public", "private"},
		},
	}

	result := gen.GenerateEnumTypes(resource)

	if len(result) != 2 {
		t.Fatalf("GenerateEnumTypes() returned %d types, want 2", len(result))
	}

	// Results should be sorted
	expectedStatus := `CREATE TYPE "post_status_enum" AS ENUM ('draft', 'published');`
	expectedVisibility := `CREATE TYPE "post_visibility_enum" AS ENUM ('public', 'private');`

	found := make(map[string]bool)
	for _, stmt := range result {
		if stmt == expectedStatus {
			found["status"] = true
		}
		if stmt == expectedVisibility {
			found["visibility"] = true
		}
	}

	if !found["status"] || !found["visibility"] {
		t.Errorf("GenerateEnumTypes() missing expected enum types. Got: %v", result)
	}
}

func TestDDLGenerator_GenerateSchema(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}, {Name: "auto"}},
	}
	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{
			BaseType:   schema.TypeEnum,
			Nullable:   false,
			EnumValues: []string{"draft", "published"},
		},
	}
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
	}

	result, err := gen.GenerateSchema(resource)
	if err != nil {
		t.Fatalf("GenerateSchema() error = %v", err)
	}

	expected := []string{
		`CREATE TYPE "post_status_enum"`,
		`CREATE TABLE IF NOT EXISTS "post"`,
		`"id" UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY`,
		`"status" post_status_enum NOT NULL`,
		`"title" VARCHAR(255) NOT NULL`,
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("GenerateSchema() missing %q\nGot:\n%s", exp, result)
		}
	}
}

func TestDDLGenerator_GenerateDropTable(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")

	result := gen.GenerateDropTable(resource)
	expected := `DROP TABLE IF EXISTS "post" CASCADE;`

	if result != expected {
		t.Errorf("GenerateDropTable() = %q, want %q", result, expected)
	}
}

func TestDDLGenerator_GenerateDropEnumTypes(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{
			EnumValues: []string{"draft", "published"},
		},
	}

	result := gen.GenerateDropEnumTypes(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateDropEnumTypes() returned %d statements, want 1", len(result))
	}

	expected := `DROP TYPE IF EXISTS "post_status_enum";`
	if result[0] != expected {
		t.Errorf("GenerateDropEnumTypes() = %q, want %q", result[0], expected)
	}
}

func TestDDLGenerator_ColumnOrdering(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Test")

	// Add fields in random order
	resource.Fields["text_field"] = &schema.Field{
		Name: "text_field",
		Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true},
	}
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["count"] = &schema.Field{
		Name: "count",
		Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
	}
	resource.Fields["json_data"] = &schema.Field{
		Name: "json_data",
		Type: &schema.TypeSpec{BaseType: schema.TypeJSONB, Nullable: true},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	// Primary key should be first
	lines := strings.Split(result, "\n")
	if !strings.Contains(lines[1], `"id" UUID`) {
		t.Errorf("Primary key should be first column, got:\n%s", result)
	}

	// Fixed-length types should come before variable-length
	idPos := strings.Index(result, `"id" UUID`)
	countPos := strings.Index(result, `"count" INTEGER`)
	textPos := strings.Index(result, `"text_field" TEXT`)
	jsonPos := strings.Index(result, `"json_data" JSONB`)

	if idPos > countPos {
		t.Error("Primary key should come before int")
	}
	if countPos > textPos {
		t.Error("Int should come before text")
	}
	if textPos > jsonPos {
		t.Error("Text should come before jsonb")
	}
}

func TestDDLGenerator_ArrayTypes(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["tags"] = &schema.Field{
		Name: "tags",
		Type: &schema.TypeSpec{
			ArrayElement: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: false,
			},
		},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	if !strings.Contains(result, `"tags" VARCHAR(255)[]`) {
		t.Errorf("GenerateCreateTable() should contain array type, got:\n%s", result)
	}
}

func TestDDLGenerator_NilResource(t *testing.T) {
	gen := NewDDLGenerator()

	_, err := gen.GenerateCreateTable(nil)
	if err == nil {
		t.Error("GenerateCreateTable() should error on nil resource")
	}
}

func TestDDLGenerator_CustomTableName(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.TableName = "blog_posts"
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	if !strings.Contains(result, `CREATE TABLE IF NOT EXISTS "blog_posts"`) {
		t.Errorf("GenerateCreateTable() should use custom table name, got:\n%s", result)
	}
}

func TestDDLGenerator_ColumnOrdering_AllTypes(t *testing.T) {
	gen := NewDDLGenerator()

	resource := schema.NewResourceSchema("Test")

	// Add all different types to test ordering priority
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	resource.Fields["flag"] = &schema.Field{
		Name: "flag",
		Type: &schema.TypeSpec{BaseType: schema.TypeBool, Nullable: false},
	}
	resource.Fields["count"] = &schema.Field{
		Name: "count",
		Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
	}
	resource.Fields["big_num"] = &schema.Field{
		Name: "big_num",
		Type: &schema.TypeSpec{BaseType: schema.TypeBigInt, Nullable: false},
	}
	resource.Fields["price"] = &schema.Field{
		Name: "price",
		Type: &schema.TypeSpec{BaseType: schema.TypeDecimal, Nullable: false},
	}
	resource.Fields["ratio"] = &schema.Field{
		Name: "ratio",
		Type: &schema.TypeSpec{BaseType: schema.TypeFloat, Nullable: false},
	}
	resource.Fields["ulid"] = &schema.Field{
		Name: "ulid",
		Type: &schema.TypeSpec{BaseType: schema.TypeULID, Nullable: false},
	}
	resource.Fields["birthday"] = &schema.Field{
		Name: "birthday",
		Type: &schema.TypeSpec{BaseType: schema.TypeDate, Nullable: false},
	}
	resource.Fields["wake_time"] = &schema.Field{
		Name: "wake_time",
		Type: &schema.TypeSpec{BaseType: schema.TypeTime, Nullable: false},
	}
	resource.Fields["created"] = &schema.Field{
		Name: "created",
		Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{BaseType: schema.TypeEmail, Nullable: false},
	}
	resource.Fields["url"] = &schema.Field{
		Name: "url",
		Type: &schema.TypeSpec{BaseType: schema.TypeURL, Nullable: false},
	}
	resource.Fields["phone"] = &schema.Field{
		Name: "phone",
		Type: &schema.TypeSpec{BaseType: schema.TypePhone, Nullable: false},
	}
	resource.Fields["short_str"] = &schema.Field{
		Name: "short_str",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
			Length:   intPtr(20),
		},
	}
	resource.Fields["long_str"] = &schema.Field{
		Name: "long_str",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
			Length:   intPtr(500),
		},
	}
	resource.Fields["content"] = &schema.Field{
		Name: "content",
		Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: false},
	}
	resource.Fields["markdown"] = &schema.Field{
		Name: "markdown",
		Type: &schema.TypeSpec{BaseType: schema.TypeMarkdown, Nullable: false},
	}
	resource.Fields["json_data"] = &schema.Field{
		Name: "json_data",
		Type: &schema.TypeSpec{BaseType: schema.TypeJSON, Nullable: true},
	}
	resource.Fields["jsonb_data"] = &schema.Field{
		Name: "jsonb_data",
		Type: &schema.TypeSpec{BaseType: schema.TypeJSONB, Nullable: true},
	}
	resource.Fields["tags"] = &schema.Field{
		Name: "tags",
		Type: &schema.TypeSpec{
			ArrayElement: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: false,
			},
		},
	}
	resource.Fields["metadata"] = &schema.Field{
		Name: "metadata",
		Type: &schema.TypeSpec{
			HashKey: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			HashValue: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		},
	}

	result, err := gen.GenerateCreateTable(resource)
	if err != nil {
		t.Fatalf("GenerateCreateTable() error = %v", err)
	}

	// Primary key should be first
	lines := strings.Split(result, "\n")
	if !strings.Contains(lines[1], `"id" UUID`) {
		t.Errorf("Primary key should be first, got:\n%s", result)
	}

	// Verify general ordering: primary, fixed-length, then variable-length
	idPos := strings.Index(result, `"id" UUID`)
	flagPos := strings.Index(result, `"flag" BOOLEAN`)
	textPos := strings.Index(result, `"content" TEXT`)
	jsonPos := strings.Index(result, `"jsonb_data" JSONB`)

	if idPos > flagPos {
		t.Errorf("Primary key should come before bool")
	}
	if flagPos > textPos {
		t.Errorf("Bool should come before text")
	}
	if textPos > jsonPos {
		t.Errorf("Text should come before jsonb")
	}
}
