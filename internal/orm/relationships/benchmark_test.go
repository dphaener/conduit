package relationships

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// BenchmarkBelongsToLoading benchmarks loading belongs-to relationships
func BenchmarkBelongsToLoading(b *testing.B) {
	db, mock := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)
	rel := schemas["Post"].Relationships["author"]

	// Create 100 posts
	posts := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		posts[i] = map[string]interface{}{
			"id":        fmt.Sprintf("post-%d", i),
			"title":     fmt.Sprintf("Post %d", i),
			"author_id": fmt.Sprintf("user-%d", i%10), // 10 unique authors
		}
	}

	// Mock query
	rows := sqlmock.NewRows([]string{"id", "name"})
	for i := 0; i < 10; i++ {
		rows.AddRow(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
	}
	mock.ExpectQuery(`SELECT \* FROM users`).WillReturnRows(rows)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	}
}

// BenchmarkHasManyLoading benchmarks loading has-many relationships
func BenchmarkHasManyLoading(b *testing.B) {
	db, mock := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)
	rel := schemas["Post"].Relationships["comments"]

	// Create 100 posts
	posts := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		posts[i] = map[string]interface{}{
			"id":    fmt.Sprintf("post-%d", i),
			"title": fmt.Sprintf("Post %d", i),
		}
	}

	// Mock query - 500 total comments
	rows := sqlmock.NewRows([]string{"id", "body", "post_id"})
	for i := 0; i < 500; i++ {
		rows.AddRow(
			fmt.Sprintf("comment-%d", i),
			fmt.Sprintf("Comment %d", i),
			fmt.Sprintf("post-%d", i%100),
		)
	}
	mock.ExpectQuery(`SELECT \* FROM comments`).WillReturnRows(rows)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	}
}

// BenchmarkHasManyThroughLoading benchmarks loading has-many-through relationships
func BenchmarkHasManyThroughLoading(b *testing.B) {
	db, mock := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)
	rel := schemas["Post"].Relationships["tags"]

	// Create 100 posts
	posts := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		posts[i] = map[string]interface{}{
			"id":    fmt.Sprintf("post-%d", i),
			"title": fmt.Sprintf("Post %d", i),
		}
	}

	// Mock query - 300 tag associations (avg 3 tags per post)
	rows := sqlmock.NewRows([]string{"id", "name", "__parent_id"})
	for i := 0; i < 300; i++ {
		rows.AddRow(
			fmt.Sprintf("tag-%d", i%20), // 20 unique tags
			fmt.Sprintf("Tag %d", i%20),
			fmt.Sprintf("post-%d", i/3), // 3 tags per post
		)
	}
	mock.ExpectQuery(`SELECT t\.\*, j\.post_id`).WillReturnRows(rows)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loader.loadHasManyThrough(ctx, posts, rel, schemas["Post"])
	}
}

// BenchmarkEagerLoadMultipleRelationships benchmarks loading multiple relationships
func BenchmarkEagerLoadMultipleRelationships(b *testing.B) {
	db, mock := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Create 50 posts
	posts := make([]map[string]interface{}, 50)
	for i := 0; i < 50; i++ {
		posts[i] = map[string]interface{}{
			"id":        fmt.Sprintf("post-%d", i),
			"title":     fmt.Sprintf("Post %d", i),
			"author_id": fmt.Sprintf("user-%d", i%5),
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mock authors
		authorRows := sqlmock.NewRows([]string{"id", "name"})
		for j := 0; j < 5; j++ {
			authorRows.AddRow(fmt.Sprintf("user-%d", j), fmt.Sprintf("User %d", j))
		}
		mock.ExpectQuery(`SELECT \* FROM users`).WillReturnRows(authorRows)

		// Mock comments
		commentRows := sqlmock.NewRows([]string{"id", "body", "post_id"})
		for j := 0; j < 150; j++ {
			commentRows.AddRow(
				fmt.Sprintf("comment-%d", j),
				fmt.Sprintf("Comment %d", j),
				fmt.Sprintf("post-%d", j%50),
			)
		}
		mock.ExpectQuery(`SELECT \* FROM comments`).WillReturnRows(commentRows)

		loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author", "comments"})
	}
}

// BenchmarkLazyRelation benchmarks lazy loading
func BenchmarkLazyRelation(b *testing.B) {
	db, mock := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)
	rel := schemas["Post"].Relationships["author"]
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lazyRel := NewLazyRelation(loader, ctx, "user-1", rel, schemas["Post"])

		mock.ExpectQuery(`SELECT \* FROM users`).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "name"}).
					AddRow("user-1", "Alice"),
			)

		lazyRel.Get()
	}
}

// BenchmarkParseInclude benchmarks include parsing
func BenchmarkParseInclude(b *testing.B) {
	includes := []string{
		"author",
		"author.posts",
		"author.posts.comments",
		"comments.author.profile",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, inc := range includes {
			parseInclude(inc)
		}
	}
}

// BenchmarkLoadContextOperations benchmarks load context operations
func BenchmarkLoadContextOperations(b *testing.B) {
	ctx := NewLoadContext(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.MarkVisited(fmt.Sprintf("Post:%d", i))
		ctx.IncrementDepth()
		ctx.DecrementDepth()
	}
}

// BenchmarkRecordExtraction benchmarks extracting nested records
func BenchmarkRecordExtraction(b *testing.B) {
	rel := &schema.Relationship{
		Type:      schema.RelationshipHasMany,
		FieldName: "comments",
	}

	// Create records with nested data
	posts := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		comments := make([]map[string]interface{}, 5)
		for j := 0; j < 5; j++ {
			comments[j] = map[string]interface{}{
				"id":   fmt.Sprintf("comment-%d-%d", i, j),
				"body": "Some comment",
			}
		}
		posts[i] = map[string]interface{}{
			"id":       fmt.Sprintf("post-%d", i),
			"comments": comments,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractNestedRecords(posts, rel)
	}
}

// BenchmarkStringConversions benchmarks utility string functions
func BenchmarkStringConversions(b *testing.B) {
	testStrings := []string{
		"User", "UserProfile", "HTTPServer", "APIKey", "PostTag",
	}

	b.Run("toSnakeCase", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				toSnakeCase(s)
			}
		}
	})

	b.Run("pluralize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				pluralize(toSnakeCase(s))
			}
		}
	})

	b.Run("toTableName", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				toTableName(s)
			}
		}
	})
}

// BenchmarkLargeDataset benchmarks loading with large datasets
func BenchmarkLargeDataset(b *testing.B) {
	sizes := []int{100, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("BelongsTo_%d_records", size), func(b *testing.B) {
			db, mock := setupTestDB(&testing.T{})
			defer db.Close()

			schemas := setupTestSchemas()
			loader := NewLoader(db, schemas)
			rel := schemas["Post"].Relationships["author"]

			posts := make([]map[string]interface{}, size)
			for i := 0; i < size; i++ {
				posts[i] = map[string]interface{}{
					"id":        fmt.Sprintf("post-%d", i),
					"author_id": fmt.Sprintf("user-%d", i%100), // 100 unique authors
				}
			}

			rows := sqlmock.NewRows([]string{"id", "name"})
			for i := 0; i < 100; i++ {
				rows.AddRow(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
			}
			mock.ExpectQuery(`SELECT \* FROM users`).WillReturnRows(rows)

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
			}
		})
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	db, _ := setupTestDB(&testing.T{})
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	b.Run("CreateLoader", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			NewLoader(db, schemas)
		}
	})

	b.Run("CreateLoadContext", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			NewLoadContext(10)
		}
	})

	b.Run("CreateLazyRelation", func(b *testing.B) {
		rel := schemas["Post"].Relationships["author"]
		ctx := context.Background()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			NewLazyRelation(loader, ctx, "user-1", rel, schemas["Post"])
		}
	})
}
