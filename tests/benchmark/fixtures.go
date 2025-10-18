package benchmark

import (
	"fmt"
	"strings"
)

// GenerateSource generates a Conduit source file with specified number of resources
func GenerateSource(resourceCount int) string {
	var sb strings.Builder

	for i := 0; i < resourceCount; i++ {
		sb.WriteString(fmt.Sprintf(`
resource Resource%d {
	id: uuid! @primary @auto
	name: string! @min(1) @max(100)
	description: text?
	count: int! @min(0)
	active: bool!
	created_at: timestamp! @auto
	updated_at: timestamp! @auto

	@before create {
		self.count = 0
	}
}

`, i))
	}

	return sb.String()
}

// GenerateLargeSource generates a source file with specified lines of code
func GenerateLargeSource(targetLOC int) string {
	// Each resource is approximately 20 lines
	resourceCount := targetLOC / 20
	if resourceCount < 1 {
		resourceCount = 1
	}

	return GenerateSource(resourceCount)
}

// SimpleResource returns a basic resource definition
func SimpleResource() string {
	return `
resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string!
	created_at: timestamp! @auto
}
`
}

// ResourceWithHooks returns a resource with lifecycle hooks
func ResourceWithHooks() string {
	return `
resource Post {
	id: uuid! @primary @auto
	title: string! @min(5) @max(200)
	slug: string! @unique
	content: text!
	view_count: int! @min(0)
	published: bool!
	created_at: timestamp! @auto
	updated_at: timestamp! @auto

	@before create {
		self.view_count = 0
		self.published = false
	}
}
`
}

// ResourceWithRelationships returns resources with relationships
func ResourceWithRelationships() string {
	return `
resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string!
	bio: text?
	created_at: timestamp! @auto
}

resource Post {
	id: uuid! @primary @auto
	title: string! @min(5) @max(200)
	content: text!
	author_id: uuid!
	published: bool!
	created_at: timestamp! @auto

	author: User! {
		foreign_key: "author_id"
		on_delete: restrict
	}
}

resource Comment {
	id: uuid! @primary @auto
	text: string! @min(1) @max(1000)
	post_id: uuid!
	author_id: uuid!
	created_at: timestamp! @auto

	post: Post! {
		foreign_key: "post_id"
		on_delete: cascade
	}

	author: User! {
		foreign_key: "author_id"
		on_delete: restrict
	}
}
`
}

// ComplexResource returns a resource with many features
func ComplexResource() string {
	return `
resource Article {
	id: uuid! @primary @auto
	title: string! @min(10) @max(200) @unique
	slug: string! @unique
	content: text! @min(100)
	excerpt: text? @max(500)
	author_id: uuid!
	category_id: uuid?
	tags: string?
	view_count: int! @min(0)
	like_count: int! @min(0)
	published: bool!
	featured: bool!
	published_at: timestamp?
	created_at: timestamp! @auto
	updated_at: timestamp! @auto

	author: User! {
		foreign_key: "author_id"
		on_delete: restrict
	}

	category: Category? {
		foreign_key: "category_id"
		on_delete: set_null
	}

	@before create {
		self.view_count = 0
		self.like_count = 0
		self.published = false
		self.featured = false
	}

}

resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string!
}

resource Category {
	id: uuid! @primary @auto
	name: string! @unique
}
`
}

// Generate1000LOC generates approximately 1000 lines of code
func Generate1000LOC() string {
	return GenerateLargeSource(1000)
}

// Generate5000LOC generates approximately 5000 lines of code
func Generate5000LOC() string {
	return GenerateLargeSource(5000)
}

// TypicalProject returns a typical project with ~10 resources
func TypicalProject() string {
	return GenerateSource(10)
}

// Generate50Resources generates exactly 50 resources for code generation benchmarks
func Generate50Resources() string {
	return GenerateSource(50)
}

// CountResources counts the number of resource definitions in source code
func CountResources(source string) int {
	return strings.Count(source, "resource ")
}

// CountLOC counts non-empty, non-comment lines
func CountLOC(source string) int {
	lines := strings.Split(source, "\n")
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "///") {
			count++
		}
	}

	return count
}
