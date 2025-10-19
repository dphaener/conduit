package relationships

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadHasOne tests has-one relationship loading
func TestLoadHasOne(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	// Add has-one relationship
	userSchema := schemas["User"]
	userSchema.Relationships = map[string]*schema.Relationship{
		"profile": {
			Type:           schema.RelationshipHasOne,
			TargetResource: "Profile",
			FieldName:      "profile",
			ForeignKey:     "user_id",
			Nullable:       true,
		},
	}

	// Create a Profile schema
	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"bio":     {Name: "bio", Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true}},
	}
	schemas["Profile"] = profileSchema

	loader := NewLoader(db, schemas)

	// Test data
	users := []map[string]interface{}{
		{"id": "user-1", "name": "Alice"},
		{"id": "user-2", "name": "Bob"},
	}

	// Mock the query for loading profiles
	mock.ExpectQuery(`SELECT DISTINCT ON \(user_id\) \* FROM profiles`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "bio"}).
				AddRow("profile-1", "user-1", "Software engineer"),
		)

	ctx := context.Background()
	rel := userSchema.Relationships["profile"]

	err := loader.loadHasOne(ctx, users, rel, userSchema)
	require.NoError(t, err)

	// Verify relationships were loaded
	assert.NotNil(t, users[0]["profile"])
	assert.Nil(t, users[1]["profile"]) // User 2 has no profile

	profile := users[0]["profile"].(map[string]interface{})
	assert.Equal(t, "Software engineer", profile["bio"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadBelongsToNullable tests nullable belongs-to relationships
func TestLoadBelongsToNullable(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Test data with some null foreign keys
	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post with author", "author_id": "user-1"},
		{"id": "post-2", "title": "Post without author", "author_id": nil},
	}

	// Mock the query for loading authors (only one author)
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]
	rel.Nullable = true // Make it nullable

	err := loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// Post 1 should have author
	assert.NotNil(t, posts[0]["author"])
	// Post 2 should have nil author
	assert.Nil(t, posts[1]["author"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasMany tests lazy loading of has-many
func TestLoadSingleHasMany(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Mock the query
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = \$1`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", "Great!", "post-1").
				AddRow("comment-2", "Thanks!", "post-1"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	comments, err := loader.loadSingleHasMany(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)

	assert.Len(t, comments, 2)
	assert.Equal(t, "Great!", comments[0]["body"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasOne tests lazy loading of has-one
func TestLoadSingleHasOne(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	// Create profile schema
	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"bio":     {Name: "bio", Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true}},
	}
	schemas["Profile"] = profileSchema

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasOne,
		TargetResource: "Profile",
		FieldName:      "profile",
		ForeignKey:     "user_id",
		Nullable:       true,
	}

	loader := NewLoader(db, schemas)

	// Mock the query
	mock.ExpectQuery(`SELECT \* FROM profiles WHERE user_id = \$1 LIMIT 1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "bio"}).
				AddRow("profile-1", "user-1", "Software engineer"),
		)

	ctx := context.Background()

	profile, err := loader.loadSingleHasOne(ctx, "user-1", rel, schemas["User"])
	require.NoError(t, err)
	require.NotNil(t, profile)

	assert.Equal(t, "Software engineer", profile["bio"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasManyThrough tests lazy loading of has-many-through
func TestLoadSingleHasManyThrough(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Mock the query
	mock.ExpectQuery(`SELECT t\.\* FROM tags t INNER JOIN post_tags j`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("tag-1", "golang").
				AddRow("tag-2", "web"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["tags"]

	tags, err := loader.loadSingleHasManyThrough(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)

	assert.Len(t, tags, 2)
	assert.Equal(t, "golang", tags[0]["name"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestEagerLoadUnknownRelationship tests error handling for unknown relationships
func TestEagerLoadUnknownRelationship(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	ctx := context.Background()

	err := loader.EagerLoad(ctx, posts, schemas["Post"], []string{"nonexistent"})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownRelationship)
}

// TestEagerLoadEmptyRecords tests loading with empty record set
func TestEagerLoadEmptyRecords(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{}

	ctx := context.Background()

	err := loader.EagerLoad(ctx, posts, schemas["Post"], []string{"author"})
	assert.NoError(t, err) // Should not error on empty records
}

// TestExtractNestedRecords tests extracting nested records
func TestExtractNestedRecords(t *testing.T) {
	// Test belongs-to
	rel := &schema.Relationship{
		Type:      schema.RelationshipBelongsTo,
		FieldName: "author",
	}

	records := []map[string]interface{}{
		{
			"id": "post-1",
			"author": map[string]interface{}{
				"id":   "user-1",
				"name": "Alice",
			},
		},
		{
			"id": "post-2",
			"author": map[string]interface{}{
				"id":   "user-1", // Same author - should deduplicate
				"name": "Alice",
			},
		},
	}

	nested := extractNestedRecords(records, rel)
	assert.Len(t, nested, 1) // Should deduplicate

	// Test has-many
	rel.Type = schema.RelationshipHasMany
	rel.FieldName = "comments"

	records = []map[string]interface{}{
		{
			"id": "post-1",
			"comments": []map[string]interface{}{
				{"id": "comment-1", "body": "Great"},
				{"id": "comment-2", "body": "Thanks"},
			},
		},
	}

	nested = extractNestedRecords(records, rel)
	assert.Len(t, nested, 2)
}

// TestLoadSingleBelongsTo tests lazy loading of belongs-to
func TestLoadSingleBelongsTo(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Mock the query
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]

	user, err := loader.loadSingleBelongsTo(ctx, "user-1", rel, schemas["Post"])
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "Alice", user["name"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleBelongsToNotFound tests lazy loading when record not found
func TestLoadSingleBelongsToNotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Mock the query returning no rows
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]
	rel.Nullable = true

	user, err := loader.loadSingleBelongsTo(ctx, "nonexistent", rel, schemas["Post"])
	require.NoError(t, err)
	assert.Nil(t, user) // Nullable, so nil is OK

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasManyEmpty tests lazy loading of empty has-many
func TestLoadSingleHasManyEmpty(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Mock the query returning no rows
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = \$1`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	comments, err := loader.loadSingleHasMany(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)

	assert.NotNil(t, comments)
	assert.Len(t, comments, 0) // Empty slice, not nil

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestToTableName tests table name conversion
func TestToTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"Post", "posts"},
		{"Category", "categories"},
		{"Box", "boxes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTableName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewLazyRelation tests creating lazy relations
func TestNewLazyRelation(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := schemas["Post"].Relationships["author"]
	ctx := context.Background()

	lazyRel := NewLazyRelation(loader, ctx, "user-1", rel, schemas["Post"])

	assert.NotNil(t, lazyRel)
	assert.False(t, lazyRel.IsLoaded())
}

// TestLoadContextDecrementDepth tests depth decrementing
func TestLoadContextDecrementDepth(t *testing.T) {
	ctx := NewLoadContext(5)

	ctx.IncrementDepth()
	ctx.IncrementDepth()

	ctx.DecrementDepth()

	// Should be back to 1
	ctx.IncrementDepth()
	assert.NoError(t, ctx.IncrementDepth())
}
