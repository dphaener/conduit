package schema

import (
	"strings"
	"testing"
)

func TestRelationshipGraph(t *testing.T) {
	t.Run("simple dependency chain", func(t *testing.T) {
		// Create schemas: Comment -> Post -> User
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

		schemas := map[string]*ResourceSchema{
			"User":    userSchema,
			"Post":    postSchema,
			"Comment": commentSchema,
		}

		graph := NewRelationshipGraph(schemas)

		// Check dependencies
		postDeps := graph.GetDependencies("Post")
		if len(postDeps) != 1 || postDeps[0] != "User" {
			t.Errorf("Post should depend on User, got %v", postDeps)
		}

		commentDeps := graph.GetDependencies("Comment")
		if len(commentDeps) != 1 || commentDeps[0] != "Post" {
			t.Errorf("Comment should depend on Post, got %v", commentDeps)
		}

		userDeps := graph.GetDependencies("User")
		if len(userDeps) != 0 {
			t.Errorf("User should have no dependencies, got %v", userDeps)
		}

		// Check dependents
		userDependents := graph.GetDependents("User")
		if len(userDependents) != 1 || userDependents[0] != "Post" {
			t.Errorf("User should have Post as dependent, got %v", userDependents)
		}
	})

	t.Run("topological sort", func(t *testing.T) {
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

		schemas := map[string]*ResourceSchema{
			"User":    userSchema,
			"Post":    postSchema,
			"Comment": commentSchema,
		}

		graph := NewRelationshipGraph(schemas)
		order, err := graph.TopologicalSort()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// User should come before Post, Post should come before Comment
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
			t.Error("User should come before Post in dependency order")
		}
		if postIdx > commentIdx {
			t.Error("Post should come before Comment in dependency order")
		}
	})

	t.Run("circular dependency detection", func(t *testing.T) {
		// Create circular dependency: A -> B -> C -> A
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
		schemaB.Relationships["c"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "C",
			FieldName:      "c",
		}

		schemaC := NewResourceSchema("C")
		schemaC.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		schemaC.Relationships["a"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "A",
			FieldName:      "a",
		}

		schemas := map[string]*ResourceSchema{
			"A": schemaA,
			"B": schemaB,
			"C": schemaC,
		}

		graph := NewRelationshipGraph(schemas)
		cycles := graph.DetectCycles()
		if len(cycles) == 0 {
			t.Error("expected to detect circular dependency")
		}

		// Topological sort should fail
		_, err := graph.TopologicalSort()
		if err == nil {
			t.Error("topological sort should fail with circular dependency")
		}
	})

	t.Run("self-referential relationship", func(t *testing.T) {
		// Category can have parent Category
		categorySchema := NewResourceSchema("Category")
		categorySchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		categorySchema.Relationships["parent"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "Category",
			FieldName:      "parent",
			Nullable:       true,
		}

		schemas := map[string]*ResourceSchema{
			"Category": categorySchema,
		}

		graph := NewRelationshipGraph(schemas)

		// Self-referential relationships create cycles
		cycles := graph.DetectCycles()
		if len(cycles) == 0 {
			t.Error("expected to detect self-referential cycle")
		}
	})
}

func TestDependencyAnalyzer(t *testing.T) {
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
	commentSchema.Relationships["author"] = &Relationship{
		Type:           RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	schemas := map[string]*ResourceSchema{
		"User":    userSchema,
		"Post":    postSchema,
		"Comment": commentSchema,
	}

	analyzer := NewDependencyAnalyzer(schemas)
	report, err := analyzer.Analyze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if report.TotalResources != 3 {
		t.Errorf("expected 3 resources, got %d", report.TotalResources)
	}

	if report.HasCycles {
		t.Error("should not have cycles")
	}

	if len(report.TopologicalOrder) != 3 {
		t.Errorf("expected 3 resources in topological order, got %d", len(report.TopologicalOrder))
	}

	// Comment should depend on both Post and User
	commentDeps := report.Dependencies["Comment"]
	if len(commentDeps) != 2 {
		t.Errorf("Comment should have 2 dependencies, got %d", len(commentDeps))
	}

	// User should have 2 dependents (Post and Comment)
	userDependents := report.Dependents["User"]
	if len(userDependents) != 2 {
		t.Errorf("User should have 2 dependents, got %d", len(userDependents))
	}

	// Check report string output
	reportStr := report.String()
	if !strings.Contains(reportStr, "Total Resources: 3") {
		t.Error("report should contain total resources")
	}
	if !strings.Contains(reportStr, "Dependency Order") {
		t.Error("report should contain dependency order")
	}
}

func TestRelationshipValidator(t *testing.T) {
	t.Run("valid relationships", func(t *testing.T) {
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

		schemas := map[string]*ResourceSchema{
			"User": userSchema,
			"Post": postSchema,
		}

		validator := NewRelationshipValidator(schemas)
		err := validator.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("circular dependency", func(t *testing.T) {
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

		schemas := map[string]*ResourceSchema{
			"A": schemaA,
			"B": schemaB,
		}

		validator := NewRelationshipValidator(schemas)
		err := validator.Validate()
		if err == nil {
			t.Error("expected error for circular dependency")
		}
	})

	t.Run("missing target resource", func(t *testing.T) {
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
			TargetResource: "NonExistent",
			FieldName:      "author",
		}

		schemas := map[string]*ResourceSchema{
			"Post": postSchema,
		}

		validator := NewRelationshipValidator(schemas)
		err := validator.Validate()
		if err == nil {
			t.Error("expected error for missing target resource")
		}
	})
}
