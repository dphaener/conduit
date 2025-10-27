package llm

import (
	"testing"
)

// TestParser_Parse_CodeBlocks tests extracting code blocks from markdown.
func TestParser_Parse_CodeBlocks(t *testing.T) {
	parser := NewParser()

	response := `Here is the code:

` + "```conduit" + `
@on create: [auth]
` + "```" + `

That's the middleware declaration.`

	parsed, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.CodeBlocks) != 1 {
		t.Fatalf("Expected 1 code block, got %d", len(parsed.CodeBlocks))
	}

	if parsed.CodeBlocks[0].Language != "conduit" {
		t.Errorf("Expected language 'conduit', got '%s'", parsed.CodeBlocks[0].Language)
	}

	if parsed.CodeBlocks[0].Content != "@on create: [auth]" {
		t.Errorf("Expected '@on create: [auth]', got '%s'", parsed.CodeBlocks[0].Content)
	}
}

// TestParser_Parse_MultipleCodeBlocks tests multiple code blocks.
func TestParser_Parse_MultipleCodeBlocks(t *testing.T) {
	parser := NewParser()

	response := `
` + "```conduit" + `
@on create: [auth]
` + "```" + `

And also:

` + "```conduit" + `
@on update: [auth]
` + "```"

	parsed, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.CodeBlocks) != 2 {
		t.Fatalf("Expected 2 code blocks, got %d", len(parsed.CodeBlocks))
	}
}

// TestParser_Parse_PlainText tests parsing plain text without code blocks.
func TestParser_Parse_PlainText(t *testing.T) {
	parser := NewParser()

	response := "@on create: [auth]"

	parsed, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.MiddlewareDeclarations) != 1 {
		t.Fatalf("Expected 1 middleware declaration, got %d", len(parsed.MiddlewareDeclarations))
	}

	if parsed.MiddlewareDeclarations[0].Operation != "create" {
		t.Errorf("Expected operation 'create', got '%s'", parsed.MiddlewareDeclarations[0].Operation)
	}

	if len(parsed.MiddlewareDeclarations[0].Middleware) != 1 {
		t.Fatalf("Expected 1 middleware, got %d", len(parsed.MiddlewareDeclarations[0].Middleware))
	}

	if parsed.MiddlewareDeclarations[0].Middleware[0] != "auth" {
		t.Errorf("Expected 'auth', got '%s'", parsed.MiddlewareDeclarations[0].Middleware[0])
	}
}

// TestParser_ExtractMiddlewareDeclarations tests extracting @on declarations.
func TestParser_ExtractMiddlewareDeclarations(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		input      string
		wantCount  int
		wantOp     string
		wantMiddle []string
	}{
		{
			name:       "simple auth",
			input:      "@on create: [auth]",
			wantCount:  1,
			wantOp:     "create",
			wantMiddle: []string{"auth"},
		},
		{
			name:       "multiple middleware",
			input:      "@on create: [auth, rate_limit(10/hour)]",
			wantCount:  1,
			wantOp:     "create",
			wantMiddle: []string{"auth", "rate_limit(10/hour)"},
		},
		{
			name:       "cache with params",
			input:      "@on list: [cache(300)]",
			wantCount:  1,
			wantOp:     "list",
			wantMiddle: []string{"cache(300)"},
		},
		{
			name:       "multiple declarations",
			input:      "@on create: [auth]\n@on update: [auth]",
			wantCount:  2,
			wantOp:     "create",
			wantMiddle: []string{"auth"},
		},
		{
			name:       "with extra whitespace",
			input:      "@on  create :  [ auth , cache(300) ]",
			wantCount:  1,
			wantOp:     "create",
			wantMiddle: []string{"auth", "cache(300)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(parsed.MiddlewareDeclarations) != tt.wantCount {
				t.Errorf("Expected %d declarations, got %d", tt.wantCount, len(parsed.MiddlewareDeclarations))
				return
			}

			decl := parsed.MiddlewareDeclarations[0]
			if decl.Operation != tt.wantOp {
				t.Errorf("Expected operation '%s', got '%s'", tt.wantOp, decl.Operation)
			}

			if len(decl.Middleware) != len(tt.wantMiddle) {
				t.Errorf("Expected %d middleware, got %d", len(tt.wantMiddle), len(decl.Middleware))
				return
			}

			for i, want := range tt.wantMiddle {
				if decl.Middleware[i] != want {
					t.Errorf("Middleware[%d]: expected '%s', got '%s'", i, want, decl.Middleware[i])
				}
			}
		})
	}
}

// TestParser_ParseMiddlewareList tests parsing middleware lists.
func TestParser_ParseMiddlewareList(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single item",
			input: "auth",
			want:  []string{"auth"},
		},
		{
			name:  "two items",
			input: "auth, cache",
			want:  []string{"auth", "cache"},
		},
		{
			name:  "with params",
			input: "auth, rate_limit(10/hour)",
			want:  []string{"auth", "rate_limit(10/hour)"},
		},
		{
			name:  "complex params",
			input: "cache(300), rate_limit(10/hour)",
			want:  []string{"cache(300)", "rate_limit(10/hour)"},
		},
		{
			name:  "extra whitespace",
			input: " auth ,  cache(300)  , rate_limit(5/min) ",
			want:  []string{"auth", "cache(300)", "rate_limit(5/min)"},
		},
		{
			name:  "empty",
			input: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.parseMiddlewareList(tt.input)

			if len(got) != len(tt.want) {
				t.Errorf("Expected %d items, got %d", len(tt.want), len(got))
				return
			}

			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("Item[%d]: expected '%s', got '%s'", i, want, got[i])
				}
			}
		})
	}
}

// TestParser_ExtractMiddleware tests the convenience method.
func TestParser_ExtractMiddleware(t *testing.T) {
	parser := NewParser()

	response := "@on create: [auth, rate_limit(10/hour)]"

	middleware, err := parser.ExtractMiddleware(response)
	if err != nil {
		t.Fatalf("ExtractMiddleware failed: %v", err)
	}

	expected := []string{"auth", "rate_limit(10/hour)"}
	if len(middleware) != len(expected) {
		t.Fatalf("Expected %d middleware, got %d", len(expected), len(middleware))
	}

	for i, want := range expected {
		if middleware[i] != want {
			t.Errorf("Middleware[%d]: expected '%s', got '%s'", i, want, middleware[i])
		}
	}
}

// TestParser_ExtractMiddleware_NoDeclaration tests error handling.
func TestParser_ExtractMiddleware_NoDeclaration(t *testing.T) {
	parser := NewParser()

	response := "No middleware declaration here"

	_, err := parser.ExtractMiddleware(response)
	if err == nil {
		t.Error("Expected error for missing declaration")
	}
}

// TestParser_HasMiddleware tests checking for middleware presence.
func TestParser_HasMiddleware(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		response  string
		operation string
		want      bool
	}{
		{
			name:      "has middleware",
			response:  "@on create: [auth]",
			operation: "create",
			want:      true,
		},
		{
			name:      "different operation",
			response:  "@on create: [auth]",
			operation: "update",
			want:      false,
		},
		{
			name:      "no middleware",
			response:  "No declaration here",
			operation: "create",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.HasMiddleware(tt.response, tt.operation)
			if got != tt.want {
				t.Errorf("Expected %v, got %v", tt.want, got)
			}
		})
	}
}

// TestNormalizeMiddlewareDeclaration tests normalization.
func TestNormalizeMiddlewareDeclaration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extra whitespace",
			input: "@on  create :  [ auth ]",
			want:  "@on create: [auth]",
		},
		{
			name:  "no spaces",
			input: "@on create:[auth]",
			want:  "@on create:[auth]",
		},
		{
			name:  "multiple spaces",
			input: "@on   create   :   [   auth   ]",
			want:  "@on create: [auth]",
		},
		{
			name:  "already normalized",
			input: "@on create: [auth]",
			want:  "@on create: [auth]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMiddlewareDeclaration(tt.input)
			if got != tt.want {
				t.Errorf("Expected '%s', got '%s'", tt.want, got)
			}
		})
	}
}

// TestParser_RealWorldResponses tests parsing realistic LLM responses.
func TestParser_RealWorldResponses(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		response string
		wantOp   string
		wantMw   []string
	}{
		{
			name: "response with explanation",
			response: `To add authentication middleware to the Comment.create operation, use:

@on create: [auth]

This ensures only authenticated users can create comments.`,
			wantOp: "create",
			wantMw: []string{"auth"},
		},
		{
			name: "response in code block",
			response: "Here's the middleware declaration:\n\n```conduit\n@on create: [auth, rate_limit(10/hour)]\n```",
			wantOp: "create",
			wantMw: []string{"auth", "rate_limit(10/hour)"},
		},
		{
			name:     "minimal response",
			response: "@on list: [cache(300)]",
			wantOp:   "list",
			wantMw:   []string{"cache(300)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.response)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(parsed.MiddlewareDeclarations) == 0 {
				t.Fatal("Expected at least one middleware declaration")
			}

			decl := parsed.MiddlewareDeclarations[0]
			if decl.Operation != tt.wantOp {
				t.Errorf("Expected operation '%s', got '%s'", tt.wantOp, decl.Operation)
			}

			if len(decl.Middleware) != len(tt.wantMw) {
				t.Errorf("Expected %d middleware, got %d", len(tt.wantMw), len(decl.Middleware))
				return
			}

			for i, want := range tt.wantMw {
				if decl.Middleware[i] != want {
					t.Errorf("Middleware[%d]: expected '%s', got '%s'", i, want, decl.Middleware[i])
				}
			}
		})
	}
}
