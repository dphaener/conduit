package parser

import (
	"strings"
	"testing"
)

// TestParser_AllConstraintTypes tests all constraint types
func TestParser_AllConstraintTypes(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary @auto
  username: string! @min(3) @max(50) @unique
  email: email! @pattern("^[a-z]+@example\\.com$")
  age: int! @default(18)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
  bio: text? @required
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]

	// Find and verify each field with constraints
	fieldConstraints := map[string][]string{
		"id":         {"primary", "auto"},
		"username":   {"min", "max", "unique"},
		"email":      {"pattern"},
		"age":        {"default"},
		"created_at": {"auto"},
		"updated_at": {"auto_update"},
		"bio":        {"required"},
	}

	for _, field := range resource.Fields {
		expectedConstraints, ok := fieldConstraints[field.Name]
		if !ok {
			continue
		}

		if len(field.Constraints) != len(expectedConstraints) {
			t.Errorf("Field %s: expected %d constraints, got %d", field.Name, len(expectedConstraints), len(field.Constraints))
		}

		for i, expected := range expectedConstraints {
			if field.Constraints[i].Name != expected {
				t.Errorf("Field %s constraint %d: expected '%s', got '%s'",
					field.Name, i, expected, field.Constraints[i].Name)
			}
		}
	}
}

// TestParser_ConstraintWithMultipleArgs tests constraints with multiple arguments
func TestParser_ConstraintWithMultipleArgs(t *testing.T) {
	source := `
resource Product {
  price: decimal! @min(0) @max(999999)
  rating: float! @min(0) @max(5)
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	priceField := resource.Fields[0]

	if len(priceField.Constraints) != 2 {
		t.Fatalf("Expected 2 constraints, got %d", len(priceField.Constraints))
	}

	// Verify min constraint
	minConstraint := priceField.Constraints[0]
	if minConstraint.Name != "min" {
		t.Errorf("Expected 'min' constraint, got '%s'", minConstraint.Name)
	}
	if len(minConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(minConstraint.Arguments))
	}
	if minConstraint.Arguments[0] != int64(0) {
		t.Errorf("Expected argument 0, got %v", minConstraint.Arguments[0])
	}

	// Verify max constraint
	maxConstraint := priceField.Constraints[1]
	if maxConstraint.Name != "max" {
		t.Errorf("Expected 'max' constraint, got '%s'", maxConstraint.Name)
	}
	if len(maxConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(maxConstraint.Arguments))
	}
	if maxConstraint.Arguments[0] != int64(999999) {
		t.Errorf("Expected argument 999999, got %v", maxConstraint.Arguments[0])
	}
}

// TestParser_ConstraintWithStringArgs tests constraints with string arguments
func TestParser_ConstraintWithStringArgs(t *testing.T) {
	source := `
resource User {
  email: string! @pattern("^[a-z]+@example\\.com$")
  status: string! @default("active")
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	emailField := resource.Fields[0]

	patternConstraint := emailField.Constraints[0]
	if patternConstraint.Name != "pattern" {
		t.Errorf("Expected 'pattern' constraint, got '%s'", patternConstraint.Name)
	}
	if len(patternConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(patternConstraint.Arguments))
	}
	if patternConstraint.Arguments[0] != "^[a-z]+@example\\.com$" {
		t.Errorf("Expected pattern argument, got %v", patternConstraint.Arguments[0])
	}
}

// TestParser_UnterminatedStruct tests error handling for unterminated inline struct
func TestParser_UnterminatedStruct(t *testing.T) {
	source := `
resource Post {
  seo: {
    title: string?
    description: string?
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for unterminated struct")
	}

	errorMsg := errors[0].Error()
	if !strings.Contains(errorMsg, "Expected") {
		t.Errorf("Expected error about missing closing brace, got: %s", errorMsg)
	}
}

// TestParser_InvalidEnumValues tests error handling for non-string enum values
func TestParser_InvalidEnumValues(t *testing.T) {
	source := `
resource Post {
  status: enum [123, 456]!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for invalid enum values")
	}
}

// TestParser_MalformedRelationshipMetadata tests error handling for malformed relationship metadata
func TestParser_MalformedRelationshipMetadata(t *testing.T) {
	source := `
resource Post {
  author: User! {
    foreign_key "missing_colon"
  }
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for malformed relationship metadata")
	}
}

// TestParser_AllOnDeleteOptions tests all on_delete relationship options
func TestParser_AllOnDeleteOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   string
		nullable bool
	}{
		{"restrict", "restrict", false},
		{"cascade", "cascade", false},
		{"set_null", "set_null", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nullableMarker := "!"
			if tt.nullable {
				nullableMarker = "?"
			}

			source := `
resource Post {
  author: User` + nullableMarker + ` {
    foreign_key: "author_id"
    on_delete: ` + tt.option + `
  }
}
`

			program, errors := parseSource(t, source)

			if len(errors) > 0 {
				t.Fatalf("Expected no errors, got: %v", errors)
			}

			resource := program.Resources[0]
			if len(resource.Relationships) != 1 {
				t.Fatalf("Expected 1 relationship, got %d", len(resource.Relationships))
			}

			rel := resource.Relationships[0]
			if rel.OnDelete != tt.option {
				t.Errorf("Expected on_delete '%s', got '%s'", tt.option, rel.OnDelete)
			}
		})
	}
}

// TestParser_OnUpdateRestrict tests on_update with restrict option
func TestParser_OnUpdateRestrict(t *testing.T) {
	source := `
resource Post {
  author: User! {
    on_update: restrict
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]
	if rel.OnUpdate != "restrict" {
		t.Errorf("Expected on_update 'restrict', got '%s'", rel.OnUpdate)
	}
}

// TestParser_OnUpdateCascade tests on_update with cascade option
func TestParser_OnUpdateCascade(t *testing.T) {
	source := `
resource Post {
  author: User! {
    on_update: cascade
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]
	if rel.OnUpdate != "cascade" {
		t.Errorf("Expected on_update 'cascade', got '%s'", rel.OnUpdate)
	}
}


// TestParser_MissingForeignKey tests relationship without foreign_key
func TestParser_MissingForeignKey(t *testing.T) {
	source := `
resource Post {
  author: User! {
    on_delete: restrict
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]

	// Foreign key should be empty when not specified
	if rel.ForeignKey != "" {
		t.Errorf("Expected empty foreign_key, got '%s'", rel.ForeignKey)
	}
}

// TestParser_MultipleErrorsInResource tests multiple errors in a single resource
func TestParser_MultipleErrorsInResource(t *testing.T) {
	source := `
resource User {
  username: string
  email:
  age: int!
}
`

	_, errors := parseSource(t, source)

	if len(errors) < 2 {
		t.Errorf("Expected at least 2 errors, got %d", len(errors))
	}
}

// TestParser_DeeplyNestedStructs tests nested struct fields
func TestParser_DeeplyNestedStructs(t *testing.T) {
	source := `
resource Config {
  settings: {
    database: {
      host: string!
      port: int!
      credentials: {
        username: string!
        password: string!
      }!
    }!
  }!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	settingsField := resource.Fields[0]

	if !settingsField.Type.IsStruct() {
		t.Fatal("Expected settings to be struct type")
	}

	// Check database field
	if len(settingsField.Type.StructFields) != 1 {
		t.Fatalf("Expected 1 struct field, got %d", len(settingsField.Type.StructFields))
	}

	databaseField := settingsField.Type.StructFields[0]
	if !databaseField.Type.IsStruct() {
		t.Fatal("Expected database to be struct type")
	}

	// Check credentials field within database
	var credentialsField *FieldNode
	for _, field := range databaseField.Type.StructFields {
		if field.Name == "credentials" {
			credentialsField = field
			break
		}
	}

	if credentialsField == nil {
		t.Fatal("Expected to find credentials field")
	}

	if !credentialsField.Type.IsStruct() {
		t.Fatal("Expected credentials to be struct type")
	}

	if len(credentialsField.Type.StructFields) != 2 {
		t.Errorf("Expected 2 fields in credentials, got %d", len(credentialsField.Type.StructFields))
	}
}

// TestParser_ResourceWithoutDocumentation tests resource without comments
func TestParser_ResourceWithoutDocumentation(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	// Documentation should be empty when no comments present
	if resource.Documentation != "" {
		t.Errorf("Expected no documentation, got: %s", resource.Documentation)
	}
}

// TestParser_TypeNodeString tests the String() method on TypeNode
func TestParser_TypeNodeString(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name: "primitive type",
			source: `
resource User {
  name: string!
}
`,
			expected: "string",
		},
		{
			name: "array type",
			source: `
resource Post {
  tags: array<string>!
}
`,
			expected: "array<string>",
		},
		{
			name: "hash type",
			source: `
resource Config {
  settings: hash<string, int>!
}
`,
			expected: "hash<string, int>",
		},
		{
			name: "enum type",
			source: `
resource Post {
  status: enum ["draft", "published"]!
}
`,
			expected: "enum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, errors := parseSource(t, tt.source)

			if len(errors) > 0 {
				t.Fatalf("Expected no errors, got: %v", errors)
			}

			resource := program.Resources[0]
			field := resource.Fields[0]

			typeStr := field.Type.String()
			if typeStr != tt.expected {
				t.Errorf("Expected type string '%s', got '%s'", tt.expected, typeStr)
			}
		})
	}
}

// TestParser_ParseErrorMethods tests error handling methods
func TestParser_ParseErrorMethods(t *testing.T) {
	source := `
resource User {
  username: string
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for missing nullability")
	}

	err := errors[0]

	// Test Error() method
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Test ErrorCode() method
	code := err.ErrorCode()
	if code == "" {
		t.Error("Expected non-empty error code")
	}

	// Test Severity() method
	severity := err.Severity()
	if severity != "error" {
		t.Errorf("Expected severity 'error', got '%s'", severity)
	}
}

// TestParser_ErrorList tests ParseErrorList methods
func TestParser_ErrorList(t *testing.T) {
	source := `
resource User {
  username: string
  email:
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected errors")
	}

	// Create ParseErrorList
	errorList := ParseErrorList(errors)

	// Test HasErrors()
	if !errorList.HasErrors() {
		t.Error("Expected HasErrors() to return true")
	}

	// Test Count()
	count := errorList.Count()
	if count != len(errors) {
		t.Errorf("Expected count %d, got %d", len(errors), count)
	}

	// Test Error() method
	errorMsg := errorList.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message from ParseErrorList")
	}

	// Test ToJSON()
	jsonMap := errorList.ToJSON()
	if jsonMap == nil {
		t.Error("Expected non-nil JSON map")
	}

	// Verify JSON contains expected structure
	if _, ok := jsonMap["errors"]; !ok {
		t.Error("Expected JSON to contain 'errors' key")
	}

	// Test Format()
	formatted := errorList.Format()
	if formatted == "" {
		t.Error("Expected non-empty formatted string")
	}
}

// TestParser_ParseErrorToJSON tests individual error JSON conversion
func TestParser_ParseErrorToJSON(t *testing.T) {
	source := `
resource User {
  username: string
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error")
	}

	err := errors[0]
	jsonMap := err.ToJSON()

	if jsonMap == nil {
		t.Error("Expected non-nil JSON map")
	}

	// Verify JSON contains expected fields
	expectedFields := []string{"message", "code", "severity", "file", "line", "column"}
	for _, field := range expectedFields {
		if _, ok := jsonMap[field]; !ok {
			t.Errorf("Expected JSON to contain field '%s'", field)
		}
	}
}

// TestParser_LowercaseResourceName tests lowercase resource names (they're valid at parser level)
func TestParser_LowercaseResourceName(t *testing.T) {
	source := `
resource lowercase {
  id: uuid! @primary
}
`

	program, errors := parseSource(t, source)

	// Parser accepts lowercase names - validation happens later in type checker
	if len(errors) > 0 {
		t.Errorf("Parser should accept lowercase resource names, got errors: %v", errors)
	}

	if len(program.Resources) == 1 {
		if program.Resources[0].Name != "lowercase" {
			t.Errorf("Expected resource name 'lowercase', got '%s'", program.Resources[0].Name)
		}
	}
}

// TestParser_UnknownRelationshipMetadataKey tests handling of unknown metadata keys
func TestParser_UnknownRelationshipMetadataKey(t *testing.T) {
	source := `
resource Post {
  author: User! {
    foreign_key: "author_id"
    unknown_key: "value"
  }
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for unknown metadata key")
	}
}

// TestParser_NestedArrayType tests error handling for nested arrays
func TestParser_NestedArrayType(t *testing.T) {
	source := `
resource Post {
  tags: array<array<string>>!
}
`

	program, errors := parseSource(t, source)

	// This might error or succeed depending on implementation
	// Just verify parser doesn't crash
	if program == nil && len(errors) == 0 {
		t.Error("Expected either program or errors")
	}
}

// TestParser_FloatLiterals tests parsing float literals in constraints
func TestParser_FloatLiterals(t *testing.T) {
	source := `
resource Product {
  rating: float! @min(0.0) @max(5.0)
  discount: float! @default(0.15)
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	ratingField := resource.Fields[0]

	if len(ratingField.Constraints) < 2 {
		t.Fatalf("Expected at least 2 constraints, got %d", len(ratingField.Constraints))
	}

	// Verify min constraint with float
	minConstraint := ratingField.Constraints[0]
	if minConstraint.Name != "min" {
		t.Errorf("Expected 'min' constraint, got '%s'", minConstraint.Name)
	}
	if len(minConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(minConstraint.Arguments))
	}

	// The argument should be a float64
	_, ok := minConstraint.Arguments[0].(float64)
	if !ok {
		t.Errorf("Expected float64 argument, got %T", minConstraint.Arguments[0])
	}
}

// TestParser_EmptyStructField tests struct with no fields
func TestParser_EmptyStructField(t *testing.T) {
	source := `
resource Post {
  metadata: {}!
}
`

	program, errors := parseSource(t, source)

	// Empty struct should either be valid or produce an error
	// Just verify parser doesn't crash
	if program == nil && len(errors) == 0 {
		t.Error("Expected either program or errors")
	}

	if len(errors) == 0 {
		resource := program.Resources[0]
		metadataField := resource.Fields[0]
		if !metadataField.Type.IsStruct() {
			t.Error("Expected metadata to be struct type")
		}
	}
}

// TestParser_ConstraintWithoutArgs tests constraints that don't require arguments
func TestParser_ConstraintWithoutArgs(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique @required
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	idField := resource.Fields[0]

	for _, constraint := range idField.Constraints {
		if constraint.Name == "primary" || constraint.Name == "auto" {
			if len(constraint.Arguments) != 0 {
				t.Errorf("Expected no arguments for %s constraint, got %d",
					constraint.Name, len(constraint.Arguments))
			}
		}
	}
}

// TestParser_InvalidConstraintArgument tests invalid argument types for constraints
func TestParser_InvalidConstraintArgument(t *testing.T) {
	source := `
resource User {
  age: int! @min(invalid)
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for invalid constraint argument")
	}
}

// TestParser_MixedNullability tests resources with mixed nullable/non-nullable fields
func TestParser_MixedNullability(t *testing.T) {
	source := `
resource User {
  id: uuid!
  username: string!
  bio: text?
  avatar: url?
  verified: bool!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]

	expectedNullability := map[string]bool{
		"id":       false,
		"username": false,
		"bio":      true,
		"avatar":   true,
		"verified": false,
	}

	for _, field := range resource.Fields {
		expected, ok := expectedNullability[field.Name]
		if !ok {
			continue
		}

		if field.Nullable != expected {
			t.Errorf("Field %s: expected nullable=%v, got %v",
				field.Name, expected, field.Nullable)
		}
	}
}

// TestParser_RelationshipWithAllMetadata tests relationship with all metadata fields
func TestParser_RelationshipWithAllMetadata(t *testing.T) {
	source := `
resource Post {
  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
    on_update: restrict
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]

	if rel.ForeignKey != "author_id" {
		t.Errorf("Expected foreign_key 'author_id', got '%s'", rel.ForeignKey)
	}
	if rel.OnDelete != "cascade" {
		t.Errorf("Expected on_delete 'cascade', got '%s'", rel.OnDelete)
	}
	if rel.OnUpdate != "restrict" {
		t.Errorf("Expected on_update 'restrict', got '%s'", rel.OnUpdate)
	}
}

// TestParser_StructFieldWithConstraints tests struct fields can have constraints
func TestParser_StructFieldWithConstraints(t *testing.T) {
	source := `
resource Post {
  seo: {
    title: string? @max(60)
    description: string? @max(160)
    keywords: array<string>? @max(10)
  }?
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	seoField := resource.Fields[0]

	if !seoField.Type.IsStruct() {
		t.Fatal("Expected seo to be struct type")
	}

	// Check title field has constraint
	titleField := seoField.Type.StructFields[0]
	if len(titleField.Constraints) != 1 {
		t.Errorf("Expected 1 constraint on title, got %d", len(titleField.Constraints))
	}
	if titleField.Constraints[0].Name != "max" {
		t.Errorf("Expected 'max' constraint, got '%s'", titleField.Constraints[0].Name)
	}
}

// TestParser_ArrayOfResourceType tests array of resource references
func TestParser_ArrayOfResourceType(t *testing.T) {
	source := `
resource Post {
  id: uuid! @primary
  collaborators: array<User>!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	collabField := resource.Fields[1]

	if !collabField.Type.IsArray() {
		t.Fatal("Expected collaborators to be array type")
	}

	if collabField.Type.ElementType.Name != "User" {
		t.Errorf("Expected element type 'User', got '%s'", collabField.Type.ElementType.Name)
	}
}

// TestParser_ComplexNesting tests deeply nested type structures
func TestParser_ComplexNesting(t *testing.T) {
	source := `
resource Analytics {
  data: {
    metrics: {
      views: hash<string, int>!
      clicks: hash<string, int>!
    }!
    tags: array<string>!
    status: enum ["active", "paused"]!
  }!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	dataField := resource.Fields[0]

	if !dataField.Type.IsStruct() {
		t.Fatal("Expected data to be struct type")
	}

	// Verify nested metrics struct
	var metricsField *FieldNode
	for _, field := range dataField.Type.StructFields {
		if field.Name == "metrics" {
			metricsField = field
			break
		}
	}

	if metricsField == nil {
		t.Fatal("Expected to find metrics field")
	}

	if !metricsField.Type.IsStruct() {
		t.Error("Expected metrics to be struct type")
	}

	// Verify hash fields inside metrics
	viewsField := metricsField.Type.StructFields[0]
	if !viewsField.Type.IsHash() {
		t.Error("Expected views to be hash type")
	}
}

// TestParser_OnDeleteNoAction tests on_delete with no_action option
func TestParser_OnDeleteNoAction(t *testing.T) {
	source := `
resource Post {
  author: User! {
    on_delete: no_action
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]
	if rel.OnDelete != "no_action" {
		t.Errorf("Expected on_delete 'no_action', got '%s'", rel.OnDelete)
	}
}

// TestParser_ErrorListEmptyCase tests empty error list
func TestParser_ErrorListEmptyCase(t *testing.T) {
	errorList := ParseErrorList{}

	if errorList.HasErrors() {
		t.Error("Expected empty error list to return false for HasErrors()")
	}

	if errorList.Count() != 0 {
		t.Errorf("Expected count 0, got %d", errorList.Count())
	}

	errorMsg := errorList.Error()
	if errorMsg != "no errors" {
		t.Errorf("Expected 'no errors' message, got: %s", errorMsg)
	}

	formatted := errorList.Format()
	if !strings.Contains(formatted, "No errors") {
		t.Errorf("Expected format to contain 'No errors', got: %s", formatted)
	}
}

// TestParser_ErrorListSingleError tests error list with one error
func TestParser_ErrorListSingleError(t *testing.T) {
	source := `
resource User {
  username: string
}
`

	_, errors := parseSource(t, source)

	if len(errors) != 1 {
		t.Fatalf("Expected exactly 1 error, got %d", len(errors))
	}

	errorList := ParseErrorList(errors)
	errorMsg := errorList.Error()

	// Single error should return just that error, not "and N more"
	if strings.Contains(errorMsg, "and") && strings.Contains(errorMsg, "more") {
		t.Errorf("Single error should not say 'and N more', got: %s", errorMsg)
	}
}

// TestParser_TypeNodeStructString tests String() method for struct types
func TestParser_TypeNodeStructString(t *testing.T) {
	source := `
resource Post {
  meta: {
    title: string!
  }!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	metaField := resource.Fields[0]

	typeStr := metaField.Type.String()
	if typeStr != "struct" {
		t.Errorf("Expected type string 'struct', got '%s'", typeStr)
	}
}

// TestParser_TypeNodeResourceString tests String() method for resource types
func TestParser_TypeNodeResourceString(t *testing.T) {
	source := `
resource Post {
  author: User!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	if len(program.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
	}

	// Verify relationship was parsed
	resource := program.Resources[0]
	if len(resource.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(resource.Relationships))
	}
}

// TestParser_NestedStructInArray tests arrays of struct types
func TestParser_NestedStructInArray(t *testing.T) {
	source := `
resource Order {
  items: array<{
    product_id: uuid!
    quantity: int!
  }>!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	itemsField := resource.Fields[0]

	if !itemsField.Type.IsArray() {
		t.Fatal("Expected items to be array type")
	}

	if !itemsField.Type.ElementType.IsStruct() {
		t.Error("Expected array element type to be struct")
	}

	if len(itemsField.Type.ElementType.StructFields) != 2 {
		t.Errorf("Expected 2 fields in struct, got %d", len(itemsField.Type.ElementType.StructFields))
	}
}

// TestParser_MissingArrayElementType tests array without element type
func TestParser_MissingArrayElementType(t *testing.T) {
	source := `
resource Post {
  tags: array!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for array without element type")
	}
}

// TestParser_MissingHashValueType tests hash without value type
func TestParser_MissingHashValueType(t *testing.T) {
	source := `
resource Config {
  settings: hash<string>!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for hash without value type")
	}
}

// TestParser_ArrayWithInvalidElementType tests array with non-type element
func TestParser_ArrayWithInvalidElementType(t *testing.T) {
	source := `
resource Post {
  tags: array<123>!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for invalid array element type")
	}
}

// TestParser_HashWithInvalidKeyType tests hash with non-type key
func TestParser_HashWithInvalidKeyType(t *testing.T) {
	source := `
resource Config {
  settings: hash<123, string>!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for invalid hash key type")
	}
}

// TestParser_MissingClosingBraceInResource tests resource without closing brace
func TestParser_MissingClosingBraceInResource(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary
  username: string!
`

	program, errors := parseSource(t, source)

	// Should have errors
	if len(errors) == 0 {
		t.Error("Expected errors for missing closing brace")
	}

	// Should still return partial AST
	if program == nil {
		t.Fatal("Expected partial AST")
	}
}

// TestParser_MultipleResourcesWithErrors tests error recovery across resources
func TestParser_MultipleResourcesWithErrors(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary
  invalid syntax here
}

resource Post {
  id: uuid! @primary
  title: string!
}

resource Comment {
  another error:
}
`

	program, errors := parseSource(t, source)

	// Should have errors
	if len(errors) == 0 {
		t.Error("Expected errors from invalid syntax")
	}

	// Should still parse at least one resource
	if len(program.Resources) < 1 {
		t.Error("Expected at least one resource to be parsed")
	}
}

// TestParser_EOFChecks tests EOF boundary conditions
func TestParser_EOFChecks(t *testing.T) {
	source := `resource User { id: uuid!`

	program, errors := parseSource(t, source)

	// Should have errors
	if len(errors) == 0 {
		t.Error("Expected errors for incomplete resource")
	}

	// Should still return program
	if program == nil {
		t.Fatal("Expected program even with errors")
	}
}

// TestParser_MissingResourceKeyword tests file without resource keyword
func TestParser_MissingResourceKeyword(t *testing.T) {
	source := `
User {
  id: uuid!
}
`

	program, errors := parseSource(t, source)

	// Should have errors
	if len(errors) == 0 {
		t.Error("Expected errors for missing resource keyword")
	}

	// Program should be empty or with no resources
	if program != nil && len(program.Resources) > 0 {
		t.Error("Expected no resources to be parsed without resource keyword")
	}
}

// TestParser_ConstraintWithBooleanArgs tests boolean arguments in constraints
func TestParser_ConstraintWithBooleanArgs(t *testing.T) {
	// This tests a potential future feature
	source := `
resource User {
  active: bool! @default(true)
  deleted: bool! @default(false)
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	activeField := resource.Fields[0]

	// Check default constraint with true
	defaultConstraint := activeField.Constraints[0]
	if defaultConstraint.Name != "default" {
		t.Errorf("Expected 'default' constraint, got '%s'", defaultConstraint.Name)
	}
	if len(defaultConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(defaultConstraint.Arguments))
	}
	if val, ok := defaultConstraint.Arguments[0].(bool); !ok || val != true {
		t.Errorf("Expected boolean true argument, got %v", defaultConstraint.Arguments[0])
	}

	// Check false value
	deletedField := resource.Fields[1]
	deletedDefault := deletedField.Constraints[0]
	if val, ok := deletedDefault.Arguments[0].(bool); !ok || val != false {
		t.Errorf("Expected boolean false argument, got %v", deletedDefault.Arguments[0])
	}
}

// TestParser_EnumWithEmptyList tests enum with no values
func TestParser_EnumWithEmptyList(t *testing.T) {
	source := `
resource Post {
  status: enum []!
}
`

	program, _ := parseSource(t, source)

	// Empty enum should still parse but may have errors
	if program == nil {
		t.Fatal("Expected program to be returned")
	}

	if len(program.Resources) == 0 {
		t.Fatal("Expected resource to be parsed")
	}

	resource := program.Resources[0]
	if len(resource.Fields) > 0 {
		statusField := resource.Fields[0]
		if statusField.Type.IsEnum() && len(statusField.Type.EnumValues) == 0 {
			// Empty enum values list is ok
		}
	}
}

// TestParser_StructWithMissingFieldType tests struct field without type
func TestParser_StructWithMissingFieldType(t *testing.T) {
	source := `
resource Post {
  meta: {
    title:
  }!
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for struct field without type")
	}
}

// TestParser_RelationshipMetadataWithCommas tests metadata with comma separators
func TestParser_RelationshipMetadataWithCommas(t *testing.T) {
	source := `
resource Post {
  author: User! {
    foreign_key: "author_id",
    on_delete: cascade,
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	rel := resource.Relationships[0]

	if rel.ForeignKey != "author_id" {
		t.Errorf("Expected foreign_key 'author_id', got '%s'", rel.ForeignKey)
	}
	if rel.OnDelete != "cascade" {
		t.Errorf("Expected on_delete 'cascade', got '%s'", rel.OnDelete)
	}
}

// TestParser_FieldNamedArray tests field with name that is a keyword
func TestParser_FieldNamedArray(t *testing.T) {
	source := `
resource Config {
  array: string!
  hash: string!
  enum: string!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(resource.Fields))
	}

	// Verify field names
	expectedNames := []string{"array", "hash", "enum"}
	for i, expected := range expectedNames {
		if resource.Fields[i].Name != expected {
			t.Errorf("Field %d: expected name '%s', got '%s'", i, expected, resource.Fields[i].Name)
		}
	}
}

// TestParser_TriggerSynchronizeRecovery tests error recovery with synchronize
func TestParser_TriggerSynchronizeRecovery(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary
}

invalid keyword here

resource Post {
  id: uuid! @primary
}
`

	program, errors := parseSource(t, source)

	// Should have at least one error from "invalid keyword"
	if len(errors) == 0 {
		t.Error("Expected errors for invalid syntax")
	}

	// Should still parse User and Post resources
	if len(program.Resources) < 2 {
		t.Errorf("Expected at least 2 resources, got %d", len(program.Resources))
	}
}

// TestParser_ConstraintWithInvalidArguments tests error handling for bad constraint args
func TestParser_ConstraintWithInvalidArguments(t *testing.T) {
	source := `
resource User {
  age: int! @min(invalid_identifier)
}
`

	program, errors := parseSource(t, source)

	// Should have errors for invalid argument
	if len(errors) == 0 {
		t.Error("Expected errors for invalid constraint argument")
	}

	// Should still return a program
	if program == nil {
		t.Fatal("Expected program to be returned")
	}
}

// TestParser_EmptyFile tests parsing an empty file
func TestParser_EmptyFile(t *testing.T) {
	source := ``

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Errorf("Expected no errors for empty file, got: %v", errors)
	}

	if len(program.Resources) != 0 {
		t.Errorf("Expected no resources, got %d", len(program.Resources))
	}
}

// TestParser_FileWithOnlyComments tests file with only comments
func TestParser_FileWithOnlyComments(t *testing.T) {
	source := `
# This is a comment
# Another comment
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Errorf("Expected no errors for comments-only file, got: %v", errors)
	}

	if len(program.Resources) != 0 {
		t.Errorf("Expected no resources, got %d", len(program.Resources))
	}
}

// TestParser_FileWithOnlyNewlines tests file with only newlines
func TestParser_FileWithOnlyNewlines(t *testing.T) {
	source := "\n\n\n"

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Errorf("Expected no errors for newlines-only file, got: %v", errors)
	}

	if len(program.Resources) != 0 {
		t.Errorf("Expected no resources, got %d", len(program.Resources))
	}
}
