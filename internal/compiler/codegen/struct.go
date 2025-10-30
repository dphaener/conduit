package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// generateStruct generates the Go struct definition for a resource
func (g *Generator) generateStruct(resource *ast.ResourceNode) error {
	// Add documentation comment if present
	if resource.Documentation != "" {
		g.writeLine("// %s", resource.Documentation)
	}

	g.writeLine("type %s struct {", resource.Name)
	g.indent++

	// Check if resource has an ID field
	hasID := false
	for _, field := range resource.Fields {
		if field.Name == "id" {
			hasID = true
			break
		}
	}

	// Collect all field information for alignment
	type fieldInfo struct {
		name string
		typ  string
		tags string
	}
	var fields []fieldInfo

	// Generate JSON:API type for this resource
	jsonapiType := g.toJSONAPIType(resource.Name)

	// Add ID field if not explicitly defined
	if !hasID {
		fields = append(fields, fieldInfo{
			name: "ID",
			typ:  "int64",
			tags: fmt.Sprintf("`jsonapi:\"primary,%s\" db:\"id\" json:\"id\"`", jsonapiType),
		})
	}

	// Collect all other fields
	for _, field := range resource.Fields {
		var tags string
		var typ string
		if field.Name == "id" {
			// ID fields always get primary tag
			tags = fmt.Sprintf("`jsonapi:\"primary,%s\" db:\"id\" json:\"id\"`", jsonapiType)
			// Make UUID IDs pointers so they can be omitted in JSON:API create requests
			typ = g.toGoType(field)
			if field.Type.Name == "uuid" {
				typ = "*" + typ
			}
		} else {
			tags = g.generateStructTags(field, resource.Name)
			typ = g.toGoType(field)
		}
		fields = append(fields, fieldInfo{
			name: g.toGoFieldName(field.Name),
			typ:  typ,
			tags: tags,
		})
	}

	// Calculate max lengths for alignment
	maxNameLen := 0
	maxTypeLen := 0
	for _, f := range fields {
		if len(f.name) > maxNameLen {
			maxNameLen = len(f.name)
		}
		if len(f.typ) > maxTypeLen {
			maxTypeLen = len(f.typ)
		}
	}

	// Generate aligned fields
	for _, f := range fields {
		namePadding := maxNameLen - len(f.name)
		typePadding := maxTypeLen - len(f.typ)
		g.writeLine("%s%s %s%s %s",
			f.name,
			strings.Repeat(" ", namePadding),
			f.typ,
			strings.Repeat(" ", typePadding),
			f.tags)
	}

	g.indent--
	g.writeLine("}")

	return nil
}

// generateTableName generates the TableName() method
func (g *Generator) generateTableName(resource *ast.ResourceNode) {
	tableName := g.toTableName(resource.Name)

	g.writeLine("// TableName returns the database table name for %s", resource.Name)
	g.writeLine("func (%s *%s) TableName() string {",
		strings.ToLower(resource.Name[0:1]), resource.Name)
	g.indent++
	g.writeLine("return %q", tableName)
	g.indent--
	g.writeLine("}")
}

// generateValidate generates the Validate() method
func (g *Generator) generateValidate(resource *ast.ResourceNode) {
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Validate validates the %s fields", resource.Name)
	g.writeLine("func (%s *%s) Validate() error {", receiverName, resource.Name)
	g.indent++

	hasValidations := false

	// Generate validations for required fields
	for _, field := range resource.Fields {
		if !field.Nullable {
			hasValidations = true
			g.generateFieldValidation(resource, field)
		}

		// Generate constraint validations
		for _, constraint := range field.Constraints {
			hasValidations = true
			g.generateConstraintValidation(resource, field, constraint)
		}
	}

	if !hasValidations {
		g.writeLine("// No validations defined")
	}

	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
}

// generateFieldValidation generates validation for a required field
func (g *Generator) generateFieldValidation(resource *ast.ResourceNode, field *ast.FieldNode) {
	receiverName := strings.ToLower(resource.Name[0:1])
	fieldName := g.toGoFieldName(field.Name)

	switch field.Type.Name {
	case "string", "text", "markdown":
		g.writeLine("if len(%s.%s) == 0 {", receiverName, fieldName)
		g.indent++
		g.writeLine("return fmt.Errorf(\"%s is required\")", field.Name)
		g.indent--
		g.writeLine("}")
	}
}

// generateConstraintValidation generates validation for a field constraint
func (g *Generator) generateConstraintValidation(resource *ast.ResourceNode, field *ast.FieldNode, constraint *ast.ConstraintNode) {
	receiverName := strings.ToLower(resource.Name[0:1])
	fieldName := g.toGoFieldName(field.Name)

	switch constraint.Name {
	case "min":
		if len(constraint.Arguments) > 0 {
			g.writeLine("if len(%s.%s) < %v {", receiverName, fieldName, extractLiteralValue(constraint.Arguments[0]))
			g.indent++
			errorMsg := constraint.Error
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("%s must be at least %v characters", field.Name, extractLiteralValue(constraint.Arguments[0]))
			}
			g.writeLine("return fmt.Errorf(%q)", errorMsg)
			g.indent--
			g.writeLine("}")
		}

	case "max":
		if len(constraint.Arguments) > 0 {
			g.writeLine("if len(%s.%s) > %v {", receiverName, fieldName, extractLiteralValue(constraint.Arguments[0]))
			g.indent++
			errorMsg := constraint.Error
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("%s must be at most %v characters", field.Name, extractLiteralValue(constraint.Arguments[0]))
			}
			g.writeLine("return fmt.Errorf(%q)", errorMsg)
			g.indent--
			g.writeLine("}")
		}
	}
}

// extractLiteralValue extracts the value from a literal expression node
func extractLiteralValue(expr ast.ExprNode) interface{} {
	if lit, ok := expr.(*ast.LiteralExpr); ok {
		return lit.Value
	}
	return nil
}
