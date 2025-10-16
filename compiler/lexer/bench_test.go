package lexer

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkLexer1000LOC benchmarks lexing 1000 lines of code
// Target: <10ms per 1000 LOC
func BenchmarkLexer1000LOC(b *testing.B) {
	source := generateConduitSource(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkLexer10000LOC benchmarks lexing 10000 lines of code
// Target: Memory < 5MB for 10k LOC
func BenchmarkLexer10000LOC(b *testing.B) {
	source := generateConduitSource(10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkKeywordLookup benchmarks keyword lookup performance
func BenchmarkKeywordLookup(b *testing.B) {
	keywords := []string{
		"resource", "before", "after", "string", "int", "bool",
		"transaction", "async", "validate", "constraint",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, kw := range keywords {
			_, _ = lookupKeyword(kw)
		}
	}
}

// BenchmarkIdentifiers benchmarks identifier scanning
func BenchmarkIdentifiers(b *testing.B) {
	identifiers := []string{
		"username", "email", "created_at", "user_id", "post_title",
		"author_name", "category_slug", "published_at", "updated_at",
	}

	source := strings.Join(identifiers, " ")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkNumbers benchmarks number scanning
func BenchmarkNumbers(b *testing.B) {
	numbers := []string{
		"42", "3.14", "1_000_000", "2.5e10", "0", "-17",
		"1000.50", "1.5e-3", "999_999", "0.001",
	}

	source := strings.Join(numbers, " ")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkStrings benchmarks string scanning
func BenchmarkStrings(b *testing.B) {
	strings := []string{
		`"hello"`, `"world"`, `"escape\nsequences"`,
		`"unicode 世界"`, `"multiline\nstring\nhere"`,
		`"Hello #{name}!"`, `"path\\to\\file"`,
	}

	source := joinStrings(strings, " ")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkComplexResource benchmarks a realistic resource definition
func BenchmarkComplexResource(b *testing.B) {
	source := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  excerpt: text? @max(500)
  content: text! @min(100)

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  tags: array<uuid>! @default([])

  status: enum ["draft", "published", "archived"]! @default("draft")
  published_at: timestamp?

  metrics: {
    view_count: int! @default(0)
    comment_count: int! @default(0)
  }! @default({})

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @has_many Comment as "comments" {
    foreign_key: "post_id"
    on_delete: cascade
  }

  @before create @transaction {
    self.slug = String.slugify(self.title)

    if self.excerpt == nil {
      self.excerpt = generate_excerpt(self.content, 200)
    }
  }

  @after create @transaction {
    if self.status == "published" && self.published_at == nil {
      self.published_at = Time.now()
    }

    @async {
      Notify.send(
        to: self.author.subscribers,
        template: "new_post"
      ) rescue |err| {
        Logger.error("Failed to notify", error: err)
      }
    }
  }

  @constraint published_requires_category {
    on: [create, update]
    when: self.status == "published"
    condition: self.category != nil
    error: "Published posts must have a category"
  }

  @computed is_published: bool! {
    return self.status == "published" &&
           self.published_at != nil &&
           self.published_at! <= Time.now()
  }

  @scope published {
    where: { status: "published" }
    order_by: "published_at DESC"
  }
}
`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkUnicodeSupport benchmarks Unicode handling
func BenchmarkUnicodeSupport(b *testing.B) {
	source := `
resource 用户 {
  名前: string!
  メール: email!
  用户名: string! @unique
  الاسم: string!
  имя: string!
}
`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkErrorRecovery benchmarks lexer with errors
func BenchmarkErrorRecovery(b *testing.B) {
	source := `
resource User {
  username: string!
  invalid ^ character
  email: email!
  more & bad $ symbols
  password: string!
}
`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		_, _ = lexer.ScanTokens()
	}
}

// BenchmarkCommentPreservation benchmarks comment handling
func BenchmarkCommentPreservation(b *testing.B) {
	source := `
# This is a comment
resource User {
  # Field comment
  username: string! # inline comment
  # Another comment
  email: email!
}
# Final comment
`

	b.Run("WithoutPreservation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			lexer := New(source, "bench.cdt")
			lexer.SetPreserveComments(false)
			_, _ = lexer.ScanTokens()
		}
	})

	b.Run("WithPreservation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			lexer := New(source, "bench.cdt")
			lexer.SetPreserveComments(true)
			_, _ = lexer.ScanTokens()
		}
	})
}

// Helper functions

// generateConduitSource generates a realistic Conduit source file with the given number of lines
func generateConduitSource(lines int) string {
	var builder strings.Builder

	resourceTemplate := `
resource User%d {
  id: uuid! @primary @auto
  username: string! @min(3) @max(50)
  email: email! @unique
  bio: text?

  profile: {
    full_name: string!
    avatar_url: url?
  }!

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.slug = String.slugify(self.username)
  }

  @after update @transaction {
    if self.email_changed? {
      Notify.send_verification(self.email)
    }
  }

  @constraint username_length {
    condition: String.length(self.username) >= 3
    error: "Username too short"
  }
}
`

	// Each resource template is approximately 30 lines
	resourcesNeeded := (lines + 29) / 30

	for i := 0; i < resourcesNeeded; i++ {
		builder.WriteString(fmt.Sprintf(resourceTemplate, i))
	}

	return builder.String()
}

// joinStrings is a helper to join strings for benchmarking
func joinStrings(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// BenchmarkMemoryAllocation specifically tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	source := generateConduitSource(1000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lexer := New(source, "bench.cdt")
		tokens, _ := lexer.ScanTokens()

		// Force token usage to prevent optimization
		_ = len(tokens)
	}
}

// BenchmarkTokenCreation benchmarks just token creation
func BenchmarkTokenCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tokens := make([]Token, 0, 1000)
		for j := 0; j < 1000; j++ {
			tokens = append(tokens, Token{
				Type:   TOKEN_IDENTIFIER,
				Lexeme: "identifier",
				Line:   1,
				Column: 1,
				File:   "test.cdt",
			})
		}
	}
}

// BenchmarkRuneConversion benchmarks string to rune conversion
func BenchmarkRuneConversion(b *testing.B) {
	source := generateConduitSource(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = []rune(source)
	}
}
