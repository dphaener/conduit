package llm

import (
	"fmt"
	"regexp"
	"strings"
)

// ParsedCode represents code extracted from an LLM response.
type ParsedCode struct {
	// Raw is the raw response from the LLM.
	Raw string

	// CodeBlocks contains all code blocks found in the response.
	CodeBlocks []CodeBlock

	// MiddlewareDeclarations contains all @on declarations found.
	MiddlewareDeclarations []MiddlewareDeclaration
}

// CodeBlock represents a single code block from markdown.
type CodeBlock struct {
	// Language is the language identifier (e.g., "conduit", "go").
	Language string

	// Content is the code content.
	Content string
}

// MiddlewareDeclaration represents a parsed @on middleware declaration.
type MiddlewareDeclaration struct {
	// Operation is the operation name (e.g., "create", "update").
	Operation string

	// Middleware is the list of middleware in order.
	Middleware []string

	// Raw is the raw declaration string.
	Raw string
}

// Parser extracts code from LLM responses.
type Parser struct{}

// NewParser creates a new parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse extracts code from an LLM response.
// It handles markdown-formatted responses with code blocks and plain text responses.
func (p *Parser) Parse(response string) (*ParsedCode, error) {
	parsed := &ParsedCode{
		Raw:                    response,
		CodeBlocks:             []CodeBlock{},
		MiddlewareDeclarations: []MiddlewareDeclaration{},
	}

	// Extract code blocks
	parsed.CodeBlocks = p.extractCodeBlocks(response)

	// Extract middleware declarations from code blocks and raw response
	parsed.MiddlewareDeclarations = p.extractMiddlewareDeclarations(response)

	// Also check inside code blocks
	for _, block := range parsed.CodeBlocks {
		decls := p.extractMiddlewareDeclarations(block.Content)
		parsed.MiddlewareDeclarations = append(parsed.MiddlewareDeclarations, decls...)
	}

	return parsed, nil
}

// extractCodeBlocks extracts all markdown code blocks from the response.
// Supports both ```language and ``` formats.
func (p *Parser) extractCodeBlocks(text string) []CodeBlock {
	var blocks []CodeBlock

	// Regex to match code blocks: ```language\ncode\n```
	re := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Content:  strings.TrimSpace(match[2]),
			})
		}
	}

	return blocks
}

// extractMiddlewareDeclarations extracts @on declarations from text.
// Handles formats like:
//   - @on create: [auth]
//   - @on update: [auth, rate_limit(10/hour)]
//   - @on list: [cache(300)]
func (p *Parser) extractMiddlewareDeclarations(text string) []MiddlewareDeclaration {
	var declarations []MiddlewareDeclaration

	// Regex to match @on declarations
	// Pattern: @on <operation>: [<middleware_list>]
	re := regexp.MustCompile(`@on\s+(\w+)\s*:\s*\[(.*?)\]`)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			operation := match[1]
			middlewareStr := match[2]

			// Parse middleware list
			middleware := p.parseMiddlewareList(middlewareStr)

			declarations = append(declarations, MiddlewareDeclaration{
				Operation:  operation,
				Middleware: middleware,
				Raw:        match[0],
			})
		}
	}

	return declarations
}

// parseMiddlewareList parses a comma-separated list of middleware.
// Handles both simple names and parameterized middleware:
//   - "auth" → ["auth"]
//   - "auth, rate_limit(10/hour)" → ["auth", "rate_limit(10/hour)"]
//   - "cache(300)" → ["cache(300)"]
func (p *Parser) parseMiddlewareList(str string) []string {
	if str == "" {
		return []string{}
	}

	// Split by comma, handling nested parentheses
	var middleware []string
	var current strings.Builder
	depth := 0

	for _, ch := range str {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				// End of middleware item
				item := strings.TrimSpace(current.String())
				if item != "" {
					middleware = append(middleware, item)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add last item
	item := strings.TrimSpace(current.String())
	if item != "" {
		middleware = append(middleware, item)
	}

	return middleware
}

// ExtractMiddleware extracts the first middleware declaration from a response.
// This is a convenience method for simple test cases.
func (p *Parser) ExtractMiddleware(response string) ([]string, error) {
	parsed, err := p.Parse(response)
	if err != nil {
		return nil, err
	}

	if len(parsed.MiddlewareDeclarations) == 0 {
		return nil, fmt.Errorf("no middleware declarations found in response")
	}

	return parsed.MiddlewareDeclarations[0].Middleware, nil
}

// ExtractMiddlewareDeclaration extracts the first full middleware declaration.
// Returns the raw @on declaration string.
func (p *Parser) ExtractMiddlewareDeclaration(response string) (string, error) {
	parsed, err := p.Parse(response)
	if err != nil {
		return "", err
	}

	if len(parsed.MiddlewareDeclarations) == 0 {
		return "", fmt.Errorf("no middleware declarations found in response")
	}

	return parsed.MiddlewareDeclarations[0].Raw, nil
}

// HasMiddleware checks if the response contains a middleware declaration
// with the given operation.
func (p *Parser) HasMiddleware(response string, operation string) bool {
	parsed, err := p.Parse(response)
	if err != nil {
		return false
	}

	for _, decl := range parsed.MiddlewareDeclarations {
		if decl.Operation == operation {
			return true
		}
	}

	return false
}

// NormalizeMiddlewareDeclaration normalizes a middleware declaration for comparison.
// It removes extra whitespace and standardizes formatting.
func NormalizeMiddlewareDeclaration(decl string) string {
	// Normalize whitespace
	decl = strings.TrimSpace(decl)
	decl = regexp.MustCompile(`\s+`).ReplaceAllString(decl, " ")

	// Normalize spacing around punctuation
	decl = strings.Replace(decl, " : ", ": ", -1)
	decl = strings.Replace(decl, " :", ":", -1)
	decl = strings.Replace(decl, ": ", ": ", -1)

	// Normalize spacing in brackets
	decl = strings.Replace(decl, "[ ", "[", -1)
	decl = strings.Replace(decl, " ]", "]", -1)

	return decl
}
