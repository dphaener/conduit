package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestConstraintGenerator_GenerateCheckConstraints_Min(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Product")
	resource.Fields["price"] = &schema.Field{
		Name: "price",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeDecimal,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 0},
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	expected := `ALTER TABLE "product" ADD CONSTRAINT "product_price_min" CHECK ("price" >= 0);`
	if result[0] != expected {
		t.Errorf("GenerateCheckConstraints() = %q, want %q", result[0], expected)
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_Max(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Product")
	resource.Fields["rating"] = &schema.Field{
		Name: "rating",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMax, Value: 5},
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	expected := `ALTER TABLE "product" ADD CONSTRAINT "product_rating_max" CHECK ("rating" <= 5);`
	if result[0] != expected {
		t.Errorf("GenerateCheckConstraints() = %q, want %q", result[0], expected)
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_TextLength(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 5},
			{Type: schema.ConstraintMax, Value: 200},
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 2 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 2", len(result))
	}

	expectedMin := `ALTER TABLE "post" ADD CONSTRAINT "post_title_min" CHECK (LENGTH("title") >= 5);`
	expectedMax := `ALTER TABLE "post" ADD CONSTRAINT "post_title_max" CHECK (LENGTH("title") <= 200);`

	found := make(map[string]bool)
	for _, stmt := range result {
		if stmt == expectedMin {
			found["min"] = true
		}
		if stmt == expectedMax {
			found["max"] = true
		}
	}

	if !found["min"] || !found["max"] {
		t.Errorf("GenerateCheckConstraints() missing expected constraints. Got: %v", result)
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_Pattern(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintPattern, Value: "^[a-z0-9-]+$"},
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	expected := `ALTER TABLE "post" ADD CONSTRAINT "post_slug_pattern" CHECK ("slug" ~ '^[a-z0-9-]+$');`
	if result[0] != expected {
		t.Errorf("GenerateCheckConstraints() = %q, want %q", result[0], expected)
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_Email(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("User")
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	if !strings.Contains(result[0], `CHECK ("email" ~`) {
		t.Errorf("GenerateCheckConstraints() should contain email regex check, got: %q", result[0])
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_URL(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Link")
	resource.Fields["url"] = &schema.Field{
		Name: "url",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeURL,
			Nullable: false,
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	if !strings.Contains(result[0], `CHECK ("url" ~ '^https?://.+')`) {
		t.Errorf("GenerateCheckConstraints() should contain URL regex check, got: %q", result[0])
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_Phone(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Contact")
	resource.Fields["phone"] = &schema.Field{
		Name: "phone",
		Type: &schema.TypeSpec{
			BaseType: schema.TypePhone,
			Nullable: false,
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	if !strings.Contains(result[0], `CHECK ("phone" ~`) {
		t.Errorf("GenerateCheckConstraints() should contain phone regex check, got: %q", result[0])
	}
}

func TestConstraintGenerator_GenerateForeignKeyConstraints(t *testing.T) {
	gen := NewConstraintGenerator()

	// Create target resource (User)
	userResource := schema.NewResourceSchema("User")
	userResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	// Create resource with FK (Post)
	postResource := schema.NewResourceSchema("Post")
	postResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
		OnDelete:       schema.CascadeRestrict,
		OnUpdate:       schema.CascadeNoAction,
	}

	registry := map[string]*schema.ResourceSchema{
		"User": userResource,
		"Post": postResource,
	}

	result, err := gen.GenerateForeignKeyConstraints(postResource, registry)
	if err != nil {
		t.Fatalf("GenerateForeignKeyConstraints() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("GenerateForeignKeyConstraints() returned %d constraints, want 1", len(result))
	}

	expected := []string{
		`ALTER TABLE "post" ADD CONSTRAINT "post_author_id_fkey"`,
		`FOREIGN KEY ("author_id") REFERENCES "user" ("id")`,
		"ON DELETE RESTRICT",
		"ON UPDATE NO ACTION",
	}

	for _, exp := range expected {
		if !strings.Contains(result[0], exp) {
			t.Errorf("GenerateForeignKeyConstraints() missing %q\nGot: %s", exp, result[0])
		}
	}
}

func TestConstraintGenerator_GenerateForeignKeyConstraints_SetNull(t *testing.T) {
	gen := NewConstraintGenerator()

	userResource := schema.NewResourceSchema("User")
	userResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	postResource := schema.NewResourceSchema("Post")
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
		OnDelete:       schema.CascadeSetNull,
		OnUpdate:       schema.CascadeNoAction,
	}

	registry := map[string]*schema.ResourceSchema{
		"User": userResource,
		"Post": postResource,
	}

	result, err := gen.GenerateForeignKeyConstraints(postResource, registry)
	if err != nil {
		t.Fatalf("GenerateForeignKeyConstraints() error = %v", err)
	}

	if !strings.Contains(result[0], "ON DELETE SET NULL") {
		t.Errorf("GenerateForeignKeyConstraints() should contain SET NULL, got: %s", result[0])
	}
}

func TestConstraintGenerator_GenerateForeignKeyConstraints_Cascade(t *testing.T) {
	gen := NewConstraintGenerator()

	userResource := schema.NewResourceSchema("User")
	userResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	postResource := schema.NewResourceSchema("Post")
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		OnDelete:       schema.CascadeCascade,
		OnUpdate:       schema.CascadeCascade,
	}

	registry := map[string]*schema.ResourceSchema{
		"User": userResource,
		"Post": postResource,
	}

	result, err := gen.GenerateForeignKeyConstraints(postResource, registry)
	if err != nil {
		t.Fatalf("GenerateForeignKeyConstraints() error = %v", err)
	}

	if !strings.Contains(result[0], "ON DELETE CASCADE") {
		t.Errorf("GenerateForeignKeyConstraints() should contain ON DELETE CASCADE, got: %s", result[0])
	}
	if !strings.Contains(result[0], "ON UPDATE CASCADE") {
		t.Errorf("GenerateForeignKeyConstraints() should contain ON UPDATE CASCADE, got: %s", result[0])
	}
}

func TestConstraintGenerator_GenerateUniqueConstraints(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("User")
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}
	resource.Fields["username"] = &schema.Field{
		Name: "username",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}

	result := gen.GenerateUniqueConstraints(resource)

	if len(result) != 2 {
		t.Fatalf("GenerateUniqueConstraints() returned %d constraints, want 2", len(result))
	}

	expectedEmail := `ALTER TABLE "user" ADD CONSTRAINT "user_email_unique" UNIQUE ("email");`
	expectedUsername := `ALTER TABLE "user" ADD CONSTRAINT "user_username_unique" UNIQUE ("username");`

	found := make(map[string]bool)
	for _, stmt := range result {
		if stmt == expectedEmail {
			found["email"] = true
		}
		if stmt == expectedUsername {
			found["username"] = true
		}
	}

	if !found["email"] || !found["username"] {
		t.Errorf("GenerateUniqueConstraints() missing expected constraints. Got: %v", result)
	}
}

func TestConstraintGenerator_GenerateAllConstraints(t *testing.T) {
	gen := NewConstraintGenerator()

	userResource := schema.NewResourceSchema("User")
	userResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	postResource := schema.NewResourceSchema("Post")
	postResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}
	postResource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 5},
		},
	}
	postResource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		OnDelete:       schema.CascadeRestrict,
	}

	registry := map[string]*schema.ResourceSchema{
		"User": userResource,
		"Post": postResource,
	}

	result, err := gen.GenerateAllConstraints(postResource, registry)
	if err != nil {
		t.Fatalf("GenerateAllConstraints() error = %v", err)
	}

	// Should have: 1 CHECK + 1 UNIQUE + 1 FK = 3 constraints
	if len(result) < 3 {
		t.Errorf("GenerateAllConstraints() returned %d constraints, want at least 3", len(result))
	}

	// Check that all types are present
	hasCheck := false
	hasUnique := false
	hasFK := false

	for _, stmt := range result {
		if strings.Contains(stmt, "CHECK") {
			hasCheck = true
		}
		if strings.Contains(stmt, "UNIQUE") {
			hasUnique = true
		}
		if strings.Contains(stmt, "FOREIGN KEY") {
			hasFK = true
		}
	}

	if !hasCheck {
		t.Error("GenerateAllConstraints() should contain CHECK constraint")
	}
	if !hasUnique {
		t.Error("GenerateAllConstraints() should contain UNIQUE constraint")
	}
	if !hasFK {
		t.Error("GenerateAllConstraints() should contain FOREIGN KEY constraint")
	}
}

func TestConstraintGenerator_GenerateDropConstraints(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 5},
		},
		Annotations: []schema.Annotation{{Name: "unique"}},
	}

	result := gen.GenerateDropConstraints(resource, nil)

	// Should have DROP for both min constraint and unique constraint
	if len(result) < 2 {
		t.Fatalf("GenerateDropConstraints() returned %d statements, want at least 2", len(result))
	}

	hasMinDrop := false
	hasUniqueDrop := false

	for _, stmt := range result {
		if strings.Contains(stmt, "post_title_min") {
			hasMinDrop = true
		}
		if strings.Contains(stmt, "post_title_unique") {
			hasUniqueDrop = true
		}
	}

	if !hasMinDrop {
		t.Error("GenerateDropConstraints() should drop min constraint")
	}
	if !hasUniqueDrop {
		t.Error("GenerateDropConstraints() should drop unique constraint")
	}
}

func TestConstraintGenerator_UnknownTargetResource(t *testing.T) {
	gen := NewConstraintGenerator()

	postResource := schema.NewResourceSchema("Post")
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "UnknownResource",
		FieldName:      "author",
	}

	registry := map[string]*schema.ResourceSchema{}

	_, err := gen.GenerateForeignKeyConstraints(postResource, registry)
	if err == nil {
		t.Error("GenerateForeignKeyConstraints() should error on unknown target resource")
	}
}

func TestConstraintGenerator_GenerateCheckConstraints_PatternWithQuotes(t *testing.T) {
	gen := NewConstraintGenerator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintPattern, Value: "^it's-test$"},
		},
	}

	result := gen.GenerateCheckConstraints(resource)

	if len(result) != 1 {
		t.Fatalf("GenerateCheckConstraints() returned %d constraints, want 1", len(result))
	}

	// Should escape the single quote
	if !strings.Contains(result[0], "it''s-test") {
		t.Errorf("GenerateCheckConstraints() should escape quotes, got: %q", result[0])
	}
}

func TestConstraintGenerator_NoAction(t *testing.T) {
	gen := NewConstraintGenerator()

	userResource := schema.NewResourceSchema("User")
	userResource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}},
	}

	postResource := schema.NewResourceSchema("Post")
	postResource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		OnDelete:       schema.CascadeNoAction,
		OnUpdate:       schema.CascadeNoAction,
	}

	registry := map[string]*schema.ResourceSchema{
		"User": userResource,
		"Post": postResource,
	}

	result, err := gen.GenerateForeignKeyConstraints(postResource, registry)
	if err != nil {
		t.Fatalf("GenerateForeignKeyConstraints() error = %v", err)
	}

	if !strings.Contains(result[0], "ON DELETE NO ACTION") {
		t.Errorf("GenerateForeignKeyConstraints() should contain NO ACTION, got: %s", result[0])
	}
	if !strings.Contains(result[0], "ON UPDATE NO ACTION") {
		t.Errorf("GenerateForeignKeyConstraints() should contain NO ACTION, got: %s", result[0])
	}
}

func TestConstraintGenerator_SQLInjectionPrevention(t *testing.T) {
	gen := NewConstraintGenerator()

	// NOTE: toSnakeCase() provides first layer of defense by sanitizing identifiers
	// QuoteIdentifier() provides second layer by escaping double quotes
	// These tests verify both layers work correctly

	t.Run("Table name with double quotes", func(t *testing.T) {
		resource := schema.NewResourceSchema("Product")
		resource.TableName = `user"table"name` // toSnakeCase strips quotes, becomes "usertablename"
		resource.Fields["price"] = &schema.Field{
			Name: "price",
			Type: &schema.TypeSpec{
				BaseType: schema.TypeDecimal,
				Nullable: false,
			},
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
			},
		}

		result := gen.GenerateCheckConstraints(resource)

		// All identifiers should be quoted
		if !strings.Contains(result[0], `ALTER TABLE "user""table""name"`) {
			t.Errorf("Should quote and escape table name with embedded quotes, got: %s", result[0])
		}
		// Verify SQL is syntactically correct with quoted identifiers
		if !strings.Contains(result[0], `ADD CONSTRAINT`) {
			t.Errorf("SQL statement malformed, got: %s", result[0])
		}
	})

	t.Run("Column names are quoted in CHECK constraints", func(t *testing.T) {
		resource := schema.NewResourceSchema("Product")
		resource.Fields["price"] = &schema.Field{
			Name: "price",
			Type: &schema.TypeSpec{
				BaseType: schema.TypeDecimal,
				Nullable: false,
			},
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
			},
		}

		result := gen.GenerateCheckConstraints(resource)

		// Column name in CHECK clause should be quoted
		if !strings.Contains(result[0], `CHECK ("price" >= 0)`) {
			t.Errorf("Column name should be quoted in CHECK, got: %s", result[0])
		}
	})

	t.Run("All identifiers quoted in FOREIGN KEY", func(t *testing.T) {
		userResource := schema.NewResourceSchema("User")
		userResource.Fields["id"] = &schema.Field{
			Name: "id",
			Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
			Annotations: []schema.Annotation{{Name: "primary"}},
		}

		postResource := schema.NewResourceSchema("Post")
		postResource.Relationships["author"] = &schema.Relationship{
			Type:           schema.RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			ForeignKey:     "author_id",
			OnDelete:       schema.CascadeRestrict,
		}

		registry := map[string]*schema.ResourceSchema{
			"User": userResource,
			"Post": postResource,
		}

		result, err := gen.GenerateForeignKeyConstraints(postResource, registry)
		if err != nil {
			t.Fatalf("GenerateForeignKeyConstraints() error = %v", err)
		}

		// All identifiers should be quoted
		requiredQuoted := []string{
			`ALTER TABLE "post"`,
			`ADD CONSTRAINT "post_author_id_fkey"`,
			`FOREIGN KEY ("author_id")`,
			`REFERENCES "user" ("id")`,
		}

		for _, expected := range requiredQuoted {
			if !strings.Contains(result[0], expected) {
				t.Errorf("Missing quoted identifier %q in: %s", expected, result[0])
			}
		}
	})

	t.Run("UNIQUE constraints quote all identifiers", func(t *testing.T) {
		resource := schema.NewResourceSchema("User")
		resource.Fields["email"] = &schema.Field{
			Name: "email",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "unique"}},
		}

		result := gen.GenerateUniqueConstraints(resource)

		// All identifiers should be quoted
		if !strings.Contains(result[0], `ALTER TABLE "user"`) {
			t.Errorf("Table name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `ADD CONSTRAINT "user_email_unique"`) {
			t.Errorf("Constraint name should be quoted, got: %s", result[0])
		}
		if !strings.Contains(result[0], `UNIQUE ("email")`) {
			t.Errorf("Column name should be quoted, got: %s", result[0])
		}
	})

	t.Run("DROP statements quote all identifiers", func(t *testing.T) {
		resource := schema.NewResourceSchema("Product")
		resource.Fields["price"] = &schema.Field{
			Name: "price",
			Type: &schema.TypeSpec{
				BaseType: schema.TypeDecimal,
				Nullable: false,
			},
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
			},
		}

		result := gen.GenerateDropConstraints(resource, nil)

		// All identifiers in DROP should be quoted
		if !strings.Contains(result[0], `ALTER TABLE "product"`) {
			t.Errorf("Table name should be quoted in DROP, got: %s", result[0])
		}
		if !strings.Contains(result[0], `DROP CONSTRAINT IF EXISTS "product_price_min"`) {
			t.Errorf("Constraint name should be quoted in DROP, got: %s", result[0])
		}
	})
}
