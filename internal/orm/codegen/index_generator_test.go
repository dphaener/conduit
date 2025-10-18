package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestIndexGenerator_GenerateIndexes_Basic(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}

	result := gen.GenerateIndexes(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateIndexes() returned %d indexes, want 1", len(result))
	}

	expected := `CREATE INDEX IF NOT EXISTS "idx_post_slug" ON "post" ("slug");`
	if result[0] != expected {
		t.Errorf("GenerateIndexes() = %q, want %q", result[0], expected)
	}
}

func TestIndexGenerator_GenerateIndexes_Unique(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("User")
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}

	result := gen.GenerateIndexes(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateIndexes() returned %d indexes, want 1", len(result))
	}

	expected := `CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_email_unique" ON "user" ("email");`
	if result[0] != expected {
		t.Errorf("GenerateIndexes() = %q, want %q", result[0], expected)
	}
}

func TestIndexGenerator_GenerateIndexes_Multiple(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}
	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}

	result := gen.GenerateIndexes(resource)

	if len(result) != 2 {
		t.Fatalf("GenerateIndexes() returned %d indexes, want 2", len(result))
	}

	// Results should be sorted
	expectedSlug := `CREATE INDEX IF NOT EXISTS "idx_post_slug" ON "post" ("slug");`
	expectedStatus := `CREATE INDEX IF NOT EXISTS "idx_post_status" ON "post" ("status");`

	found := make(map[string]bool)
	for _, stmt := range result {
		if stmt == expectedSlug {
			found["slug"] = true
		}
		if stmt == expectedStatus {
			found["status"] = true
		}
	}

	if !found["slug"] || !found["status"] {
		t.Errorf("GenerateIndexes() missing expected indexes. Got: %v", result)
	}
}

func TestIndexGenerator_GenerateForeignKeyIndexes(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	result := gen.GenerateForeignKeyIndexes(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateForeignKeyIndexes() returned %d indexes, want 1", len(result))
	}

	expected := `CREATE INDEX IF NOT EXISTS "idx_post_author_id" ON "post" ("author_id");`
	if result[0] != expected {
		t.Errorf("GenerateForeignKeyIndexes() = %q, want %q", result[0], expected)
	}
}

func TestIndexGenerator_GenerateForeignKeyIndexes_DefaultFK(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		// ForeignKey not specified, should default to user_id
	}

	result := gen.GenerateForeignKeyIndexes(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateForeignKeyIndexes() returned %d indexes, want 1", len(result))
	}

	expected := `CREATE INDEX IF NOT EXISTS "idx_post_user_id" ON "post" ("user_id");`
	if result[0] != expected {
		t.Errorf("GenerateForeignKeyIndexes() = %q, want %q", result[0], expected)
	}
}

func TestIndexGenerator_GenerateForeignKeyIndexes_SkipExisting(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["author"] = &schema.Field{
		Name: "author",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}}, // Already has index
	}
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	result := gen.GenerateForeignKeyIndexes(resource)

	// Should not create duplicate index since field already has @index
	if len(result) != 0 {
		t.Errorf("GenerateForeignKeyIndexes() should skip existing index, got %d indexes", len(result))
	}
}

func TestIndexGenerator_GenerateAllIndexes(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	result := gen.GenerateAllIndexes(resource)

	// Should have: 1 regular index + 1 unique index + 1 FK index = 3
	if len(result) != 3 {
		t.Fatalf("GenerateAllIndexes() returned %d indexes, want 3", len(result))
	}

	hasSlug := false
	hasEmail := false
	hasFK := false

	for _, stmt := range result {
		if strings.Contains(stmt, "idx_post_slug") {
			hasSlug = true
		}
		if strings.Contains(stmt, "idx_post_email_unique") {
			hasEmail = true
		}
		if strings.Contains(stmt, "idx_post_author_id") {
			hasFK = true
		}
	}

	if !hasSlug {
		t.Error("GenerateAllIndexes() missing slug index")
	}
	if !hasEmail {
		t.Error("GenerateAllIndexes() missing email unique index")
	}
	if !hasFK {
		t.Error("GenerateAllIndexes() missing FK index")
	}
}

func TestIndexGenerator_GenerateDropIndexes(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	result := gen.GenerateDropIndexes(resource)

	// Should drop all 3 indexes
	if len(result) != 3 {
		t.Fatalf("GenerateDropIndexes() returned %d statements, want 3", len(result))
	}

	hasSlugDrop := false
	hasEmailDrop := false
	hasFKDrop := false

	for _, stmt := range result {
		if strings.Contains(stmt, `DROP INDEX IF EXISTS "idx_post_slug"`) {
			hasSlugDrop = true
		}
		if strings.Contains(stmt, `DROP INDEX IF EXISTS "idx_post_email_unique"`) {
			hasEmailDrop = true
		}
		if strings.Contains(stmt, `DROP INDEX IF EXISTS "idx_post_author_id"`) {
			hasFKDrop = true
		}
	}

	if !hasSlugDrop {
		t.Error("GenerateDropIndexes() missing slug drop")
	}
	if !hasEmailDrop {
		t.Error("GenerateDropIndexes() missing email drop")
	}
	if !hasFKDrop {
		t.Error("GenerateDropIndexes() missing FK drop")
	}
}

func TestIndexGenerator_GenerateCompositeIndex(t *testing.T) {
	gen := NewIndexGenerator()

	result := gen.GenerateCompositeIndex("post", "idx_post_author_status", []string{"author_id", "status"}, false)

	expected := `CREATE INDEX IF NOT EXISTS "idx_post_author_status" ON "post" ("author_id", "status");`
	if result != expected {
		t.Errorf("GenerateCompositeIndex() = %q, want %q", result, expected)
	}
}

func TestIndexGenerator_GenerateCompositeIndex_Unique(t *testing.T) {
	gen := NewIndexGenerator()

	result := gen.GenerateCompositeIndex("user", "idx_user_email_tenant", []string{"email", "tenant_id"}, true)

	expected := `CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_email_tenant" ON "user" ("email", "tenant_id");`
	if result != expected {
		t.Errorf("GenerateCompositeIndex() = %q, want %q", result, expected)
	}
}

func TestIndexGenerator_GeneratePartialIndex(t *testing.T) {
	gen := NewIndexGenerator()

	result := gen.GeneratePartialIndex("post", "idx_post_published", "published_at", "published_at IS NOT NULL")

	expected := `CREATE INDEX IF NOT EXISTS "idx_post_published" ON "post" ("published_at") WHERE published_at IS NOT NULL;`
	if result != expected {
		t.Errorf("GeneratePartialIndex() = %q, want %q", result, expected)
	}
}

func TestIndexGenerator_GenerateExpressionIndex(t *testing.T) {
	gen := NewIndexGenerator()

	result := gen.GenerateExpressionIndex("user", "idx_user_email_lower", "LOWER(email)")

	expected := `CREATE INDEX IF NOT EXISTS "idx_user_email_lower" ON "user" (LOWER(email));`
	if result != expected {
		t.Errorf("GenerateExpressionIndex() = %q, want %q", result, expected)
	}
}

func TestIndexGenerator_NoDuplicates(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["author_id"] = &schema.Field{
		Name: "author_id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "index"}},
	}
	// Same field as FK
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author_id",
		ForeignKey:     "author_id",
	}

	result := gen.GenerateAllIndexes(resource)

	// Should only have 1 index (field index), FK index should be skipped
	if len(result) != 1 {
		t.Errorf("GenerateAllIndexes() should deduplicate indexes, got %d", len(result))
	}
}

func TestIndexGenerator_HasManyNotIndexed(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("User")
	resource.Relationships["posts"] = &schema.Relationship{
		Type:           schema.RelationshipHasMany,
		TargetResource: "Post",
		FieldName:      "posts",
	}

	result := gen.GenerateForeignKeyIndexes(resource)

	// has_many relationships should not create indexes
	if len(result) != 0 {
		t.Errorf("GenerateForeignKeyIndexes() should not index has_many, got %d indexes", len(result))
	}
}

func TestIndexGenerator_MultipleRelationships(t *testing.T) {
	gen := NewIndexGenerator()

	resource := schema.NewResourceSchema("Comment")
	resource.Relationships["post"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "Post",
		FieldName:      "post",
	}
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	result := gen.GenerateForeignKeyIndexes(resource)

	// Should create 2 indexes
	if len(result) != 2 {
		t.Fatalf("GenerateForeignKeyIndexes() returned %d indexes, want 2", len(result))
	}

	hasPost := false
	hasAuthor := false

	for _, stmt := range result {
		if strings.Contains(stmt, "post_id") {
			hasPost = true
		}
		if strings.Contains(stmt, "user_id") {
			hasAuthor = true
		}
	}

	if !hasPost || !hasAuthor {
		t.Errorf("GenerateForeignKeyIndexes() missing expected indexes. Got: %v", result)
	}
}

func TestIndexGenerator_SQLInjectionPrevention(t *testing.T) {
	gen := NewIndexGenerator()

	// NOTE: toSnakeCase() provides first layer of defense by sanitizing identifiers
	// QuoteIdentifier() provides second layer by escaping double quotes
	// These tests verify both layers work correctly

	t.Run("All identifiers quoted in CREATE INDEX", func(t *testing.T) {
		resource := schema.NewResourceSchema("Post")
		resource.Fields["slug"] = &schema.Field{
			Name: "slug",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "index"}},
		}

		result := gen.GenerateIndexes(resource)

		// All identifiers should be quoted
		if !strings.Contains(result[0], `CREATE INDEX IF NOT EXISTS "idx_post_slug"`) {
			t.Errorf("Index name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `ON "post"`) {
			t.Errorf("Table name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `("slug")`) {
			t.Errorf("Column name should be quoted, got: %s", result[0])
		}
	})

	t.Run("UNIQUE index quotes all identifiers", func(t *testing.T) {
		resource := schema.NewResourceSchema("User")
		resource.Fields["email"] = &schema.Field{
			Name: "email",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "unique"}},
		}

		result := gen.GenerateIndexes(resource)

		// All identifiers should be quoted
		if !strings.Contains(result[0], `CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_email_unique"`) {
			t.Errorf("Index name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `ON "user"`) {
			t.Errorf("Table name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `("email")`) {
			t.Errorf("Column name should be quoted, got: %s", result[0])
		}
	})

	t.Run("Foreign key index quotes all identifiers", func(t *testing.T) {
		resource := schema.NewResourceSchema("Post")
		resource.Relationships["author"] = &schema.Relationship{
			Type:           schema.RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			ForeignKey:     "author_id",
		}

		result := gen.GenerateForeignKeyIndexes(resource)

		// All identifiers should be quoted
		if !strings.Contains(result[0], `CREATE INDEX IF NOT EXISTS "idx_post_author_id"`) {
			t.Errorf("Index name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `ON "post"`) {
			t.Errorf("Table name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `("author_id")`) {
			t.Errorf("Column name should be quoted, got: %s", result[0])
		}
	})

	t.Run("Composite index quotes all columns", func(t *testing.T) {
		result := gen.GenerateCompositeIndex("posts", "idx_post_author_status", []string{"author_id", "status"}, false)

		// All identifiers should be quoted including each column
		requiredQuoted := []string{
			`CREATE INDEX IF NOT EXISTS "idx_post_author_status"`,
			`ON "posts"`,
			`("author_id", "status")`,
		}

		for _, expected := range requiredQuoted {
			if !strings.Contains(result, expected) {
				t.Errorf("Missing quoted identifier %q in: %s", expected, result)
			}
		}
	})

	t.Run("DROP INDEX quotes identifier", func(t *testing.T) {
		resource := schema.NewResourceSchema("Post")
		resource.Fields["slug"] = &schema.Field{
			Name: "slug",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "index"}},
		}

		result := gen.GenerateDropIndexes(resource)

		// Index name should be quoted in DROP
		if !strings.Contains(result[0], `DROP INDEX IF EXISTS "idx_post_slug"`) {
			t.Errorf("Index name should be quoted in DROP, got: %s", result[0])
		}
	})

	t.Run("Partial index quotes all identifiers", func(t *testing.T) {
		result := gen.GeneratePartialIndex("posts", "idx_post_published", "published_at", "published_at IS NOT NULL")

		// All identifiers should be quoted
		requiredQuoted := []string{
			`CREATE INDEX IF NOT EXISTS "idx_post_published"`,
			`ON "posts"`,
			`("published_at")`,
			`WHERE published_at IS NOT NULL`, // WHERE clause not quoted
		}

		for _, expected := range requiredQuoted {
			if !strings.Contains(result, expected) {
				t.Errorf("Missing expected content %q in: %s", expected, result)
			}
		}
	})

	t.Run("Expression index quotes table and index names", func(t *testing.T) {
		result := gen.GenerateExpressionIndex("users", "idx_user_email_lower", "LOWER(email)")

		// Table and index names should be quoted, expression preserved
		if !strings.Contains(result, `CREATE INDEX IF NOT EXISTS "idx_user_email_lower"`) {
			t.Errorf("Index name should be quoted, got: %s", result)
		}
		if !strings.Contains(result, `ON "users"`) {
			t.Errorf("Table name should be quoted, got: %s", result)
		}
		if !strings.Contains(result, "(LOWER(email))") {
			t.Errorf("Expression should be preserved, got: %s", result)
		}
	})

	t.Run("Double quotes in identifiers are escaped", func(t *testing.T) {
		// This tests QuoteIdentifier's escaping of embedded double quotes
		result := gen.GenerateCompositeIndex(`table"name`, `idx"name`, []string{`col"1`, `col"2`}, false)

		// Double quotes should be escaped by doubling them
		if !strings.Contains(result, `"table""name"`) {
			t.Errorf("Table name with quotes should be escaped, got: %s", result)
		}
		if !strings.Contains(result, `"idx""name"`) {
			t.Errorf("Index name with quotes should be escaped, got: %s", result)
		}
		if !strings.Contains(result, `"col""1"`) {
			t.Errorf("Column 1 with quotes should be escaped, got: %s", result)
		}
		if !strings.Contains(result, `"col""2"`) {
			t.Errorf("Column 2 with quotes should be escaped, got: %s", result)
		}
	})
}
