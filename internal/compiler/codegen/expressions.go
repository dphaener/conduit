package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// generateExpr generates Go code for an expression
// Returns the generated code as a string
func (g *Generator) generateExpr(expr ast.ExprNode) string {
	if expr == nil {
		return "nil"
	}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return g.generateLiteral(e)

	case *ast.IdentifierExpr:
		return g.generateIdentifier(e)

	case *ast.SelfExpr:
		return g.generateSelf(e)

	case *ast.FieldAccessExpr:
		return g.generateFieldAccess(e)

	case *ast.SafeNavigationExpr:
		return g.generateSafeNavigation(e)

	case *ast.CallExpr:
		return g.generateCall(e)

	case *ast.BinaryExpr:
		return g.generateBinary(e)

	case *ast.UnaryExpr:
		return g.generateUnary(e)

	case *ast.LogicalExpr:
		return g.generateLogical(e)

	case *ast.NullCoalesceExpr:
		return g.generateNullCoalesce(e)

	case *ast.ArrayLiteralExpr:
		return g.generateArrayLiteral(e)

	case *ast.HashLiteralExpr:
		return g.generateHashLiteral(e)

	case *ast.IndexExpr:
		return g.generateIndex(e)

	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", g.generateExpr(e.Expr))

	case *ast.InterpolatedStringExpr:
		return g.generateInterpolatedString(e)

	default:
		// Unknown expression type - return placeholder
		return fmt.Sprintf("/* TODO: unsupported expression type %T */", expr)
	}
}

// generateLiteral generates Go code for a literal expression
func (g *Generator) generateLiteral(lit *ast.LiteralExpr) string {
	if lit.Value == nil {
		return "nil"
	}

	switch v := lit.Value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// Use %g to avoid trailing zeros
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// generateIdentifier generates Go code for an identifier
func (g *Generator) generateIdentifier(ident *ast.IdentifierExpr) string {
	// Convert to Go naming conventions if needed
	// For now, keep as-is (might need to handle snake_case -> camelCase)
	return ident.Name
}

// generateSelf generates Go code for the 'self' keyword
func (g *Generator) generateSelf(self *ast.SelfExpr) string {
	// In Go methods, 'self' is typically the receiver (e.g., 'p' for *Post)
	// We'll use a conventional lowercase first letter of the type
	// The actual receiver name should be determined by context
	// For now, return a placeholder that should be set by the caller
	return "self"
}

// generateFieldAccess generates Go code for field access
func (g *Generator) generateFieldAccess(fa *ast.FieldAccessExpr) string {
	obj := g.generateExpr(fa.Object)
	fieldName := g.toGoFieldName(fa.Field)
	return fmt.Sprintf("%s.%s", obj, fieldName)
}

// generateSafeNavigation generates Go code for safe navigation (?.)
func (g *Generator) generateSafeNavigation(sn *ast.SafeNavigationExpr) string {
	obj := g.generateExpr(sn.Object)
	fieldName := g.toGoFieldName(sn.Field)

	// Generate safe navigation pattern in Go
	// if obj != nil { return obj.Field } else { return nil }
	// For inline expression, we can use a helper function or generate inline check
	// For simplicity, generate an inline ternary-style expression using Go idioms

	// For simple cases, just check nil
	return fmt.Sprintf("(func() interface{} { if %s != nil { return %s.%s }; return nil })()",
		obj, obj, fieldName)
}

// generateCall generates Go code for function calls
func (g *Generator) generateCall(call *ast.CallExpr) string {
	if call.Namespace != "" {
		// Namespaced function call (e.g., String.slugify())
		// Map Conduit stdlib to Go stdlib implementations
		return g.generateStdlibCall(call)
	}

	// Regular function call
	args := make([]string, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = g.generateExpr(arg)
	}

	return fmt.Sprintf("%s(%s)", call.Function, strings.Join(args, ", "))
}

// generateStdlibCall generates Go code for stdlib function calls
func (g *Generator) generateStdlibCall(call *ast.CallExpr) string {
	fullName := call.Namespace + "." + call.Function

	// Generate arguments
	args := make([]string, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = g.generateExpr(arg)
	}
	argsStr := strings.Join(args, ", ")

	// Map Conduit stdlib functions to Go implementations
	switch fullName {
	// String namespace
	case "String.slugify":
		g.imports["strings"] = true
		g.imports["regexp"] = true
		return fmt.Sprintf("stdlib.StringSlugify(%s)", argsStr)
	case "String.upcase":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ToUpper(%s)", argsStr)
	case "String.downcase":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ToLower(%s)", argsStr)
	case "String.replace":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ReplaceAll(%s)", argsStr)
	case "String.from_int":
		g.imports["strconv"] = true
		return fmt.Sprintf("strconv.FormatInt(%s, 10)", argsStr)
	case "String.length":
		return fmt.Sprintf("len(%s)", argsStr)

	// Time namespace
	case "Time.now":
		g.imports["time"] = true
		return "time.Now()"
	case "Time.format":
		g.imports["time"] = true
		return fmt.Sprintf("stdlib.TimeFormat(%s)", argsStr)
	case "Time.parse":
		g.imports["time"] = true
		return fmt.Sprintf("stdlib.TimeParse(%s)", argsStr)

	// Array namespace
	case "Array.first":
		return fmt.Sprintf("stdlib.ArrayFirst(%s)", argsStr)
	case "Array.last":
		return fmt.Sprintf("stdlib.ArrayLast(%s)", argsStr)
	case "Array.count":
		return fmt.Sprintf("len(%s)", argsStr)

	// Context namespace
	case "Context.current_user":
		return "stdlib.ContextCurrentUser(ctx)"
	case "Context.request_id":
		return "stdlib.ContextRequestID(ctx)"

	// Logger namespace
	case "Logger.info":
		g.imports["log"] = true
		return fmt.Sprintf("log.Printf(\"INFO: \" + %s)", argsStr)
	case "Logger.error":
		g.imports["log"] = true
		return fmt.Sprintf("log.Printf(\"ERROR: \" + %s)", argsStr)

	default:
		// Unknown stdlib function
		return fmt.Sprintf("stdlib.%s_%s(%s)", call.Namespace, call.Function, argsStr)
	}
}

// generateBinary generates Go code for binary expressions
func (g *Generator) generateBinary(bin *ast.BinaryExpr) string {
	left := g.generateExpr(bin.Left)
	right := g.generateExpr(bin.Right)

	// Map Conduit operators to Go operators
	op := bin.Operator
	switch op {
	case "**":
		// Exponentiation - use math.Pow
		g.imports["math"] = true
		return fmt.Sprintf("math.Pow(%s, %s)", left, right)
	default:
		// Most operators are the same in Go
		return fmt.Sprintf("%s %s %s", left, op, right)
	}
}

// generateUnary generates Go code for unary expressions
func (g *Generator) generateUnary(un *ast.UnaryExpr) string {
	operand := g.generateExpr(un.Operand)

	switch un.Operator {
	case "!":
		// Unwrap operator in Conduit - in Go this is dereferencing for pointers
		// or checking .Valid for sql.Null* types
		// For simplicity, assume it's a force unwrap and generate a dereference
		return fmt.Sprintf("*(%s)", operand)
	case "-":
		return fmt.Sprintf("-%s", operand)
	case "not":
		return fmt.Sprintf("!(%s)", operand)
	default:
		return fmt.Sprintf("%s%s", un.Operator, operand)
	}
}

// generateLogical generates Go code for logical expressions
func (g *Generator) generateLogical(log *ast.LogicalExpr) string {
	left := g.generateExpr(log.Left)
	right := g.generateExpr(log.Right)

	// Map Conduit logical operators to Go
	op := log.Operator
	switch op {
	case "and":
		op = "&&"
	case "or":
		op = "||"
	}

	return fmt.Sprintf("%s %s %s", left, op, right)
}

// generateNullCoalesce generates Go code for null coalescing (??)
func (g *Generator) generateNullCoalesce(nc *ast.NullCoalesceExpr) string {
	left := g.generateExpr(nc.Left)
	right := g.generateExpr(nc.Right)

	// Generate null coalescing pattern in Go
	// For sql.Null* types: if left.Valid { left.Type } else { right }
	// For pointers: if left != nil { *left } else { right }
	// For simplicity, generate a function-style expression

	return fmt.Sprintf("(func() interface{} { if %s != nil { return %s }; return %s })()",
		left, left, right)
}

// generateArrayLiteral generates Go code for array literals
func (g *Generator) generateArrayLiteral(arr *ast.ArrayLiteralExpr) string {
	if len(arr.Elements) == 0 {
		return "[]interface{}{}"
	}

	elements := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		elements[i] = g.generateExpr(elem)
	}

	// For now, use []interface{} - a real implementation would infer the element type
	return fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", "))
}

// generateHashLiteral generates Go code for hash literals
func (g *Generator) generateHashLiteral(hash *ast.HashLiteralExpr) string {
	if len(hash.Pairs) == 0 {
		return "map[string]interface{}{}"
	}

	pairs := make([]string, len(hash.Pairs))
	for i, pair := range hash.Pairs {
		key := g.generateExpr(pair.Key)
		value := g.generateExpr(pair.Value)
		pairs[i] = fmt.Sprintf("%s: %s", key, value)
	}

	// For now, use map[string]interface{} - a real implementation would infer types
	return fmt.Sprintf("map[string]interface{}{%s}", strings.Join(pairs, ", "))
}

// generateIndex generates Go code for indexing operations
func (g *Generator) generateIndex(idx *ast.IndexExpr) string {
	obj := g.generateExpr(idx.Object)
	index := g.generateExpr(idx.Index)

	return fmt.Sprintf("%s[%s]", obj, index)
}

// generateInterpolatedString generates Go code for interpolated strings
func (g *Generator) generateInterpolatedString(is *ast.InterpolatedStringExpr) string {
	if len(is.Parts) == 0 {
		return `""`
	}

	// Use fmt.Sprintf for string interpolation
	g.imports["fmt"] = true

	var formatParts []string
	var args []string

	for _, part := range is.Parts {
		if lit, ok := part.(*ast.LiteralExpr); ok {
			if str, ok := lit.Value.(string); ok {
				// String literal part
				formatParts = append(formatParts, str)
				continue
			}
		}

		// Expression part - add %v placeholder
		formatParts = append(formatParts, "%v")
		args = append(args, g.generateExpr(part))
	}

	if len(args) == 0 {
		// No interpolation needed
		return fmt.Sprintf("%q", strings.Join(formatParts, ""))
	}

	formatStr := strings.Join(formatParts, "")
	argsStr := strings.Join(args, ", ")
	return fmt.Sprintf("fmt.Sprintf(%q, %s)", formatStr, argsStr)
}
