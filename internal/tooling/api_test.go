package tooling

import (
	"strings"
	"testing"
)

func TestAPICreation(t *testing.T) {
	api := NewAPI()
	if api == nil {
		t.Fatal("NewAPI() returned nil")
	}

	if api.documents == nil {
		t.Error("API documents map is nil")
	}

	if api.symbolIndex == nil {
		t.Error("API symbolIndex is nil")
	}

	if api.config == nil {
		t.Error("API config is nil")
	}
}

func TestAPIWithCustomConfig(t *testing.T) {
	config := &Config{
		CacheSize:                50,
		EnableIncrementalParsing: false,
	}

	api := NewAPIWithConfig(config)
	if api == nil {
		t.Fatal("NewAPIWithConfig() returned nil")
	}

	if api.config.CacheSize != 50 {
		t.Errorf("Expected CacheSize=50, got %d", api.config.CacheSize)
	}

	if api.config.EnableIncrementalParsing {
		t.Error("Expected EnableIncrementalParsing=false")
	}
}

func TestParseFile(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  created_at: timestamp! @auto
}
`

	doc, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	if doc == nil {
		t.Fatal("ParseFile() returned nil document")
	}

	if doc.URI != "test.cdt" {
		t.Errorf("Expected URI='test.cdt', got '%s'", doc.URI)
	}

	if doc.Content != source {
		t.Error("Document content doesn't match source")
	}

	if doc.AST == nil {
		t.Fatal("Document AST is nil")
	}

	if len(doc.AST.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(doc.AST.Resources))
	}

	resource := doc.AST.Resources[0]
	if resource.Name != "User" {
		t.Errorf("Expected resource name='User', got '%s'", resource.Name)
	}

	if len(doc.ParseErrors) != 0 {
		t.Errorf("Unexpected parse errors: %v", doc.ParseErrors)
	}
}

func TestParseFileWithErrors(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email string! @unique
  name: string!
}
`

	doc, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	if len(doc.ParseErrors) == 0 {
		t.Error("Expected parse errors for invalid syntax")
	}
}

func TestUpdateDocument(t *testing.T) {
	api := NewAPI()

	source1 := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	// Initial parse
	doc1, err := api.ParseFile("test.cdt", source1)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	if doc1.Version != 1 {
		t.Errorf("Expected version=1, got %d", doc1.Version)
	}

	source2 := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	// Update document
	doc2, err := api.UpdateDocument("test.cdt", source2, 2)
	if err != nil {
		t.Fatalf("UpdateDocument() failed: %v", err)
	}

	if doc2.Version != 2 {
		t.Errorf("Expected version=2, got %d", doc2.Version)
	}

	if len(doc2.AST.Resources[0].Fields) != 3 {
		t.Errorf("Expected 3 fields after update, got %d", len(doc2.AST.Resources[0].Fields))
	}
}

func TestUpdateDocumentUnchanged(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
}
`

	// Initial parse
	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Update with same content
	doc2, err := api.UpdateDocument("test.cdt", source, 2)
	if err != nil {
		t.Fatalf("UpdateDocument() failed: %v", err)
	}

	// Should return cached document with updated version
	if doc2.Version != 2 {
		t.Errorf("Expected updated version=2, got %d", doc2.Version)
	}
}

func TestGetDocument(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Get existing document
	doc, exists := api.GetDocument("test.cdt")
	if !exists {
		t.Error("Expected document to exist")
	}

	if doc.URI != "test.cdt" {
		t.Errorf("Expected URI='test.cdt', got '%s'", doc.URI)
	}

	// Get non-existing document
	_, exists = api.GetDocument("nonexistent.cdt")
	if exists {
		t.Error("Expected document to not exist")
	}
}

func TestCloseDocument(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Verify document exists
	_, exists := api.GetDocument("test.cdt")
	if !exists {
		t.Error("Expected document to exist before closing")
	}

	// Close document
	api.CloseDocument("test.cdt")

	// Verify document no longer exists
	_, exists = api.GetDocument("test.cdt")
	if exists {
		t.Error("Expected document to not exist after closing")
	}
}

func TestGetDiagnostics(t *testing.T) {
	api := NewAPI()

	// Valid source
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	diagnostics := api.GetDiagnostics("test.cdt")
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for valid source, got %d", len(diagnostics))
	}
}

func TestGetDiagnosticsWithErrors(t *testing.T) {
	api := NewAPI()

	// Invalid source (missing colon)
	source := `
resource User {
  id uuid! @primary @auto
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	diagnostics := api.GetDiagnostics("test.cdt")
	if len(diagnostics) == 0 {
		t.Error("Expected diagnostics for invalid source")
	}

	// Check diagnostic properties
	diag := diagnostics[0]
	if diag.Severity != DiagnosticSeverityError {
		t.Errorf("Expected severity=Error, got %v", diag.Severity)
	}

	if diag.Source != "conduit" {
		t.Errorf("Expected source='conduit', got '%s'", diag.Source)
	}
}

func TestGetHover(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Hover on resource name
	hover, err := api.GetHover("test.cdt", Position{Line: 1, Character: 10})
	if err != nil {
		t.Fatalf("GetHover() failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information")
	}

	if !strings.Contains(hover.Contents, "User") {
		t.Errorf("Expected hover to contain 'User', got: %s", hover.Contents)
	}
}

func TestGetHoverNoSymbol(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Hover on empty space
	hover, err := api.GetHover("test.cdt", Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("GetHover() failed: %v", err)
	}

	if hover != nil {
		t.Error("Expected no hover information for empty space")
	}
}

func TestGetCompletions(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto

}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Get completions inside resource body
	completions, err := api.GetCompletions("test.cdt", Position{Line: 3, Character: 2})
	if err != nil {
		t.Fatalf("GetCompletions() failed: %v", err)
	}

	if len(completions) == 0 {
		t.Error("Expected completion items")
	}
}

func TestGetCompletionsType(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  email:
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Get completions after colon (type position)
	completions, err := api.GetCompletions("test.cdt", Position{Line: 2, Character: 9})
	if err != nil {
		t.Fatalf("GetCompletions() failed: %v", err)
	}

	if len(completions) == 0 {
		t.Fatal("Expected type completions")
	}

	// Verify we got type completions
	hasString := false
	for _, c := range completions {
		if c.Label == "string" && c.Kind == CompletionKindType {
			hasString = true
			break
		}
	}

	if !hasString {
		t.Error("Expected 'string' type in completions")
	}
}

func TestGetDefinition(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Get definition of resource name
	loc, err := api.GetDefinition("test.cdt", Position{Line: 1, Character: 10})
	if err != nil {
		t.Fatalf("GetDefinition() failed: %v", err)
	}

	if loc == nil {
		t.Fatal("Expected definition location")
	}

	if loc.URI != "test.cdt" {
		t.Errorf("Expected URI='test.cdt', got '%s'", loc.URI)
	}
}

func TestGetReferences(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	// Get references to resource
	refs, err := api.GetReferences("test.cdt", Position{Line: 1, Character: 10})
	if err != nil {
		t.Fatalf("GetReferences() failed: %v", err)
	}

	if len(refs) == 0 {
		t.Error("Expected at least one reference")
	}
}

func TestGetDocumentSymbols(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	_, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	symbols, err := api.GetDocumentSymbols("test.cdt")
	if err != nil {
		t.Fatalf("GetDocumentSymbols() failed: %v", err)
	}

	if len(symbols) == 0 {
		t.Fatal("Expected document symbols")
	}

	// Verify we have resource and fields
	hasResource := false
	fieldCount := 0

	for _, sym := range symbols {
		if sym.Kind == SymbolKindResource && sym.Name == "User" {
			hasResource = true
		}
		if sym.Kind == SymbolKindField {
			fieldCount++
		}
	}

	if !hasResource {
		t.Error("Expected User resource symbol")
	}

	if fieldCount != 3 {
		t.Errorf("Expected 3 field symbols, got %d", fieldCount)
	}
}

func TestThreadSafety(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
}
`

	// Test concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(n int) {
			uri := "test.cdt"
			_, err := api.ParseFile(uri, source)
			if err != nil {
				t.Errorf("ParseFile() failed in goroutine %d: %v", n, err)
			}

			_, _ = api.GetDocument(uri)
			_ = api.GetDiagnostics(uri)
			_, _ = api.GetDocumentSymbols(uri)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestComplexResource(t *testing.T) {
	api := NewAPI()

	source := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  content: text!
  published_at: timestamp?

  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
  }

  @before create {
    if self.published_at == nil {
      self.published_at = Time.now()
    }
  }

  @validate published_content {
    condition: self.published_at != nil
    error: "Must set publish date"
  }

  @constraint min_content_length {
    on: [create, update]
    condition: String.length(self.content) >= 100
    error: "Content must be at least 100 characters"
  }
}
`

	doc, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	if doc.AST == nil {
		t.Fatal("Document AST is nil")
	}

	resource := doc.AST.Resources[0]
	if resource.Name != "Post" {
		t.Errorf("Expected resource name='Post', got '%s'", resource.Name)
	}

	if len(resource.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(resource.Fields))
	}

	if len(resource.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(resource.Relationships))
	}

	if len(resource.Hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(resource.Hooks))
	}

	if len(resource.Validations) != 1 {
		t.Errorf("Expected 1 validation, got %d", len(resource.Validations))
	}

	if len(resource.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(resource.Constraints))
	}

	// Verify symbols extracted correctly
	if len(doc.Symbols) == 0 {
		t.Error("Expected symbols to be extracted")
	}
}
