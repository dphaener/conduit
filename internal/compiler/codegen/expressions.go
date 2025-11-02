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
	// ============================================================================
	// String namespace - string manipulation functions
	// ============================================================================
	case "String.slugify":
		// Use runtime stdlib implementation
		g.imports["github.com/conduit-lang/conduit/pkg/runtime"] = true
		return fmt.Sprintf("runtime.StringSlugify(%s)", argsStr)
	case "String.capitalize":
		// Custom implementation - capitalize first letter
		return fmt.Sprintf("stdlib.StringCapitalize(%s)", argsStr)
	case "String.upcase":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ToUpper(%s)", argsStr)
	case "String.downcase":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ToLower(%s)", argsStr)
	case "String.trim":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.TrimSpace(%s)", argsStr)
	case "String.truncate":
		// Custom implementation - truncate with ellipsis handling
		return fmt.Sprintf("stdlib.StringTruncate(%s)", argsStr)
	case "String.split":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.Split(%s)", argsStr)
	case "String.join":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.Join(%s)", argsStr)
	case "String.replace":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.ReplaceAll(%s)", argsStr)
	case "String.starts_with?":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.HasPrefix(%s)", argsStr)
	case "String.ends_with?":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.HasSuffix(%s)", argsStr)
	case "String.includes?":
		g.imports["strings"] = true
		return fmt.Sprintf("strings.Contains(%s)", argsStr)
	case "String.length":
		// Use len() for string length
		return fmt.Sprintf("len(%s)", argsStr)

	// ============================================================================
	// Text namespace - text processing functions
	// ============================================================================
	case "Text.calculate_reading_time":
		// Custom implementation - words per minute calculation
		return fmt.Sprintf("stdlib.TextCalculateReadingTime(%s)", argsStr)
	case "Text.word_count":
		// Custom implementation - count words in text
		return fmt.Sprintf("stdlib.TextWordCount(%s)", argsStr)
	case "Text.character_count":
		// Use len() for character count
		return fmt.Sprintf("len(%s)", argsStr)
	case "Text.excerpt":
		// Custom implementation - extract excerpt with word boundaries
		return fmt.Sprintf("stdlib.TextExcerpt(%s)", argsStr)

	// ============================================================================
	// Number namespace - numeric operations
	// ============================================================================
	case "Number.format":
		// Custom implementation - format float with decimals
		return fmt.Sprintf("stdlib.NumberFormat(%s)", argsStr)
	case "Number.round":
		// Custom implementation - round to precision
		return fmt.Sprintf("stdlib.NumberRound(%s)", argsStr)
	case "Number.abs":
		g.imports["math"] = true
		return fmt.Sprintf("math.Abs(%s)", argsStr)
	case "Number.ceil":
		g.imports["math"] = true
		return fmt.Sprintf("int(math.Ceil(%s))", argsStr)
	case "Number.floor":
		g.imports["math"] = true
		return fmt.Sprintf("int(math.Floor(%s))", argsStr)
	case "Number.min":
		g.imports["math"] = true
		return fmt.Sprintf("math.Min(%s)", argsStr)
	case "Number.max":
		g.imports["math"] = true
		return fmt.Sprintf("math.Max(%s)", argsStr)

	// ============================================================================
	// Array namespace - array/slice operations
	// ============================================================================
	case "Array.first":
		// Custom implementation - return first element or nil
		return fmt.Sprintf("stdlib.ArrayFirst(%s)", argsStr)
	case "Array.last":
		// Custom implementation - return last element or nil
		return fmt.Sprintf("stdlib.ArrayLast(%s)", argsStr)
	case "Array.length":
		// Use len() for array length
		return fmt.Sprintf("len(%s)", argsStr)
	case "Array.empty?":
		// Check if length is zero
		return fmt.Sprintf("len(%s) == 0", argsStr)
	case "Array.includes?":
		// Custom implementation - check if array contains element
		return fmt.Sprintf("stdlib.ArrayIncludes(%s)", argsStr)
	case "Array.unique":
		// Custom implementation - remove duplicates
		return fmt.Sprintf("stdlib.ArrayUnique(%s)", argsStr)
	case "Array.sort":
		// Custom implementation - sort array
		return fmt.Sprintf("stdlib.ArraySort(%s)", argsStr)
	case "Array.reverse":
		// Custom implementation - reverse array
		return fmt.Sprintf("stdlib.ArrayReverse(%s)", argsStr)
	case "Array.push":
		// Use append for push
		return fmt.Sprintf("append(%s)", argsStr)
	case "Array.concat":
		// Use append for concatenation
		return fmt.Sprintf("append(%s)", argsStr)
	case "Array.map":
		// Custom implementation - map over array with function
		return fmt.Sprintf("stdlib.ArrayMap(%s)", argsStr)
	case "Array.filter":
		// Custom implementation - filter array with predicate
		return fmt.Sprintf("stdlib.ArrayFilter(%s)", argsStr)
	case "Array.reduce":
		// Custom implementation - reduce array with accumulator
		return fmt.Sprintf("stdlib.ArrayReduce(%s)", argsStr)
	case "Array.count":
		// Use len() for count (alias of length)
		return fmt.Sprintf("len(%s)", argsStr)
	case "Array.contains":
		// Custom implementation - check if array contains element (alias of includes?)
		return fmt.Sprintf("stdlib.ArrayIncludes(%s)", argsStr)

	// ============================================================================
	// Hash namespace - map operations
	// ============================================================================
	case "Hash.keys":
		// Custom implementation - extract keys from map
		return fmt.Sprintf("stdlib.HashKeys(%s)", argsStr)
	case "Hash.values":
		// Custom implementation - extract values from map
		return fmt.Sprintf("stdlib.HashValues(%s)", argsStr)
	case "Hash.merge":
		// Custom implementation - merge two maps
		return fmt.Sprintf("stdlib.HashMerge(%s)", argsStr)
	case "Hash.has_key?":
		// Custom implementation - check if key exists
		return fmt.Sprintf("stdlib.HashHasKey(%s)", argsStr)
	case "Hash.get":
		// Custom implementation - get value with default
		return fmt.Sprintf("stdlib.HashGet(%s)", argsStr)

	// ============================================================================
	// Time namespace - date/time operations
	// ============================================================================
	case "Time.now":
		g.imports["time"] = true
		return "time.Now()"
	case "Time.today":
		// Custom implementation - return today's date (truncated to day)
		return "stdlib.TimeToday()"
	case "Time.parse":
		// Custom implementation - parse time with optional format
		return fmt.Sprintf("stdlib.TimeParse(%s)", argsStr)
	case "Time.format":
		// Custom implementation - format time to string
		return fmt.Sprintf("stdlib.TimeFormat(%s)", argsStr)
	case "Time.year":
		// Extract year from time
		return fmt.Sprintf("(%s).Year()", args[0])
	case "Time.month":
		// Extract month from time (as int)
		return fmt.Sprintf("int((%s).Month())", args[0])
	case "Time.day":
		// Extract day from time
		return fmt.Sprintf("(%s).Day()", args[0])

	// ============================================================================
	// UUID namespace - UUID operations
	// ============================================================================
	case "UUID.generate":
		// Custom implementation or use google/uuid
		return "stdlib.UUIDGenerate()"
	case "UUID.validate":
		// Custom implementation - validate UUID string
		return fmt.Sprintf("stdlib.UUIDValidate(%s)", argsStr)
	case "UUID.parse":
		// Custom implementation - parse UUID string (returns nil on error)
		return fmt.Sprintf("stdlib.UUIDParse(%s)", argsStr)

	// ============================================================================
	// Random namespace - random value generation
	// ============================================================================
	case "Random.int":
		// Custom implementation - random int in range
		return fmt.Sprintf("stdlib.RandomInt(%s)", argsStr)
	case "Random.float":
		// Custom implementation - random float in range
		return fmt.Sprintf("stdlib.RandomFloat(%s)", argsStr)
	case "Random.uuid":
		// Same as UUID.generate
		return "stdlib.UUIDGenerate()"
	case "Random.hex":
		// Custom implementation - random hex string
		return fmt.Sprintf("stdlib.RandomHex(%s)", argsStr)
	case "Random.alphanumeric":
		// Custom implementation - random alphanumeric string
		return fmt.Sprintf("stdlib.RandomAlphanumeric(%s)", argsStr)

	// ============================================================================
	// Crypto namespace - cryptographic operations
	// ============================================================================
	case "Crypto.hash":
		// Custom implementation - hash data with algorithm (sha256, bcrypt, etc.)
		return fmt.Sprintf("stdlib.CryptoHash(%s)", argsStr)
	case "Crypto.compare":
		// Custom implementation - constant-time comparison for hashes
		return fmt.Sprintf("stdlib.CryptoCompare(%s)", argsStr)

	// ============================================================================
	// HTML namespace - HTML processing
	// ============================================================================
	case "HTML.strip_tags":
		// Custom implementation - remove HTML tags
		return fmt.Sprintf("stdlib.HTMLStripTags(%s)", argsStr)
	case "HTML.escape":
		g.imports["html"] = true
		return fmt.Sprintf("html.EscapeString(%s)", argsStr)
	case "HTML.unescape":
		g.imports["html"] = true
		return fmt.Sprintf("html.UnescapeString(%s)", argsStr)

	// ============================================================================
	// JSON namespace - JSON operations
	// ============================================================================
	case "JSON.parse":
		// Custom implementation - parse JSON string
		return fmt.Sprintf("stdlib.JSONParse(%s)", argsStr)
	case "JSON.stringify":
		// Custom implementation - stringify to JSON (with optional pretty)
		return fmt.Sprintf("stdlib.JSONStringify(%s)", argsStr)
	case "JSON.validate":
		// Custom implementation - validate JSON string
		return fmt.Sprintf("stdlib.JSONValidate(%s)", argsStr)

	// ============================================================================
	// Regex namespace - regular expression operations
	// ============================================================================
	case "Regex.match":
		// Custom implementation - find regex matches
		return fmt.Sprintf("stdlib.RegexMatch(%s)", argsStr)
	case "Regex.replace":
		// Custom implementation - regex replace
		return fmt.Sprintf("stdlib.RegexReplace(%s)", argsStr)
	case "Regex.test":
		// Custom implementation - test if regex matches
		return fmt.Sprintf("stdlib.RegexTest(%s)", argsStr)
	case "Regex.split":
		// Custom implementation - split by regex pattern
		return fmt.Sprintf("stdlib.RegexSplit(%s)", argsStr)

	// ============================================================================
	// Logger namespace - logging operations
	// ============================================================================
	case "Logger.warn":
		g.imports["log"] = true
		return fmt.Sprintf("log.Printf(\"WARN: %s\", %s)", argsStr)
	case "Logger.debug":
		g.imports["log"] = true
		return fmt.Sprintf("log.Printf(\"DEBUG: %s\", %s)", argsStr)

	// ============================================================================
	// Context namespace - request context operations
	// ============================================================================
	case "Context.current_user":
		// Custom implementation - extract current user from context
		return "stdlib.ContextCurrentUser(ctx)"
	case "Context.authenticated?":
		// Custom implementation - check if user is authenticated
		return "stdlib.ContextAuthenticated(ctx)"
	case "Context.headers":
		// Custom implementation - extract request headers from context
		return "stdlib.ContextHeaders(ctx)"
	case "Context.request_id":
		// Custom implementation - extract request ID from context
		return "stdlib.ContextRequestID(ctx)"

	// ============================================================================
	// Env namespace - environment variable operations
	// ============================================================================
	case "Env.get":
		// Custom implementation - get env var with optional default
		return fmt.Sprintf("stdlib.EnvGet(%s)", argsStr)
	case "Env.has?":
		// Custom implementation - check if env var exists
		return fmt.Sprintf("stdlib.EnvHas(%s)", argsStr)

	default:
		// Unknown stdlib function - generate fallback
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
