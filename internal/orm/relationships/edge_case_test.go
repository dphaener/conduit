package relationships

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadSingleInvalidRelationType tests error handling for invalid relationship types
func TestLoadSingleInvalidRelationType(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := &schema.Relationship{
		Type:           schema.RelationType(999), // Invalid type
		TargetResource: "User",
		FieldName:      "invalid",
	}

	ctx := context.Background()

	_, err := loader.LoadSingle(ctx, "user-1", rel, schemas["Post"])
	assert.ErrorIs(t, err, ErrInvalidRelationType)
}

// TestLoadRelationshipInvalidType tests error handling in loadRelationship
func TestLoadRelationshipInvalidType(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := &schema.Relationship{
		Type:           schema.RelationType(999),
		TargetResource: "User",
		FieldName:      "invalid",
	}

	records := []map[string]interface{}{
		{"id": "post-1"},
	}

	ctx := context.Background()

	err := loader.loadRelationship(ctx, records, rel, schemas["Post"])
	assert.ErrorIs(t, err, ErrInvalidRelationType)
}

// TestLoadHasManyWithOrderBy tests has-many with order by clause
func TestLoadHasManyWithOrderBy(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	// Set order by
	rel := schemas["Post"].Relationships["comments"]
	rel.OrderBy = "created_at DESC"

	// Mock should include ORDER BY
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\) ORDER BY created_at DESC`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-2", "Second", "post-1").
				AddRow("comment-1", "First", "post-1"),
		)

	ctx := context.Background()

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	comments := posts[0]["comments"].([]map[string]interface{})
	assert.Len(t, comments, 2)
	assert.Equal(t, "Second", comments[0]["body"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasManyWithOrderBy tests lazy loading has-many with order by
func TestLoadSingleHasManyWithOrderBy(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := schemas["Post"].Relationships["comments"]
	rel.OrderBy = "created_at ASC"

	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = \$1 ORDER BY created_at ASC`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", "First", "post-1"),
		)

	ctx := context.Background()

	comments, err := loader.loadSingleHasMany(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)
	assert.Len(t, comments, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadHasManyThroughWithOrderBy tests has-many-through with order by
func TestLoadHasManyThroughWithOrderBy(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	rel := schemas["Post"].Relationships["tags"]
	rel.OrderBy = "name ASC"

	mock.ExpectQuery(`SELECT t\.\*, j\.post_id as __parent_id .* ORDER BY name ASC`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "__parent_id"}).
				AddRow("tag-1", "a-tag", "post-1"),
		)

	ctx := context.Background()

	err := loader.loadHasManyThrough(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	tags := posts[0]["tags"].([]map[string]interface{})
	assert.Len(t, tags, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasManyThroughWithOrderBy tests lazy loading has-many-through with order by
func TestLoadSingleHasManyThroughWithOrderBy(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := schemas["Post"].Relationships["tags"]
	rel.OrderBy = "name DESC"

	mock.ExpectQuery(`SELECT t\.\* FROM tags t .* ORDER BY name DESC`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("tag-1", "golang"),
		)

	ctx := context.Background()

	tags, err := loader.loadSingleHasManyThrough(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)
	assert.Len(t, tags, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestEagerLoadWithContextDepthTracking tests depth tracking in nested loading
func TestEagerLoadWithContextDepthTracking(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post", "author_id": "user-1"},
	}

	mock.ExpectQuery(`SELECT \* FROM users`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	ctx := context.Background()
	loadCtx := NewLoadContext(5)

	err := loader.EagerLoadWithContext(ctx, posts, schemas["Post"], []string{"author"}, loadCtx)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadBelongsToMissingSchema tests error when target schema not found
func TestLoadBelongsToMissingSchema(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "author_id": "user-1"},
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "NonexistentResource",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	ctx := context.Background()

	err := loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource")
}

// TestLoadHasManyMissingSchema tests error when target schema not found
func TestLoadHasManyMissingSchema(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1"},
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasMany,
		TargetResource: "NonexistentResource",
		FieldName:      "items",
		ForeignKey:     "post_id",
	}

	ctx := context.Background()

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource")
}

// TestExtractNestedRecordsWithNilData tests extracting when relationship data is nil
func TestExtractNestedRecordsWithNilData(t *testing.T) {
	rel := &schema.Relationship{
		Type:      schema.RelationshipBelongsTo,
		FieldName: "author",
	}

	records := []map[string]interface{}{
		{"id": "post-1", "author": nil},
		{"id": "post-2"}, // Missing field entirely
	}

	nested := extractNestedRecords(records, rel)
	assert.Len(t, nested, 0)
}

// TestExtractNestedRecordsWithoutID tests extracting records without IDs
func TestExtractNestedRecordsWithoutID(t *testing.T) {
	rel := &schema.Relationship{
		Type:      schema.RelationshipHasMany,
		FieldName: "comments",
	}

	records := []map[string]interface{}{
		{
			"id": "post-1",
			"comments": []map[string]interface{}{
				{"body": "No ID"},
			},
		},
	}

	nested := extractNestedRecords(records, rel)
	assert.Len(t, nested, 0) // Should skip records without IDs
}

// TestParseIncludeEdgeCases tests parseInclude with edge cases
func TestParseIncludeEdgeCases(t *testing.T) {
	// Empty string
	rel, nested := parseInclude("")
	assert.Equal(t, "", rel)
	assert.Nil(t, nested)

	// Just dots
	rel, nested = parseInclude("...")
	assert.Equal(t, "", rel)
	assert.Equal(t, []string{".."}, nested)
}

// TestLoadSingleBelongsToNonNullable tests error when nullable=false and not found
func TestLoadSingleBelongsToNonNullable(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]
	rel.Nullable = false

	_, err := loader.loadSingleBelongsTo(ctx, "nonexistent", rel, schemas["Post"])
	assert.ErrorIs(t, err, ErrNoRecords)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasOneNonNullable tests error when nullable=false and not found
func TestLoadSingleHasOneNonNullable(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
	}
	schemas["Profile"] = profileSchema

	loader := NewLoader(db, schemas)

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasOne,
		TargetResource: "Profile",
		FieldName:      "profile",
		ForeignKey:     "user_id",
		Nullable:       false,
	}

	mock.ExpectQuery(`SELECT \* FROM profiles WHERE user_id = \$1 LIMIT 1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id"}),
		)

	ctx := context.Background()

	_, err := loader.loadSingleHasOne(ctx, "user-1", rel, schemas["User"])
	assert.ErrorIs(t, err, ErrNoRecords)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadBelongsToQueryError tests database query error handling
func TestLoadBelongsToQueryError(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "author_id": "user-1"},
	}

	mock.ExpectQuery(`SELECT \* FROM users`).
		WillReturnError(assert.AnError)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["author"]

	err := loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	assert.Error(t, err)
}

// TestScanRowsWithByteConversion tests byte array to string conversion
func TestScanRowsWithByteConversion(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1"},
	}

	// Return data with byte arrays
	mock.ExpectQuery(`SELECT \* FROM comments`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", []byte("Text as bytes"), "post-1"),
		)

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	comments := posts[0]["comments"].([]map[string]interface{})
	assert.Equal(t, "Text as bytes", comments[0]["body"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
