package parser

import (
	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
)

// Expression parsing using Pratt parsing / precedence climbing
//
// Expression grammar (from lowest to highest precedence):
// expression     → assignment
// assignment     → IDENTIFIER "=" expression | nullCoalesce
// nullCoalesce   → logicalOr ( "??" logicalOr )*
// logicalOr      → logicalAnd ( ("||" | "or") logicalAnd )*
// logicalAnd     → equality ( ("&&" | "and") equality )*
// equality       → comparison ( ( "==" | "!=" ) comparison )*
// comparison     → term ( ( ">" | ">=" | "<" | "<=" ) term )*
// term           → factor ( ( "+" | "-" ) factor )*
// factor         → exponentiation ( ( "*" | "/" | "%" ) exponentiation )*
// exponentiation → unary ( "**" exponentiation )?   [right-associative]
// unary          → ( "!" | "-" | "not" ) unary | call
// call           → primary ( "(" arguments? ")" | "." IDENTIFIER | "?." IDENTIFIER | "[" expression "]" )*
// primary        → literal | IDENTIFIER | "self" | "(" expression ")" | arrayLiteral | hashLiteral

// parseExpression is the entry point for expression parsing
func (p *Parser) parseExpression() ast.ExprNode {
	return p.parseAssignment()
}

// parseAssignment handles assignment expressions
func (p *Parser) parseAssignment() ast.ExprNode {
	return p.parseNullCoalesce()
}

// parseNullCoalesce handles null coalescing operator (??)
func (p *Parser) parseNullCoalesce() ast.ExprNode {
	expr := p.parseLogicalOr()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_DOUBLE_QUESTION) {
		operator := p.previous()
		right := p.parseLogicalOr()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.NullCoalesceExpr{
			Left:  expr,
			Right: right,
			Loc:   ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseLogicalOr handles logical OR expressions
func (p *Parser) parseLogicalOr() ast.ExprNode {
	expr := p.parseLogicalAnd()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_DOUBLE_PIPE, lexer.TOKEN_OR) {
		operator := p.previous()
		right := p.parseLogicalAnd()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.LogicalExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseLogicalAnd handles logical AND expressions
func (p *Parser) parseLogicalAnd() ast.ExprNode {
	expr := p.parseEquality()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_DOUBLE_AMP, lexer.TOKEN_AND) {
		operator := p.previous()
		right := p.parseEquality()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.LogicalExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseEquality handles equality operators (==, !=)
func (p *Parser) parseEquality() ast.ExprNode {
	expr := p.parseComparison()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_EQ, lexer.TOKEN_NEQ) {
		operator := p.previous()
		right := p.parseComparison()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseComparison handles comparison operators (<, >, <=, >=)
func (p *Parser) parseComparison() ast.ExprNode {
	expr := p.parseTerm()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_LT, lexer.TOKEN_GT, lexer.TOKEN_LTE, lexer.TOKEN_GTE) {
		operator := p.previous()
		right := p.parseTerm()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseTerm handles addition and subtraction
func (p *Parser) parseTerm() ast.ExprNode {
	expr := p.parseFactor()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_PLUS, lexer.TOKEN_MINUS) {
		operator := p.previous()
		right := p.parseFactor()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseFactor handles multiplication, division, and modulo
func (p *Parser) parseFactor() ast.ExprNode {
	expr := p.parseExponentiation()
	if expr == nil {
		return nil
	}

	for p.match(lexer.TOKEN_STAR, lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT) {
		operator := p.previous()
		right := p.parseExponentiation()
		if right == nil {
			return expr // Return what we have so far
		}
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseExponentiation handles exponentiation (right-associative)
func (p *Parser) parseExponentiation() ast.ExprNode {
	expr := p.parseUnary()

	// Right-associative: recursively parse the right side
	if p.match(lexer.TOKEN_DOUBLE_STAR) {
		operator := p.previous()
		right := p.parseExponentiation() // Recursive call for right-associativity
		return &ast.BinaryExpr{
			Left:     expr,
			Operator: operator.Lexeme,
			Right:    right,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return expr
}

// parseUnary handles unary operators (!, -, not)
func (p *Parser) parseUnary() ast.ExprNode {
	if p.match(lexer.TOKEN_BANG, lexer.TOKEN_MINUS, lexer.TOKEN_NOT) {
		operator := p.previous()
		operand := p.parseUnary()
		return &ast.UnaryExpr{
			Operator: operator.Lexeme,
			Operand:  operand,
			Loc:      ast.TokenLocation(operator),
		}
	}

	return p.parseCall()
}

// parseCall handles function calls, field access, and indexing
func (p *Parser) parseCall() ast.ExprNode {
	expr := p.parsePrimary()

	for {
		if p.match(lexer.TOKEN_LPAREN) {
			expr = p.finishCall(expr)
		} else if p.match(lexer.TOKEN_DOT) {
			// Field access or method call
			nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected field name after '.'")
			if nameToken.Type == lexer.TOKEN_ERROR {
				return expr
			}

			// Check if this is a namespaced function call
			if identExpr, ok := expr.(*ast.IdentifierExpr); ok {
				// This is a namespaced call like String.slugify()
				if p.match(lexer.TOKEN_LPAREN) {
					args := p.parseArguments()
					if !p.match(lexer.TOKEN_RPAREN) {
						p.error(p.peek(), "Expected ')' after arguments")
					}

					expr = &ast.CallExpr{
						Namespace: identExpr.Name,
						Function:  nameToken.Lexeme,
						Arguments: args,
						Loc:       identExpr.Loc,
					}
				} else {
					// Just field access
					expr = &ast.FieldAccessExpr{
						Object: expr,
						Field:  nameToken.Lexeme,
						Loc:    expr.Location(),
					}
				}
			} else {
				// Regular field access
				expr = &ast.FieldAccessExpr{
					Object: expr,
					Field:  nameToken.Lexeme,
					Loc:    expr.Location(),
				}
			}
		} else if p.match(lexer.TOKEN_SAFE_NAV) {
			// Safe navigation (?.)
			nameToken := p.consume(lexer.TOKEN_IDENTIFIER, "Expected field name after '?.'")
			if nameToken.Type == lexer.TOKEN_ERROR {
				return expr
			}

			expr = &ast.SafeNavigationExpr{
				Object: expr,
				Field:  nameToken.Lexeme,
				Loc:    expr.Location(),
			}
		} else if p.match(lexer.TOKEN_LBRACKET) {
			// Indexing
			index := p.parseExpression()
			if index == nil {
				p.error(p.peek(), "Expected expression inside brackets")
				return expr
			}

			if !p.match(lexer.TOKEN_RBRACKET) {
				p.error(p.peek(), "Expected ']' after index")
			}

			expr = &ast.IndexExpr{
				Object: expr,
				Index:  index,
				Loc:    expr.Location(),
			}
		} else {
			break
		}
	}

	return expr
}

// finishCall completes parsing a function call
func (p *Parser) finishCall(callee ast.ExprNode) ast.ExprNode {
	args := p.parseArguments()

	if !p.match(lexer.TOKEN_RPAREN) {
		p.error(p.peek(), "Expected ')' after arguments")
	}

	// Handle different callee types
	if fieldAccess, ok := callee.(*ast.FieldAccessExpr); ok {
		// Method call or namespaced function
		if identExpr, ok := fieldAccess.Object.(*ast.IdentifierExpr); ok {
			// Namespaced function call
			return &ast.CallExpr{
				Namespace: identExpr.Name,
				Function:  fieldAccess.Field,
				Arguments: args,
				Loc:       callee.Location(),
			}
		}
	}

	if identExpr, ok := callee.(*ast.IdentifierExpr); ok {
		// Simple function call
		return &ast.CallExpr{
			Function:  identExpr.Name,
			Arguments: args,
			Loc:       callee.Location(),
		}
	}

	// This shouldn't normally happen for valid Conduit syntax
	p.error(p.previous(), "Invalid function call")
	return callee
}

// parseArguments parses a comma-separated list of arguments
func (p *Parser) parseArguments() []ast.ExprNode {
	args := make([]ast.ExprNode, 0)

	if p.check(lexer.TOKEN_RPAREN) {
		return args
	}

	for {
		arg := p.parseExpression()
		if arg != nil {
			args = append(args, arg)
		}

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	return args
}

// parsePrimary handles primary expressions
func (p *Parser) parsePrimary() ast.ExprNode {
	loc := ast.TokenLocation(p.peek())

	// Literals
	if p.match(lexer.TOKEN_TRUE) {
		return &ast.LiteralExpr{
			Value: true,
			Loc:   loc,
		}
	}

	if p.match(lexer.TOKEN_FALSE) {
		return &ast.LiteralExpr{
			Value: false,
			Loc:   loc,
		}
	}

	if p.match(lexer.TOKEN_NULL) {
		return &ast.LiteralExpr{
			Value: nil,
			Loc:   loc,
		}
	}

	if p.match(lexer.TOKEN_INT_LITERAL) {
		return &ast.LiteralExpr{
			Value: p.previous().Literal,
			Loc:   loc,
		}
	}

	if p.match(lexer.TOKEN_FLOAT_LITERAL) {
		return &ast.LiteralExpr{
			Value: p.previous().Literal,
			Loc:   loc,
		}
	}

	if p.match(lexer.TOKEN_STRING_LITERAL) {
		return &ast.LiteralExpr{
			Value: p.previous().Literal,
			Loc:   loc,
		}
	}

	// Self keyword
	if p.match(lexer.TOKEN_SELF) {
		return &ast.SelfExpr{
			Loc: loc,
		}
	}

	// Identifier
	if p.match(lexer.TOKEN_IDENTIFIER) {
		return &ast.IdentifierExpr{
			Name: p.previous().Lexeme,
			Loc:  loc,
		}
	}

	// Parenthesized expression
	if p.match(lexer.TOKEN_LPAREN) {
		expr := p.parseExpression()
		if !p.match(lexer.TOKEN_RPAREN) {
			p.error(p.peek(), "Expected ')' after expression")
		}
		return &ast.ParenExpr{
			Expr: expr,
			Loc:  loc,
		}
	}

	// Array literal
	if p.match(lexer.TOKEN_LBRACKET) {
		return p.parseArrayLiteral(loc)
	}

	// Hash literal
	if p.match(lexer.TOKEN_LBRACE) {
		return p.parseHashLiteral(loc)
	}

	p.error(p.peek(), "Expected expression")
	return nil
}

// parseArrayLiteral parses an array literal [1, 2, 3]
func (p *Parser) parseArrayLiteral(loc ast.SourceLocation) ast.ExprNode {
	elements := make([]ast.ExprNode, 0)

	if p.check(lexer.TOKEN_RBRACKET) {
		p.advance()
		return &ast.ArrayLiteralExpr{
			Elements: elements,
			Loc:      loc,
		}
	}

	for {
		elem := p.parseExpression()
		if elem != nil {
			elements = append(elements, elem)
		}

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}

		// Allow trailing comma
		if p.check(lexer.TOKEN_RBRACKET) {
			break
		}
	}

	if !p.match(lexer.TOKEN_RBRACKET) {
		p.error(p.peek(), "Expected ']' after array elements")
	}

	return &ast.ArrayLiteralExpr{
		Elements: elements,
		Loc:      loc,
	}
}

// parseHashLiteral parses a hash literal {key: value}
func (p *Parser) parseHashLiteral(loc ast.SourceLocation) ast.ExprNode {
	pairs := make([]ast.HashPair, 0)

	if p.check(lexer.TOKEN_RBRACE) {
		p.advance()
		return &ast.HashLiteralExpr{
			Pairs: pairs,
			Loc:   loc,
		}
	}

	for {
		pairLoc := ast.TokenLocation(p.peek())

		// Parse key (can be identifier or expression)
		var key ast.ExprNode
		if p.check(lexer.TOKEN_IDENTIFIER) {
			keyToken := p.advance()
			key = &ast.IdentifierExpr{
				Name: keyToken.Lexeme,
				Loc:  pairLoc,
			}
		} else {
			key = p.parseExpression()
			if key == nil {
				p.error(p.peek(), "Expected hash key")
				break
			}
		}

		if !p.match(lexer.TOKEN_COLON) {
			p.error(p.peek(), "Expected ':' after hash key")
			break
		}

		value := p.parseExpression()
		if value == nil {
			p.error(p.peek(), "Expected hash value")
			break
		}

		pairs = append(pairs, ast.HashPair{
			Key:   key,
			Value: value,
			Loc:   pairLoc,
		})

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}

		// Allow trailing comma
		if p.check(lexer.TOKEN_RBRACE) {
			break
		}
	}

	if !p.match(lexer.TOKEN_RBRACE) {
		p.error(p.peek(), "Expected '}' after hash pairs")
	}

	return &ast.HashLiteralExpr{
		Pairs: pairs,
		Loc:   loc,
	}
}
