package parser

import (
	"fmt"
	"github.com/conduit-lang/conduit/compiler/lexer"
)

// parseType parses a type definition
// Types can be:
// - Primitives: string, int, bool, etc.
// - Arrays: array<T>
// - Hashes: hash<K,V>
// - Enums: enum ["value1", "value2"]
// - Inline structs: { field: type, ... }
// - Resource references: User, Post, etc.
func (p *Parser) parseType() (TypeNode, bool) {
	typeStart := p.peek()

	// Check for primitive types
	if p.isPrimitiveType(p.peek().Type) {
		typeName := p.advance().Lexeme
		return NewPrimitiveType(typeName, TokenToLocation(typeStart)), true
	}

	// Check for array type
	if p.match(lexer.TOKEN_ARRAY) {
		return p.parseArrayType(typeStart)
	}

	// Check for hash type
	if p.match(lexer.TOKEN_HASH) {
		return p.parseHashType(typeStart)
	}

	// Check for enum type
	if p.match(lexer.TOKEN_ENUM) {
		return p.parseEnumType(typeStart)
	}

	// Check for inline struct
	if p.check(lexer.TOKEN_LBRACE) {
		return p.parseStructType(typeStart)
	}

	// Check for resource reference (identifier starting with capital)
	if p.check(lexer.TOKEN_IDENTIFIER) {
		name := p.advance().Lexeme
		// Resource names should start with capital letter
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			return NewResourceType(name, TokenToLocation(typeStart)), true
		}
		p.addError(ParseError{
			Message:  fmt.Sprintf("Invalid type: %s. Resource names must start with a capital letter.", name),
			Location: TokenToLocation(typeStart),
		})
		return TypeNode{}, false
	}

	p.addError(ParseError{
		Message:  fmt.Sprintf("Expected type, got: %s", p.peek().Lexeme),
		Location: TokenToLocation(p.peek()),
	})
	return TypeNode{}, false
}

// parseArrayType parses an array type: array<ElementType>
func (p *Parser) parseArrayType(startToken lexer.Token) (TypeNode, bool) {
	// Consume '<'
	if _, ok := p.consume(lexer.TOKEN_LESS, "Expected '<' after 'array'"); !ok {
		return TypeNode{}, false
	}

	// Parse element type
	elementType, ok := p.parseType()
	if !ok {
		return TypeNode{}, false
	}

	// Consume '>'
	if _, ok := p.consume(lexer.TOKEN_GREATER, "Expected '>' after array element type"); !ok {
		return TypeNode{}, false
	}

	return NewArrayType(elementType, TokenToLocation(startToken)), true
}

// parseHashType parses a hash type: hash<KeyType, ValueType>
func (p *Parser) parseHashType(startToken lexer.Token) (TypeNode, bool) {
	// Consume '<'
	if _, ok := p.consume(lexer.TOKEN_LESS, "Expected '<' after 'hash'"); !ok {
		return TypeNode{}, false
	}

	// Parse key type
	keyType, ok := p.parseType()
	if !ok {
		return TypeNode{}, false
	}

	// Consume ','
	if _, ok := p.consume(lexer.TOKEN_COMMA, "Expected ',' between hash key and value types"); !ok {
		return TypeNode{}, false
	}

	// Parse value type
	valueType, ok := p.parseType()
	if !ok {
		return TypeNode{}, false
	}

	// Consume '>'
	if _, ok := p.consume(lexer.TOKEN_GREATER, "Expected '>' after hash value type"); !ok {
		return TypeNode{}, false
	}

	return NewHashType(keyType, valueType, TokenToLocation(startToken)), true
}

// parseEnumType parses an enum type: enum ["value1", "value2", ...]
func (p *Parser) parseEnumType(startToken lexer.Token) (TypeNode, bool) {
	// Consume '['
	if _, ok := p.consume(lexer.TOKEN_LBRACKET, "Expected '[' after 'enum'"); !ok {
		return TypeNode{}, false
	}

	values := []string{}

	// Parse enum values
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		// Parse string literal
		value, ok := p.parseStringLiteral()
		if !ok {
			return TypeNode{}, false
		}
		values = append(values, value)

		// Optional comma
		if !p.check(lexer.TOKEN_RBRACKET) {
			if _, ok := p.consume(lexer.TOKEN_COMMA, "Expected ',' or ']' in enum definition"); !ok {
				return TypeNode{}, false
			}
		}
	}

	// Consume ']'
	if _, ok := p.consume(lexer.TOKEN_RBRACKET, "Expected ']' after enum values"); !ok {
		return TypeNode{}, false
	}

	if len(values) == 0 {
		p.addError(ParseError{
			Message:  "Enum must have at least one value",
			Location: TokenToLocation(startToken),
		})
		return TypeNode{}, false
	}

	return NewEnumType(values, TokenToLocation(startToken)), true
}

// parseStructType parses an inline struct type: { field: type, ... }
func (p *Parser) parseStructType(startToken lexer.Token) (TypeNode, bool) {
	// Consume '{'
	if _, ok := p.consume(lexer.TOKEN_LBRACE, "Expected '{'"); !ok {
		return TypeNode{}, false
	}

	p.skipNewlines()

	fields := []*FieldNode{}

	// Parse struct fields
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		p.skipNewlines()

		if p.check(lexer.TOKEN_RBRACE) {
			break
		}

		field := p.parseStructField()
		if field != nil {
			fields = append(fields, field)
		} else {
			// Skip to next field or closing brace
			p.skipUntilNewlineOrBrace()
		}

		p.skipNewlines()
	}

	// Consume '}'
	if _, ok := p.consume(lexer.TOKEN_RBRACE, "Expected '}' after struct fields"); !ok {
		return TypeNode{}, false
	}

	if len(fields) == 0 {
		p.addError(ParseError{
			Message:  "Inline struct must have at least one field",
			Location: TokenToLocation(startToken),
		})
		return TypeNode{}, false
	}

	return NewStructType(fields, TokenToLocation(startToken)), true
}

// parseStructField parses a field inside an inline struct
func (p *Parser) parseStructField() *FieldNode {
	fieldStart := p.peek()

	// Get field name
	name, ok := p.parseIdentifier()
	if !ok {
		return nil
	}

	// Consume ':'
	if _, ok := p.consume(lexer.TOKEN_COLON, "Expected ':' after field name"); !ok {
		return nil
	}

	// Parse type
	fieldType, ok := p.parseType()
	if !ok {
		return nil
	}

	// Parse nullability
	nullable := false
	if p.match(lexer.TOKEN_QUESTION) {
		nullable = true
	} else if p.match(lexer.TOKEN_BANG) {
		nullable = false
	} else {
		p.addError(ParseError{
			Message:  fmt.Sprintf("Missing nullability indicator (! or ?) for field '%s'", name),
			Location: TokenToLocation(p.previous()),
		})
		nullable = false
	}

	field := NewFieldNode(name, fieldType, nullable, TokenToLocation(fieldStart))

	// Parse optional constraints
	for p.check(lexer.TOKEN_AT) {
		constraint := p.parseConstraint()
		if constraint != nil {
			field.AddConstraint(constraint)
		}
	}

	// Optional comma
	p.match(lexer.TOKEN_COMMA)

	return field
}
