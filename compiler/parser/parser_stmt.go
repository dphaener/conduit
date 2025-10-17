package parser

import "github.com/conduit-lang/conduit/compiler/lexer"

// parseStatement parses a single statement
func (p *Parser) parseStatement() StmtNode {
	p.skipNewlines()

	startToken := p.peek()
	loc := TokenToLocation(startToken)

	// Let statement: let x = expr
	if p.match(lexer.TOKEN_LET) {
		return p.parseLetStatement()
	}

	// Return statement
	if p.match(lexer.TOKEN_RETURN) {
		return p.parseReturnStatement()
	}

	// If statement
	if p.check(lexer.TOKEN_IF) {
		return p.parseIfStatement()
	}

	// Unless statement
	if p.check(lexer.TOKEN_UNLESS) {
		return p.parseUnlessStatement()
	}

	// Expression statement (including assignments)
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	return NewExprStmt(expr, loc)
}

// parseStatementBlock parses a block of statements (inside {})
func (p *Parser) parseStatementBlock() []StmtNode {
	statements := []StmtNode{}

	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACE) {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		} else {
			// Error recovery: if parseStatement() returns nil, advance past the
			// problematic token to avoid infinite loops
			p.advance()
		}

		p.skipNewlines()

		// Statements can be separated by newlines or semicolons
		// Newlines are handled by skipNewlines()
	}

	return statements
}

// parseLetStatement parses a let statement (already consumed 'let')
func (p *Parser) parseLetStatement() StmtNode {
	loc := TokenToLocation(p.previous())

	if !p.check(lexer.TOKEN_IDENTIFIER) {
		p.addError(ParseError{
			Message:  "Expected variable name after 'let'",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	name := p.advance().Lexeme

	if _, ok := p.consume(lexer.TOKEN_EQUAL, "Expected '=' after variable name"); !ok {
		return nil
	}

	value := p.parseExpression()
	if value == nil {
		p.addError(ParseError{
			Message:  "Expected expression after '='",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	return NewLetStmt(name, value, loc)
}

// parseReturnStatement parses a return statement (already consumed 'return')
func (p *Parser) parseReturnStatement() StmtNode {
	loc := TokenToLocation(p.previous())

	// Check if there's a value to return
	// If next token is newline or }, return without value
	if p.check(lexer.TOKEN_NEWLINE) || p.check(lexer.TOKEN_RBRACE) || p.isAtEnd() {
		return NewReturnStmt(nil, loc)
	}

	value := p.parseExpression()
	if value == nil {
		// Error already reported by parseExpression
		return nil
	}

	return NewReturnStmt(value, loc)
}

// parseIfStatement parses an if statement
func (p *Parser) parseIfStatement() StmtNode {
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

	return NewIfStmt(condition, thenBody, elsifBranches, elseBody, loc)
}

// parseUnlessStatement parses an unless statement
func (p *Parser) parseUnlessStatement() StmtNode {
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

	return NewUnlessStmt(condition, body, loc)
}
