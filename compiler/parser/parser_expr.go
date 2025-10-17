package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/conduit-lang/conduit/compiler/lexer"
)

// Operator precedence levels (higher number = higher precedence)
const (
	PREC_NONE       = iota
	PREC_ASSIGNMENT // = (right associative)
	PREC_TERNARY    // ?: (right associative)
	PREC_COALESCE   // ?? (right associative)
	PREC_OR         // ||
	PREC_AND        // &&
	PREC_EQUALITY   // == !=
	PREC_COMPARISON // < <= > >=
	PREC_TERM       // + -
	PREC_FACTOR     // * / %
	PREC_EXPONENT   // ** (right associative)
	PREC_UNARY      // ! - (unary)
	PREC_CALL       // () [] . ?. (left associative)
	PREC_PRIMARY    // literals, identifiers
)

// parseExpression parses an expression with the minimum precedence
func (p *Parser) parseExpression() ExprNode {
	return p.parseExpressionWithPrecedence(PREC_NONE)
}

// parseExpressionWithPrecedence implements Pratt parsing / operator precedence climbing
func (p *Parser) parseExpressionWithPrecedence(minPrec int) ExprNode {
	// Parse prefix expression (unary, primary)
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// Parse infix expressions based on precedence
	for {
		precedence := p.getCurrentPrecedence()
		if precedence < minPrec {
			break
		}

		// Handle right-associative operators
		if p.isRightAssociative() {
			precedence--
		}

		// Save current position to detect if parseInfixExpression consumed anything
		oldPosition := p.current
		left = p.parseInfixExpression(left, precedence)
		if left == nil {
			break
		}

		// If no token was consumed, we're done with infix operations
		if p.current == oldPosition {
			break
		}
	}

	return left
}

// parsePrefixExpression parses prefix expressions (unary, primary)
func (p *Parser) parsePrefixExpression() ExprNode {
	startToken := p.peek()
	loc := TokenToLocation(startToken)

	// Unary operators: ! -
	if p.match(lexer.TOKEN_BANG, lexer.TOKEN_MINUS) {
		operator := p.previous().Type
		operand := p.parseExpressionWithPrecedence(PREC_UNARY)
		if operand == nil {
			p.addError(ParseError{
				Message:  "Expected expression after unary operator",
				Location: loc,
			})
			return nil
		}
		return NewUnaryExpr(operator, operand, loc)
	}

	// Primary expressions
	return p.parsePrimaryExpression()
}

// parsePrimaryExpression parses primary expressions
func (p *Parser) parsePrimaryExpression() ExprNode {
	startToken := p.peek()
	loc := TokenToLocation(startToken)

	// Literals
	switch {
	case p.check(lexer.TOKEN_INT_LITERAL):
		token := p.advance()
		return NewLiteralExpr(token.Literal, loc)

	case p.check(lexer.TOKEN_FLOAT_LITERAL):
		token := p.advance()
		return NewLiteralExpr(token.Literal, loc)

	case p.check(lexer.TOKEN_STRING_LITERAL):
		return p.parseStringLiteralOrInterpolation()

	case p.match(lexer.TOKEN_TRUE):
		return NewLiteralExpr(true, loc)

	case p.match(lexer.TOKEN_FALSE):
		return NewLiteralExpr(false, loc)

	case p.match(lexer.TOKEN_NIL):
		return NewLiteralExpr(nil, loc)

	case p.match(lexer.TOKEN_SELF):
		return NewSelfExpr(loc)

	case p.check(lexer.TOKEN_LBRACKET):
		return p.parseArrayLiteral()

	case p.check(lexer.TOKEN_LBRACE):
		return p.parseHashLiteral()

	case p.check(lexer.TOKEN_LPAREN):
		return p.parseGroupExpression()

	case p.check(lexer.TOKEN_IF):
		return p.parseIfExpression()

	case p.check(lexer.TOKEN_UNLESS):
		return p.parseUnlessExpression()

	case p.check(lexer.TOKEN_MATCH):
		return p.parseMatchExpression()

	case p.check(lexer.TOKEN_IDENTIFIER):
		return p.parseIdentifierOrCall()
	}

	p.addError(ParseError{
		Message:  fmt.Sprintf("Expected expression, got %s", p.peek().Type),
		Location: loc,
	})
	return nil
}

// parseInfixExpression parses infix expressions
func (p *Parser) parseInfixExpression(left ExprNode, precedence int) ExprNode {
	loc := TokenToLocation(p.peek())

	// Assignment: =
	if p.match(lexer.TOKEN_EQUAL) {
		right := p.parseExpressionWithPrecedence(precedence)
		if right == nil {
			p.addError(ParseError{
				Message:  "Expected expression after '='",
				Location: loc,
			})
			return left
		}
		return NewAssignmentExpr(left, right, loc)
	}

	// Ternary: ? :
	if p.match(lexer.TOKEN_QUESTION) {
		trueExpr := p.parseExpression()
		if trueExpr == nil {
			p.addError(ParseError{
				Message:  "Expected expression after '?'",
				Location: loc,
			})
			return left
		}

		if _, ok := p.consume(lexer.TOKEN_COLON, "Expected ':' in ternary expression"); !ok {
			return left
		}

		falseExpr := p.parseExpressionWithPrecedence(precedence)
		if falseExpr == nil {
			p.addError(ParseError{
				Message:  "Expected expression after ':'",
				Location: loc,
			})
			return left
		}

		return NewTernaryExpr(left, trueExpr, falseExpr, loc)
	}

	// Null coalescing: ??
	if p.match(lexer.TOKEN_QUESTION_QUESTION) {
		right := p.parseExpressionWithPrecedence(precedence)
		if right == nil {
			p.addError(ParseError{
				Message:  "Expected expression after '??'",
				Location: loc,
			})
			return left
		}
		return NewCoalesceExpr(left, right, loc)
	}

	// Binary operators
	binaryOps := []lexer.TokenType{
		lexer.TOKEN_PIPE_PIPE, lexer.TOKEN_AMPERSAND_AMPERSAND,
		lexer.TOKEN_EQUAL_EQUAL, lexer.TOKEN_BANG_EQUAL,
		lexer.TOKEN_LESS, lexer.TOKEN_LESS_EQUAL,
		lexer.TOKEN_GREATER, lexer.TOKEN_GREATER_EQUAL,
		lexer.TOKEN_PLUS, lexer.TOKEN_MINUS,
		lexer.TOKEN_STAR, lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT,
		lexer.TOKEN_STAR_STAR,
		lexer.TOKEN_IN,
	}

	for _, op := range binaryOps {
		if p.match(op) {
			right := p.parseExpressionWithPrecedence(precedence + 1)
			if right == nil {
				p.addError(ParseError{
					Message:  fmt.Sprintf("Expected expression after '%s'", p.previous().Lexeme),
					Location: loc,
				})
				return left
			}
			return NewBinaryExpr(left, op, right, loc)
		}
	}

	// Postfix operators: () [] . ?.
	if p.match(lexer.TOKEN_LPAREN) {
		return p.parseCallExpression(left)
	}

	if p.match(lexer.TOKEN_LBRACKET) {
		return p.parseIndexExpression(left)
	}

	if p.match(lexer.TOKEN_DOT) {
		return p.parseFieldAccessOrMethodCall(left)
	}

	if p.match(lexer.TOKEN_QUESTION_DOT) {
		return p.parseSafeNavigation(left)
	}

	return left
}

// parseStringLiteralOrInterpolation parses a string literal, checking for interpolation
func (p *Parser) parseStringLiteralOrInterpolation() ExprNode {
	token := p.advance()
	loc := TokenToLocation(token)

	// Check if the string contains interpolation markers #{}
	str := token.Literal.(string)
	if !strings.Contains(str, "#{") {
		// Simple string literal
		return NewLiteralExpr(str, loc)
	}

	// Parse string interpolation
	parts := []ExprNode{}
	current := ""
	i := 0

	for i < len(str) {
		if i < len(str)-1 && str[i] == '#' && str[i+1] == '{' {
			// Found interpolation start
			if current != "" {
				parts = append(parts, NewLiteralExpr(current, loc))
				current = ""
			}

			// Find matching }
			i += 2
			depth := 1
			exprStr := ""
			for i < len(str) && depth > 0 {
				if str[i] == '{' {
					depth++
				} else if str[i] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				exprStr += string(str[i])
				i++
			}

			// Parse the expression inside #{}
			// TODO(CON-13): String interpolation parser is simplified
			// Currently only handles simple identifiers in #{} blocks
			// Full implementation needs lexer support to tokenize interpolated expressions
			// Examples that won't work yet:
			//   - Field access: "Hello #{user.name}"
			//   - Function calls: "Total: $#{String.format(order.total)}"
			//   - Binary expressions: "Sum: #{a + b}"
			//
			// Proper fix requires lexer changes to emit special tokens for #{} blocks
			if exprStr != "" {
				// For now, try to parse as a complete mini-expression
				exprLexer := lexer.New(exprStr, "interpolation")
				exprTokens, _ := exprLexer.ScanTokens()

				if len(exprTokens) > 0 {
					exprParser := New(exprTokens)
					expr := exprParser.parseExpression()

					if expr != nil && len(exprParser.errors) == 0 {
						parts = append(parts, expr)
					} else {
						// Fallback to identifier for simple cases
						parts = append(parts, NewIdentifierExpr(exprStr, loc))
					}
				}
			}
			i++ // Skip closing }
		} else {
			current += string(str[i])
			i++
		}
	}

	if current != "" {
		parts = append(parts, NewLiteralExpr(current, loc))
	}

	if len(parts) == 1 {
		// If only one part and it's a literal, just return the literal
		if lit, ok := parts[0].(*LiteralExpr); ok {
			return lit
		}
	}

	return NewStringInterpolationExpr(parts, loc)
}

// parseArrayLiteral parses an array literal [1, 2, 3]
func (p *Parser) parseArrayLiteral() ExprNode {
	startToken := p.peek()
	loc := TokenToLocation(startToken)

	p.consume(lexer.TOKEN_LBRACKET, "Expected '['")

	elements := []ExprNode{}

	// Handle empty array
	if p.check(lexer.TOKEN_RBRACKET) {
		p.advance()
		return NewArrayLiteralExpr(elements, loc)
	}

	// Parse elements
	for {
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACKET) {
			break
		}

		elem := p.parseExpression()
		if elem == nil {
			p.addError(ParseError{
				Message:  "Expected expression in array literal",
				Location: TokenToLocation(p.peek()),
			})
			return nil
		}
		elements = append(elements, elem)

		p.skipNewlines()

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	p.skipNewlines()
	p.consume(lexer.TOKEN_RBRACKET, "Expected ']' after array elements")

	return NewArrayLiteralExpr(elements, loc)
}

// parseHashLiteral parses a hash literal {name: "Alice", age: 30}
func (p *Parser) parseHashLiteral() ExprNode {
	startToken := p.peek()
	loc := TokenToLocation(startToken)

	p.consume(lexer.TOKEN_LBRACE, "Expected '{'")

	pairs := []HashPair{}

	// Handle empty hash
	if p.check(lexer.TOKEN_RBRACE) {
		p.advance()
		return NewHashLiteralExpr(pairs, loc)
	}

	// Parse key-value pairs
	for {
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACE) {
			break
		}

		// Parse key (identifier or string literal)
		var key string
		if p.check(lexer.TOKEN_IDENTIFIER) {
			key = p.advance().Lexeme
		} else if p.check(lexer.TOKEN_STRING_LITERAL) {
			token := p.advance()
			key = token.Literal.(string)
		} else {
			p.addError(ParseError{
				Message:  "Expected identifier or string as hash key",
				Location: TokenToLocation(p.peek()),
			})
			return nil
		}

		p.consume(lexer.TOKEN_COLON, "Expected ':' after hash key")

		value := p.parseExpression()
		if value == nil {
			p.addError(ParseError{
				Message:  "Expected expression as hash value",
				Location: TokenToLocation(p.peek()),
			})
			return nil
		}

		pairs = append(pairs, HashPair{Key: key, Value: value})

		p.skipNewlines()

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	p.skipNewlines()
	p.consume(lexer.TOKEN_RBRACE, "Expected '}' after hash elements")

	return NewHashLiteralExpr(pairs, loc)
}

// parseGroupExpression parses a parenthesized expression
func (p *Parser) parseGroupExpression() ExprNode {
	loc := TokenToLocation(p.peek())
	p.consume(lexer.TOKEN_LPAREN, "Expected '('")

	expr := p.parseExpression()
	if expr == nil {
		p.addError(ParseError{
			Message:  "Expected expression inside parentheses",
			Location: loc,
		})
		return nil
	}

	p.consume(lexer.TOKEN_RPAREN, "Expected ')' after expression")

	return NewGroupExpr(expr, loc)
}

// parseIdentifierOrCall parses an identifier, which might be the start of a namespace call
func (p *Parser) parseIdentifierOrCall() ExprNode {
	loc := TokenToLocation(p.peek())
	name := p.advance().Lexeme

	// Check if this is a namespaced call (e.g., String.slugify())
	if p.check(lexer.TOKEN_DOT) {
		p.advance() // consume .

		if !p.check(lexer.TOKEN_IDENTIFIER) {
			p.addError(ParseError{
				Message:  "Expected function name after namespace",
				Location: TokenToLocation(p.peek()),
			})
			return NewIdentifierExpr(name, loc)
		}

		functionName := p.advance().Lexeme

		// Must be followed by (
		if !p.check(lexer.TOKEN_LPAREN) {
			// This is actually field access, not a call
			// Create a namespace identifier and field access
			namespaceExpr := NewIdentifierExpr(name, loc)
			return NewFieldAccessExpr(namespaceExpr, functionName, loc)
		}

		// Parse function call arguments
		p.advance() // consume (
		args := p.parseArguments()
		p.consume(lexer.TOKEN_RPAREN, "Expected ')' after arguments")

		return NewCallExpr(name, functionName, args, loc)
	}

	return NewIdentifierExpr(name, loc)
}

// parseCallExpression parses a function call (already consumed '(')
func (p *Parser) parseCallExpression(callee ExprNode) ExprNode {
	loc := callee.GetLocation()

	args := p.parseArguments()
	p.consume(lexer.TOKEN_RPAREN, "Expected ')' after arguments")

	// Determine if this is a method call or function call
	if fieldAccess, ok := callee.(*FieldAccessExpr); ok {
		// It's a method call: object.method()
		return NewMethodCallExpr(fieldAccess.Object, fieldAccess.Field, args, loc)
	}

	// It's a function call on an identifier or expression
	// This shouldn't happen in normal Conduit code (all stdlib is namespaced)
	// but we handle it for completeness
	if ident, ok := callee.(*IdentifierExpr); ok {
		return NewCallExpr("", ident.Name, args, loc)
	}

	p.addError(ParseError{
		Message:  "Invalid function call",
		Location: loc,
	})
	return callee
}

// parseArguments parses function arguments (already consumed '(')
func (p *Parser) parseArguments() []ExprNode {
	args := []ExprNode{}

	if p.check(lexer.TOKEN_RPAREN) {
		return args
	}

	for {
		p.skipNewlines()

		arg := p.parseExpression()
		if arg == nil {
			break
		}
		args = append(args, arg)

		p.skipNewlines()

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	return args
}

// parseIndexExpression parses array/hash indexing (already consumed '[')
func (p *Parser) parseIndexExpression(object ExprNode) ExprNode {
	loc := object.GetLocation()

	index := p.parseExpression()
	if index == nil {
		p.addError(ParseError{
			Message:  "Expected expression inside brackets",
			Location: loc,
		})
		return object
	}

	p.consume(lexer.TOKEN_RBRACKET, "Expected ']' after index")

	return NewIndexExpr(object, index, loc)
}

// parseFieldAccessOrMethodCall parses field access or method call (already consumed '.')
func (p *Parser) parseFieldAccessOrMethodCall(object ExprNode) ExprNode {
	loc := object.GetLocation()

	if !p.check(lexer.TOKEN_IDENTIFIER) {
		p.addError(ParseError{
			Message:  "Expected field or method name after '.'",
			Location: TokenToLocation(p.peek()),
		})
		return object
	}

	fieldName := p.advance().Lexeme

	// Check if this is a method call (followed by '(')
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance() // consume (
		args := p.parseArguments()
		p.consume(lexer.TOKEN_RPAREN, "Expected ')' after arguments")
		return NewMethodCallExpr(object, fieldName, args, loc)
	}

	// It's field access
	return NewFieldAccessExpr(object, fieldName, loc)
}

// parseSafeNavigation parses safe navigation (already consumed '?.')
func (p *Parser) parseSafeNavigation(object ExprNode) ExprNode {
	loc := object.GetLocation()

	if !p.check(lexer.TOKEN_IDENTIFIER) {
		p.addError(ParseError{
			Message:  "Expected field name after '?.'",
			Location: TokenToLocation(p.peek()),
		})
		return object
	}

	fieldName := p.advance().Lexeme

	return NewSafeNavigationExpr(object, fieldName, loc)
}

// parseIfExpression parses an if expression
func (p *Parser) parseIfExpression() ExprNode {
	loc := TokenToLocation(p.peek())
	p.consume(lexer.TOKEN_IF, "Expected 'if'")

	condition := p.parseExpression()
	if condition == nil {
		p.addError(ParseError{
			Message:  "Expected condition after 'if'",
			Location: loc,
		})
		return nil
	}

	p.consume(lexer.TOKEN_LBRACE, "Expected '{' after if condition")
	thenBody := p.parseStatementBlock()
	p.consume(lexer.TOKEN_RBRACE, "Expected '}' after if body")

	// Parse elsif branches
	elsifBranches := []ElsifBranch{}
	for p.match(lexer.TOKEN_ELSIF) {
		elsifCondition := p.parseExpression()
		if elsifCondition == nil {
			p.addError(ParseError{
				Message:  "Expected condition after 'elsif'",
				Location: TokenToLocation(p.peek()),
			})
			break
		}

		p.consume(lexer.TOKEN_LBRACE, "Expected '{' after elsif condition")
		elsifBody := p.parseStatementBlock()
		p.consume(lexer.TOKEN_RBRACE, "Expected '}' after elsif body")

		elsifBranches = append(elsifBranches, ElsifBranch{
			Condition: elsifCondition,
			Body:      elsifBody,
		})
	}

	// Parse else branch
	var elseBody []StmtNode
	if p.match(lexer.TOKEN_ELSE) {
		p.consume(lexer.TOKEN_LBRACE, "Expected '{' after else")
		elseBody = p.parseStatementBlock()
		p.consume(lexer.TOKEN_RBRACE, "Expected '}' after else body")
	}

	return NewIfExpr(condition, thenBody, elsifBranches, elseBody, loc)
}

// parseUnlessExpression parses an unless expression
func (p *Parser) parseUnlessExpression() ExprNode {
	loc := TokenToLocation(p.peek())
	p.consume(lexer.TOKEN_UNLESS, "Expected 'unless'")

	condition := p.parseExpression()
	if condition == nil {
		p.addError(ParseError{
			Message:  "Expected condition after 'unless'",
			Location: loc,
		})
		return nil
	}

	p.consume(lexer.TOKEN_LBRACE, "Expected '{' after unless condition")
	body := p.parseStatementBlock()
	p.consume(lexer.TOKEN_RBRACE, "Expected '}' after unless body")

	return NewUnlessExpr(condition, body, loc)
}

// parseMatchExpression parses a match expression
func (p *Parser) parseMatchExpression() ExprNode {
	loc := TokenToLocation(p.peek())
	p.consume(lexer.TOKEN_MATCH, "Expected 'match'")

	value := p.parseExpression()
	if value == nil {
		p.addError(ParseError{
			Message:  "Expected expression after 'match'",
			Location: loc,
		})
		return nil
	}

	p.consume(lexer.TOKEN_LBRACE, "Expected '{' after match value")

	cases := []MatchCase{}
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		p.skipNewlines()

		// Parse pattern (string literal)
		if !p.check(lexer.TOKEN_STRING_LITERAL) {
			p.addError(ParseError{
				Message:  "Expected string literal as match pattern",
				Location: TokenToLocation(p.peek()),
			})
			break
		}
		pattern := p.advance().Literal.(string)

		p.consume(lexer.TOKEN_FAT_ARROW, "Expected '=>' after match pattern")

		// Parse expression
		expr := p.parseExpression()
		if expr == nil {
			p.addError(ParseError{
				Message:  "Expected expression after '=>'",
				Location: TokenToLocation(p.peek()),
			})
			break
		}

		cases = append(cases, MatchCase{Pattern: pattern, Expr: expr})

		p.skipNewlines()

		// Optional comma
		p.match(lexer.TOKEN_COMMA)
	}

	p.consume(lexer.TOKEN_RBRACE, "Expected '}' after match cases")

	return NewMatchExpr(value, cases, loc)
}

// getCurrentPrecedence returns the precedence of the current token
func (p *Parser) getCurrentPrecedence() int {
	switch p.peek().Type {
	case lexer.TOKEN_EQUAL:
		return PREC_ASSIGNMENT
	case lexer.TOKEN_QUESTION:
		return PREC_TERNARY
	case lexer.TOKEN_QUESTION_QUESTION:
		return PREC_COALESCE
	case lexer.TOKEN_PIPE_PIPE:
		return PREC_OR
	case lexer.TOKEN_AMPERSAND_AMPERSAND:
		return PREC_AND
	case lexer.TOKEN_EQUAL_EQUAL, lexer.TOKEN_BANG_EQUAL:
		return PREC_EQUALITY
	case lexer.TOKEN_LESS, lexer.TOKEN_LESS_EQUAL, lexer.TOKEN_GREATER, lexer.TOKEN_GREATER_EQUAL, lexer.TOKEN_IN:
		return PREC_COMPARISON
	case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS:
		return PREC_TERM
	case lexer.TOKEN_STAR, lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT:
		return PREC_FACTOR
	case lexer.TOKEN_STAR_STAR:
		return PREC_EXPONENT
	case lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET, lexer.TOKEN_DOT, lexer.TOKEN_QUESTION_DOT:
		return PREC_CALL
	default:
		// Non-operators (EOF, literals, etc.) have no precedence
		// Return -1 to ensure the loop terminates
		return -1
	}
}

// isRightAssociative returns true if the current operator is right-associative
func (p *Parser) isRightAssociative() bool {
	switch p.peek().Type {
	case lexer.TOKEN_EQUAL, lexer.TOKEN_QUESTION, lexer.TOKEN_QUESTION_QUESTION, lexer.TOKEN_STAR_STAR:
		return true
	default:
		return false
	}
}

// parseIntegerLiteral parses an integer literal and returns int64
func parseIntegerLiteral(s string) (int64, error) {
	// Remove underscores for readability (e.g., 1_000_000)
	s = strings.ReplaceAll(s, "_", "")
	return strconv.ParseInt(s, 10, 64)
}

// parseFloatLiteral parses a float literal and returns float64
func parseFloatLiteral(s string) (float64, error) {
	// Remove underscores for readability
	s = strings.ReplaceAll(s, "_", "")
	return strconv.ParseFloat(s, 64)
}
