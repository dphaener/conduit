package typechecker

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TypeChecker performs type checking and nullability analysis on a Conduit AST
type TypeChecker struct {
	// Resource registry - maps resource name to resource definition
	resources map[string]*ast.ResourceNode

	// Current resource being type-checked
	currentResource *ast.ResourceNode

	// Current scope - maps variable names to their types
	currentScope map[string]Type

	// Custom functions defined in resources
	customFunctions map[string]*Function

	// Accumulated errors
	errors ErrorList
}

// NewTypeChecker creates a new type checker
func NewTypeChecker() *TypeChecker {
	return &TypeChecker{
		resources:       make(map[string]*ast.ResourceNode),
		currentScope:    make(map[string]Type),
		customFunctions: make(map[string]*Function),
		errors:          make(ErrorList, 0),
	}
}

// CheckProgram is the main entry point for type checking
// It type-checks all resources in the program and returns any errors found
func (tc *TypeChecker) CheckProgram(prog *ast.Program) ErrorList {
	// First pass: Register all resources
	for _, resource := range prog.Resources {
		tc.resources[resource.Name] = resource
	}

	// Second pass: Type check each resource
	for _, resource := range prog.Resources {
		tc.checkResource(resource)
	}

	return tc.errors
}

// checkResource type-checks a single resource
func (tc *TypeChecker) checkResource(resource *ast.ResourceNode) {
	tc.currentResource = resource

	// Check all fields
	for _, field := range resource.Fields {
		tc.checkField(field)
	}

	// Check all hooks
	for _, hook := range resource.Hooks {
		tc.checkHook(hook)
	}

	// Check all validations
	for _, validation := range resource.Validations {
		tc.checkValidation(validation)
	}

	// Check all constraints
	for _, constraint := range resource.Constraints {
		tc.checkConstraint(constraint)
	}

	// Check computed fields
	for _, computed := range resource.Computed {
		tc.checkComputed(computed)
	}

	// Check relationships
	for _, relationship := range resource.Relationships {
		tc.checkRelationship(relationship)
	}

	// Reset current resource
	tc.currentResource = nil
}

// checkField type-checks a field definition
func (tc *TypeChecker) checkField(field *ast.FieldNode) {
	// Verify the field type is valid
	_, err := TypeFromASTNode(field.Type, field.Nullable)
	if err != nil {
		tc.errors = append(tc.errors, &TypeError{
			Code:     ErrUndefinedType,
			Type:     "invalid_field_type",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Invalid type for field %s: %s", field.Name, err.Error()),
			Location: field.Location(),
		})
		return
	}

	// Check field-level constraints
	for _, constraint := range field.Constraints {
		tc.checkFieldConstraint(field, constraint)
	}

	// Check default value if present
	if field.Default != nil {
		tc.checkDefaultValue(field)
	}
}

// checkFieldConstraint validates a field-level constraint
func (tc *TypeChecker) checkFieldConstraint(field *ast.FieldNode, constraint *ast.ConstraintNode) {
	fieldType, err := TypeFromASTNode(field.Type, field.Nullable)
	if err != nil {
		return // Already reported in checkField
	}

	switch constraint.Name {
	case "min", "max":
		// @min and @max only work on numeric and string types
		if prim, ok := fieldType.(*PrimitiveType); ok {
			if prim.Name != typeInt && prim.Name != typeFloat && prim.Name != "string" && prim.Name != "text" {
				tc.errors = append(tc.errors, NewInvalidConstraintType(
					constraint.Location(),
					constraint.Name,
					fieldType,
					"only valid for int, float, string, or text types",
				))
			}

			// Check argument type
			if len(constraint.Arguments) > 0 {
				argType, err := tc.inferExpr(constraint.Arguments[0])
				if err == nil {
					var expectedType Type
					if prim.Name == typeInt || prim.Name == typeFloat {
						expectedType = NewPrimitiveType(prim.Name, false)
					} else {
						expectedType = NewPrimitiveType("int", false)
					}

					if !expectedType.IsAssignableFrom(argType) {
						tc.errors = append(tc.errors, NewConstraintTypeMismatch(
							constraint.Location(),
							constraint.Name,
							expectedType,
							argType,
						))
					}
				}
			}
		} else {
			tc.errors = append(tc.errors, NewInvalidConstraintType(
				constraint.Location(),
				constraint.Name,
				fieldType,
				"only valid for primitive types",
			))
		}

	case "pattern":
		// @pattern only works on string types
		if prim, ok := fieldType.(*PrimitiveType); ok {
			if prim.Name != "string" && prim.Name != "text" {
				tc.errors = append(tc.errors, NewInvalidConstraintType(
					constraint.Location(),
					"pattern",
					fieldType,
					"only valid for string or text types",
				))
			}
		}

	case "unique", "primary", "auto", "auto_update":
		// These are always valid

	case "default":
		// Check that default value matches field type
		if len(constraint.Arguments) > 0 {
			argType, err := tc.inferExpr(constraint.Arguments[0])
			if err == nil {
				if !fieldType.IsAssignableFrom(argType) {
					tc.errors = append(tc.errors, NewConstraintTypeMismatch(
						constraint.Location(),
						"default",
						fieldType,
						argType,
					))
				}
			}
		}
	}
}

// checkDefaultValue validates a field's default value
func (tc *TypeChecker) checkDefaultValue(field *ast.FieldNode) {
	fieldType, err := TypeFromASTNode(field.Type, field.Nullable)
	if err != nil {
		return
	}

	defaultType, err := tc.inferExpr(field.Default)
	if err != nil {
		tc.errors = append(tc.errors, &TypeError{
			Code:     ErrTypeMismatch,
			Type:     "invalid_default",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Invalid default value for field %s: %s", field.Name, err.Error()),
			Location: field.Location(),
		})
		return
	}

	if !fieldType.IsAssignableFrom(defaultType) {
		tc.errors = append(tc.errors, NewTypeMismatch(
			field.Location(),
			fieldType,
			defaultType,
			fmt.Sprintf("default value for field %s", field.Name),
		))
	}
}

// checkHook type-checks a lifecycle hook
func (tc *TypeChecker) checkHook(hook *ast.HookNode) {
	// Create a new scope for the hook
	oldScope := tc.currentScope
	tc.currentScope = make(map[string]Type)

	// Add 'self' to scope if we're in a resource
	if tc.currentResource != nil {
		tc.currentScope["self"] = NewResourceType(tc.currentResource.Name, false)
	}

	// Type check all statements in the hook
	for _, stmt := range hook.Body {
		tc.checkStmt(stmt)
	}

	// Restore old scope
	tc.currentScope = oldScope
}

// checkValidation type-checks a validation block
func (tc *TypeChecker) checkValidation(validation *ast.ValidationNode) {
	// Create scope for validation
	oldScope := tc.currentScope
	tc.currentScope = make(map[string]Type)

	if tc.currentResource != nil {
		tc.currentScope["self"] = NewResourceType(tc.currentResource.Name, false)
	}

	// The condition must be a boolean expression
	if validation.Condition != nil {
		condType, err := tc.inferExpr(validation.Condition)
		if err == nil {
			expectedBool := NewPrimitiveType("bool", false)
			if !expectedBool.IsAssignableFrom(condType) {
				tc.errors = append(tc.errors, NewTypeMismatch(
					validation.Location(),
					expectedBool,
					condType,
					"validation condition",
				))
			}
		}
	}

	tc.currentScope = oldScope
}

// checkConstraint type-checks a constraint block
func (tc *TypeChecker) checkConstraint(constraint *ast.ConstraintNode) {
	// Create scope for constraint
	oldScope := tc.currentScope
	tc.currentScope = make(map[string]Type)

	if tc.currentResource != nil {
		tc.currentScope["self"] = NewResourceType(tc.currentResource.Name, false)
	}

	// Check 'when' condition if present
	if constraint.When != nil {
		whenType, err := tc.inferExpr(constraint.When)
		if err == nil {
			expectedBool := NewPrimitiveType("bool", false)
			if !expectedBool.IsAssignableFrom(whenType) {
				tc.errors = append(tc.errors, NewTypeMismatch(
					constraint.Location(),
					expectedBool,
					whenType,
					"constraint when condition",
				))
			}
		}
	}

	// Check main condition
	if constraint.Condition != nil {
		condType, err := tc.inferExpr(constraint.Condition)
		if err == nil {
			expectedBool := NewPrimitiveType("bool", false)
			if !expectedBool.IsAssignableFrom(condType) {
				tc.errors = append(tc.errors, NewTypeMismatch(
					constraint.Location(),
					expectedBool,
					condType,
					"constraint condition",
				))
			}
		}
	}

	tc.currentScope = oldScope
}

// checkComputed type-checks a computed field
func (tc *TypeChecker) checkComputed(computed *ast.ComputedNode) {
	// Create scope for computed field
	oldScope := tc.currentScope
	tc.currentScope = make(map[string]Type)

	if tc.currentResource != nil {
		tc.currentScope["self"] = NewResourceType(tc.currentResource.Name, false)
	}

	// Infer the type of the computed expression
	if computed.Body != nil {
		bodyType, err := tc.inferExpr(computed.Body)
		if err == nil {
			// Check that it matches the declared type
			declaredType, err := TypeFromASTNode(computed.Type, false)
			if err == nil {
				if !declaredType.IsAssignableFrom(bodyType) {
					tc.errors = append(tc.errors, NewTypeMismatch(
						computed.Location(),
						declaredType,
						bodyType,
						fmt.Sprintf("computed field %s", computed.Name),
					))
				}
			}
		}
	}

	tc.currentScope = oldScope
}

// checkRelationship validates a relationship between resources
func (tc *TypeChecker) checkRelationship(rel *ast.RelationshipNode) {
	// Verify the referenced resource exists
	targetResource, exists := tc.resources[rel.Type]
	if !exists {
		tc.errors = append(tc.errors, &TypeError{
			Code:     ErrUndefinedType,
			Type:     "undefined_resource",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Relationship %s references undefined resource: %s", rel.Name, rel.Type),
			Location: rel.Location(),
		})
		return
	}

	// Validate on_delete rules
	validOnDelete := map[string]bool{
		"cascade":  true,
		"restrict": true,
		"nullify":  true,
		"":         true, // Empty is valid (defaults to restrict)
	}
	if !validOnDelete[rel.OnDelete] {
		tc.errors = append(tc.errors, &TypeError{
			Code:     ErrInvalidConstraintType,
			Type:     "invalid_on_delete",
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"Invalid on_delete rule '%s' for relationship %s. Valid values: cascade, restrict, nullify",
				rel.OnDelete, rel.Name,
			),
			Location: rel.Location(),
		})
	}

	// Verify nullify is only used with nullable relationships
	if rel.OnDelete == "nullify" && !rel.Nullable {
		tc.errors = append(tc.errors, &TypeError{
			Code:     ErrInvalidConstraintType,
			Type:     "invalid_nullify",
			Severity: SeverityError,
			Message:  fmt.Sprintf("on_delete: nullify requires nullable relationship (use %s? instead of %s!)", rel.Type, rel.Type),
			Location: rel.Location(),
		})
	}

	// Note: For has-many-through relationships, we're not currently validating
	// that the through table exists as a resource, since it might be defined
	// as a pure join table in migrations. This could be enhanced in the future
	// to check migration files.

	// Suppress unused variable warning
	_ = targetResource
}

// checkStmt type-checks a statement
func (tc *TypeChecker) checkStmt(stmt ast.StmtNode) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		_, _ = tc.inferExpr(s.Expr)

	case *ast.AssignmentStmt:
		tc.checkAssignment(s)

	case *ast.LetStmt:
		tc.checkLet(s)

	case *ast.ReturnStmt:
		if s.Value != nil {
			_, _ = tc.inferExpr(s.Value)
		}

	case *ast.IfStmt:
		tc.checkIf(s)

	case *ast.BlockStmt:
		for _, stmt := range s.Statements {
			tc.checkStmt(stmt)
		}

	case *ast.RescueStmt:
		for _, stmt := range s.Try {
			tc.checkStmt(stmt)
		}
		for _, stmt := range s.RescueBody {
			tc.checkStmt(stmt)
		}

	case *ast.MatchStmt:
		tc.checkMatch(s)
	}
}

// checkAssignment enforces nullability rules for assignments
func (tc *TypeChecker) checkAssignment(assign *ast.AssignmentStmt) {
	// Infer the type of the value being assigned
	valueType, err := tc.inferExpr(assign.Value)
	if err != nil {
		return
	}

	// Determine the type of the target
	var targetType Type

	switch target := assign.Target.(type) {
	case *ast.FieldAccessExpr:
		// Assignment to a field (e.g., self.field = value)
		targetType, err = tc.inferFieldAccess(target)
		if err != nil {
			return
		}

	case *ast.IdentifierExpr:
		// Assignment to a variable
		var ok bool
		targetType, ok = tc.currentScope[target.Name]
		if !ok {
			// Undefined variable - this is an error
			tc.errors = append(tc.errors, &TypeError{
				Code:     ErrUndefinedField,
				Type:     "undefined_variable",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Undefined variable: %s", target.Name),
				Location: assign.Location(),
			})
			return
		}

	default:
		// Invalid assignment target
		return
	}

	// Check nullability: nullable cannot assign to required
	if !targetType.IsAssignableFrom(valueType) {
		// Check if it's a nullability violation specifically
		if !targetType.IsNullable() && valueType.IsNullable() {
			tc.errors = append(tc.errors, NewNullabilityViolation(
				assign.Location(),
				targetType,
				valueType,
			))
		} else {
			tc.errors = append(tc.errors, NewTypeMismatch(
				assign.Location(),
				targetType,
				valueType,
				"assignment",
			))
		}
	}
}

// checkLet type-checks a let statement
func (tc *TypeChecker) checkLet(let *ast.LetStmt) {
	// Infer the type of the value
	valueType, err := tc.inferExpr(let.Value)
	if err != nil {
		return
	}

	// If type is explicitly declared, check compatibility
	if let.Type != nil {
		declaredType, err := TypeFromASTNode(let.Type, false)
		if err != nil {
			tc.errors = append(tc.errors, &TypeError{
				Code:     ErrUndefinedType,
				Type:     "invalid_variable_type",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Invalid type for variable %s: %s", let.Name, err.Error()),
				Location: let.Location(),
			})
			return
		}

		if !declaredType.IsAssignableFrom(valueType) {
			tc.errors = append(tc.errors, NewTypeMismatch(
				let.Location(),
				declaredType,
				valueType,
				fmt.Sprintf("variable %s", let.Name),
			))
			return
		}

		// Add to scope with declared type
		tc.currentScope[let.Name] = declaredType
	} else {
		// Add to scope with inferred type
		tc.currentScope[let.Name] = valueType
	}
}

// checkIf type-checks an if statement
func (tc *TypeChecker) checkIf(ifStmt *ast.IfStmt) {
	// Check condition
	if ifStmt.Condition != nil {
		condType, err := tc.inferExpr(ifStmt.Condition)
		if err == nil {
			expectedBool := NewPrimitiveType("bool", false)
			if !expectedBool.IsAssignableFrom(condType) {
				tc.errors = append(tc.errors, NewTypeMismatch(
					ifStmt.Location(),
					expectedBool,
					condType,
					"if condition",
				))
			}
		}
	}

	// Check then branch
	for _, stmt := range ifStmt.ThenBranch {
		tc.checkStmt(stmt)
	}

	// Check elsif branches
	for _, elsif := range ifStmt.ElsIfBranches {
		if elsif.Condition != nil {
			condType, err := tc.inferExpr(elsif.Condition)
			if err == nil {
				expectedBool := NewPrimitiveType("bool", false)
				if !expectedBool.IsAssignableFrom(condType) {
					tc.errors = append(tc.errors, NewTypeMismatch(
						elsif.Loc,
						expectedBool,
						condType,
						"elsif condition",
					))
				}
			}
		}
		for _, stmt := range elsif.Body {
			tc.checkStmt(stmt)
		}
	}

	// Check else branch
	for _, stmt := range ifStmt.ElseBranch {
		tc.checkStmt(stmt)
	}
}

// checkMatch type-checks a match statement
func (tc *TypeChecker) checkMatch(match *ast.MatchStmt) {
	// Infer type of value being matched
	_, _ = tc.inferExpr(match.Value)

	// Check each case
	for _, matchCase := range match.Cases {
		// Check pattern
		_, _ = tc.inferExpr(matchCase.Pattern)

		// Check body
		for _, stmt := range matchCase.Body {
			tc.checkStmt(stmt)
		}
	}
}
