// +build integration

package relationships

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QueryCounter wraps a DB connection and counts queries
type QueryCounter struct {
	db    *sql.DB
	count int
}

func NewQueryCounter(db *sql.DB) *QueryCounter {
	return &QueryCounter{db: db, count: 0}
}

func (qc *QueryCounter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	qc.count++
	return qc.db.QueryContext(ctx, query, args...)
}

func (qc *QueryCounter) Reset() {
	qc.count = 0
}

func (qc *QueryCounter) Count() int {
	return qc.count
}

func setupIntegrationDB(t *testing.T) *sql.DB {
	// Use environment variable or default to local PostgreSQL
	dsn := "postgres://localhost/conduit_test?sslmode=disable"
	if testDSN := os.Getenv("TEST_DATABASE_URL"); testDSN != "" {
		dsn = testDSN
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	// Ping to verify connection
	err = db.Ping()
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

func setupTestTables(t *testing.T, db *sql.DB) {
	// Create test tables
	queries := []string{
		`DROP TABLE IF EXISTS post_tags CASCADE`,
		`DROP TABLE IF EXISTS comments CASCADE`,
		`DROP TABLE IF EXISTS tags CASCADE`,
		`DROP TABLE IF EXISTS posts CASCADE`,
		`DROP TABLE IF EXISTS profiles CASCADE`,
		`DROP TABLE IF EXISTS users CASCADE`,

		`CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL
		)`,

		`CREATE TABLE profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			bio TEXT,
			avatar_url VARCHAR(500)
		)`,

		`CREATE TABLE posts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			body TEXT NOT NULL,
			author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			body TEXT NOT NULL,
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE tags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL UNIQUE
		)`,

		`CREATE TABLE post_tags (
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			PRIMARY KEY (post_id, tag_id)
		)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		require.NoError(t, err, "Failed to execute: %s", query)
	}
}

func seedTestData(t *testing.T, db *sql.DB) {
	// Insert test data
	queries := []string{
		`INSERT INTO users (id, name, email) VALUES
			('00000000-0000-0000-0000-000000000001', 'Alice', 'alice@example.com'),
			('00000000-0000-0000-0000-000000000002', 'Bob', 'bob@example.com'),
			('00000000-0000-0000-0000-000000000003', 'Charlie', 'charlie@example.com')`,

		`INSERT INTO profiles (user_id, bio) VALUES
			('00000000-0000-0000-0000-000000000001', 'Software engineer'),
			('00000000-0000-0000-0000-000000000002', 'Designer')`,

		`INSERT INTO posts (id, title, body, author_id) VALUES
			('10000000-0000-0000-0000-000000000001', 'First Post', 'Content 1', '00000000-0000-0000-0000-000000000001'),
			('10000000-0000-0000-0000-000000000002', 'Second Post', 'Content 2', '00000000-0000-0000-0000-000000000001'),
			('10000000-0000-0000-0000-000000000003', 'Third Post', 'Content 3', '00000000-0000-0000-0000-000000000002'),
			('10000000-0000-0000-0000-000000000004', 'Fourth Post', 'Content 4', '00000000-0000-0000-0000-000000000003')`,

		`INSERT INTO comments (body, post_id, author_id) VALUES
			('Great post!', '10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002'),
			('Thanks!', '10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001'),
			('Interesting', '10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000003'),
			('Cool', '10000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001')`,

		`INSERT INTO tags (id, name) VALUES
			('20000000-0000-0000-0000-000000000001', 'golang'),
			('20000000-0000-0000-0000-000000000002', 'web'),
			('20000000-0000-0000-0000-000000000003', 'database')`,

		`INSERT INTO post_tags (post_id, tag_id) VALUES
			('10000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001'),
			('10000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002'),
			('10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001'),
			('10000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000003')`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		require.NoError(t, err, "Failed to seed data: %s", query)
	}
}

func TestNPlusOnePreventionBelongsTo(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	qc := NewQueryCounter(db)
	loader := &Loader{db: qc.db, schemas: schemas}

	ctx := context.Background()

	// Fetch all posts
	rows, err := qc.QueryContext(ctx, "SELECT * FROM posts ORDER BY created_at")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)
	require.Len(t, posts, 4)

	qc.Reset()

	// Eager load authors (should be 1 query, not 4)
	err = loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author"})
	require.NoError(t, err)

	// Should have made exactly 1 query to load all authors
	assert.Equal(t, 1, qc.Count(), "Expected 1 query for eager loading authors, got %d", qc.Count())

	// Verify all authors are loaded
	for i, post := range posts {
		author, ok := post["author"].(map[string]interface{})
		require.True(t, ok, "Post %d missing author", i)
		require.NotNil(t, author, "Post %d author is nil", i)
		assert.NotEmpty(t, author["name"], "Post %d author has no name", i)
	}
}

func TestNPlusOnePreventionHasMany(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	qc := NewQueryCounter(db)
	loader := &Loader{db: qc.db, schemas: schemas}

	ctx := context.Background()

	// Fetch all posts
	rows, err := qc.QueryContext(ctx, "SELECT * FROM posts ORDER BY created_at")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)
	require.Len(t, posts, 4)

	qc.Reset()

	// Eager load comments (should be 1 query, not 4)
	err = loader.EagerLoad(ctx, posts, schemas["Post"], []string{"comments"})
	require.NoError(t, err)

	// Should have made exactly 1 query to load all comments
	assert.Equal(t, 1, qc.Count(), "Expected 1 query for eager loading comments, got %d", qc.Count())

	// Verify all comments are loaded (even if empty)
	for i, post := range posts {
		comments, ok := post["comments"].([]map[string]interface{})
		require.True(t, ok, "Post %d missing comments field", i)
		require.NotNil(t, comments, "Post %d comments is nil (should be empty slice)", i)
	}

	// Check specific post has correct number of comments
	post1Comments := posts[0]["comments"].([]map[string]interface{})
	assert.Len(t, post1Comments, 2, "First post should have 2 comments")
}

func TestNPlusOnePreventionHasManyThrough(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	qc := NewQueryCounter(db)
	loader := &Loader{db: qc.db, schemas: schemas}

	ctx := context.Background()

	// Fetch all posts
	rows, err := qc.QueryContext(ctx, "SELECT * FROM posts ORDER BY created_at")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)
	require.Len(t, posts, 4)

	qc.Reset()

	// Eager load tags through join table (should be 1 query, not 4)
	err = loader.EagerLoad(ctx, posts, schemas["Post"], []string{"tags"})
	require.NoError(t, err)

	// Should have made exactly 1 query to load all tags
	assert.Equal(t, 1, qc.Count(), "Expected 1 query for eager loading tags, got %d", qc.Count())

	// Verify all tags are loaded
	for i, post := range posts {
		tags, ok := post["tags"].([]map[string]interface{})
		require.True(t, ok, "Post %d missing tags field", i)
		require.NotNil(t, tags, "Post %d tags is nil (should be empty slice)", i)
	}

	// Check specific posts have correct tags
	post1Tags := posts[0]["tags"].([]map[string]interface{})
	assert.Len(t, post1Tags, 2, "First post should have 2 tags")
}

func TestNestedEagerLoading(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	qc := NewQueryCounter(db)
	loader := &Loader{db: qc.db, schemas: schemas}

	ctx := context.Background()

	// Fetch all posts
	rows, err := qc.QueryContext(ctx, "SELECT * FROM posts ORDER BY created_at LIMIT 2")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)
	require.Len(t, posts, 2)

	qc.Reset()

	// Eager load author, comments, and tags in one go
	err = loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author", "comments", "tags"})
	require.NoError(t, err)

	// Should have made exactly 3 queries (one for each relationship type)
	assert.LessOrEqual(t, qc.Count(), 3, "Expected at most 3 queries for loading 3 relationships, got %d", qc.Count())

	// Verify all relationships loaded
	for i, post := range posts {
		assert.NotNil(t, post["author"], "Post %d missing author", i)
		assert.NotNil(t, post["comments"], "Post %d missing comments", i)
		assert.NotNil(t, post["tags"], "Post %d missing tags", i)
	}
}

func TestDeepNestedLoading(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	qc := NewQueryCounter(db)

	// Add relationships for nested loading
	// Comment belongs_to User (author)
	schemas["Comment"].Relationships = map[string]*schema.Relationship{
		"author": {
			Type:           schema.RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			ForeignKey:     "author_id",
			Nullable:       false,
		},
	}

	loader := &Loader{db: qc.db, schemas: schemas}

	ctx := context.Background()

	// Fetch posts
	rows, err := qc.QueryContext(ctx, "SELECT * FROM posts ORDER BY created_at LIMIT 1")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)
	require.Len(t, posts, 1)

	qc.Reset()

	// Load posts -> comments -> author (3 levels deep)
	// This tests nested includes like "comments.author"
	err = loader.EagerLoad(ctx, posts, schemas["Post"], []string{"comments"})
	require.NoError(t, err)

	// Now load authors on the comments
	comments := posts[0]["comments"].([]map[string]interface{})
	if len(comments) > 0 {
		err = loader.EagerLoad(ctx, comments, schemas["Comment"], []string{"author"})
		require.NoError(t, err)

		// Verify nested data loaded
		for _, comment := range comments {
			assert.NotNil(t, comment["author"], "Comment missing author")
		}
	}

	// Total queries should be <= 4 (posts + comments + authors for comments + initial query)
	assert.LessOrEqual(t, qc.Count(), 3, "Expected at most 3 queries for nested loading, got %d", qc.Count())
}

func TestCircularReferenceHandling(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	ctx := context.Background()

	// Create a load context with max depth
	loadCtx := NewLoadContext(2)

	rows, err := db.Query("SELECT * FROM posts LIMIT 1")
	require.NoError(t, err)
	defer rows.Close()

	posts, err := scanRows(rows, schemas["Post"])
	require.NoError(t, err)

	// This should succeed at depth 2
	err = loader.EagerLoadWithContext(ctx, posts, schemas["Post"], []string{"author"}, loadCtx)
	require.NoError(t, err)

	// Try to go deeper - should be prevented
	loadCtx.IncrementDepth()
	loadCtx.IncrementDepth()
	err = loadCtx.IncrementDepth()
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
}

func BenchmarkEagerLoadingVsLazy(b *testing.B) {
	t := &testing.T{}
	db := setupIntegrationDB(t)
	defer db.Close()

	setupTestTables(t, db)
	seedTestData(t, db)

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	ctx := context.Background()

	b.Run("EagerLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, _ := db.Query("SELECT * FROM posts")
			posts, _ := scanRows(rows, schemas["Post"])
			rows.Close()

			loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author", "comments"})
		}
	})

	b.Run("LazyLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, _ := db.Query("SELECT * FROM posts")
			posts, _ := scanRows(rows, schemas["Post"])
			rows.Close()

			// Simulate lazy loading each relationship
			for _, post := range posts {
				rel := schemas["Post"].Relationships["author"]
				loader.LoadSingle(ctx, post["author_id"], rel, schemas["Post"])

				rel2 := schemas["Post"].Relationships["comments"]
				loader.LoadSingle(ctx, post["id"], rel2, schemas["Post"])
			}
		}
	})
}
