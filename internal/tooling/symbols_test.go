package tooling

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestSymbolIndex(t *testing.T) {
	index := NewSymbolIndex()

	if index == nil {
		t.Fatal("NewSymbolIndex() returned nil")
	}

	if index.symbols == nil {
		t.Error("Symbol index map is nil")
	}
}

func TestSymbolIndexing(t *testing.T) {
	index := NewSymbolIndex()

	symbols := []*Symbol{
		{
			Name: "User",
			Kind: SymbolKindResource,
			Range: Range{
				Start: Position{Line: 0, Character: 9},
				End:   Position{Line: 0, Character: 13},
			},
		},
		{
			Name:          "email",
			Kind:          SymbolKindField,
			ContainerName: "User",
			Range: Range{
				Start: Position{Line: 1, Character: 2},
				End:   Position{Line: 1, Character: 7},
			},
		},
	}

	index.Index("test.cdt", symbols)

	// Verify symbols were indexed
	def := index.FindDefinition("User")
	if def == nil {
		t.Fatal("Expected to find User definition")
	}

	if def.Name != "User" {
		t.Errorf("Expected name='User', got '%s'", def.Name)
	}

	if def.URI != "test.cdt" {
		t.Errorf("Expected URI='test.cdt', got '%s'", def.URI)
	}
}

func TestSymbolIndexRemoveDocument(t *testing.T) {
	index := NewSymbolIndex()

	symbols := []*Symbol{
		{
			Name: "User",
			Kind: SymbolKindResource,
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 4},
			},
		},
	}

	index.Index("test.cdt", symbols)

	// Verify symbol exists
	def := index.FindDefinition("User")
	if def == nil {
		t.Fatal("Expected to find User definition before removal")
	}

	// Remove document
	index.RemoveDocument("test.cdt")

	// Verify symbol no longer exists
	def = index.FindDefinition("User")
	if def != nil {
		t.Error("Expected User definition to be removed")
	}
}

func TestSymbolIndexReindexing(t *testing.T) {
	index := NewSymbolIndex()

	// Initial indexing
	symbols1 := []*Symbol{
		{
			Name: "User",
			Kind: SymbolKindResource,
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 4},
			},
		},
	}

	index.Index("test.cdt", symbols1)

	// Reindex with different symbols
	symbols2 := []*Symbol{
		{
			Name: "Post",
			Kind: SymbolKindResource,
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 4},
			},
		},
	}

	index.Index("test.cdt", symbols2)

	// Verify old symbol is gone
	def := index.FindDefinition("User")
	if def != nil {
		t.Error("Expected User definition to be removed after reindexing")
	}

	// Verify new symbol exists
	def = index.FindDefinition("Post")
	if def == nil {
		t.Fatal("Expected to find Post definition after reindexing")
	}
}

func TestFindReferences(t *testing.T) {
	index := NewSymbolIndex()

	symbols := []*Symbol{
		{
			Name: "User",
			Kind: SymbolKindResource,
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 4},
			},
		},
		{
			Name: "User",
			Kind: SymbolKindField, // Reference to User type
			Range: Range{
				Start: Position{Line: 5, Character: 0},
				End:   Position{Line: 5, Character: 4},
			},
		},
	}

	index.Index("test.cdt", symbols)

	refs := index.FindReferences("User")
	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}

	for _, ref := range refs {
		if ref.URI != "test.cdt" {
			t.Errorf("Expected URI='test.cdt', got '%s'", ref.URI)
		}
	}
}

func TestExtractSymbolsFromResource(t *testing.T) {
	api := NewAPI()

	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!

  posts: array<Post>! {
    foreign_key: "author_id"
  }

  @before create {
    self.name = String.trim(self.name)
  }

  @computed displayName: string! {
    String.uppercase(self.name)
  }

  @scope active {
    self.deleted_at == nil
  }
}
`

	doc, err := api.ParseFile("test.cdt", source)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	symbols := doc.Symbols
	if len(symbols) == 0 {
		t.Fatal("Expected symbols to be extracted")
	}

	// Count symbols by kind
	counts := make(map[SymbolKind]int)
	for _, sym := range symbols {
		counts[sym.Kind]++
	}

	if counts[SymbolKindResource] != 1 {
		t.Errorf("Expected 1 resource symbol, got %d", counts[SymbolKindResource])
	}

	if counts[SymbolKindField] != 3 {
		t.Errorf("Expected 3 field symbols, got %d", counts[SymbolKindField])
	}

	if counts[SymbolKindRelationship] != 1 {
		t.Errorf("Expected 1 relationship symbol, got %d", counts[SymbolKindRelationship])
	}

	if counts[SymbolKindHook] != 1 {
		t.Errorf("Expected 1 hook symbol, got %d", counts[SymbolKindHook])
	}

	if counts[SymbolKindComputed] != 1 {
		t.Errorf("Expected 1 computed symbol, got %d", counts[SymbolKindComputed])
	}

	if counts[SymbolKindScope] != 1 {
		t.Errorf("Expected 1 scope symbol, got %d", counts[SymbolKindScope])
	}
}

func TestPositionInRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		r        Range
		expected bool
	}{
		{
			name: "position inside range",
			pos:  Position{Line: 5, Character: 10},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: true,
		},
		{
			name: "position at start",
			pos:  Position{Line: 5, Character: 5},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: true,
		},
		{
			name: "position at end",
			pos:  Position{Line: 5, Character: 15},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: true,
		},
		{
			name: "position before range",
			pos:  Position{Line: 5, Character: 3},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: false,
		},
		{
			name: "position after range",
			pos:  Position{Line: 5, Character: 20},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: false,
		},
		{
			name: "position on different line before",
			pos:  Position{Line: 3, Character: 10},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: false,
		},
		{
			name: "position on different line after",
			pos:  Position{Line: 7, Character: 10},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 15},
			},
			expected: false,
		},
		{
			name: "multi-line range inside",
			pos:  Position{Line: 6, Character: 10},
			r: Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 7, Character: 15},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := positionInRange(tt.pos, tt.r)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		expected string
	}{
		{
			name: "primitive required",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "string",
				Nullable: false,
			},
			expected: "string!",
		},
		{
			name: "primitive optional",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "int",
				Nullable: true,
			},
			expected: "int?",
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeArray,
				Name:     "array",
				Nullable: false,
				ElementType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
			},
			expected: "array<string!>!",
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeHash,
				Name:     "hash",
				Nullable: false,
				KeyType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				ValueType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
			},
			expected: "hash<string!, int!>!",
		},
		{
			name: "enum type",
			typeNode: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				Name:       "enum",
				Nullable:   false,
				EnumValues: []string{"active", "inactive", "pending"},
			},
			expected: `enum["active", "inactive", "pending"]!`,
		},
		{
			name: "resource type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeResource,
				Name:     "User",
				Nullable: false,
			},
			expected: "User!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatType(tt.typeNode)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestRelationshipKindString(t *testing.T) {
	tests := []struct {
		kind     ast.RelationshipKind
		expected string
	}{
		{ast.RelationshipBelongsTo, "belongs_to"},
		{ast.RelationshipHasMany, "has_many"},
		{ast.RelationshipHasManyThrough, "has_many_through"},
		{ast.RelationshipHasOne, "has_one"},
	}

	for _, tt := range tests {
		result := relationshipKindString(tt.kind)
		if result != tt.expected {
			t.Errorf("Expected '%s', got '%s'", tt.expected, result)
		}
	}
}
