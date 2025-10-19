package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestChangeTrackingGenerator_Generate(t *testing.T) {
	resource := createTestResourceForChangeTracking()
	generator := NewChangeTrackingGenerator()

	code, err := generator.Generate(resource)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if code == "" {
		t.Fatal("Generated code is empty")
	}

	// Verify the generated code contains expected elements
	expectedElements := []string{
		"// Change Tracking Methods",
		"TitleChanged()",
		"PreviousTitle()",
		"SetTitle(",
		"StatusChanged()",
		"PreviousStatus()",
		"SetStatus(",
		"Changed(field string)",
		"ChangedFields()",
		"HasChanges()",
		"Reload(",
		"GetChangedData()",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}
}

func TestChangeTrackingGenerator_GenerateFieldChangedMethod(t *testing.T) {
	resource := &schema.ResourceSchema{
		Name:   "Post",
		Fields: make(map[string]*schema.Field),
	}

	generator := NewChangeTrackingGenerator()
	code := generator.generateFieldChangedMethod(resource, "title")

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) TitleChanged() bool") {
		t.Error("Missing correct method signature")
	}

	// Verify it uses the change tracker
	if !strings.Contains(code, `tracker.Changed("title")`) {
		t.Error("Missing change tracker call")
	}

	// Verify documentation
	if !strings.Contains(code, "// TitleChanged") {
		t.Error("Missing documentation comment")
	}
}

func TestChangeTrackingGenerator_GeneratePreviousValueMethod(t *testing.T) {
	tests := []struct {
		name       string
		fieldName  string
		fieldType  *schema.TypeSpec
		wantReturn string
	}{
		{
			name:      "required string field",
			fieldName: "title",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: false,
			},
			wantReturn: "string",
		},
		{
			name:      "optional string field",
			fieldName: "bio",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: true,
			},
			wantReturn: "*string",
		},
		{
			name:      "required int field",
			fieldName: "count",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeInt,
				Nullable: false,
			},
			wantReturn: "int",
		},
		{
			name:      "optional int field",
			fieldName: "views",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeInt,
				Nullable: true,
			},
			wantReturn: "*int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &schema.ResourceSchema{Name: "Post"}
			field := &schema.Field{
				Name: tt.fieldName,
				Type: tt.fieldType,
			}

			generator := NewChangeTrackingGenerator()
			code := generator.generatePreviousValueMethod(resource, tt.fieldName, field)

			methodName := "Previous" + toPascalCase(tt.fieldName)
			expectedSig := fmt.Sprintf("func (r *Post) %s() %s", methodName, tt.wantReturn)

			if !strings.Contains(code, expectedSig) {
				t.Errorf("Missing expected signature: %s\nGenerated:\n%s", expectedSig, code)
			}

			// Verify it uses the change tracker
			if !strings.Contains(code, "tracker.PreviousValue") {
				t.Error("Missing change tracker call")
			}
		})
	}
}

func TestChangeTrackingGenerator_GenerateSetterMethod(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldType *schema.TypeSpec
		wantParam string
	}{
		{
			name:      "required string field",
			fieldName: "title",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: false,
			},
			wantParam: "string",
		},
		{
			name:      "optional string field",
			fieldName: "bio",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: true,
			},
			wantParam: "*string",
		},
		{
			name:      "required bool field",
			fieldName: "published",
			fieldType: &schema.TypeSpec{
				BaseType: schema.TypeBool,
				Nullable: false,
			},
			wantParam: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &schema.ResourceSchema{Name: "Post"}
			field := &schema.Field{
				Name: tt.fieldName,
				Type: tt.fieldType,
			}

			generator := NewChangeTrackingGenerator()
			code := generator.generateSetterMethod(resource, tt.fieldName, field)

			methodName := "Set" + toPascalCase(tt.fieldName)
			expectedSig := fmt.Sprintf("func (r *Post) %s(value %s)", methodName, tt.wantParam)

			if !strings.Contains(code, expectedSig) {
				t.Errorf("Missing expected signature: %s\nGenerated:\n%s", expectedSig, code)
			}

			// Verify it updates the field
			pascalField := toPascalCase(tt.fieldName)
			if !strings.Contains(code, fmt.Sprintf("r.%s = value", pascalField)) {
				t.Error("Missing field assignment")
			}

			// Verify it uses the change tracker
			if !strings.Contains(code, "tracker.SetFieldValue") {
				t.Error("Missing change tracker call")
			}
		})
	}
}

func TestChangeTrackingGenerator_GenerateChangedMethod(t *testing.T) {
	resource := &schema.ResourceSchema{Name: "Post"}
	generator := NewChangeTrackingGenerator()

	code := generator.generateChangedMethod(resource)

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) Changed(field string) bool") {
		t.Error("Missing correct method signature")
	}

	// Verify it uses the change tracker
	if !strings.Contains(code, "tracker.Changed(field)") {
		t.Error("Missing change tracker call")
	}
}

func TestChangeTrackingGenerator_GenerateChangedFieldsMethod(t *testing.T) {
	resource := &schema.ResourceSchema{Name: "Post"}
	generator := NewChangeTrackingGenerator()

	code := generator.generateChangedFieldsMethod(resource)

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) ChangedFields() []string") {
		t.Error("Missing correct method signature")
	}

	// Verify it uses the change tracker
	if !strings.Contains(code, "tracker.ChangedFields()") {
		t.Error("Missing change tracker call")
	}
}

func TestChangeTrackingGenerator_GenerateHasChangesMethod(t *testing.T) {
	resource := &schema.ResourceSchema{Name: "Post"}
	generator := NewChangeTrackingGenerator()

	code := generator.generateHasChangesMethod(resource)

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) HasChanges() bool") {
		t.Error("Missing correct method signature")
	}

	// Verify it uses the change tracker
	if !strings.Contains(code, "tracker.HasChanges()") {
		t.Error("Missing change tracker call")
	}
}

func TestChangeTrackingGenerator_GenerateReloadMethod(t *testing.T) {
	resource := &schema.ResourceSchema{
		Name:   "Post",
		Fields: make(map[string]*schema.Field),
	}

	generator := NewChangeTrackingGenerator()
	code := generator.generateReloadMethod(resource)

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) Reload(ctx context.Context, db *sql.DB) error") {
		t.Error("Missing correct method signature")
	}

	// Verify it creates CRUD operations
	if !strings.Contains(code, "crud.NewOperations") {
		t.Error("Missing CRUD operations creation")
	}

	// Verify it calls Find
	if !strings.Contains(code, "ops.Find(ctx, r.ID)") {
		t.Error("Missing Find call")
	}

	// Verify it resets change tracking
	if !strings.Contains(code, "tracker.Reset()") {
		t.Error("Missing change tracker reset")
	}
}

func TestChangeTrackingGenerator_GenerateGetChangedDataMethod(t *testing.T) {
	resource := &schema.ResourceSchema{Name: "Post"}
	generator := NewChangeTrackingGenerator()

	code := generator.generateGetChangedDataMethod(resource)

	// Verify method signature
	if !strings.Contains(code, "func (r *Post) GetChangedData() map[string]interface{}") {
		t.Error("Missing correct method signature")
	}

	// Verify it uses the change tracker
	if !strings.Contains(code, "tracker.GetChangedData()") {
		t.Error("Missing change tracker call")
	}
}

func TestChangeTrackingGenerator_GenerateChangeTrackerField(t *testing.T) {
	generator := NewChangeTrackingGenerator()
	code := generator.GenerateChangeTrackerField()

	// Verify field declarations
	if !strings.Contains(code, "__changeTracker__ *tracking.ChangeTracker") {
		t.Error("Missing change tracker field")
	}

	if !strings.Contains(code, "__changeTrackerMu__ sync.RWMutex") {
		t.Error("Missing mutex field")
	}
}

func TestChangeTrackingGenerator_GenerateChangeTrackerAccessor(t *testing.T) {
	resource := &schema.ResourceSchema{Name: "Post"}
	generator := NewChangeTrackingGenerator()

	code := generator.GenerateChangeTrackerAccessor(resource)

	// Verify accessor method
	if !strings.Contains(code, "func (r *Post) changeTracker() (*tracking.ChangeTracker, bool)") {
		t.Error("Missing changeTracker accessor")
	}

	// Verify init method
	if !strings.Contains(code, "func (r *Post) initChangeTracker(original, current map[string]interface{})") {
		t.Error("Missing initChangeTracker method")
	}

	// Verify reset method
	if !strings.Contains(code, "func (r *Post) resetChangeTracker()") {
		t.Error("Missing resetChangeTracker method")
	}

	// Verify mutex usage
	if !strings.Contains(code, "__changeTrackerMu__.RLock()") {
		t.Error("Missing read lock in accessor")
	}

	if !strings.Contains(code, "__changeTrackerMu__.Lock()") {
		t.Error("Missing write lock in init/reset")
	}
}

func TestChangeTrackingGenerator_SkipsInternalFields(t *testing.T) {
	resource := &schema.ResourceSchema{
		Name: "Post",
		Fields: map[string]*schema.Field{
			"id": {
				Name: "id",
				Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
			},
			"created_at": {
				Name: "created_at",
				Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
			},
			"updated_at": {
				Name: "updated_at",
				Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
			},
			"title": {
				Name: "title",
				Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			},
		},
	}

	generator := NewChangeTrackingGenerator()
	code, err := generator.Generate(resource)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should not generate methods for internal fields
	if strings.Contains(code, "IdChanged()") {
		t.Error("Should not generate IdChanged method")
	}
	if strings.Contains(code, "CreatedAtChanged()") {
		t.Error("Should not generate CreatedAtChanged method")
	}
	if strings.Contains(code, "UpdatedAtChanged()") {
		t.Error("Should not generate UpdatedAtChanged method")
	}

	// Should generate methods for regular fields
	if !strings.Contains(code, "TitleChanged()") {
		t.Error("Should generate TitleChanged method")
	}
}

func TestChangeTrackingGenerator_MultipleFields(t *testing.T) {
	resource := &schema.ResourceSchema{
		Name: "Post",
		Fields: map[string]*schema.Field{
			"title": {
				Name: "title",
				Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			},
			"content": {
				Name: "content",
				Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: false},
			},
			"status": {
				Name: "status",
				Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			},
		},
	}

	generator := NewChangeTrackingGenerator()
	code, err := generator.Generate(resource)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify methods for all fields
	expectedMethods := []string{
		"TitleChanged()",
		"PreviousTitle()",
		"SetTitle(",
		"ContentChanged()",
		"PreviousContent()",
		"SetContent(",
		"StatusChanged()",
		"PreviousStatus()",
		"SetStatus(",
	}

	for _, method := range expectedMethods {
		if !strings.Contains(code, method) {
			t.Errorf("Missing expected method: %s", method)
		}
	}
}

func TestChangeTrackingGenerator_NullableFields(t *testing.T) {
	resource := &schema.ResourceSchema{
		Name: "Post",
		Fields: map[string]*schema.Field{
			"bio": {
				Name: "bio",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeText,
					Nullable: true,
				},
			},
		},
	}

	generator := NewChangeTrackingGenerator()
	code, err := generator.Generate(resource)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify nullable handling in PreviousValue
	if !strings.Contains(code, "*string") {
		t.Error("Should use pointer type for nullable field")
	}

	if !strings.Contains(code, "if val == nil") {
		t.Error("Should handle nil values for nullable fields")
	}
}

// Helper function to create a test resource for change tracking tests
func createTestResourceForChangeTracking() *schema.ResourceSchema {
	return &schema.ResourceSchema{
		Name: "Post",
		Fields: map[string]*schema.Field{
			"id": {
				Name: "id",
				Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
			},
			"title": {
				Name: "title",
				Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			},
			"content": {
				Name: "content",
				Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: false},
			},
			"status": {
				Name: "status",
				Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			},
			"created_at": {
				Name: "created_at",
				Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
			},
		},
		Relationships: make(map[string]*schema.Relationship),
		Hooks:         make(map[schema.HookType][]*schema.Hook),
		Scopes:        make(map[string]*schema.Scope),
		Location:      ast.SourceLocation{},
	}
}
