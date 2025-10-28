package codegen

import (
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// generateHooks generates lifecycle hook methods for a resource
func (g *Generator) generateHooks(resource *ast.ResourceNode) string {
	if len(resource.Hooks) == 0 {
		return ""
	}

	var hookCode strings.Builder

	for _, hook := range resource.Hooks {
		code := g.generateHook(resource, hook)
		if code != "" {
			hookCode.WriteString(code)
			hookCode.WriteString("\n\n")
		}
	}

	return hookCode.String()
}

// generateHook generates a single lifecycle hook method
func (g *Generator) generateHook(resource *ast.ResourceNode, hook *ast.HookNode) string {
	receiverName := strings.ToLower(resource.Name[0:1])

	// Generate method name: BeforeCreate, AfterUpdate, etc.
	methodName := strings.Title(hook.Timing) + strings.Title(hook.Event)

	// Build method signature
	g.reset()
	g.writeLine("// %s is called %s %s", methodName, hook.Timing, hook.Event)
	g.writeLine("func (%s *%s) %s(ctx context.Context, db *sql.DB) error {",
		receiverName, resource.Name, methodName)
	g.indent++

	// Generate transaction wrapper if needed
	if hook.IsTransaction {
		g.writeLine("tx, err := db.Begin()")
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to begin transaction: %w\", err)")
		g.indent--
		g.writeLine("}")
		g.writeLine("defer tx.Rollback()")
		g.writeLine("")
	}

	// Generate hook body statements
	asyncFound := false
	for _, stmt := range hook.Body {
		// Check if this is an async block
		if blockStmt, ok := stmt.(*ast.BlockStmt); ok && blockStmt.IsAsync {
			asyncFound = true
			g.generateAsyncBlock(resource, blockStmt)
		} else {
			g.generateStatement(resource, stmt)
		}
	}

	// If we have a transaction, commit it before async blocks
	if hook.IsTransaction && !asyncFound {
		g.writeLine("")
		g.writeLine("if err := tx.Commit(); err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to commit transaction: %w\", err)")
		g.indent--
		g.writeLine("}")
	} else if hook.IsTransaction && asyncFound {
		// Commit before async block was generated inline
	}

	g.writeLine("")
	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")

	return g.buf.String()
}

// generateAsyncBlock generates code for an @async block
func (g *Generator) generateAsyncBlock(resource *ast.ResourceNode, block *ast.BlockStmt) {
	receiverName := strings.ToLower(resource.Name[0:1])

	// If we're in a transaction context, commit before spawning goroutine
	// This is a simplification - in production, we'd check parent context
	g.writeLine("// Async block - runs in background after response")
	g.writeLine("go func() {")
	g.indent++

	// Copy receiver for use in goroutine
	g.writeLine("// Copy resource for async access")
	g.writeLine("asyncResource := *%s", receiverName)
	g.writeLine("")

	// Generate async block statements
	for _, stmt := range block.Statements {
		g.generateStatement(resource, stmt)
	}

	g.indent--
	g.writeLine("}()")
}

// generateStatement generates Go code for a statement
func (g *Generator) generateStatement(resource *ast.ResourceNode, stmt ast.StmtNode) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		exprCode := g.generateExpr(s.Expr)
		// Replace "self" with actual receiver name
		receiverName := strings.ToLower(resource.Name[0:1])
		exprCode = strings.ReplaceAll(exprCode, "self", receiverName)
		g.writeLine("%s", exprCode)

	case *ast.AssignmentStmt:
		g.generateAssignment(resource, s)

	case *ast.LetStmt:
		g.generateLet(resource, s)

	case *ast.IfStmt:
		g.generateIf(resource, s)

	case *ast.ReturnStmt:
		g.generateReturn(resource, s)

	case *ast.BlockStmt:
		// Nested block (non-async)
		if s.IsAsync {
			g.generateAsyncBlock(resource, s)
		} else {
			for _, innerStmt := range s.Statements {
				g.generateStatement(resource, innerStmt)
			}
		}

	case *ast.RescueStmt:
		g.generateRescue(resource, s)

	default:
		g.writeLine("// TODO: Unsupported statement type: %T", stmt)
	}
}

// generateAssignment generates an assignment statement
func (g *Generator) generateAssignment(resource *ast.ResourceNode, stmt *ast.AssignmentStmt) {
	receiverName := strings.ToLower(resource.Name[0:1])

	target := g.generateExpr(stmt.Target)
	target = strings.ReplaceAll(target, "self", receiverName)

	value := g.generateExpr(stmt.Value)
	value = strings.ReplaceAll(value, "self", receiverName)

	g.writeLine("%s = %s", target, value)
}

// generateLet generates a let statement (local variable)
func (g *Generator) generateLet(resource *ast.ResourceNode, stmt *ast.LetStmt) {
	receiverName := strings.ToLower(resource.Name[0:1])

	value := g.generateExpr(stmt.Value)
	value = strings.ReplaceAll(value, "self", receiverName)

	// Determine type annotation if provided
	if stmt.Type != nil {
		// Generate type annotation
		// For now, use type inference with :=
		g.writeLine("%s := %s", stmt.Name, value)
	} else {
		g.writeLine("%s := %s", stmt.Name, value)
	}
}

// generateIf generates an if statement
func (g *Generator) generateIf(resource *ast.ResourceNode, stmt *ast.IfStmt) {
	receiverName := strings.ToLower(resource.Name[0:1])

	condition := g.generateExpr(stmt.Condition)
	condition = strings.ReplaceAll(condition, "self", receiverName)

	g.writeLine("if %s {", condition)
	g.indent++
	for _, thenStmt := range stmt.ThenBranch {
		g.generateStatement(resource, thenStmt)
	}
	g.indent--

	// Generate elsif branches
	for _, elsifBranch := range stmt.ElsIfBranches {
		elsifCond := g.generateExpr(elsifBranch.Condition)
		elsifCond = strings.ReplaceAll(elsifCond, "self", receiverName)

		g.writeLine("} else if %s {", elsifCond)
		g.indent++
		for _, elsifStmt := range elsifBranch.Body {
			g.generateStatement(resource, elsifStmt)
		}
		g.indent--
	}

	// Generate else branch
	if len(stmt.ElseBranch) > 0 {
		g.writeLine("} else {")
		g.indent++
		for _, elseStmt := range stmt.ElseBranch {
			g.generateStatement(resource, elseStmt)
		}
		g.indent--
	}

	g.writeLine("}")
}

// generateReturn generates a return statement
func (g *Generator) generateReturn(resource *ast.ResourceNode, stmt *ast.ReturnStmt) {
	receiverName := strings.ToLower(resource.Name[0:1])

	if stmt.Value != nil {
		value := g.generateExpr(stmt.Value)
		value = strings.ReplaceAll(value, "self", receiverName)
		g.writeLine("return %s", value)
	} else {
		g.writeLine("return nil")
	}
}

// generateRescue generates error handling with rescue block
func (g *Generator) generateRescue(resource *ast.ResourceNode, stmt *ast.RescueStmt) {
	// Generate try block as a function call that returns error
	g.writeLine("{")
	g.indent++

	// Generate try statements
	for _, tryStmt := range stmt.Try {
		g.generateStatement(resource, tryStmt)
	}

	g.indent--
	g.writeLine("}")

	// Generate rescue block
	g.writeLine("if err := recover(); err != nil {")
	g.indent++

	// Assign error to variable if specified
	if stmt.ErrorVar != "" {
		g.imports["fmt"] = true
		g.writeLine("%s := fmt.Errorf(\"%v\", err)", stmt.ErrorVar)
	}

	// Generate rescue body
	for _, rescueStmt := range stmt.RescueBody {
		g.generateStatement(resource, rescueStmt)
	}

	g.indent--
	g.writeLine("}")
}
