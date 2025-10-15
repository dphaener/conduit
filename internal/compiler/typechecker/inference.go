package typechecker

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// inferExpr infers the type of an expression
func (tc *TypeChecker) inferExpr(expr ast.ExprNode) (Type, error) {
	if expr == nil {
		return nil, fmt.Errorf("nil expression")
	}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return tc.inferLiteral(e)

	case *ast.IdentifierExpr:
		return tc.inferIdentifier(e)

	case *ast.SelfExpr:
		return tc.inferSelf(e)

	case *ast.FieldAccessExpr:
		return tc.inferFieldAccess(e)

	case *ast.SafeNavigationExpr:
		return tc.inferSafeNavigation(e)

	case *ast.CallExpr:
		return tc.inferCall(e)

	case *ast.BinaryExpr:
		return tc.inferBinary(e)

	case *ast.UnaryExpr:
		return tc.inferUnary(e)

	case *ast.LogicalExpr:
		return tc.inferLogical(e)

	case *ast.NullCoalesceExpr:
		return tc.inferNullCoalesce(e)

	case *ast.ArrayLiteralExpr:
		return tc.inferArrayLiteral(e)

	case *ast.HashLiteralExpr:
		return tc.inferHashLiteral(e)

	case *ast.IndexExpr:
		return tc.inferIndex(e)

	case *ast.ParenExpr:
		return tc.inferExpr(e.Expr)

	case *ast.InterpolatedStringExpr:
		return NewPrimitiveType("string", false), nil

	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// inferLiteral infers the type of a literal expression
func (tc *TypeChecker) inferLiteral(lit *ast.LiteralExpr) (Type, error) {
	if lit.Value == nil {
		// nil literal - we'll treat this as a special case
		// The type will be determined by context
		return NewPrimitiveType("nil", true), nil
	}

	switch lit.Value.(type) {
	case string:
		return NewPrimitiveType("string", false), nil
	case int, int64:
		return NewPrimitiveType(typeInt, false), nil
	case float64:
		return NewPrimitiveType(typeFloat, false), nil
	case bool:
		return NewPrimitiveType("bool", false), nil
	default:
		return nil, fmt.Errorf("unknown literal type: %T", lit.Value)
	}
}

// inferIdentifier infers the type of an identifier
func (tc *TypeChecker) inferIdentifier(ident *ast.IdentifierExpr) (Type, error) {
	// Look up in local scope first
	if typ, ok := tc.currentScope[ident.Name]; ok {
		return typ, nil
	}

	// Check if it's a resource name
	if res, ok := tc.resources[ident.Name]; ok {
		return NewResourceType(res.Name, false), nil
	}

	return nil, fmt.Errorf("undefined identifier: %s", ident.Name)
}

// inferSelf infers the type of the 'self' keyword
func (tc *TypeChecker) inferSelf(self *ast.SelfExpr) (Type, error) {
	if tc.currentResource == nil {
		return nil, fmt.Errorf("'self' used outside of resource context")
	}

	return NewResourceType(tc.currentResource.Name, false), nil
}

// inferFieldAccess infers the type of a field access expression
func (tc *TypeChecker) inferFieldAccess(fa *ast.FieldAccessExpr) (Type, error) {
	// Infer the type of the object being accessed
	objType, err := tc.inferExpr(fa.Object)
	if err != nil {
		return nil, err
	}

	// Handle different object types
	switch t := objType.(type) {
	case *ResourceType:
		// Look up field in any resource using the resource registry
		fieldType, err := tc.lookupResourceField(t.Name, fa.Field)
		if err != nil {
			return nil, NewUndefinedField(fa.Location(), fa.Field, t.Name)
		}
		return fieldType, nil

	case *StructType:
		// Look up field in struct
		fieldType, ok := t.GetField(fa.Field)
		if !ok {
			return nil, NewUndefinedField(fa.Location(), fa.Field, t.String())
		}
		return fieldType, nil

	default:
		return nil, fmt.Errorf("cannot access field '%s' on type %s", fa.Field, objType.String())
	}
}

// inferSafeNavigation infers the type of a safe navigation expression
// Safe navigation always returns a nullable type
func (tc *TypeChecker) inferSafeNavigation(sn *ast.SafeNavigationExpr) (Type, error) {
	// Infer the type of the object
	objType, err := tc.inferExpr(sn.Object)
	if err != nil {
		return nil, err
	}

	// The object must be nullable for safe navigation to make sense
	if !objType.IsNullable() {
		// This is a warning, not an error - safe navigation on required type is unnecessary
		tc.errors = append(tc.errors, &TypeError{
			Code:       ErrUnnecessaryUnwrap,
			Type:       "unnecessary_safe_navigation",
			Severity:   SeverityWarning,
			Message:    "Unnecessary safe navigation on required type",
			Location:   sn.Location(),
			Suggestion: "Use . instead of ?. for required types",
		})
	}

	// Look up the field type similar to field access
	var fieldType Type
	switch t := objType.(type) {
	case *ResourceType:
		fieldType, err = tc.lookupResourceField(t.Name, sn.Field)
		if err != nil {
			return nil, err
		}

	case *StructType:
		var ok bool
		fieldType, ok = t.GetField(sn.Field)
		if !ok {
			return nil, NewUndefinedField(sn.Location(), sn.Field, t.String())
		}

	default:
		return nil, fmt.Errorf("cannot use safe navigation on type %s", objType.String())
	}

	// Safe navigation always returns nullable
	return fieldType.MakeNullable(), nil
}

// inferCall infers the type of a function call
func (tc *TypeChecker) inferCall(call *ast.CallExpr) (Type, error) {
	// Look up stdlib function
	if call.Namespace != "" {
		fn, ok := LookupStdlibFunction(call.Namespace, call.Function)
		if !ok {
			tc.errors = append(tc.errors, NewUndefinedFunction(call.Location(), call.Namespace, call.Function))
			return NewPrimitiveType("unknown", false), nil
		}

		// Type check arguments
		if len(call.Arguments) != len(fn.Parameters) {
			// Check if extra args are optional params
			requiredCount := 0
			for _, p := range fn.Parameters {
				if !p.Optional {
					requiredCount++
				}
			}

			if len(call.Arguments) < requiredCount || len(call.Arguments) > len(fn.Parameters) {
				tc.errors = append(tc.errors, NewInvalidArgumentCount(
					call.Location(),
					fn.FullName(),
					len(fn.Parameters),
					len(call.Arguments),
				))
			}
		}

		// Type check each argument
		for i, arg := range call.Arguments {
			if i >= len(fn.Parameters) {
				break
			}

			argType, err := tc.inferExpr(arg)
			if err != nil {
				return nil, err
			}

			expectedType := fn.Parameters[i].Type
			if !expectedType.IsAssignableFrom(argType) {
				tc.errors = append(tc.errors, NewInvalidArgumentType(
					call.Location(),
					fn.FullName(),
					i,
					expectedType,
					argType,
				))
			}
		}

		return fn.ReturnType, nil
	}

	// Custom function - would need to be looked up in custom function registry
	// For now, return unknown type
	tc.errors = append(tc.errors, NewUndefinedFunction(call.Location(), "", call.Function))
	return NewPrimitiveType("unknown", false), nil
}

// inferBinary infers the type of a binary expression
func (tc *TypeChecker) inferBinary(bin *ast.BinaryExpr) (Type, error) {
	leftType, err := tc.inferExpr(bin.Left)
	if err != nil {
		return nil, err
	}

	rightType, err := tc.inferExpr(bin.Right)
	if err != nil {
		return nil, err
	}

	switch bin.Operator {
	case "+", "-", "*", "/", "%":
		// Arithmetic operators - both sides must be numeric
		if !tc.isNumeric(leftType) || !tc.isNumeric(rightType) {
			tc.errors = append(tc.errors, NewInvalidBinaryOp(bin.Location(), bin.Operator, leftType, rightType))
			return NewPrimitiveType("unknown", false), nil
		}

		// Result is float if either operand is float, otherwise int
		if tc.isFloat(leftType) || tc.isFloat(rightType) {
			return NewPrimitiveType(typeFloat, false), nil
		}
		return NewPrimitiveType(typeInt, false), nil

	case "==", "!=", "<", ">", "<=", ">=":
		// Comparison operators - return bool
		return NewPrimitiveType("bool", false), nil

	case "**":
		// Exponentiation - return float
		return NewPrimitiveType(typeFloat, false), nil

	default:
		tc.errors = append(tc.errors, NewInvalidBinaryOp(bin.Location(), bin.Operator, leftType, rightType))
		return NewPrimitiveType("unknown", false), nil
	}
}

// inferUnary infers the type of a unary expression
func (tc *TypeChecker) inferUnary(un *ast.UnaryExpr) (Type, error) {
	operandType, err := tc.inferExpr(un.Operand)
	if err != nil {
		return nil, err
	}

	switch un.Operator {
	case "!":
		// Unwrap operator - converts nullable to required
		if !operandType.IsNullable() {
			tc.errors = append(tc.errors, NewUnnecessaryUnwrap(un.Location(), operandType))
		}
		return operandType.MakeRequired(), nil

	case "-":
		// Negation - must be numeric
		if !tc.isNumeric(operandType) {
			tc.errors = append(tc.errors, NewInvalidUnaryOp(un.Location(), un.Operator, operandType))
			return NewPrimitiveType("unknown", false), nil
		}
		return operandType, nil

	case "not":
		// Logical not - return bool
		return NewPrimitiveType("bool", false), nil

	default:
		tc.errors = append(tc.errors, NewInvalidUnaryOp(un.Location(), un.Operator, operandType))
		return NewPrimitiveType("unknown", false), nil
	}
}

// inferLogical infers the type of a logical expression
func (tc *TypeChecker) inferLogical(log *ast.LogicalExpr) (Type, error) {
	// Type check both sides
	_, err := tc.inferExpr(log.Left)
	if err != nil {
		return nil, err
	}

	_, err = tc.inferExpr(log.Right)
	if err != nil {
		return nil, err
	}

	// Logical operators always return bool
	return NewPrimitiveType("bool", false), nil
}

// inferNullCoalesce infers the type of a null coalescing expression
func (tc *TypeChecker) inferNullCoalesce(nc *ast.NullCoalesceExpr) (Type, error) {
	leftType, err := tc.inferExpr(nc.Left)
	if err != nil {
		return nil, err
	}

	_, err = tc.inferExpr(nc.Right)
	if err != nil {
		return nil, err
	}

	// Left must be nullable for ?? to make sense
	if !leftType.IsNullable() {
		tc.errors = append(tc.errors, &TypeError{
			Code:       ErrUnnecessaryUnwrap,
			Type:       "unnecessary_coalesce",
			Severity:   SeverityWarning,
			Message:    "Null coalescing operator used on required type",
			Location:   nc.Location(),
			Suggestion: "Remove ?? as left side cannot be nil",
		})
	}

	// Result type is the required version of the left type (or right type if different)
	// This is simplified - in reality we'd want to ensure right type is assignable to left
	return leftType.MakeRequired(), nil
}

// inferArrayLiteral infers the type of an array literal
func (tc *TypeChecker) inferArrayLiteral(arr *ast.ArrayLiteralExpr) (Type, error) {
	if len(arr.Elements) == 0 {
		// Empty array - would need type annotation in real implementation
		// For now, return array<any>!
		return NewArrayType(NewPrimitiveType("any", false), false), nil
	}

	// Infer element type from first element
	elemType, err := tc.inferExpr(arr.Elements[0])
	if err != nil {
		return nil, err
	}

	// Verify all elements have compatible types
	for i, elem := range arr.Elements[1:] {
		eType, err := tc.inferExpr(elem)
		if err != nil {
			return nil, err
		}

		if !elemType.IsAssignableFrom(eType) {
			return nil, fmt.Errorf("array element %d has incompatible type: expected %s, got %s",
				i+1, elemType.String(), eType.String())
		}
	}

	return NewArrayType(elemType, false), nil
}

// inferHashLiteral infers the type of a hash literal
func (tc *TypeChecker) inferHashLiteral(hash *ast.HashLiteralExpr) (Type, error) {
	if len(hash.Pairs) == 0 {
		// Empty hash
		return NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("any", false), false), nil
	}

	// Infer key and value types from first pair
	firstPair := hash.Pairs[0]
	keyType, err := tc.inferExpr(firstPair.Key)
	if err != nil {
		return nil, err
	}

	valueType, err := tc.inferExpr(firstPair.Value)
	if err != nil {
		return nil, err
	}

	// Verify all pairs have compatible types
	for i, pair := range hash.Pairs[1:] {
		kType, err := tc.inferExpr(pair.Key)
		if err != nil {
			return nil, err
		}

		vType, err := tc.inferExpr(pair.Value)
		if err != nil {
			return nil, err
		}

		if !keyType.IsAssignableFrom(kType) {
			return nil, fmt.Errorf("hash key %d has incompatible type", i+1)
		}

		if !valueType.IsAssignableFrom(vType) {
			return nil, fmt.Errorf("hash value %d has incompatible type", i+1)
		}
	}

	return NewHashType(keyType, valueType, false), nil
}

// inferIndex infers the type of an index expression
func (tc *TypeChecker) inferIndex(idx *ast.IndexExpr) (Type, error) {
	objType, err := tc.inferExpr(idx.Object)
	if err != nil {
		return nil, err
	}

	switch t := objType.(type) {
	case *ArrayType:
		// Array indexing returns element type (always nullable - could be out of bounds)
		return t.ElementType.MakeNullable(), nil

	case *HashType:
		// Hash indexing returns value type (always nullable - key might not exist)
		return t.ValueType.MakeNullable(), nil

	default:
		tc.errors = append(tc.errors, NewInvalidIndexOp(idx.Location(), objType))
		return NewPrimitiveType("unknown", false), nil
	}
}

// lookupResourceField looks up a field in a resource by name
func (tc *TypeChecker) lookupResourceField(resourceName, fieldName string) (Type, error) {
	// Find the resource
	resource, ok := tc.resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("undefined resource: %s", resourceName)
	}

	// Find the field
	for _, field := range resource.Fields {
		if field.Name == fieldName {
			return TypeFromASTNode(field.Type, field.Nullable)
		}
	}

	return nil, fmt.Errorf("field %s not found in resource %s", fieldName, resourceName)
}

// isNumeric checks if a type is numeric (int or float)
func (tc *TypeChecker) isNumeric(t Type) bool {
	prim, ok := t.(*PrimitiveType)
	if !ok {
		return false
	}
	return prim.Name == typeInt || prim.Name == typeFloat
}

// isFloat checks if a type is float
func (tc *TypeChecker) isFloat(t Type) bool {
	prim, ok := t.(*PrimitiveType)
	if !ok {
		return false
	}
	return prim.Name == typeFloat
}
