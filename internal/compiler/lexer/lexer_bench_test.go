package lexer

import (
	"strings"
	"testing"
)

// Generate a sample resource with n fields
func generateResource(fields int) string {
	var sb strings.Builder
	sb.WriteString(`/// Sample resource for benchmarking
resource TestResource {
  id: uuid! @primary @auto
`)

	for i := 0; i < fields; i++ {
		sb.WriteString("  field_")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString(": string! @min(1) @max(100)\n")
	}

	sb.WriteString(`
  @before create @transaction {
    self.slug = String.slugify(self.name)
    self.created_at = Time.now()
  }

  @after create @transaction {
    Logger.info("Created resource", id: self.id)

    @async {
      SearchIndex.update(self) rescue |err| {
        Logger.error("Search index failed", error: err)
      }
    }
  }

  @constraint name_required {
    on: [create, update]
    condition: String.length(self.name) > 0
    error: "Name is required"
  }

  @computed full_path: string! {
    return "/resources/" + self.slug
  }

  @scope active {
    where: { status: "active" }
    order_by: "created_at DESC"
  }
}
`)

	return sb.String()
}

// Generate multiple resources
func generateMultipleResources(count, fieldsPerResource int) string {
	var sb strings.Builder

	for i := 0; i < count; i++ {
		sb.WriteString(generateResource(fieldsPerResource))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// Benchmark simple tokenization
func BenchmarkLexer_Simple(b *testing.B) {
	source := `resource User {
  id: uuid! @primary @auto
  username: string! @unique @min(3) @max(50)
  email: email! @unique
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark 10 fields resource
func BenchmarkLexer_10Fields(b *testing.B) {
	source := generateResource(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark 50 fields resource
func BenchmarkLexer_50Fields(b *testing.B) {
	source := generateResource(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark 100 fields resource
func BenchmarkLexer_100Fields(b *testing.B) {
	source := generateResource(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark 1000 LOC (target: < 10ms)
func BenchmarkLexer_1000LOC(b *testing.B) {
	// Generate approximately 1000 lines
	source := generateMultipleResources(10, 50)
	lines := strings.Count(source, "\n")

	b.Logf("Generated %d lines of code", lines)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark 10K LOC
func BenchmarkLexer_10KLOC(b *testing.B) {
	// Generate approximately 10,000 lines
	source := generateMultipleResources(100, 50)
	lines := strings.Count(source, "\n")

	b.Logf("Generated %d lines of code", lines)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark keyword lookup
func BenchmarkLexer_Keywords(b *testing.B) {
	source := strings.Repeat("resource on after before transaction async rescue ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark identifiers
func BenchmarkLexer_Identifiers(b *testing.B) {
	source := strings.Repeat("user_name post_title author_id created_at ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark string literals
func BenchmarkLexer_Strings(b *testing.B) {
	source := `"hello" "world" "test string" "another string with spaces"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark long strings
func BenchmarkLexer_LongStrings(b *testing.B) {
	longString := strings.Repeat("a", 1000)
	source := `"` + longString + `"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark strings with escapes
func BenchmarkLexer_StringsWithEscapes(b *testing.B) {
	source := `"hello\nworld" "tab\tseparated" "quote\"inside" "backslash\\here"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark numbers
func BenchmarkLexer_Numbers(b *testing.B) {
	source := `42 3.14 1_000_000 2.5e10 0.001`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark operators
func BenchmarkLexer_Operators(b *testing.B) {
	source := `== != <= >= && || ?? ?. -> + - * / %`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark annotations
func BenchmarkLexer_Annotations(b *testing.B) {
	source := `@primary @auto @unique @min @max @default @before @after @transaction`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark comments
func BenchmarkLexer_Comments(b *testing.B) {
	source := strings.Repeat("# This is a comment\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark complex expressions
func BenchmarkLexer_ComplexExpressions(b *testing.B) {
	source := `self.price * (1.0 - self.discount / 100.0) + self.tax ?? 0.0`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark full blog post resource
func BenchmarkLexer_FullBlogPost(b *testing.B) {
	source := `/// Blog post with content, categorization, and publishing workflow
resource Post {
  id: uuid! @primary @auto

  // Content
  title: string! @min(5) @max(200)
  slug: string! @unique
  excerpt: text? @max(500)
  content: text! @min(100)

  // Author & Categorization
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  tags: array<uuid>! @default([])

  // SEO
  seo: {
    meta_title: string? @max(60)
    meta_description: string? @max(160)
    og_image: url?
  }? @default({})

  // Status & Visibility
  status: enum ["draft", "published", "archived"]! @default("draft")
  visibility: enum ["public", "private"]! @default("public")

  // Metrics
  metrics: {
    view_count: int! @default(0)
    comment_count: int! @default(0)
    like_count: int! @default(0)
  }! @default({})

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationships
  @has_many Comment as "comments" {
    foreign_key: "post_id"
    on_delete: cascade
    order_by: "created_at DESC"
  }

  // Lifecycle
  @before create @transaction {
    self.slug = String.slugify(self.title)

    if self.excerpt == nil {
      self.excerpt = Text.excerpt(self.content, 200)
    }
  }

  @after create @transaction {
    if self.status == "published" {
      self.published_at = Time.now()

      @async {
        Notify.send(
          to: self.author.subscribers,
          template: "new_post"
        ) rescue |err| {
          Logger.error("Failed to notify", error: err)
        }
      }
    }
  }

  @after update @transaction {
    if self.content_changed? {
      Revision.create!({
        post_id: self.id,
        content: self.previous_value(:content)
      })
    }

    if self.status_changed_to?("published") {
      self.published_at = Time.now()

      @async {
        SearchIndex.update(self) rescue |err| {
          Logger.warn("Search index failed", error: err)
        }
      }
    }
  }

  // Constraints
  @constraint published_requires_category {
    on: [create, update]
    when: self.status == "published"
    condition: self.category != nil
    error: "Published posts must have a category"
  }

  @invariant metrics_non_negative {
    condition:
      self.metrics.view_count >= 0 &&
      self.metrics.comment_count >= 0 &&
      self.metrics.like_count >= 0
    error: "Metrics cannot be negative"
  }

  // Computed
  @computed is_published: bool! {
    return self.status == "published" && self.published_at <= Time.now()
  }

  @computed url: string! {
    return "/blog/" + self.slug
  }

  // Middleware
  @on list: [cache(300), filter_by_status]
  @on get: [increment_metric(:view_count), cache(600)]
  @on create: [auth, rate_limit(5, per: "hour")]
  @on update: [auth, author_or_editor]
  @on delete: [auth, author_or_admin]

  // Scopes
  @scope published {
    where: { status: "published", published_at: { lte: Time.now() } }
    order_by: "published_at DESC"
  }

  @scope search(query: string) {
    where: {
      or: [
        { title: { ilike: query } },
        { content: { ilike: query } }
      ],
      status: "published"
    }
  }
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Benchmark memory allocation
func BenchmarkLexer_Memory(b *testing.B) {
	source := generateMultipleResources(10, 50)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lexer := New(source)
		lexer.ScanTokens()
	}
}

// Performance test - use benchmarks instead
// Run with: go test -bench=BenchmarkLexer_1000LOC -benchtime=100x
// Target: < 10ms per 1000 LOC
