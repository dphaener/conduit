package parser

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/compiler/lexer"
)

// parseResource parses a resource definition
// resource User {
//   fields...
//   relationships...
// }
func (p *Parser) parseResource() *ResourceNode {
	p.skipNewlines()

	// Collect leading comments before resource
	leadingComment := p.collectLeadingComments()

	// Note: Documentation (///) is not currently used in Conduit
	// All comments use # syntax and are captured as LeadingComment
	doc := ""

	// Consume 'resource' keyword
	resourceToken, ok := p.consume(lexer.TOKEN_RESOURCE, "Expected 'resource' keyword")
	if !ok {
		return nil
	}

	// Get resource name
	name, ok := p.parseIdentifier()
	if !ok {
		p.synchronize()
		return nil
	}

	// Create resource node
	resource := NewResourceNode(name, doc, TokenToLocation(resourceToken))
	resource.LeadingComment = leadingComment

	// Consume opening brace
	if _, ok := p.consume(lexer.TOKEN_LBRACE, "Expected '{' after resource name"); !ok {
		p.synchronize()
		return nil
	}

	p.skipNewlines()

	// Parse resource body
	maxLoopIter := 10000
	loopIter := 0
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() && loopIter < maxLoopIter {
		loopIter++
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACE) {
			break
		}

		// Skip comments - they'll be collected by the next field/relationship/hook
		if p.check(lexer.TOKEN_COMMENT) {
			p.advance()
			continue
		}

		// Check for resource-level annotations starting with @
		// Examples: @before create { }, @constraint name { }, @after delete { }
		if p.check(lexer.TOKEN_AT) {
			// Look ahead to determine what type of annotation this is
			nextPos := p.current + 1
			if nextPos < len(p.tokens) {
				nextTok := p.tokens[nextPos]

				// Parse hooks (@before, @after)
				if nextTok.Type == lexer.TOKEN_BEFORE || nextTok.Type == lexer.TOKEN_AFTER {
					hook := p.parseHook()
					if hook != nil {
						resource.AddHook(hook)
					}
					continue
				}

				// Parse custom constraints (@constraint)
				if nextTok.Type == lexer.TOKEN_CONSTRAINT {
					constraint := p.parseCustomConstraint()
					if constraint != nil {
						resource.AddCustomConstraint(constraint)
					}
					continue
				}
			}

			// Unknown annotation - skip it
			p.addError(ParseError{
				Message:  fmt.Sprintf("Unknown resource-level annotation: %s", p.peek().Lexeme),
				Location: TokenToLocation(p.peek()),
			})
			p.advance() // Skip @
			p.skipUntilNewlineOrBrace()
			continue
		}

		// Otherwise, parse field or relationship
		// Field names can be identifiers OR type keywords (like "email", "string", etc.)
		if p.check(lexer.TOKEN_IDENTIFIER) || p.canBeFieldName() {
			field := p.parseFieldOrRelationship(resource)
			if field != nil {
				// Field successfully parsed - it's already added to resource in parseFieldOrRelationship
			}
		} else {
			p.addError(ParseError{
				Message:  fmt.Sprintf("Unexpected token in resource body: %s", p.peek().Lexeme),
				Location: TokenToLocation(p.peek()),
			})
			// Skip the unexpected token and any following block
			p.advance()
			// If followed by a block, skip it
			if p.check(lexer.TOKEN_LBRACE) {
				p.skipBalancedBlock()
			}
		}

		p.skipNewlines()
	}

	if loopIter >= maxLoopIter {
		p.addError(ParseError{
			Message:  "Infinite loop detected in parseResource body",
			Location: TokenToLocation(p.peek()),
		})
	}

	// Consume closing brace
	if _, ok := p.consume(lexer.TOKEN_RBRACE, "Expected '}' after resource body"); !ok {
		return resource // Return partial AST
	}

	return resource
}

// parseFieldOrRelationship parses a field or relationship definition
// This determines if it's a field or relationship based on the type
func (p *Parser) parseFieldOrRelationship(resource *ResourceNode) *FieldNode {
	// Collect any leading comments before the field
	leadingComment := p.collectLeadingComments()

	fieldStart := p.peek()

	// Get field name
	name, ok := p.parseIdentifier()
	if !ok {
		p.skipUntilNewlineOrBrace()
		return nil
	}

	// Consume colon
	if _, ok := p.consume(lexer.TOKEN_COLON, "Expected ':' after field name"); !ok {
		p.skipUntilNewlineOrBrace()
		return nil
	}

	// Parse type
	fieldType, ok := p.parseType()
	if !ok {
		p.skipUntilNewlineOrBrace()
		return nil
	}

	// Parse nullability (! or ?)
	// Note: The lexer may include ? in the identifier for predicates (e.g., "Category?")
	// If the type is a resource reference and the lexer included ?, we need to handle it
	nullable := false
	hasNullabilityMarker := false

	// Check if the type already has nullability from lexer (resource names ending with ?)
	if fieldType.IsResource() && len(fieldType.Name) > 0 && fieldType.Name[len(fieldType.Name)-1] == '?' {
		// Strip the ? from the type name
		fieldType.Name = fieldType.Name[:len(fieldType.Name)-1]
		nullable = true
		hasNullabilityMarker = true
	}

	// Otherwise check for separate nullability token
	if !hasNullabilityMarker {
		if p.match(lexer.TOKEN_QUESTION) {
			nullable = true
			hasNullabilityMarker = true
		} else if p.match(lexer.TOKEN_BANG) {
			nullable = false
			hasNullabilityMarker = true
		}
	}

	if !hasNullabilityMarker {
		p.addError(ParseError{
			Message:  fmt.Sprintf("Missing nullability indicator (! or ?) for field '%s'", name),
			Location: TokenToLocation(p.previous()),
		})
		// Continue parsing with default (required)
		nullable = false
	}

	// Check if this is a relationship (resource reference)
	if fieldType.IsResource() {
		rel := p.parseRelationshipMetadata(name, fieldType.Name, nullable, fieldStart)
		if rel != nil {
			rel.LeadingComment = leadingComment
			// Check for trailing comment
			rel.TrailingComment = p.consumeTrailingComment()
			resource.AddRelationship(rel)
		}
		return nil
	}

	// Create field node
	field := NewFieldNode(name, fieldType, nullable, TokenToLocation(fieldStart))
	field.LeadingComment = leadingComment

	// Parse field constraints
	for p.check(lexer.TOKEN_AT) {
		// Check if this is actually a resource-level annotation (not a field constraint)
		// Look ahead to see if the next token after @ is a resource-level keyword
		nextPos := p.current + 1
		if nextPos < len(p.tokens) {
			nextTok := p.tokens[nextPos]
			// If it's a resource-level annotation keyword, stop parsing field constraints
			if nextTok.Type == lexer.TOKEN_BEFORE || nextTok.Type == lexer.TOKEN_AFTER ||
				nextTok.Type == lexer.TOKEN_CONSTRAINT {
				break
			}
		}

		constraint := p.parseConstraint()
		if constraint != nil {
			field.AddConstraint(constraint)
		}
	}

	// Check for trailing comment before newline
	field.TrailingComment = p.consumeTrailingComment()

	// Add field to resource
	resource.AddField(field)

	// Expect newline or closing brace
	p.expectNewlineOrEOF()

	return field
}

// parseRelationshipMetadata parses optional relationship metadata
// author: User! { foreign_key: "author_id", on_delete: cascade }
func (p *Parser) parseRelationshipMetadata(name, targetType string, nullable bool, startToken lexer.Token) *RelationshipNode {
	rel := NewRelationshipNode(name, targetType, nullable, TokenToLocation(startToken))

	// Skip newlines before checking for metadata block
	p.skipNewlines()

	// Check for optional metadata block
	if p.match(lexer.TOKEN_LBRACE) {
		p.skipNewlines()

		// Parse metadata key-value pairs
		for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
			p.skipNewlines()

			if p.check(lexer.TOKEN_RBRACE) {
				break
			}

			// Parse metadata key
			// Metadata keys can be identifiers or specific keywords
			var key string
			if p.check(lexer.TOKEN_IDENTIFIER) {
				key = p.advance().Lexeme
			} else if p.check(lexer.TOKEN_FOREIGN_KEY) {
				key = "foreign_key"
				p.advance()
			} else if p.check(lexer.TOKEN_ON_DELETE) {
				key = "on_delete"
				p.advance()
			} else if p.check(lexer.TOKEN_ON_UPDATE) {
				key = "on_update"
				p.advance()
			} else {
				p.addError(ParseError{
					Message:  "Expected metadata key",
					Location: TokenToLocation(p.peek()),
				})
				p.skipUntilNewlineOrBrace()
				continue
			}

			// Consume colon
			if _, ok := p.consume(lexer.TOKEN_COLON, "Expected ':' after metadata key"); !ok {
				p.skipUntilNewlineOrBrace()
				continue
			}

			// Parse metadata value
			switch key {
			case "foreign_key":
				if value, ok := p.parseStringLiteral(); ok {
					rel.ForeignKey = value
				}
			case "on_delete":
				if p.check(lexer.TOKEN_IDENTIFIER) {
					value := p.advance().Lexeme
					rel.OnDelete = value
				} else if p.check(lexer.TOKEN_RESTRICT) {
					rel.OnDelete = "restrict"
					p.advance()
				} else if p.check(lexer.TOKEN_CASCADE) {
					rel.OnDelete = "cascade"
					p.advance()
				} else if p.check(lexer.TOKEN_SET_NULL) {
					rel.OnDelete = "set_null"
					p.advance()
				} else if p.check(lexer.TOKEN_NO_ACTION) {
					rel.OnDelete = "no_action"
					p.advance()
				}
			case "on_update":
				if p.check(lexer.TOKEN_IDENTIFIER) {
					value := p.advance().Lexeme
					rel.OnUpdate = value
				} else if p.check(lexer.TOKEN_CASCADE) {
					rel.OnUpdate = "cascade"
					p.advance()
				} else if p.check(lexer.TOKEN_RESTRICT) {
					rel.OnUpdate = "restrict"
					p.advance()
				}
			default:
				p.addError(ParseError{
					Message:  fmt.Sprintf("Unknown relationship metadata key: %s", key),
					Location: TokenToLocation(p.previous()),
				})
			}

			// Optional comma
			p.match(lexer.TOKEN_COMMA)
			p.skipNewlines()
		}

		// Consume closing brace
		p.consume(lexer.TOKEN_RBRACE, "Expected '}' after relationship metadata")
	}

	// Expect newline or closing brace
	p.expectNewlineOrEOF()

	return rel
}

// parseConstraint parses a field constraint annotation
// @min(5) @max(200) @unique
func (p *Parser) parseConstraint() *ConstraintNode {
	constraintStart := p.peek()

	// Consume @
	if !p.match(lexer.TOKEN_AT) {
		return nil
	}

	// Get constraint name
	var constraintName string
	var args []interface{}

	// Check for known constraint keywords
	if p.match(lexer.TOKEN_MIN) {
		constraintName = "min"
	} else if p.match(lexer.TOKEN_MAX) {
		constraintName = "max"
	} else if p.match(lexer.TOKEN_UNIQUE) {
		constraintName = "unique"
	} else if p.match(lexer.TOKEN_PRIMARY) {
		constraintName = "primary"
	} else if p.match(lexer.TOKEN_AUTO) {
		constraintName = "auto"
	} else if p.match(lexer.TOKEN_AUTO_UPDATE) {
		constraintName = "auto_update"
	} else if p.match(lexer.TOKEN_DEFAULT) {
		constraintName = "default"
	} else if p.match(lexer.TOKEN_PATTERN) {
		constraintName = "pattern"
	} else if p.match(lexer.TOKEN_REQUIRED) {
		constraintName = "required"
	} else if p.check(lexer.TOKEN_IDENTIFIER) {
		constraintName = p.advance().Lexeme
	} else {
		p.addError(ParseError{
			Message:  "Expected constraint name after '@'",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Check for arguments
	if p.match(lexer.TOKEN_LPAREN) {
		// Parse constraint arguments
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			// Parse argument value
			if p.check(lexer.TOKEN_INT_LITERAL) {
				if val, ok := p.parseIntLiteral(); ok {
					args = append(args, val)
				}
			} else if p.check(lexer.TOKEN_FLOAT_LITERAL) {
				if val, ok := p.parseFloatLiteral(); ok {
					args = append(args, val)
				}
			} else if p.check(lexer.TOKEN_STRING_LITERAL) {
				if val, ok := p.parseStringLiteral(); ok {
					args = append(args, val)
				}
			} else if p.check(lexer.TOKEN_TRUE) {
				args = append(args, true)
				p.advance()
			} else if p.check(lexer.TOKEN_FALSE) {
				args = append(args, false)
				p.advance()
			} else {
				p.addError(ParseError{
					Message:  "Invalid constraint argument",
					Location: TokenToLocation(p.peek()),
				})
				p.advance()
			}

			// Optional comma
			if !p.check(lexer.TOKEN_RPAREN) {
				p.match(lexer.TOKEN_COMMA)
			}
		}

		p.consume(lexer.TOKEN_RPAREN, "Expected ')' after constraint arguments")
	}

	return NewConstraintNode(constraintName, args, TokenToLocation(constraintStart))
}

// skipUntilNewlineOrBrace skips tokens until newline or brace
func (p *Parser) skipUntilNewlineOrBrace() {
	for !p.isAtEnd() && !p.check(lexer.TOKEN_NEWLINE) &&
		!p.check(lexer.TOKEN_RBRACE) && !p.check(lexer.TOKEN_LBRACE) {
		p.advance()
	}
	p.skipNewlines()
}

// parseHook parses a resource-level hook (@before or @after)
// @before create { body content }
// @after update { body content }
func (p *Parser) parseHook() *HookNode {
	hookStart := p.peek()

	// Consume @ token (already checked by caller)
	p.advance()

	// Get hook type (before or after)
	var hookType string
	if p.match(lexer.TOKEN_BEFORE) {
		hookType = "before"
	} else if p.match(lexer.TOKEN_AFTER) {
		hookType = "after"
	} else {
		p.addError(ParseError{
			Message:  "Expected 'before' or 'after' after '@'",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Get trigger (e.g., "create", "update", "delete")
	// The trigger can be an identifier or one of the operation keywords
	trigger := ""
	if p.check(lexer.TOKEN_IDENTIFIER) {
		trigger = p.advance().Lexeme
	} else if p.check(lexer.TOKEN_CREATE) {
		trigger = "create"
		p.advance()
	} else if p.check(lexer.TOKEN_UPDATE) {
		trigger = "update"
		p.advance()
	} else if p.check(lexer.TOKEN_DELETE) {
		trigger = "delete"
		p.advance()
	} else if p.check(lexer.TOKEN_SAVE) {
		trigger = "save"
		p.advance()
	} else {
		p.addError(ParseError{
			Message:  fmt.Sprintf("Expected trigger name after '@%s'", hookType),
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Skip newlines before block
	p.skipNewlines()

	// Consume opening brace and save its position for body extraction
	openBracePos := -1
	if p.match(lexer.TOKEN_LBRACE) {
		// Save the end position of the opening brace for offset-based extraction
		if p.current > 0 {
			openBracePos = p.tokens[p.current-1].End
		}
	} else {
		p.addError(ParseError{
			Message:  fmt.Sprintf("Expected '{' after '@%s %s'", hookType, trigger),
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Capture body as raw string
	body := p.captureBlockBodyFrom(openBracePos)

	// Consume closing brace (already consumed by captureBlockBody)

	p.skipNewlines()

	return NewHookNode(hookType, trigger, body, TokenToLocation(hookStart))
}

// parseCustomConstraint parses a resource-level custom constraint
// @constraint name { body content }
func (p *Parser) parseCustomConstraint() *CustomConstraintNode {
	constraintStart := p.peek()

	// Consume @ token (already checked by caller)
	p.advance()

	// Consume 'constraint' keyword
	if !p.match(lexer.TOKEN_CONSTRAINT) {
		p.addError(ParseError{
			Message:  "Expected 'constraint' after '@'",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Get constraint name
	name := ""
	if p.check(lexer.TOKEN_IDENTIFIER) {
		name = p.advance().Lexeme
	} else {
		p.addError(ParseError{
			Message:  "Expected constraint name after '@constraint'",
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Skip newlines before block
	p.skipNewlines()

	// Consume opening brace and save its position for body extraction
	openBracePos := -1
	if p.match(lexer.TOKEN_LBRACE) {
		// Save the end position of the opening brace for offset-based extraction
		if p.current > 0 {
			openBracePos = p.tokens[p.current-1].End
		}
	} else {
		p.addError(ParseError{
			Message:  fmt.Sprintf("Expected '{' after '@constraint %s'", name),
			Location: TokenToLocation(p.peek()),
		})
		return nil
	}

	// Capture body as raw string
	body := p.captureBlockBodyFrom(openBracePos)

	// Consume closing brace (already consumed by captureBlockBody)

	p.skipNewlines()

	return NewCustomConstraintNode(name, body, TokenToLocation(constraintStart))
}

// captureBlockBody captures the raw content of a block as a string
// Assumes opening brace has been consumed
// Consumes tokens up to and including the closing brace
func (p *Parser) captureBlockBody() string {
	return p.captureBlockBodyFrom(-1)
}

// captureBlockBodyFrom captures block body starting from the given source position
// If startPos is -1, uses the current token's start position
// Consumes tokens up to and including the closing brace
func (p *Parser) captureBlockBodyFrom(startPos int) string {
	// If we have source text available, use offset-based extraction to preserve formatting
	if p.source != "" {
		return p.captureBlockBodyWithOffsets(startPos)
	}

	// Fallback to token-based reconstruction if source is not available
	return p.captureBlockBodyFromTokens()
}

// captureBlockBodyWithOffsets extracts the block body directly from source text
// preserving all original formatting including indentation
// startPos is the position in source to start extracting from (typically end of opening brace)
// If startPos is -1, uses the current token's start position
func (p *Parser) captureBlockBodyWithOffsets(startPos int) string {
	// Use provided start position or default to current token's start
	if startPos == -1 {
		startPos = p.peek().Start
	}

	depth := 1
	maxIter := 10000
	iter := 0
	var endTok lexer.Token

	for !p.isAtEnd() && depth > 0 && iter < maxIter {
		iter++
		tok := p.peek()

		if tok.Type == lexer.TOKEN_LBRACE {
			depth++
			p.advance()
		} else if tok.Type == lexer.TOKEN_RBRACE {
			depth--
			if depth == 0 {
				endTok = tok
			}
			p.advance()
		} else {
			endTok = tok
			p.advance()
		}
	}

	if iter >= maxIter {
		p.addError(ParseError{
			Message:  "Infinite loop detected while capturing block body",
			Location: TokenToLocation(p.peek()),
		})
		return ""
	}

	// Extract the substring from source using offsets
	// Extract from startPos to the start of the closing brace to preserve formatting
	if startPos < endTok.Start && endTok.Start <= len(p.source) {
		body := p.source[startPos:endTok.Start]
		// Trim trailing whitespace and single leading newline (from opening brace line)
		body = strings.TrimRight(body, " \t\n\r")
		// Remove single leading newline if present (common after opening brace)
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		}
		return body
	}

	return ""
}

// captureBlockBodyFromTokens reconstructs body from tokens (fallback for when source is unavailable)
func (p *Parser) captureBlockBodyFromTokens() string {
	var bodyTokens []lexer.Token
	depth := 1
	maxIter := 10000
	iter := 0

	for !p.isAtEnd() && depth > 0 && iter < maxIter {
		iter++
		tok := p.peek()

		if tok.Type == lexer.TOKEN_LBRACE {
			depth++
			bodyTokens = append(bodyTokens, tok)
			p.advance()
		} else if tok.Type == lexer.TOKEN_RBRACE {
			depth--
			if depth > 0 {
				bodyTokens = append(bodyTokens, tok)
			}
			p.advance()
		} else {
			bodyTokens = append(bodyTokens, tok)
			p.advance()
		}
	}

	if iter >= maxIter {
		p.addError(ParseError{
			Message:  "Infinite loop detected while capturing block body",
			Location: TokenToLocation(p.peek()),
		})
		return ""
	}

	// Reconstruct body from tokens
	var body strings.Builder
	for i, tok := range bodyTokens {
		// Add the token lexeme
		body.WriteString(tok.Lexeme)

		// Add spacing between tokens where appropriate
		if i < len(bodyTokens)-1 {
			nextTok := bodyTokens[i+1]

			// Add newline if this token is a newline
			if tok.Type == lexer.TOKEN_NEWLINE {
				// Newline already in lexeme
				continue
			}

			// Add space between most tokens (but not before/after certain punctuation)
			if !isNoSpaceAfter(tok.Type) && !isNoSpaceBefore(nextTok.Type) {
				body.WriteString(" ")
			}
		}
	}

	return strings.TrimSpace(body.String())
}

// isNoSpaceAfter returns true if no space should be added after this token type
func isNoSpaceAfter(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TOKEN_LPAREN ||
		tokenType == lexer.TOKEN_LBRACKET ||
		tokenType == lexer.TOKEN_DOT ||
		tokenType == lexer.TOKEN_NEWLINE
}

// isNoSpaceBefore returns true if no space should be added before this token type
func isNoSpaceBefore(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TOKEN_RPAREN ||
		tokenType == lexer.TOKEN_RBRACKET ||
		tokenType == lexer.TOKEN_COMMA ||
		tokenType == lexer.TOKEN_DOT ||
		tokenType == lexer.TOKEN_NEWLINE
}

// skipBalancedBlock skips a balanced block delimited by braces { ... }
func (p *Parser) skipBalancedBlock() {
	if !p.match(lexer.TOKEN_LBRACE) {
		return
	}

	depth := 1
	maxIter := 10000 // Safety limit to prevent infinite loops
	iter := 0
	for !p.isAtEnd() && depth > 0 && iter < maxIter {
		iter++
		tok := p.peek()
		if tok.Type == lexer.TOKEN_LBRACE {
			depth++
			p.advance()
		} else if tok.Type == lexer.TOKEN_RBRACE {
			depth--
			p.advance()
		} else {
			p.advance()
		}
	}

	if iter >= maxIter {
		p.addError(ParseError{
			Message:  "Infinite loop detected while skipping balanced block",
			Location: TokenToLocation(p.peek()),
		})
	}
}
