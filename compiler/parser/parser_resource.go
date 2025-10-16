package parser

import (
	"fmt"
	"github.com/conduit-lang/conduit/compiler/lexer"
)

// parseResource parses a resource definition
// resource User {
//   fields...
//   relationships...
// }
func (p *Parser) parseResource() *ResourceNode {
	p.skipNewlines()

	// Get documentation before 'resource' keyword
	doc := p.parseDocumentation()

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

	// Consume opening brace
	if _, ok := p.consume(lexer.TOKEN_LBRACE, "Expected '{' after resource name"); !ok {
		p.synchronize()
		return nil
	}

	p.skipNewlines()

	// Parse resource body
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACE) {
			break
		}

		// Check for annotations starting with @
		if p.check(lexer.TOKEN_AT) {
			// Future: parse hooks, constraints, etc.
			// For MVP, skip these
			p.advance() // consume @
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
			p.skipUntilNewlineOrBrace()
		}

		p.skipNewlines()
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
			resource.AddRelationship(rel)
		}
		return nil
	}

	// Create field node
	field := NewFieldNode(name, fieldType, nullable, TokenToLocation(fieldStart))

	// Parse field constraints
	for p.check(lexer.TOKEN_AT) {
		constraint := p.parseConstraint()
		if constraint != nil {
			field.AddConstraint(constraint)
		}
	}

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
