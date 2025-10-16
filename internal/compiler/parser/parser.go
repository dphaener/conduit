package parser

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
)

const (
	hookTimingBefore = "before"
	hookTimingAfter  = "after"
)

// Parser transforms a stream of tokens into an Abstract Syntax Tree (AST)
type Parser struct {
	tokens  []lexer.Token
	current int
	errors  []ParseError
}

// New creates a new parser for the given token stream
func New(tokens []lexer.Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
		errors:  make([]ParseError, 0),
	}
}

// Parse parses the token stream and returns the AST and any errors
func (p *Parser) Parse() (*ast.Program, []ParseError) {
	program := &ast.Program{
		Resources: make([]*ast.ResourceNode, 0),
	}

	for !p.isAtEnd() {
		if resource := p.parseResource(); resource != nil {
			program.Resources = append(program.Resources, resource)
		}
	}

	return program, p.errors
}

// parseResource parses a resource definition
func (p *Parser) parseResource() *ast.ResourceNode {
	// Expect 'resource' keyword
	resourceToken := p.consume(lexer.TOKEN_RESOURCE, "Expected 'resource' keyword")
	if resourceToken.Type == lexer.TOKEN_ERROR {
		p.synchronize()
		return nil
	}

	// Expect resource name
	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected resource name")
	if nameToken.Type == lexer.TOKEN_ERROR {
		p.synchronize()
		return nil
	}

	// Expect opening brace
	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' after resource name")
		p.synchronize()
		return nil
	}

	resource := &ast.ResourceNode{
		Name:          nameToken.Lexeme,
		Fields:        make([]*ast.FieldNode, 0),
		Hooks:         make([]*ast.HookNode, 0),
		Validations:   make([]*ast.ValidationNode, 0),
		Constraints:   make([]*ast.ConstraintNode, 0),
		Relationships: make([]*ast.RelationshipNode, 0),
		Scopes:        make([]*ast.ScopeNode, 0),
		Computed:      make([]*ast.ComputedNode, 0),
		Operations:    make([]string, 0),
		Middleware:    make([]string, 0),
		Loc:           ast.TokenLocation(resourceToken),
	}

	// Parse resource body
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		// Check for annotations
		if p.isResourceAnnotationToken() {
			p.parseResourceAnnotation(resource)
		} else if p.isFieldNameToken() {
			// Field or relationship
			if field := p.parseField(); field != nil {
				// Check if it's a relationship based on type
				if p.isRelationshipField(field) {
					resource.Relationships = append(resource.Relationships, p.fieldToRelationship(field))
				} else {
					resource.Fields = append(resource.Fields, field)
				}
			}
		} else {
			// Unexpected token
			p.error(p.peek(), fmt.Sprintf("Unexpected token in resource body: %s", p.peek().Lexeme))
			p.advance()
		}
	}

	// Expect closing brace
	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after resource body")
	}

	return resource
}

// parseResourceAnnotation parses annotations at the resource level
func (p *Parser) parseResourceAnnotation(resource *ast.ResourceNode) {
	annotationToken := p.advance()
	annotationName := annotationToken.Lexeme

	// Map token types to annotation names for keywords
	if annotationToken.Type != lexer.TOKEN_IDENTIFIER {
		annotationName = p.getAnnotationName(annotationToken.Type)
	}

	switch annotationName {
	case hookTimingBefore, hookTimingAfter:
		if hook := p.parseHook(annotationToken); hook != nil {
			resource.Hooks = append(resource.Hooks, hook)
		}
	case "validate":
		if validation := p.parseValidation(); validation != nil {
			resource.Validations = append(resource.Validations, validation)
		}
	case "constraint":
		if constraint := p.parseConstraintBlock(); constraint != nil {
			resource.Constraints = append(resource.Constraints, constraint)
		}
	case "scope":
		if scope := p.parseScope(); scope != nil {
			resource.Scopes = append(resource.Scopes, scope)
		}
	case "computed":
		if computed := p.parseComputed(); computed != nil {
			resource.Computed = append(resource.Computed, computed)
		}
	case "operations":
		resource.Operations = p.parseOperations()
	case "middleware":
		resource.Middleware = p.parseMiddleware()
	default:
		p.error(annotationToken, fmt.Sprintf("Unknown resource annotation: @%s", annotationName))
	}
}

// parseField parses a field declaration
func (p *Parser) parseField() *ast.FieldNode {
	// Accept any token that can be a field name (identifiers or type keywords)
	nameToken := p.consumeFieldName()
	if nameToken.Type == lexer.TOKEN_ERROR {
		p.synchronizeToNextField()
		return nil
	}

	// Expect colon
	if !p.match(lexer.TOKEN_COLON) {
		p.error(p.peek(), "Expected ':' after field name")
		p.synchronizeToNextField()
		return nil
	}

	// Parse type
	fieldType := p.parseType()
	if fieldType == nil {
		p.synchronizeToNextField()
		return nil
	}

	nullable := false
	if p.previous().Type == lexer.TOKEN_QUESTION {
		nullable = true
	}

	field := &ast.FieldNode{
		Name:        nameToken.Lexeme,
		Type:        fieldType,
		Nullable:    nullable,
		Constraints: make([]*ast.ConstraintNode, 0),
		Loc:         ast.TokenLocation(nameToken),
	}

	// Parse field constraints
	for p.isFieldConstraintToken() {
		if constraint := p.parseFieldConstraint(); constraint != nil {
			field.Constraints = append(field.Constraints, constraint)
		}
	}

	// Check for relationship body
	if p.check(lexer.TOKEN_LBRACE) {
		// This is a relationship, not a simple field
		// The relationship body will be parsed by fieldToRelationship
		return field
	}

	return field
}

// parseType parses a type specification
func (p *Parser) parseType() *ast.TypeNode {
	loc := ast.TokenLocation(p.peek())

	// Check for primitive types
	if p.isPrimitiveType() {
		return p.parsePrimitiveType(loc)
	}

	// Check for array type
	if p.match(lexer.TOKEN_ARRAY) {
		return p.parseArrayType(loc)
	}

	// Check for hash type
	if p.match(lexer.TOKEN_HASH) {
		return p.parseHashType(loc)
	}

	// Check for enum type
	if p.match(lexer.TOKEN_ENUM) {
		return p.parseEnumType(loc)
	}

	// Check for resource type (identifier)
	if p.check(lexer.TOKEN_IDENTIFIER) {
		return p.parseResourceType(loc)
	}

	p.error(p.peek(), "Expected type name")
	return nil
}

// parsePrimitiveType parses a primitive type with nullability marker
func (p *Parser) parsePrimitiveType(loc ast.SourceLocation) *ast.TypeNode {
	typeToken := p.advance()
	typeNode := &ast.TypeNode{
		Kind: ast.TypePrimitive,
		Name: p.getTypeName(typeToken.Type),
		Loc:  loc,
	}

	p.parseNullabilityMarker(typeNode)
	return typeNode
}

// parseArrayType parses an array type (array<T>)
func (p *Parser) parseArrayType(loc ast.SourceLocation) *ast.TypeNode {
	if !p.match(lexer.TOKEN_LT) {
		p.error(p.peek(), "Expected '<' after 'array'")
		return nil
	}

	elementType := p.parseType()
	if elementType == nil {
		return nil
	}

	if !p.match(lexer.TOKEN_GT) {
		p.error(p.peek(), "Expected '>' after array element type")
		return nil
	}

	typeNode := &ast.TypeNode{
		Kind:        ast.TypeArray,
		Name:        "array",
		ElementType: elementType,
		Loc:         loc,
	}

	p.parseNullabilityMarker(typeNode)
	return typeNode
}

// parseHashType parses a hash type (hash<K, V>)
func (p *Parser) parseHashType(loc ast.SourceLocation) *ast.TypeNode {
	if !p.match(lexer.TOKEN_LT) {
		p.error(p.peek(), "Expected '<' after 'hash'")
		return nil
	}

	keyType := p.parseType()
	if keyType == nil {
		return nil
	}

	if !p.match(lexer.TOKEN_COMMA) {
		p.error(p.peek(), "Expected ',' after hash key type")
		return nil
	}

	valueType := p.parseType()
	if valueType == nil {
		return nil
	}

	if !p.match(lexer.TOKEN_GT) {
		p.error(p.peek(), "Expected '>' after hash value type")
		return nil
	}

	typeNode := &ast.TypeNode{
		Kind:      ast.TypeHash,
		Name:      "hash",
		KeyType:   keyType,
		ValueType: valueType,
		Loc:       loc,
	}

	p.parseNullabilityMarker(typeNode)
	return typeNode
}

// parseEnumType parses an enum type (enum["value1", "value2"])
func (p *Parser) parseEnumType(loc ast.SourceLocation) *ast.TypeNode {
	if !p.match(lexer.TOKEN_LBRACKET) {
		p.error(p.peek(), "Expected '[' after 'enum'")
		return nil
	}

	enumValues := make([]string, 0)
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		valueToken := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected string literal in enum")
		if valueToken.Type == lexer.TOKEN_ERROR {
			return nil
		}

		if str, ok := valueToken.Literal.(string); ok {
			enumValues = append(enumValues, str)
		} else {
			enumValues = append(enumValues, valueToken.Lexeme)
		}

		if !p.check(lexer.TOKEN_RBRACKET) {
			if !p.match(lexer.TOKEN_COMMA) {
				p.error(p.peek(), "Expected ',' or ']' after enum value")
				return nil
			}
		}
	}

	if !p.match(lexer.TOKEN_RBRACKET) {
		p.error(p.peek(), "Expected ']' after enum values")
		return nil
	}

	typeNode := &ast.TypeNode{
		Kind:       ast.TypeEnum,
		Name:       "enum",
		EnumValues: enumValues,
		Loc:        loc,
	}

	p.parseNullabilityMarker(typeNode)
	return typeNode
}

// parseResourceType parses a resource type (identifier)
func (p *Parser) parseResourceType(loc ast.SourceLocation) *ast.TypeNode {
	typeToken := p.advance()
	typeNode := &ast.TypeNode{
		Kind: ast.TypeResource,
		Name: typeToken.Lexeme,
		Loc:  loc,
	}

	p.parseNullabilityMarker(typeNode)
	return typeNode
}

// parseNullabilityMarker checks and sets the nullability marker (! or ?)
func (p *Parser) parseNullabilityMarker(typeNode *ast.TypeNode) {
	if p.match(lexer.TOKEN_BANG) {
		typeNode.Nullable = false
	} else if p.match(lexer.TOKEN_QUESTION) {
		typeNode.Nullable = true
	} else {
		p.error(p.peek(), "Type must have nullability marker (! or ?)")
	}
}

// parseFieldConstraint parses a field-level constraint annotation
func (p *Parser) parseFieldConstraint() *ast.ConstraintNode {
	nameToken := p.advance()
	constraintName := nameToken.Lexeme

	// Map token types to constraint names for annotation keywords
	if nameToken.Type != lexer.TOKEN_IDENTIFIER {
		constraintName = p.getAnnotationName(nameToken.Type)
	}

	constraint := &ast.ConstraintNode{
		Name:      constraintName,
		Arguments: make([]ast.ExprNode, 0),
		Loc:       ast.TokenLocation(nameToken),
	}

	// Parse constraint arguments
	if p.match(lexer.TOKEN_LPAREN) {
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			arg := p.parseExpression()
			if arg != nil {
				constraint.Arguments = append(constraint.Arguments, arg)
			}

			if !p.check(lexer.TOKEN_RPAREN) {
				if !p.match(lexer.TOKEN_COMMA) {
					p.error(p.peek(), "Expected ',' or ')' after constraint argument")
					break
				}
			}
		}

		if !p.match(lexer.TOKEN_RPAREN) {
			p.error(p.peek(), "Expected ')' after constraint arguments")
		}
	}

	return constraint
}

// parseHook parses a lifecycle hook
func (p *Parser) parseHook(timingToken lexer.Token) *ast.HookNode {
	timing := timingToken.Lexeme
	if timingToken.Type == lexer.TOKEN_BEFORE {
		timing = hookTimingBefore
	} else if timingToken.Type == lexer.TOKEN_AFTER {
		timing = hookTimingAfter
	}

	// Expect event (create, update, delete, save)
	eventToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected hook event (create, update, delete, save)")
	if eventToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	hook := &ast.HookNode{
		Timing:        timing,
		Event:         eventToken.Lexeme,
		Middleware:    make([]string, 0),
		IsAsync:       false,
		IsTransaction: false,
		Body:          make([]ast.StmtNode, 0),
		Loc:           ast.TokenLocation(timingToken),
	}

	// Parse optional modifiers (transaction and async are represented as tokens)
	for p.check(lexer.TOKEN_TRANSACTION) || p.check(lexer.TOKEN_ASYNC) {
		modifierToken := p.advance()

		switch modifierToken.Type {
		case lexer.TOKEN_TRANSACTION:
			hook.IsTransaction = true
		case lexer.TOKEN_ASYNC:
			hook.IsAsync = true
		default:
			p.error(modifierToken, "Expected hook modifier (@transaction or @async)")
		}
	}

	// Parse hook body
	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for hook body")
		return nil
	}

	hook.Body = p.parseStatementList()

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after hook body")
	}

	return hook
}

// parseValidation parses a validation block
func (p *Parser) parseValidation() *ast.ValidationNode {
	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected validation name")
	if nameToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	validation := &ast.ValidationNode{
		Name: nameToken.Lexeme,
		Loc:  ast.TokenLocation(nameToken),
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for validation block")
		return nil
	}

	// Parse validation body (expect 'condition' and optionally 'error')
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		if p.match(lexer.TOKEN_CONDITION) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'condition'")
				continue
			}
			validation.Condition = p.parseExpression()
		} else if p.match(lexer.TOKEN_ERROR_KW) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'error'")
				continue
			}
			errorToken := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected error message string")
			if errorToken.Type != lexer.TOKEN_ERROR {
				if str, ok := errorToken.Literal.(string); ok {
					validation.Error = str
				} else {
					validation.Error = errorToken.Lexeme
				}
			}
		} else {
			p.error(p.peek(), "Expected 'condition' or 'error' in validation block")
			p.advance()
		}
	}

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after validation block")
	}

	return validation
}

// parseConstraintBlock parses a constraint block
func (p *Parser) parseConstraintBlock() *ast.ConstraintNode {
	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected constraint name")
	if nameToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	constraint := &ast.ConstraintNode{
		Name:      nameToken.Lexeme,
		Arguments: make([]ast.ExprNode, 0),
		On:        make([]string, 0),
		Loc:       ast.TokenLocation(nameToken),
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for constraint block")
		return nil
	}

	// Parse constraint body
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		if p.match(lexer.TOKEN_ON) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'on'")
				continue
			}

			// Parse list of events
			if !p.match(lexer.TOKEN_LBRACKET) {
				p.error(p.peek(), "Expected '[' for event list")
				continue
			}

			for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
				eventToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected event name")
				if eventToken.Type != lexer.TOKEN_ERROR {
					constraint.On = append(constraint.On, eventToken.Lexeme)
				}

				if !p.check(lexer.TOKEN_RBRACKET) {
					if !p.match(lexer.TOKEN_COMMA) {
						p.error(p.peek(), "Expected ',' or ']' after event")
						break
					}
				}
			}

			if !p.match(lexer.TOKEN_RBRACKET) {
				p.error(p.peek(), "Expected ']' after event list")
			}
		} else if p.match(lexer.TOKEN_WHEN) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'when'")
				continue
			}
			constraint.When = p.parseExpression()
		} else if p.match(lexer.TOKEN_CONDITION) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'condition'")
				continue
			}
			constraint.Condition = p.parseExpression()
		} else if p.match(lexer.TOKEN_ERROR_KW) {
			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after 'error'")
				continue
			}
			errorToken := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected error message string")
			if errorToken.Type != lexer.TOKEN_ERROR {
				if str, ok := errorToken.Literal.(string); ok {
					constraint.Error = str
				} else {
					constraint.Error = errorToken.Lexeme
				}
			}
		} else {
			p.error(p.peek(), "Expected 'on', 'when', 'condition', or 'error' in constraint block")
			p.advance()
		}
	}

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after constraint block")
	}

	return constraint
}

// parseScope parses a scope definition
func (p *Parser) parseScope() *ast.ScopeNode {
	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected scope name")
	if nameToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	scope := &ast.ScopeNode{
		Name:      nameToken.Lexeme,
		Arguments: make([]*ast.ArgumentNode, 0),
		Loc:       ast.TokenLocation(nameToken),
	}

	// Parse optional arguments
	if p.match(lexer.TOKEN_LPAREN) {
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			argNameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected argument name")
			if argNameToken.Type == lexer.TOKEN_ERROR {
				break
			}

			arg := &ast.ArgumentNode{
				Name: argNameToken.Lexeme,
				Loc:  ast.TokenLocation(argNameToken),
			}

			// Optional type annotation
			if p.match(lexer.TOKEN_COLON) {
				arg.Type = p.parseType()
			}

			scope.Arguments = append(scope.Arguments, arg)

			if !p.check(lexer.TOKEN_RPAREN) {
				if !p.match(lexer.TOKEN_COMMA) {
					p.error(p.peek(), "Expected ',' or ')' after argument")
					break
				}
			}
		}

		if !p.match(lexer.TOKEN_RPAREN) {
			p.error(p.peek(), "Expected ')' after arguments")
		}
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for scope body")
		return nil
	}

	// For now, parse the scope body as a single expression
	scope.Condition = p.parseExpression()

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after scope body")
	}

	return scope
}

// parseComputed parses a computed field
func (p *Parser) parseComputed() *ast.ComputedNode {
	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected computed field name")
	if nameToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	if !p.match(lexer.TOKEN_COLON) {
		p.error(p.peek(), "Expected ':' after computed field name")
		return nil
	}

	fieldType := p.parseType()
	if fieldType == nil {
		return nil
	}

	computed := &ast.ComputedNode{
		Name: nameToken.Lexeme,
		Type: fieldType,
		Loc:  ast.TokenLocation(nameToken),
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for computed field body")
		return nil
	}

	computed.Body = p.parseExpression()

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after computed field body")
	}

	return computed
}

// parseOperations parses the @operations annotation
func (p *Parser) parseOperations() []string {
	if !p.match(lexer.TOKEN_LBRACKET) {
		p.error(p.peek(), "Expected '[' after @operations")
		return nil
	}

	operations := make([]string, 0)
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		opToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected operation name")
		if opToken.Type != lexer.TOKEN_ERROR {
			operations = append(operations, opToken.Lexeme)
		}

		if !p.check(lexer.TOKEN_RBRACKET) {
			if !p.match(lexer.TOKEN_COMMA) {
				p.error(p.peek(), "Expected ',' or ']' after operation")
				break
			}
		}
	}

	if !p.match(lexer.TOKEN_RBRACKET) {
		p.error(p.peek(), "Expected ']' after operations")
	}

	return operations
}

// parseMiddleware parses the @middleware annotation
func (p *Parser) parseMiddleware() []string {
	if !p.match(lexer.TOKEN_LBRACKET) {
		p.error(p.peek(), "Expected '[' after @middleware")
		return nil
	}

	middleware := make([]string, 0)
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		mwToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected middleware name")
		if mwToken.Type != lexer.TOKEN_ERROR {
			middleware = append(middleware, mwToken.Lexeme)
		}

		if !p.check(lexer.TOKEN_RBRACKET) {
			if !p.match(lexer.TOKEN_COMMA) {
				p.error(p.peek(), "Expected ',' or ']' after middleware")
				break
			}
		}
	}

	if !p.match(lexer.TOKEN_RBRACKET) {
		p.error(p.peek(), "Expected ']' after middleware")
	}

	return middleware
}

// parseStatementList parses a list of statements
func (p *Parser) parseStatementList() []ast.StmtNode {
	statements := make([]ast.StmtNode, 0)

	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		if stmt := p.parseStatement(); stmt != nil {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// parseStatement parses a single statement
func (p *Parser) parseStatement() ast.StmtNode {
	// Check for control flow
	if p.match(lexer.TOKEN_IF) {
		return p.parseIfStatement()
	}

	if p.match(lexer.TOKEN_RETURN) {
		return p.parseReturnStatement()
	}

	if p.match(lexer.TOKEN_LET) {
		return p.parseLetStatement()
	}

	if p.match(lexer.TOKEN_MATCH) {
		return p.parseMatchStatement()
	}

	// Check for async block
	if p.check(lexer.TOKEN_ASYNC) {
		return p.parseAsyncBlock()
	}

	// Try to parse as expression statement or assignment
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	// Check if it's an assignment
	if p.match(lexer.TOKEN_EQUALS) {
		value := p.parseExpression()
		if value == nil {
			p.error(p.peek(), "Expected expression after '='")
			return nil
		}

		return &ast.AssignmentStmt{
			Target: expr,
			Value:  value,
			Loc:    expr.Location(),
		}
	}

	// Expression statement
	return &ast.ExprStmt{
		Expr: expr,
		Loc:  expr.Location(),
	}
}

// parseIfStatement parses an if/elsif/else statement
func (p *Parser) parseIfStatement() ast.StmtNode {
	loc := ast.TokenLocation(p.previous())

	condition := p.parseExpression()
	if condition == nil {
		p.error(p.peek(), "Expected condition after 'if'")
		return nil
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for if body")
		return nil
	}

	thenBranch := p.parseStatementList()

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after if body")
	}

	ifStmt := &ast.IfStmt{
		Condition:     condition,
		ThenBranch:    thenBranch,
		ElsIfBranches: make([]*ast.ElsIfBranch, 0),
		Loc:           loc,
	}

	// Parse elsif branches
	for p.match(lexer.TOKEN_ELSIF) {
		elsifLoc := ast.TokenLocation(p.previous())
		elsifCondition := p.parseExpression()
		if elsifCondition == nil {
			p.error(p.peek(), "Expected condition after 'elsif'")
			break
		}

		if !p.match(lexer.TOKEN_LBRACE) {
			p.error(p.peek(), "Expected '{' for elsif body")
			break
		}

		elsifBody := p.parseStatementList()

		if !p.match(lexer.TOKEN_RBRACE) {
			p.error(p.peek(), "Expected '}' after elsif body")
		}

		ifStmt.ElsIfBranches = append(ifStmt.ElsIfBranches, &ast.ElsIfBranch{
			Condition: elsifCondition,
			Body:      elsifBody,
			Loc:       elsifLoc,
		})
	}

	// Parse else branch
	if p.match(lexer.TOKEN_ELSE) {
		if !p.match(lexer.TOKEN_LBRACE) {
			p.error(p.peek(), "Expected '{' for else body")
			return ifStmt
		}

		ifStmt.ElseBranch = p.parseStatementList()

		if !p.match(lexer.TOKEN_RBRACE) {
			p.error(p.peek(), "Expected '}' after else body")
		}
	}

	return ifStmt
}

// parseReturnStatement parses a return statement
func (p *Parser) parseReturnStatement() ast.StmtNode {
	loc := ast.TokenLocation(p.previous())

	var value ast.ExprNode
	if !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		value = p.parseExpression()
	}

	return &ast.ReturnStmt{
		Value: value,
		Loc:   loc,
	}
}

// parseLetStatement parses a let statement
func (p *Parser) parseLetStatement() ast.StmtNode {
	loc := ast.TokenLocation(p.previous())

	nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected variable name after 'let'")
	if nameToken.Type == lexer.TOKEN_ERROR {
		return nil
	}

	letStmt := &ast.LetStmt{
		Name: nameToken.Lexeme,
		Loc:  loc,
	}

	// Optional type annotation
	if p.match(lexer.TOKEN_COLON) {
		letStmt.Type = p.parseType()
	}

	// Expect assignment
	if !p.match(lexer.TOKEN_EQUALS) {
		p.error(p.peek(), "Expected '=' after variable name")
		return nil
	}

	letStmt.Value = p.parseExpression()
	if letStmt.Value == nil {
		p.error(p.peek(), "Expected expression after '='")
		return nil
	}

	return letStmt
}

// parseMatchStatement parses a match statement
func (p *Parser) parseMatchStatement() ast.StmtNode {
	loc := ast.TokenLocation(p.previous())

	value := p.parseExpression()
	if value == nil {
		p.error(p.peek(), "Expected expression after 'match'")
		return nil
	}

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for match body")
		return nil
	}

	matchStmt := &ast.MatchStmt{
		Value: value,
		Cases: make([]*ast.MatchCase, 0),
		Loc:   loc,
	}

	// Parse cases
	for p.match(lexer.TOKEN_WHEN) {
		caseLoc := ast.TokenLocation(p.previous())
		pattern := p.parseExpression()
		if pattern == nil {
			p.error(p.peek(), "Expected pattern after 'when'")
			continue
		}

		if !p.match(lexer.TOKEN_LBRACE) {
			p.error(p.peek(), "Expected '{' for case body")
			continue
		}

		caseBody := p.parseStatementList()

		if !p.match(lexer.TOKEN_RBRACE) {
			p.error(p.peek(), "Expected '}' after case body")
		}

		matchStmt.Cases = append(matchStmt.Cases, &ast.MatchCase{
			Pattern: pattern,
			Body:    caseBody,
			Loc:     caseLoc,
		})
	}

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after match body")
	}

	return matchStmt
}

// parseAsyncBlock parses an @async block
func (p *Parser) parseAsyncBlock() ast.StmtNode {
	asyncToken := p.advance() // consume TOKEN_ASYNC
	loc := ast.TokenLocation(asyncToken)

	if !p.match(lexer.TOKEN_LBRACE) {
		p.error(p.peek(), "Expected '{' for async block")
		return nil
	}

	statements := p.parseStatementList()

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after async block")
	}

	return &ast.BlockStmt{
		Statements: statements,
		IsAsync:    true,
		Loc:        loc,
	}
}

// Helper methods

// isRelationshipField checks if a field is actually a relationship
func (p *Parser) isRelationshipField(field *ast.FieldNode) bool {
	// Check if the next token is a brace (relationship body)
	return p.check(lexer.TOKEN_LBRACE)
}

// fieldToRelationship converts a field to a relationship
func (p *Parser) fieldToRelationship(field *ast.FieldNode) *ast.RelationshipNode {
	relationship := &ast.RelationshipNode{
		Name:     field.Name,
		Type:     field.Type.Name,
		Nullable: field.Nullable,
		Loc:      field.Loc,
	}

	// Parse relationship body
	if p.match(lexer.TOKEN_LBRACE) {
		for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
			keyToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected relationship property")
			if keyToken.Type == lexer.TOKEN_ERROR {
				break
			}

			if !p.match(lexer.TOKEN_COLON) {
				p.error(p.peek(), "Expected ':' after property name")
				continue
			}

			switch keyToken.Lexeme {
			case "foreign_key":
				fkToken := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected string literal for foreign_key")
				if fkToken.Type != lexer.TOKEN_ERROR {
					if str, ok := fkToken.Literal.(string); ok {
						relationship.ForeignKey = str
					} else {
						relationship.ForeignKey = fkToken.Lexeme
					}
				}
			case "on_delete":
				odToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected identifier for on_delete")
				if odToken.Type != lexer.TOKEN_ERROR {
					relationship.OnDelete = odToken.Lexeme
				}
			case "through":
				throughToken := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected string literal for through")
				if throughToken.Type != lexer.TOKEN_ERROR {
					if str, ok := throughToken.Literal.(string); ok {
						relationship.Through = str
					} else {
						relationship.Through = throughToken.Lexeme
					}
					relationship.Kind = ast.RelationshipHasManyThrough
				}
			default:
				p.error(keyToken, fmt.Sprintf("Unknown relationship property: %s", keyToken.Lexeme))
			}
		}

		if !p.match(lexer.TOKEN_RBRACE) {
			p.error(p.peek(), "Expected '}' after relationship body")
		}
	}

	// Determine relationship kind based on type
	if relationship.Kind == ast.RelationshipHasManyThrough {
		// Already set via 'through' property
	} else if field.Type.Kind == ast.TypeArray {
		relationship.Kind = ast.RelationshipHasMany
	} else {
		// Default to BelongsTo for scalar resource types
		// TODO: Add support for HasOne relationships (requires 'kind' property or FK direction)
		relationship.Kind = ast.RelationshipBelongsTo
	}

	return relationship
}

// isPrimitiveType checks if the current token is a primitive type
func (p *Parser) isPrimitiveType() bool {
	return p.check(lexer.TOKEN_STRING) ||
		p.check(lexer.TOKEN_TEXT) ||
		p.check(lexer.TOKEN_MARKDOWN) ||
		p.check(lexer.TOKEN_INT) ||
		p.check(lexer.TOKEN_FLOAT) ||
		p.check(lexer.TOKEN_DECIMAL) ||
		p.check(lexer.TOKEN_BOOL) ||
		p.check(lexer.TOKEN_TIMESTAMP) ||
		p.check(lexer.TOKEN_DATE) ||
		p.check(lexer.TOKEN_TIME) ||
		p.check(lexer.TOKEN_UUID) ||
		p.check(lexer.TOKEN_ULID) ||
		p.check(lexer.TOKEN_EMAIL) ||
		p.check(lexer.TOKEN_URL) ||
		p.check(lexer.TOKEN_PHONE) ||
		p.check(lexer.TOKEN_JSON)
}

// isFieldConstraintToken checks if the current token is a field constraint annotation
func (p *Parser) isFieldConstraintToken() bool {
	return p.check(lexer.TOKEN_PRIMARY) ||
		p.check(lexer.TOKEN_AUTO) ||
		p.check(lexer.TOKEN_AUTO_UPDATE) ||
		p.check(lexer.TOKEN_UNIQUE) ||
		p.check(lexer.TOKEN_DEFAULT) ||
		p.check(lexer.TOKEN_MIN) ||
		p.check(lexer.TOKEN_MAX) ||
		p.check(lexer.TOKEN_PATTERN)
}

// isResourceAnnotationToken checks if the current token is a resource-level annotation
func (p *Parser) isResourceAnnotationToken() bool {
	return p.check(lexer.TOKEN_BEFORE) ||
		p.check(lexer.TOKEN_AFTER) ||
		p.check(lexer.TOKEN_VALIDATE) ||
		p.check(lexer.TOKEN_CONSTRAINT) ||
		p.check(lexer.TOKEN_SCOPE) ||
		p.check(lexer.TOKEN_COMPUTED) ||
		p.check(lexer.TOKEN_OPERATIONS) ||
		p.check(lexer.TOKEN_MIDDLEWARE)
}

// isFieldNameToken checks if the current token can be used as a field name
// Field names can be identifiers OR type keywords (like "email", "url", "phone")
func (p *Parser) isFieldNameToken() bool {
	return p.check(lexer.TOKEN_IDENTIFIER) || p.isPrimitiveType()
}

// consumeFieldName consumes a field name token (identifier or type keyword)
func (p *Parser) consumeFieldName() lexer.Token {
	if p.isFieldNameToken() {
		return p.advance()
	}
	p.error(p.peek(), "Expected field name")
	return lexer.Token{Type: lexer.TOKEN_ERROR}
}

// getTypeName maps token types to type names
func (p *Parser) getTypeName(tokenType lexer.TokenType) string {
	typeNames := map[lexer.TokenType]string{
		lexer.TOKEN_STRING:    "string",
		lexer.TOKEN_TEXT:      "text",
		lexer.TOKEN_MARKDOWN:  "markdown",
		lexer.TOKEN_INT:       "int",
		lexer.TOKEN_FLOAT:     "float",
		lexer.TOKEN_DECIMAL:   "decimal",
		lexer.TOKEN_BOOL:      "bool",
		lexer.TOKEN_TIMESTAMP: "timestamp",
		lexer.TOKEN_DATE:      "date",
		lexer.TOKEN_TIME:      "time",
		lexer.TOKEN_UUID:      "uuid",
		lexer.TOKEN_ULID:      "ulid",
		lexer.TOKEN_EMAIL:     "email",
		lexer.TOKEN_URL:       "url",
		lexer.TOKEN_PHONE:     "phone",
		lexer.TOKEN_JSON:      "json",
	}

	if name, ok := typeNames[tokenType]; ok {
		return name
	}

	return "unknown"
}

// getAnnotationName maps token types to annotation names
func (p *Parser) getAnnotationName(tokenType lexer.TokenType) string {
	annotationNames := map[lexer.TokenType]string{
		lexer.TOKEN_BEFORE:      hookTimingBefore,
		lexer.TOKEN_AFTER:       hookTimingAfter,
		lexer.TOKEN_VALIDATE:    "validate",
		lexer.TOKEN_CONSTRAINT:  "constraint",
		lexer.TOKEN_SCOPE:       "scope",
		lexer.TOKEN_COMPUTED:    "computed",
		lexer.TOKEN_OPERATIONS:  "operations",
		lexer.TOKEN_MIDDLEWARE:  "middleware",
		lexer.TOKEN_PRIMARY:     "primary",
		lexer.TOKEN_AUTO:        "auto",
		lexer.TOKEN_AUTO_UPDATE: "auto_update",
		lexer.TOKEN_UNIQUE:      "unique",
		lexer.TOKEN_DEFAULT:     "default",
		lexer.TOKEN_MIN:         "min",
		lexer.TOKEN_MAX:         "max",
		lexer.TOKEN_PATTERN:     "pattern",
		lexer.TOKEN_TRANSACTION: "transaction",
		lexer.TOKEN_ASYNC:       "async",
	}

	if name, ok := annotationNames[tokenType]; ok {
		return name
	}

	return "unknown"
}

// Token stream navigation

// peek returns the current token without advancing
func (p *Parser) peek() lexer.Token {
	if len(p.tokens) == 0 {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	if p.current >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.current]
}

// previous returns the most recently consumed token
func (p *Parser) previous() lexer.Token {
	if len(p.tokens) == 0 || p.current == 0 {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.current-1]
}

// advance consumes the current token and returns it
func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

// check returns true if the current token matches the given type
func (p *Parser) check(tokenType lexer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

// match consumes the token if it matches any of the given types
func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

// consume advances if the next token matches, otherwise reports an error
func (p *Parser) consume(tokenType lexer.TokenType, message string) lexer.Token {
	if p.check(tokenType) {
		return p.advance()
	}

	p.error(p.peek(), message)
	return lexer.Token{Type: lexer.TOKEN_ERROR}
}

// isAtEnd returns true if we've reached the end of the token stream
func (p *Parser) isAtEnd() bool {
	return p.current >= len(p.tokens) || p.tokens[p.current].Type == lexer.TOKEN_EOF
}

// Error handling

// error records a parse error
func (p *Parser) error(token lexer.Token, message string) {
	p.errors = append(p.errors, NewParseError(message, token))
}

// synchronize implements panic mode error recovery
func (p *Parser) synchronize() {
	p.advance()

	for !p.isAtEnd() {
		// Synchronize on resource boundaries
		if p.check(lexer.TOKEN_RESOURCE) {
			return
		}

		p.advance()
	}
}

// synchronizeToNextField synchronizes to the next field declaration
func (p *Parser) synchronizeToNextField() {
	p.advance()

	for !p.isAtEnd() {
		// Stop at identifiers (potential field names) or closing brace
		if p.check(lexer.TOKEN_IDENTIFIER) || p.check(lexer.TOKEN_RBRACE) || p.check(lexer.TOKEN_AT) {
			return
		}

		p.advance()
	}
}
