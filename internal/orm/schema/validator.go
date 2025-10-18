// Package schema provides validation for ResourceSchema
package schema

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// ValidationError represents a schema validation error with context
type ValidationError struct {
	Resource string
	Field    string
	Message  string
	Location ast.SourceLocation
	Hint     string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	var b strings.Builder

	if e.Location.Line > 0 {
		b.WriteString(fmt.Sprintf("line %d, column %d: ", e.Location.Line, e.Location.Column))
	}

	if e.Resource != "" {
		b.WriteString(e.Resource)
		if e.Field != "" {
			b.WriteString(".")
			b.WriteString(e.Field)
		}
		b.WriteString(": ")
	}

	b.WriteString(e.Message)

	if e.Hint != "" {
		b.WriteString("\n  hint: ")
		b.WriteString(e.Hint)
	}

	return b.String()
}

// SchemaValidator validates resource schemas
type SchemaValidator struct {
	schemas  map[string]*ResourceSchema
	errors   []*ValidationError
	warnings []string
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas:  make(map[string]*ResourceSchema),
		errors:   make([]*ValidationError, 0),
		warnings: make([]string, 0),
	}
}

// ValidateStructural validates a single resource schema without cross-resource checks
// This is used during registration to allow forward references
func (v *SchemaValidator) ValidateStructural(schema *ResourceSchema) error {
	v.errors = make([]*ValidationError, 0)

	// Validate nullability
	if err := v.validateNullability(schema); err != nil {
		return err
	}

	// Validate primary key
	if err := v.validatePrimaryKey(schema); err != nil {
		return err
	}

	// Validate fields
	if err := v.validateFields(schema); err != nil {
		return err
	}

	// Skip relationship validation - that happens in ValidateAll()

	// Validate constraints
	if err := v.validateConstraints(schema); err != nil {
		return err
	}

	// Validate type compatibility
	if err := v.validateTypeCompatibility(schema); err != nil {
		return err
	}

	if len(v.errors) > 0 {
		var errMsgs []string
		for _, err := range v.errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("schema validation failed with %d errors:\n%s",
			len(v.errors), strings.Join(errMsgs, "\n"))
	}

	return nil
}

// Validate validates a single resource schema
func (v *SchemaValidator) Validate(schema *ResourceSchema, registry map[string]*ResourceSchema) error {
	v.schemas = registry
	v.errors = make([]*ValidationError, 0)

	// Validate nullability
	if err := v.validateNullability(schema); err != nil {
		return err
	}

	// Validate primary key
	if err := v.validatePrimaryKey(schema); err != nil {
		return err
	}

	// Validate fields
	if err := v.validateFields(schema); err != nil {
		return err
	}

	// Validate relationships
	if err := v.validateRelationships(schema); err != nil {
		return err
	}

	// Validate constraints
	if err := v.validateConstraints(schema); err != nil {
		return err
	}

	// Validate type compatibility
	if err := v.validateTypeCompatibility(schema); err != nil {
		return err
	}

	if len(v.errors) > 0 {
		return fmt.Errorf("schema validation failed with %d errors", len(v.errors))
	}

	return nil
}

// validateNullability ensures all types have explicit nullability markers
func (v *SchemaValidator) validateNullability(schema *ResourceSchema) error {
	for name, field := range schema.Fields {
		if err := v.validateTypeNullability(schema.Name, name, field.Type); err != nil {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Field:    name,
				Message:  err.Error(),
				Location: field.Location,
				Hint:     "Add ! for required or ? for optional",
			})
		}

		// Check for optional fields with defaults (warning)
		if field.Type.Nullable && field.Type.Default != nil {
			v.warnings = append(v.warnings,
				fmt.Sprintf("%s.%s is optional but has a default value", schema.Name, name))
		}
	}

	return nil
}

// validateTypeNullability recursively validates nullability for complex types
func (v *SchemaValidator) validateTypeNullability(resource, field string, typeSpec *TypeSpec) error {
	// Array element types must have explicit nullability
	if typeSpec.ArrayElement != nil {
		if err := v.validateTypeNullability(resource, field+"[]", typeSpec.ArrayElement); err != nil {
			return err
		}
	}

	// Hash key and value types must have explicit nullability
	if typeSpec.HashKey != nil {
		if err := v.validateTypeNullability(resource, field+".key", typeSpec.HashKey); err != nil {
			return err
		}
	}
	if typeSpec.HashValue != nil {
		if err := v.validateTypeNullability(resource, field+".value", typeSpec.HashValue); err != nil {
			return err
		}
	}

	// Struct fields must have explicit nullability
	if len(typeSpec.StructFields) > 0 {
		for structFieldName, structFieldType := range typeSpec.StructFields {
			if err := v.validateTypeNullability(resource, field+"."+structFieldName, structFieldType); err != nil {
				return err
			}
		}
	}

	return nil
}

// validatePrimaryKey ensures the resource has exactly one primary key
func (v *SchemaValidator) validatePrimaryKey(schema *ResourceSchema) error {
	primaryKeys := make([]*Field, 0)

	for _, field := range schema.Fields {
		for _, annotation := range field.Annotations {
			if annotation.Name == "primary" {
				primaryKeys = append(primaryKeys, field)
			}
		}
	}

	if len(primaryKeys) == 0 {
		v.errors = append(v.errors, &ValidationError{
			Resource: schema.Name,
			Message:  "resource must have a primary key",
			Location: schema.Location,
			Hint:     "Add a field with @primary annotation, e.g.: id: uuid! @primary @auto",
		})
	} else if len(primaryKeys) > 1 {
		v.errors = append(v.errors, &ValidationError{
			Resource: schema.Name,
			Message:  fmt.Sprintf("resource has %d primary keys, expected 1", len(primaryKeys)),
			Location: schema.Location,
			Hint:     "Only one field should have @primary annotation",
		})
	} else {
		// Primary key must be non-nullable
		pk := primaryKeys[0]
		if pk.Type.Nullable {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Field:    pk.Name,
				Message:  "primary key must be non-nullable (!)",
				Location: pk.Location,
				Hint:     fmt.Sprintf("Change %s: %s to %s: %s!", pk.Name, pk.Type.String(), pk.Name, strings.TrimSuffix(pk.Type.String(), "?")),
			})
		}
	}

	return nil
}

// validateFields validates field constraints and types
func (v *SchemaValidator) validateFields(schema *ResourceSchema) error {
	for name, field := range schema.Fields {
		// Validate constraint compatibility with field type
		for _, constraint := range field.Constraints {
			if err := v.validateFieldConstraint(schema.Name, name, field.Type, constraint); err != nil {
				v.errors = append(v.errors, &ValidationError{
					Resource: schema.Name,
					Field:    name,
					Message:  err.Error(),
					Location: constraint.Location,
				})
			}
		}
	}

	return nil
}

// validateFieldConstraint checks if a constraint is compatible with a field type
func (v *SchemaValidator) validateFieldConstraint(resource, field string, typeSpec *TypeSpec, constraint Constraint) error {
	switch constraint.Type {
	case ConstraintMin, ConstraintMax:
		if !typeSpec.IsNumeric() && !typeSpec.IsText() {
			return fmt.Errorf("@%s constraint only applies to numeric and text types", constraint.Type)
		}

	case ConstraintPattern:
		if !typeSpec.IsText() {
			return fmt.Errorf("@pattern constraint only applies to text types")
		}

	case ConstraintUnique, ConstraintIndex:
		// These can't be applied to text or json fields in most databases
		if typeSpec.BaseType == TypeText || typeSpec.BaseType == TypeJSON || typeSpec.BaseType == TypeJSONB {
			return fmt.Errorf("@%s constraint cannot be applied to %s type", constraint.Type, typeSpec.BaseType)
		}
	}

	return nil
}

// validateRelationships validates relationship definitions
func (v *SchemaValidator) validateRelationships(schema *ResourceSchema) error {
	for name, rel := range schema.Relationships {
		// Target resource must exist
		target, exists := v.schemas[rel.TargetResource]
		if !exists {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Field:    name,
				Message:  fmt.Sprintf("references unknown resource %s", rel.TargetResource),
				Location: rel.Location,
				Hint:     "Ensure the target resource is defined",
			})
			continue
		}

		// Validate cascade actions
		if err := v.validateCascadeAction(schema.Name, name, rel); err != nil {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Field:    name,
				Message:  err.Error(),
				Location: rel.Location,
			})
		}

		// For has_many_through, validate join table
		if rel.Type == RelationshipHasManyThrough {
			if rel.JoinTable == "" && rel.ThroughResource == "" {
				v.errors = append(v.errors, &ValidationError{
					Resource: schema.Name,
					Field:    name,
					Message:  "has_many_through relationship requires join_table or through resource",
					Location: rel.Location,
				})
			}
		}

		// Validate foreign key target exists
		if rel.Type == RelationshipBelongsTo {
			// Ensure target has a primary key
			if _, err := target.GetPrimaryKey(); err != nil {
				v.errors = append(v.errors, &ValidationError{
					Resource: schema.Name,
					Field:    name,
					Message:  fmt.Sprintf("target resource %s has no primary key", rel.TargetResource),
					Location: rel.Location,
				})
			}
		}
	}

	return nil
}

// validateCascadeAction validates cascade actions for relationships
func (v *SchemaValidator) validateCascadeAction(resource, field string, rel *Relationship) error {
	// set_null requires nullable relationship
	if rel.OnDelete == CascadeSetNull && !rel.Nullable {
		return fmt.Errorf("on_delete: set_null requires nullable relationship (use %s: %s?)", field, rel.TargetResource)
	}

	return nil
}

// validateConstraints validates constraint blocks
func (v *SchemaValidator) validateConstraints(schema *ResourceSchema) error {
	for _, constraint := range schema.ConstraintBlocks {
		// Validate 'on' events
		for _, event := range constraint.On {
			if event != "create" && event != "update" && event != "delete" {
				v.errors = append(v.errors, &ValidationError{
					Resource: schema.Name,
					Message:  fmt.Sprintf("constraint %s: invalid event '%s'", constraint.Name, event),
					Location: constraint.Location,
					Hint:     "Valid events are: create, update, delete",
				})
			}
		}

		// Ensure condition is present
		if constraint.Condition == nil {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Message:  fmt.Sprintf("constraint %s: missing condition", constraint.Name),
				Location: constraint.Location,
			})
		}

		// Ensure error message is present
		if constraint.Error == "" {
			v.errors = append(v.errors, &ValidationError{
				Resource: schema.Name,
				Message:  fmt.Sprintf("constraint %s: missing error message", constraint.Name),
				Location: constraint.Location,
			})
		}
	}

	return nil
}

// validateTypeCompatibility validates default values and constraint values match field types
func (v *SchemaValidator) validateTypeCompatibility(schema *ResourceSchema) error {
	for name, field := range schema.Fields {
		// Validate default value type matches field type
		if field.Type.Default != nil {
			if err := v.checkTypeMatch(field.Type, field.Type.Default); err != nil {
				v.errors = append(v.errors, &ValidationError{
					Resource: schema.Name,
					Field:    name,
					Message:  fmt.Sprintf("default value type mismatch: %s", err),
					Location: field.Location,
				})
			}
		}

		// Validate constraint values match field type
		for _, constraint := range field.Constraints {
			if constraint.Value != nil {
				if err := v.checkConstraintValueType(field.Type, constraint); err != nil {
					v.errors = append(v.errors, &ValidationError{
						Resource: schema.Name,
						Field:    name,
						Message:  err.Error(),
						Location: constraint.Location,
					})
				}
			}
		}
	}

	return nil
}

// checkTypeMatch validates a value matches a type specification
func (v *SchemaValidator) checkTypeMatch(typeSpec *TypeSpec, value interface{}) error {
	switch typeSpec.BaseType {
	case TypeInt, TypeBigInt:
		if _, ok := value.(int); !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
	case TypeFloat, TypeDecimal:
		switch value.(type) {
		case float64, float32:
			return nil
		default:
			return fmt.Errorf("expected float, got %T", value)
		}
	case TypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case TypeString, TypeText, TypeMarkdown, TypeEmail, TypeURL, TypePhone:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case TypeEnum:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for enum, got %T", value)
		}
		// Check if value is in enum values
		found := false
		for _, enumVal := range typeSpec.EnumValues {
			if enumVal == strVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value %s not in enum values %v", strVal, typeSpec.EnumValues)
		}
	}

	return nil
}

// checkConstraintValueType validates constraint value matches field type
func (v *SchemaValidator) checkConstraintValueType(typeSpec *TypeSpec, constraint Constraint) error {
	switch constraint.Type {
	case ConstraintMin, ConstraintMax:
		if typeSpec.IsNumeric() {
			switch constraint.Value.(type) {
			case int, float64, float32:
				return nil
			default:
				return fmt.Errorf("@%s value must be numeric for numeric field", constraint.Type)
			}
		} else if typeSpec.IsText() {
			if _, ok := constraint.Value.(int); !ok {
				return fmt.Errorf("@%s value must be int for text field (length)", constraint.Type)
			}
		}
	case ConstraintPattern:
		if _, ok := constraint.Value.(string); !ok {
			return fmt.Errorf("@pattern value must be a string (regex pattern)")
		}
	}

	return nil
}

// Errors returns all validation errors
func (v *SchemaValidator) Errors() []*ValidationError {
	return v.errors
}

// Warnings returns all validation warnings
func (v *SchemaValidator) Warnings() []string {
	return v.warnings
}
