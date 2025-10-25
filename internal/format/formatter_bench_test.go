package format

import (
	"testing"
)

var benchmarkInput = `/// User resource with authentication
resource User {
  id: uuid! @primary @auto
  name: string! @min(2) @max(100)
  email: string! @unique @pattern("^[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,}$")
  bio: text?
  age: integer? @min(13) @max(120)
  created_at: datetime! @auto
  updated_at: datetime! @auto_update
}

/// Blog post
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  status: enum("draft", "published", "archived")!
  tags: array<string>!
  metadata: hash<string, string>?

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: datetime! @auto
  updated_at: datetime! @auto_update
}

/// Comment on a post
resource Comment {
  id: uuid! @primary @auto
  content: text! @min(1) @max(1000)

  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: datetime! @auto
}
`

func BenchmarkFormatter(b *testing.B) {
	config := DefaultConfig()
	formatter := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(benchmarkInput)
		if err != nil {
			b.Fatalf("Formatting failed: %v", err)
		}
	}
}

func BenchmarkFormatterSmall(b *testing.B) {
	input := `resource User {
id: uuid! @primary @auto
name: string!
}`

	config := DefaultConfig()
	formatter := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(input)
		if err != nil {
			b.Fatalf("Formatting failed: %v", err)
		}
	}
}

func BenchmarkFormatterLarge(b *testing.B) {
	// Generate a large input with many resources
	input := ""
	for i := 0; i < 50; i++ {
		input += `resource Resource` + string(rune('A'+i%26)) + ` {
  id: uuid! @primary @auto
  field1: string! @min(1) @max(100)
  field2: text?
  field3: integer! @min(0) @max(1000)
  field4: datetime! @auto
  field5: array<string>!
}

`
	}

	config := DefaultConfig()
	formatter := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(input)
		if err != nil {
			b.Fatalf("Formatting failed: %v", err)
		}
	}
}

func BenchmarkFormatterWithAlignment(b *testing.B) {
	config := DefaultConfig()
	config.AlignFields = true
	formatter := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(benchmarkInput)
		if err != nil {
			b.Fatalf("Formatting failed: %v", err)
		}
	}
}

func BenchmarkFormatterWithoutAlignment(b *testing.B) {
	config := DefaultConfig()
	config.AlignFields = false
	formatter := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(benchmarkInput)
		if err != nil {
			b.Fatalf("Formatting failed: %v", err)
		}
	}
}
