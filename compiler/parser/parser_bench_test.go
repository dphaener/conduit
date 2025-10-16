package parser

import (
	"github.com/conduit-lang/conduit/compiler/lexer"
	"strings"
	"testing"
)

// BenchmarkParser_SimpleResource benchmarks parsing a simple resource
func BenchmarkParser_SimpleResource(b *testing.B) {
	source := `
resource User {
  id: uuid! @primary @auto
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  created_at: timestamp! @auto
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_ComplexResource benchmarks parsing a complex resource
func BenchmarkParser_ComplexResource(b *testing.B) {
	source := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  excerpt: text?

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  tags: array<uuid>!

  seo: {
    title: string? @max(60)
    description: string? @max(160)
    image: url?
  }?

  status: enum ["draft", "published", "archived"]! @default("draft")

  metrics: {
    view_count: int! @default(0)
    comment_count: int! @default(0)
    like_count: int! @default(0)
  }!

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
  published_at: timestamp?
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_MultipleResources benchmarks parsing multiple resources
func BenchmarkParser_MultipleResources(b *testing.B) {
	source := `
resource User {
  id: uuid! @primary
  username: string! @unique
  email: email! @unique
}

resource Post {
  id: uuid! @primary
  title: string!
  author: User!
}

resource Comment {
  id: uuid! @primary
  content: text!
  post: Post!
  author: User!
}

resource Tag {
  id: uuid! @primary
  name: string! @unique
}

resource Category {
  id: uuid! @primary
  name: string!
  slug: string! @unique
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_LargeFile benchmarks parsing a large file (~1000 lines)
func BenchmarkParser_LargeFile(b *testing.B) {
	// Generate a large file with multiple resources
	resourceTemplate := `
resource Resource%d {
  id: uuid! @primary @auto
  name: string! @min(3) @max(100)
  description: text?
  status: enum ["active", "inactive"]!
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
  metadata: hash<string, string>!
  tags: array<string>!
  count: int! @default(0)
  settings: {
    enabled: bool!
    priority: int!
  }!
}
`

	var builder strings.Builder
	// Generate 50 resources (approx 1000 lines)
	for i := 0; i < 50; i++ {
		builder.WriteString(strings.ReplaceAll(resourceTemplate, "%d", string(rune('0'+i%10))))
	}

	source := builder.String()
	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_PrimitiveTypes benchmarks parsing all primitive types
func BenchmarkParser_PrimitiveTypes(b *testing.B) {
	source := `
resource AllTypes {
  str: string!
  txt: text!
  num: int!
  flt: float!
  dec: decimal!
  flag: bool!
  ts: timestamp!
  dt: date!
  tm: time!
  uid: uuid!
  ulid: ulid!
  mail: email!
  link: url!
  phone: phone!
  data: json!
  md: markdown!
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_StructuralTypes benchmarks parsing structural types
func BenchmarkParser_StructuralTypes(b *testing.B) {
	source := `
resource StructuralTypes {
  simple_array: array<string>!
  nested_array: array<array<int>>!
  simple_hash: hash<string, int>!
  complex_hash: hash<uuid, string>!
  enum_field: enum ["one", "two", "three"]!
  inline_struct: {
    field1: string!
    field2: int!
    nested: {
      deep: bool!
    }!
  }!
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_Relationships benchmarks parsing relationships
func BenchmarkParser_Relationships(b *testing.B) {
	source := `
resource Post {
  id: uuid! @primary

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  parent: Post? {
    foreign_key: "parent_id"
    on_delete: cascade
  }
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}

// BenchmarkParser_Constraints benchmarks parsing field constraints
func BenchmarkParser_Constraints(b *testing.B) {
	source := `
resource User {
  username: string! @unique @min(3) @max(50) @pattern("[a-z0-9_]+")
  email: email! @unique @required
  age: int! @min(18) @max(120) @default(18)
  score: float! @min(0.0) @max(100.0)
  active: bool! @default(true)
}
`

	l := lexer.New(source, "bench.cdt")
	tokens, _ := l.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(tokens)
		p.Parse()
	}
}
