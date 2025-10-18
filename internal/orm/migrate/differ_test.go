package migrate

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestDiffer_ComputeDiff_AddResource(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{}
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:   "User",
			Fields: map[string]*schema.Field{},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Type != ChangeAddResource {
		t.Errorf("Expected ChangeAddResource, got %v", change.Type)
	}

	if change.Resource != "User" {
		t.Errorf("Expected resource 'User', got %s", change.Resource)
	}

	if change.Breaking {
		t.Error("Adding a resource should not be breaking")
	}

	if change.DataLoss {
		t.Error("Adding a resource should not cause data loss")
	}
}

func TestDiffer_ComputeDiff_DropResource(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:   "User",
			Fields: map[string]*schema.Field{},
		},
	}
	newSchemas := map[string]*schema.ResourceSchema{}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Type != ChangeDropResource {
		t.Errorf("Expected ChangeDropResource, got %v", change.Type)
	}

	if !change.Breaking {
		t.Error("Dropping a resource should be breaking")
	}

	if !change.DataLoss {
		t.Error("Dropping a resource should cause data loss")
	}
}

func TestDiffer_ComputeDiff_AddField(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
				"name": {
					Name: "name",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: true, // Nullable field
					},
				},
			},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Type != ChangeAddField {
		t.Errorf("Expected ChangeAddField, got %v", change.Type)
	}

	if change.Field != "name" {
		t.Errorf("Expected field 'name', got %s", change.Field)
	}

	if change.Breaking {
		t.Error("Adding a nullable field should not be breaking")
	}
}

func TestDiffer_ComputeDiff_AddFieldNonNullableWithoutDefault(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:   "User",
			Fields: map[string]*schema.Field{},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"name": {
					Name: "name",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false, // Non-nullable
						Default:  nil,   // No default
					},
				},
			},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	change := changes[0]
	if !change.Breaking {
		t.Error("Adding a non-nullable field without default should be breaking")
	}
}

func TestDiffer_ComputeDiff_DropField(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeEmail,
						Nullable: false,
					},
				},
			},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:   "User",
			Fields: map[string]*schema.Field{},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	change := changes[0]
	if change.Type != ChangeDropField {
		t.Errorf("Expected ChangeDropField, got %v", change.Type)
	}

	if !change.Breaking {
		t.Error("Dropping a field should be breaking")
	}

	if !change.DataLoss {
		t.Error("Dropping a field should cause data loss")
	}
}

func TestDiffer_ComputeDiff_ModifyFieldNullability(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeEmail,
						Nullable: true, // Optional
					},
				},
			},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeEmail,
						Nullable: false, // Required
					},
				},
			},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Type != ChangeModifyField {
		t.Errorf("Expected ChangeModifyField, got %v", change.Type)
	}

	if !change.Breaking {
		t.Error("Making a field non-nullable should be breaking")
	}
}

func TestDiffer_ComputeDiff_ModifyFieldType(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"age": {
					Name: "age",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeInt,
						Nullable: false,
					},
				},
			},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"age": {
					Name: "age",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
				},
			},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	change := changes[0]
	if !change.Breaking {
		t.Error("Changing field type should be breaking")
	}

	if !change.DataLoss {
		t.Error("Changing field type may cause data loss")
	}
}

func TestDiffer_ComputeDiff_AddRelationship(t *testing.T) {
	oldSchemas := map[string]*schema.ResourceSchema{
		"Post": {
			Name:          "Post",
			Fields:        map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	newSchemas := map[string]*schema.ResourceSchema{
		"Post": {
			Name:   "Post",
			Fields: map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{
				"author": {
					Type:           schema.RelationshipBelongsTo,
					FieldName:      "author",
					TargetResource: "User",
					ForeignKey:     "author_id",
					OnDelete:       schema.CascadeRestrict,
				},
			},
		},
	}

	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Type != ChangeAddRelationship {
		t.Errorf("Expected ChangeAddRelationship, got %v", change.Type)
	}

	if change.Relation != "author" {
		t.Errorf("Expected relation 'author', got %s", change.Relation)
	}

	if change.Breaking {
		t.Error("Adding a relationship should not be breaking")
	}
}

func TestDiffer_ComputeDiff_NoChanges(t *testing.T) {
	schemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
		},
	}

	differ := NewDiffer(schemas, schemas)
	changes := differ.ComputeDiff()

	if len(changes) != 0 {
		t.Errorf("Expected no changes, got %d", len(changes))
	}
}

func TestGenerateMigrationName(t *testing.T) {
	tests := []struct {
		name     string
		changes  []SchemaChange
		expected string
	}{
		{
			name:     "no changes",
			changes:  []SchemaChange{},
			expected: "no_changes",
		},
		{
			name: "add resource",
			changes: []SchemaChange{
				{Type: ChangeAddResource, Resource: "User"},
			},
			expected: "add_resource_User",
		},
		{
			name: "drop resource",
			changes: []SchemaChange{
				{Type: ChangeDropResource, Resource: "Post"},
			},
			expected: "drop_resource_Post",
		},
		{
			name: "add field",
			changes: []SchemaChange{
				{Type: ChangeAddField, Resource: "User", Field: "email"},
			},
			expected: "add_User.email",
		},
		{
			name: "multiple changes",
			changes: []SchemaChange{
				{Type: ChangeAddResource, Resource: "User"},
				{Type: ChangeAddField, Resource: "Post", Field: "title"},
			},
			expected: "add_resource_User_Post.title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMigrationName(tt.changes)
			if result != tt.expected {
				t.Errorf("Expected name %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDiffer_FieldComparison(t *testing.T) {
	differ := NewDiffer(nil, nil)

	tests := []struct {
		name     string
		old      *schema.Field
		new      *schema.Field
		equal    bool
		breaking bool
		dataLoss bool
	}{
		{
			name: "identical fields",
			old: &schema.Field{
				Name: "email",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeEmail,
					Nullable: false,
				},
			},
			new: &schema.Field{
				Name: "email",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeEmail,
					Nullable: false,
				},
			},
			equal:    true,
			breaking: false,
			dataLoss: false,
		},
		{
			name: "optional to required",
			old: &schema.Field{
				Name: "email",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeEmail,
					Nullable: true,
				},
			},
			new: &schema.Field{
				Name: "email",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeEmail,
					Nullable: false,
				},
			},
			equal:    false,
			breaking: true,
			dataLoss: false,
		},
		{
			name: "type change",
			old: &schema.Field{
				Name: "value",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeInt,
					Nullable: false,
				},
			},
			new: &schema.Field{
				Name: "value",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeString,
					Nullable: false,
				},
			},
			equal:    false,
			breaking: true,
			dataLoss: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := differ.fieldsEqual(tt.old, tt.new)
			if equal != tt.equal {
				t.Errorf("fieldsEqual: expected %v, got %v", tt.equal, equal)
			}

			if !equal {
				breaking := differ.isBreakingFieldChange(tt.old, tt.new)
				if breaking != tt.breaking {
					t.Errorf("isBreakingFieldChange: expected %v, got %v", tt.breaking, breaking)
				}

				dataLoss := differ.causesDataLoss(tt.old, tt.new)
				if dataLoss != tt.dataLoss {
					t.Errorf("causesDataLoss: expected %v, got %v", tt.dataLoss, dataLoss)
				}
			}
		})
	}
}
