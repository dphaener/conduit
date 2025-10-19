package relationships

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func setupTestSchemas() map[string]*schema.ResourceSchema {
	userSchema := schema.NewResourceSchema("User")
	userSchema.Fields = map[string]*schema.Field{
		"id":   {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"name": {Name: "name", Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false}},
	}

	postSchema := schema.NewResourceSchema("Post")
	postSchema.Fields = map[string]*schema.Field{
		"id":        {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"title":     {Name: "title", Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false}},
		"author_id": {Name: "author_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
	}
	postSchema.Relationships = map[string]*schema.Relationship{
		"author": {
			Type:           schema.RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			ForeignKey:     "author_id",
			Nullable:       false,
		},
	}

	commentSchema := schema.NewResourceSchema("Comment")
	commentSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"body":    {Name: "body", Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: false}},
		"post_id": {Name: "post_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
	}

	// Add has-many relationship to post
	postSchema.Relationships["comments"] = &schema.Relationship{
		Type:           schema.RelationshipHasMany,
		TargetResource: "Comment",
		FieldName:      "comments",
		ForeignKey:     "post_id",
		Nullable:       false,
	}

	tagSchema := schema.NewResourceSchema("Tag")
	tagSchema.Fields = map[string]*schema.Field{
		"id":   {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"name": {Name: "name", Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false}},
	}

	// Add has-many-through relationship
	postSchema.Relationships["tags"] = &schema.Relationship{
		Type:           schema.RelationshipHasManyThrough,
		TargetResource: "Tag",
		FieldName:      "tags",
		ForeignKey:     "post_id",
		AssociationKey: "tag_id",
		JoinTable:      "post_tags",
		Nullable:       false,
	}

	return map[string]*schema.ResourceSchema{
		"User":    userSchema,
		"Post":    postSchema,
		"Comment": commentSchema,
		"Tag":     tagSchema,
	}
}

func TestLoadBelongsTo(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "First Post", "author_id": "user-1"},
		{"id": "post-2", "title": "Second Post", "author_id": "user-2"},
		{"id": "post-3", "title": "Third Post", "author_id": "user-1"}, // Same author
	}

	// Mock the query for loading authors
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice").
				AddRow("user-2", "Bob"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]

	err := loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// Verify relationships were loaded
	assert.NotNil(t, posts[0]["author"])
	assert.NotNil(t, posts[1]["author"])
	assert.NotNil(t, posts[2]["author"])

	// Verify same author is referenced (should be same object)
	author1 := posts[0]["author"].(map[string]interface{})
	author3 := posts[2]["author"].(map[string]interface{})
	assert.Equal(t, "Alice", author1["name"])
	assert.Equal(t, "Alice", author3["name"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadHasMany(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "First Post"},
		{"id": "post-2", "title": "Second Post"},
	}

	// Mock the query for loading comments
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", "Great post!", "post-1").
				AddRow("comment-2", "Thanks!", "post-1").
				AddRow("comment-3", "Interesting", "post-2"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// Verify relationships were loaded
	comments1 := posts[0]["comments"].([]map[string]interface{})
	comments2 := posts[1]["comments"].([]map[string]interface{})

	assert.Len(t, comments1, 2)
	assert.Len(t, comments2, 1)
	assert.Equal(t, "Great post!", comments1[0]["body"])
	assert.Equal(t, "Interesting", comments2[0]["body"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadHasManyEmpty(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data - post with no comments
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Lonely Post"},
	}

	// Mock the query returning no comments
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// Verify empty slice, not nil
	comments := posts[0]["comments"]
	assert.NotNil(t, comments)
	assert.IsType(t, []map[string]interface{}{}, comments)
	assert.Len(t, comments, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadHasManyThrough(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "First Post"},
		{"id": "post-2", "title": "Second Post"},
	}

	// Mock the query for loading tags through join table
	mock.ExpectQuery(`SELECT t\.\*, j\.post_id as __parent_id`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "__parent_id"}).
				AddRow("tag-1", "golang", "post-1").
				AddRow("tag-2", "web", "post-1").
				AddRow("tag-1", "golang", "post-2"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["tags"]

	err := loader.loadHasManyThrough(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// Verify relationships were loaded
	tags1 := posts[0]["tags"].([]map[string]interface{})
	tags2 := posts[1]["tags"].([]map[string]interface{})

	assert.Len(t, tags1, 2)
	assert.Len(t, tags2, 1)
	assert.Equal(t, "golang", tags1[0]["name"])
	assert.Equal(t, "web", tags1[1]["name"])

	// Verify __parent_id was removed
	_, hasParentID := tags1[0]["__parent_id"]
	assert.False(t, hasParentID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEagerLoadWithNestedRelationships(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "First Post", "author_id": "user-1"},
	}

	// Mock loading authors
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	// Mock loading comments
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", "Great!", "post-1"),
		)

	ctx := context.Background()

	err := loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author", "comments"})
	require.NoError(t, err)

	// Verify both relationships were loaded
	assert.NotNil(t, posts[0]["author"])
	assert.NotNil(t, posts[0]["comments"])

	author := posts[0]["author"].(map[string]interface{})
	assert.Equal(t, "Alice", author["name"])

	comments := posts[0]["comments"].([]map[string]interface{})
	assert.Len(t, comments, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadContextMaxDepth(t *testing.T) {
	ctx := NewLoadContext(3)

	// Increment depth 3 times should succeed
	assert.NoError(t, ctx.IncrementDepth())
	assert.NoError(t, ctx.IncrementDepth())
	assert.NoError(t, ctx.IncrementDepth())

	// 4th increment should fail
	err := ctx.IncrementDepth()
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
}

func TestLoadContextCircularReference(t *testing.T) {
	ctx := NewLoadContext(10)

	// Mark resource as visited
	visited := ctx.MarkVisited("Post:1")
	assert.True(t, visited)

	// Try to visit again
	visited = ctx.MarkVisited("Post:1")
	assert.False(t, visited, "should detect circular reference")
}

func TestLazyRelation(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := schemas["Post"].Relationships["author"]
	ctx := context.Background()

	lazyRel := NewLazyRelation(loader, ctx, "user-1", rel, schemas["Post"])

	// First access should load
	assert.False(t, lazyRel.IsLoaded())

	mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	value, err := lazyRel.Get()
	require.NoError(t, err)
	assert.NotNil(t, value)
	assert.True(t, lazyRel.IsLoaded())

	// Second access should use cached value (no new query)
	value2, err := lazyRel.Get()
	require.NoError(t, err)
	assert.Equal(t, value, value2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParseInclude(t *testing.T) {
	tests := []struct {
		name             string
		include          string
		expectedRel      string
		expectedNested   []string
		expectedHasNested bool
	}{
		{
			name:             "simple relationship",
			include:          "author",
			expectedRel:      "author",
			expectedNested:   nil,
			expectedHasNested: false,
		},
		{
			name:             "nested relationship",
			include:          "author.posts",
			expectedRel:      "author",
			expectedNested:   []string{"posts"},
			expectedHasNested: true,
		},
		{
			name:             "deeply nested",
			include:          "author.posts.comments",
			expectedRel:      "author",
			expectedNested:   []string{"posts.comments"},
			expectedHasNested: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, nested := parseInclude(tt.include)
			assert.Equal(t, tt.expectedRel, rel)
			if tt.expectedHasNested {
				assert.NotNil(t, nested)
				assert.Equal(t, tt.expectedNested, nested)
			} else {
				assert.Nil(t, nested)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserProfile", "user_profile"},
		{"HTTPServer", "http_server"},
		{"APIKey", "api_key"},
		{"PostTag", "post_tag"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"post", "posts"},
		{"category", "categories"},
		{"box", "boxes"},
		{"buzz", "buzzes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pluralize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
