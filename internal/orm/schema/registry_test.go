package schema

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestRegistry(t *testing.T) {
	t.Run("register and get schema", func(t *testing.T) {
		registry := NewRegistry()

		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		err := registry.Register(schema)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		retrieved, exists := registry.Get("Post")
		if !exists {
			t.Error("schema should exist")
		}
		if retrieved.Name != "Post" {
			t.Errorf("expected Post, got %s", retrieved.Name)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		registry := NewRegistry()

		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		registry.Register(schema)
		err := registry.Register(schema)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})

	t.Run("list schemas", func(t *testing.T) {
		registry := NewRegistry()

		for _, name := range []string{"User", "Post", "Comment"} {
			schema := NewResourceSchema(name)
			schema.Fields["id"] = &Field{
				Name: "id",
				Type: &TypeSpec{
					BaseType: TypeUUID,
					Nullable: false,
				},
				Annotations: []Annotation{{Name: "primary"}},
			}
			registry.Register(schema)
		}

		names := registry.List()
		if len(names) != 3 {
			t.Errorf("expected 3 schemas, got %d", len(names))
		}

		expectedNames := map[string]bool{
			"User":    false,
			"Post":    false,
			"Comment": false,
		}

		for _, name := range names {
			if _, ok := expectedNames[name]; ok {
				expectedNames[name] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("expected %s in list", name)
			}
		}
	})

	t.Run("count and exists", func(t *testing.T) {
		registry := NewRegistry()

		if registry.Count() != 0 {
			t.Error("empty registry should have count 0")
		}

		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		registry.Register(schema)

		if registry.Count() != 1 {
			t.Error("registry should have count 1")
		}

		if !registry.Exists("Post") {
			t.Error("Post should exist")
		}

		if registry.Exists("NonExistent") {
			t.Error("NonExistent should not exist")
		}
	})

	t.Run("clear", func(t *testing.T) {
		registry := NewRegistry()

		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		registry.Register(schema)

		registry.Clear()

		if registry.Count() != 0 {
			t.Error("cleared registry should have count 0")
		}
	})

	t.Run("get all schemas", func(t *testing.T) {
		registry := NewRegistry()

		for _, name := range []string{"User", "Post"} {
			schema := NewResourceSchema(name)
			schema.Fields["id"] = &Field{
				Name: "id",
				Type: &TypeSpec{
					BaseType: TypeUUID,
					Nullable: false,
				},
				Annotations: []Annotation{{Name: "primary"}},
			}
			registry.Register(schema)
		}

		all := registry.All()
		if len(all) != 2 {
			t.Errorf("expected 2 schemas, got %d", len(all))
		}

		// Verify it's a copy
		delete(all, "User")
		if !registry.Exists("User") {
			t.Error("deleting from All() result should not affect registry")
		}
	})
}

func TestRegistryDependencyOrder(t *testing.T) {
	registry := NewRegistry()

	// Create schemas with dependencies
	userSchema := NewResourceSchema("User")
	userSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}

	postSchema := NewResourceSchema("Post")
	postSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}
	postSchema.Relationships["author"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	commentSchema := NewResourceSchema("Comment")
	commentSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}
	commentSchema.Relationships["post"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "Post",
		FieldName:      "post",
	}

	registry.Register(userSchema)
	registry.Register(postSchema)
	registry.Register(commentSchema)

	order, err := registry.GetDependencyOrder()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify order: User before Post before Comment
	userIdx := -1
	postIdx := -1
	commentIdx := -1

	for i, name := range order {
		switch name {
		case "User":
			userIdx = i
		case "Post":
			postIdx = i
		case "Comment":
			commentIdx = i
		}
	}

	if userIdx > postIdx {
		t.Error("User should come before Post")
	}
	if postIdx > commentIdx {
		t.Error("Post should come before Comment")
	}
}

func TestRegistryValidation(t *testing.T) {
	t.Run("validate all with circular dependency", func(t *testing.T) {
		registry := NewRegistry()

		schemaA := NewResourceSchema("A")
		schemaA.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		schemaA.Relationships["b"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "B",
			FieldName:      "b",
		}

		schemaB := NewResourceSchema("B")
		schemaB.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		schemaB.Relationships["a"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "A",
			FieldName:      "a",
		}

		errA := registry.Register(schemaA)
		if errA != nil {
			t.Logf("Register(A) error: %v", errA)
		}
		errB := registry.Register(schemaB)
		if errB != nil {
			t.Logf("Register(B) error: %v", errB)
		}

		t.Logf("Registry has %d schemas", registry.Count())

		err := registry.ValidateAll()
		t.Logf("ValidateAll error: %v", err)
		if err == nil {
			t.Error("expected error for circular dependency")
		}
	})

	t.Run("validate all success", func(t *testing.T) {
		registry := NewRegistry()

		userSchema := NewResourceSchema("User")
		userSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		postSchema := NewResourceSchema("Post")
		postSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		postSchema.Relationships["author"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
		}

		registry.Register(userSchema)
		registry.Register(postSchema)

		err := registry.ValidateAll()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestRegistryAnalyzeDependencies(t *testing.T) {
	registry := NewRegistry()

	userSchema := NewResourceSchema("User")
	userSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}

	postSchema := NewResourceSchema("Post")
	postSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType: TypeUUID,
			Nullable: false,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}
	postSchema.Relationships["author"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	registry.Register(userSchema)
	registry.Register(postSchema)

	report, err := registry.AnalyzeDependencies()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if report.TotalResources != 2 {
		t.Errorf("expected 2 resources, got %d", report.TotalResources)
	}

	if report.HasCycles {
		t.Error("should not have cycles")
	}
}

func TestRegistryStats(t *testing.T) {
	registry := NewRegistry()

	userSchema := NewResourceSchema("User")
	userSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType:       TypeUUID,
			Nullable:       false,
			NullabilitySet: true,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}
	userSchema.Fields["email"] = &Field{
		Name: "email",
		Type: &TypeSpec{
			BaseType:       TypeEmail,
			Nullable:       false,
			NullabilitySet: true,
		},
	}

	postSchema := NewResourceSchema("Post")
	postSchema.Fields["id"] = &Field{
		Name: "id",
		Type: &TypeSpec{
			BaseType:       TypeUUID,
			Nullable:       false,
			NullabilitySet: true,
		},
		Annotations: []Annotation{{Name: "primary"}},
	}
	postSchema.Relationships["author"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}
	postSchema.Hooks[BeforeCreate] = []*Hook{
		{
			Type: BeforeCreate,
			Body: []ast.StmtNode{},
		},
	}
	postSchema.Scopes["published"] = &Scope{
		Name: "published",
	}
	postSchema.Computed["is_published"] = &ComputedField{
		Name: "is_published",
	}
	postSchema.ConstraintBlocks = append(postSchema.ConstraintBlocks, &ConstraintBlock{
		Name:      "title_required",
		On:        []string{"create", "update"},
		Condition: &ast.LiteralExpr{Value: true},
		Error:     "Title is required",
	})

	errUser := registry.Register(userSchema)
	if errUser != nil {
		t.Fatalf("failed to register User: %v", errUser)
	}
	errPost := registry.Register(postSchema)
	if errPost != nil {
		t.Fatalf("failed to register Post: %v", errPost)
	}

	stats := registry.GetStats()

	if stats.TotalResources != 2 {
		t.Errorf("expected 2 resources, got %d", stats.TotalResources)
	}

	// User has id and email, Post has id
	if stats.TotalFields != 3 {
		t.Errorf("expected 3 fields, got %d", stats.TotalFields)
	}

	if stats.TotalRelationships != 1 {
		t.Errorf("expected 1 relationship, got %d", stats.TotalRelationships)
	}

	if stats.TotalHooks != 1 {
		t.Errorf("expected 1 hook, got %d", stats.TotalHooks)
	}

	if stats.ResourcesWithHooks != 1 {
		t.Errorf("expected 1 resource with hooks, got %d", stats.ResourcesWithHooks)
	}

	if stats.TotalScopes != 1 {
		t.Errorf("expected 1 scope, got %d", stats.TotalScopes)
	}

	if stats.TotalComputedFields != 1 {
		t.Errorf("expected 1 computed field, got %d", stats.TotalComputedFields)
	}

	if stats.TotalConstraints != 1 {
		t.Errorf("expected 1 constraint, got %d", stats.TotalConstraints)
	}
}
