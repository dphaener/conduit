package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// GenerateValidationMethod generates a Validate() method for a resource
func GenerateValidationMethod(resource *schema.ResourceSchema) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("// Validate validates all field constraints for %s\n", resource.Name))
	b.WriteString(fmt.Sprintf("func (r *%s) Validate() error {\n", resource.Name))
	b.WriteString("\terrs := validation.NewValidationErrors()\n\n")

	// Generate field-level validations
	for fieldName, field := range resource.Fields {
		if len(field.Constraints) == 0 && !field.Type.IsValidated() && !field.Type.Nullable {
			continue
		}

		b.WriteString(fmt.Sprintf("\t// Validate %s\n", fieldName))

		// Check nullability for required fields
		if !field.Type.Nullable {
			b.WriteString(fmt.Sprintf("\tif r.%s == nil {\n", toGoFieldName(fieldName)))
			b.WriteString(fmt.Sprintf("\t\terrs.Add(\"%s\", \"is required\")\n", fieldName))
			b.WriteString("\t} else {\n")
		} else {
			b.WriteString(fmt.Sprintf("\tif r.%s != nil {\n", toGoFieldName(fieldName)))
		}

		// Generate constraint validations
		for _, constraint := range field.Constraints {
			b.WriteString(generateConstraintValidation(fieldName, &constraint, field.Type))
		}

		// Generate type-specific validations
		if field.Type.BaseType == schema.TypeEmail {
			b.WriteString(fmt.Sprintf("\t\tif err := validation.EmailValidator{}.Validate(r.%s); err != nil {\n", toGoFieldName(fieldName)))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", err.Error())\n", fieldName))
			b.WriteString("\t\t}\n")
		} else if field.Type.BaseType == schema.TypeURL {
			b.WriteString(fmt.Sprintf("\t\tif err := validation.URLValidator{}.Validate(r.%s); err != nil {\n", toGoFieldName(fieldName)))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", err.Error())\n", fieldName))
			b.WriteString("\t\t}\n")
		}

		b.WriteString("\t}\n\n")
	}

	b.WriteString("\tif errs.HasErrors() {\n")
	b.WriteString("\t\treturn errs\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")

	return b.String()
}

// generateConstraintValidation generates validation code for a specific constraint
func generateConstraintValidation(fieldName string, constraint *schema.Constraint, fieldType *schema.TypeSpec) string {
	var b strings.Builder
	goFieldName := toGoFieldName(fieldName)

	switch constraint.Type {
	case schema.ConstraintMin:
		if fieldType.IsNumeric() {
			b.WriteString(fmt.Sprintf("\t\tif r.%s < %v {\n", goFieldName, constraint.Value))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", \"must be at least %v\")\n", fieldName, constraint.Value))
			b.WriteString("\t\t}\n")
		} else if fieldType.IsText() {
			b.WriteString(fmt.Sprintf("\t\tif len(r.%s) < %v {\n", goFieldName, constraint.Value))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", \"must be at least %v characters\")\n", fieldName, constraint.Value))
			b.WriteString("\t\t}\n")
		}

	case schema.ConstraintMax:
		if fieldType.IsNumeric() {
			b.WriteString(fmt.Sprintf("\t\tif r.%s > %v {\n", goFieldName, constraint.Value))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", \"must be at most %v\")\n", fieldName, constraint.Value))
			b.WriteString("\t\t}\n")
		} else if fieldType.IsText() {
			b.WriteString(fmt.Sprintf("\t\tif len(r.%s) > %v {\n", goFieldName, constraint.Value))
			b.WriteString(fmt.Sprintf("\t\t\terrs.Add(\"%s\", \"must be at most %v characters\")\n", fieldName, constraint.Value))
			b.WriteString("\t\t}\n")
		}

	case schema.ConstraintPattern:
		// For pattern constraints, we'd need to generate regex validation
		// This would require storing compiled regex patterns or compiling at runtime
		b.WriteString(fmt.Sprintf("\t\t// TODO: Pattern validation for %s\n", fieldName))
	}

	return b.String()
}

// GenerateValidateConstraintsMethod generates ValidateConstraints() method for custom constraint blocks
func GenerateValidateConstraintsMethod(resource *schema.ResourceSchema) string {
	if len(resource.ConstraintBlocks) == 0 && len(resource.Invariants) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("// ValidateConstraints validates custom constraint blocks for %s\n", resource.Name))
	b.WriteString(fmt.Sprintf("func (r *%s) ValidateConstraints(ctx context.Context) error {\n", resource.Name))
	b.WriteString("\terrs := validation.NewValidationErrors()\n\n")

	// Generate constraint block validations
	for _, constraint := range resource.ConstraintBlocks {
		b.WriteString(fmt.Sprintf("\t// Constraint: %s\n", constraint.Name))
		b.WriteString("\t// TODO: Implement constraint evaluation\n")
		b.WriteString(fmt.Sprintf("\t// When: %v\n", constraint.When))
		b.WriteString(fmt.Sprintf("\t// Condition: %v\n", constraint.Condition))
		b.WriteString(fmt.Sprintf("\t// Error: %s\n\n", constraint.Error))
	}

	// Generate invariant validations
	for _, invariant := range resource.Invariants {
		b.WriteString(fmt.Sprintf("\t// Invariant: %s\n", invariant.Name))
		b.WriteString("\t// TODO: Implement invariant evaluation\n")
		b.WriteString(fmt.Sprintf("\t// Condition: %v\n", invariant.Condition))
		b.WriteString(fmt.Sprintf("\t// Error: %s\n\n", invariant.Error))
	}

	b.WriteString("\tif errs.HasErrors() {\n")
	b.WriteString("\t\treturn errs\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")

	return b.String()
}

// toGoFieldName converts a field name to Go field name (PascalCase)
func toGoFieldName(name string) string {
	// Split by underscores
	parts := strings.Split(name, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
