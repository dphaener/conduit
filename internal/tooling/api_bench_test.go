package tooling

import (
	"fmt"
	"testing"
)

// Benchmark parsing a simple resource
func BenchmarkParseSimpleResource(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.ParseFile(fmt.Sprintf("test%d.cdt", i), source)
	}
}

// Benchmark parsing a complex resource
func BenchmarkParseComplexResource(b *testing.B) {
	api := NewAPI()
	source := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  published_at: timestamp?
  view_count: int! @default(0)
  status: enum["draft", "published", "archived"]!

  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
  }

  tags: array<Tag>! {
    through: "post_tags"
  }

  @before create {
    self.slug = String.slugify(self.title)
    if self.published_at == nil {
      self.status = "draft"
    }
  }

  @before update {
    if self.status == "published" && self.published_at == nil {
      self.published_at = Time.now()
    }
  }

  @validate published_content {
    condition: self.status == "published"
    error: "Published posts must have publish date"
  }

  @constraint min_content_length {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts need at least 500 characters"
  }

  @computed readTime: int! {
    Math.round(String.length(self.content) / 200)
  }

  @scope published {
    self.status == "published"
  }

  @scope recent {
    self.published_at > Time.now() - Duration.days(30)
  }
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.ParseFile(fmt.Sprintf("test%d.cdt", i), source)
	}
}

// Benchmark hover operation (target: <50ms)
func BenchmarkGetHover(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  bio: text?
  created_at: timestamp! @auto
}
`

	_, _ = api.ParseFile("test.cdt", source)

	// Position on "email" field
	pos := Position{Line: 3, Character: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetHover("test.cdt", pos)
	}

	// Verify performance requirement (<50ms per operation)
	if b.N > 0 {
		nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
		msPerOp := nsPerOp / 1000000
		if msPerOp > 50 {
			b.Errorf("GetHover took %dms per operation, expected <50ms", msPerOp)
		}
	}
}

// Benchmark completion operation (target: <50ms)
func BenchmarkGetCompletions(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto

}
`

	_, _ = api.ParseFile("test.cdt", source)

	// Position inside resource body
	pos := Position{Line: 3, Character: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetCompletions("test.cdt", pos)
	}

	// Verify performance requirement (<50ms per operation)
	if b.N > 0 {
		nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
		msPerOp := nsPerOp / 1000000
		if msPerOp > 50 {
			b.Errorf("GetCompletions took %dms per operation, expected <50ms", msPerOp)
		}
	}
}

// Benchmark type completions
func BenchmarkGetCompletionsType(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  email:
}
`

	_, _ = api.ParseFile("test.cdt", source)

	// Position after colon (type position)
	pos := Position{Line: 2, Character: 9}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetCompletions("test.cdt", pos)
	}
}

// Benchmark definition lookup (target: <50ms)
func BenchmarkGetDefinition(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, _ = api.ParseFile("test.cdt", source)

	// Position on resource name
	pos := Position{Line: 1, Character: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetDefinition("test.cdt", pos)
	}

	// Verify performance requirement (<50ms per operation)
	if b.N > 0 {
		nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
		msPerOp := nsPerOp / 1000000
		if msPerOp > 50 {
			b.Errorf("GetDefinition took %dms per operation, expected <50ms", msPerOp)
		}
	}
}

// Benchmark references lookup
func BenchmarkGetReferences(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, _ = api.ParseFile("test.cdt", source)

	// Position on resource name
	pos := Position{Line: 1, Character: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetReferences("test.cdt", pos)
	}
}

// Benchmark document symbols
func BenchmarkGetDocumentSymbols(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  bio: text?
  created_at: timestamp! @auto

  posts: array<Post>! {
    foreign_key: "author_id"
  }

  @before create {
    self.email = String.lowercase(self.email)
  }

  @computed postCount: int! {
    Array.length(self.posts)
  }
}
`

	_, _ = api.ParseFile("test.cdt", source)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.GetDocumentSymbols("test.cdt")
	}
}

// Benchmark diagnostics retrieval
func BenchmarkGetDiagnostics(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	_, _ = api.ParseFile("test.cdt", source)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = api.GetDiagnostics("test.cdt")
	}
}

// Benchmark document update
func BenchmarkUpdateDocument(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
}
`

	_, _ = api.ParseFile("test.cdt", source)

	updatedSource := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.UpdateDocument("test.cdt", updatedSource, i+2)
	}
}

// Benchmark concurrent access
func BenchmarkConcurrentAccess(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

	_, _ = api.ParseFile("test.cdt", source)
	pos := Position{Line: 1, Character: 10}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = api.GetDiagnostics("test.cdt")
			_, _ = api.GetHover("test.cdt", pos)
			_, _ = api.GetDocumentSymbols("test.cdt")
		}
	})
}

// Benchmark symbol indexing
func BenchmarkSymbolIndexing(b *testing.B) {
	index := NewSymbolIndex()

	symbols := []*Symbol{
		{Name: "User", Kind: SymbolKindResource, Range: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 4}}},
		{Name: "id", Kind: SymbolKindField, Range: Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 2}}},
		{Name: "email", Kind: SymbolKindField, Range: Range{Start: Position{Line: 2, Character: 0}, End: Position{Line: 2, Character: 5}}},
		{Name: "name", Kind: SymbolKindField, Range: Range{Start: Position{Line: 3, Character: 0}, End: Position{Line: 3, Character: 4}}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Index(fmt.Sprintf("test%d.cdt", i), symbols)
	}
}

// Benchmark symbol lookup
func BenchmarkSymbolLookup(b *testing.B) {
	index := NewSymbolIndex()

	// Index many symbols
	for i := 0; i < 100; i++ {
		symbols := []*Symbol{
			{
				Name: fmt.Sprintf("Resource%d", i),
				Kind: SymbolKindResource,
				Range: Range{
					Start: Position{Line: i, Character: 0},
					End:   Position{Line: i, Character: 4},
				},
			},
		}
		index.Index(fmt.Sprintf("test%d.cdt", i), symbols)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = index.FindDefinition(fmt.Sprintf("Resource%d", i%100))
	}
}

// Benchmark completion context detection
func BenchmarkCompletionContext(b *testing.B) {
	api := NewAPI()
	source := `
resource User {
  email: string!
  name:
}
`

	doc, _ := api.ParseFile("test.cdt", source)
	pos := Position{Line: 3, Character: 9}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = api.getCompletionContext(doc, pos)
	}
}

// Benchmark multiple documents
func BenchmarkMultipleDocuments(b *testing.B) {
	api := NewAPI()

	sources := []string{
		`resource User { id: uuid! @primary @auto email: string! @unique }`,
		`resource Post { id: uuid! @primary @auto title: string! content: text! }`,
		`resource Comment { id: uuid! @primary @auto body: text! }`,
		`resource Tag { id: uuid! @primary @auto name: string! @unique }`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, source := range sources {
			_, _ = api.ParseFile(fmt.Sprintf("test%d_%d.cdt", i, j), source)
		}
	}
}
