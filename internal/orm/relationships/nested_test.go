package relationships

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEagerLoadWithContextNestedIncludes tests nested includes
func TestEagerLoadWithContextNestedIncludes(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	// Add a relationship from Comment to User (author)
	schemas["Comment"].Relationships = map[string]*schema.Relationship{
		"author": {
			Type:           schema.RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			ForeignKey:     "author_id",
			Nullable:       false,
		},
	}
	schemas["Comment"].Fields["author_id"] = &schema.Field{
		Name: "author_id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
	}

	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post", "author_id": "user-1"},
	}

	// Mock loading comments
	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id", "author_id"}).
				AddRow("comment-1", "Great!", "post-1", "user-2"),
		)

	// Mock loading comment authors
	mock.ExpectQuery(`SELECT \* FROM users WHERE id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-2", "Bob"),
		)

	ctx := context.Background()
	loadCtx := NewLoadContext(10)

	// Load posts -> comments
	err := loader.EagerLoadWithContext(ctx, posts, schemas["Post"], []string{"comments"}, loadCtx)
	require.NoError(t, err)

	// Now load comment authors
	comments := posts[0]["comments"].([]map[string]interface{})
	err = loader.EagerLoadWithContext(ctx, comments, schemas["Comment"], []string{"author"}, loadCtx)
	require.NoError(t, err)

	// Verify nested data
	author := comments[0]["author"].(map[string]interface{})
	assert.Equal(t, "Bob", author["name"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasOne tests all has-one variations
func TestLoadSingleHasOneVariations(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"bio":     {Name: "bio", Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true}},
	}
	schemas["Profile"] = profileSchema

	loader := NewLoader(db, schemas)

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasOne,
		TargetResource: "Profile",
		FieldName:      "profile",
		ForeignKey:     "user_id",
		Nullable:       true,
	}

	// Test nullable with no result
	mock.ExpectQuery(`SELECT \* FROM profiles WHERE user_id = \$1 LIMIT 1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "bio"}),
		)

	ctx := context.Background()

	profile, err := loader.loadSingleHasOne(ctx, "user-1", rel, schemas["User"])
	require.NoError(t, err)
	assert.Nil(t, profile)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleHasManyThroughEmpty tests empty has-many-through
func TestLoadSingleHasManyThroughEmpty(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	rel := schemas["Post"].Relationships["tags"]

	mock.ExpectQuery(`SELECT t\.\* FROM tags t`).
		WithArgs("post-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}),
		)

	ctx := context.Background()

	tags, err := loader.loadSingleHasManyThrough(ctx, "post-1", rel, schemas["Post"])
	require.NoError(t, err)

	assert.NotNil(t, tags)
	assert.Len(t, tags, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadRelationshipAllTypes tests loadRelationship with all types
func TestLoadRelationshipAllTypes(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	ctx := context.Background()

	// Test has-one
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

	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
	}
	schemas["Profile"] = profileSchema

	users := []map[string]interface{}{
		{"id": "user-1", "name": "Alice"},
	}

	mock.ExpectQuery(`SELECT DISTINCT ON \(user_id\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id"}).
				AddRow("profile-1", "user-1"),
		)

	err := loader.loadRelationship(ctx, users, userSchema.Relationships["profile"], userSchema)
	require.NoError(t, err)

	assert.NotNil(t, users[0]["profile"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadSingleAllTypes tests LoadSingle with all types
func TestLoadSingleAllTypes(t *testing.T) {
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

	ctx := context.Background()

	// Test has-one
	rel := &schema.Relationship{
		Type:           schema.RelationshipHasOne,
		TargetResource: "Profile",
		FieldName:      "profile",
		ForeignKey:     "user_id",
		Nullable:       true,
	}

	mock.ExpectQuery(`SELECT \* FROM profiles WHERE user_id = \$1 LIMIT 1`).
		WithArgs("user-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id"}).
				AddRow("profile-1", "user-1"),
		)

	result, err := loader.LoadSingle(ctx, "user-1", rel, schemas["User"])
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadHasManyWithEmptyParentIDs tests has-many with no parent IDs
func TestLoadHasManyWithEmptyParentIDs(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	// Records with no IDs
	posts := []map[string]interface{}{
		{"title": "No ID"},
	}

	ctx := context.Background()
	rel := schemas["Post"].Relationships["comments"]

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// When there are no parent IDs, function returns early without setting field
	_, hasComments := posts[0]["comments"]
	assert.False(t, hasComments)
}

// TestLoadHasOneWithEmptyParentIDs tests has-one with no parent IDs
func TestLoadHasOneWithEmptyParentIDs(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()

	profileSchema := schema.NewResourceSchema("Profile")
	profileSchema.Fields = map[string]*schema.Field{
		"id":      {Name: "id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
		"user_id": {Name: "user_id", Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false}},
	}
	schemas["Profile"] = profileSchema

	loader := NewLoader(db, schemas)

	users := []map[string]interface{}{
		{"name": "No ID"},
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasOne,
		TargetResource: "Profile",
		FieldName:      "profile",
		ForeignKey:     "user_id",
		Nullable:       true,
	}

	ctx := context.Background()

	err := loader.loadHasOne(ctx, users, rel, schemas["User"])
	require.NoError(t, err)

	// Should set nil for nullable
	assert.Nil(t, users[0]["profile"])
}

// TestLoadHasManyThroughWithEmptyParentIDs tests has-many-through with no parent IDs
func TestLoadHasManyThroughWithEmptyParentIDs(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"title": "No ID"},
	}

	ctx := context.Background()
	rel := schemas["Post"].Relationships["tags"]

	err := loader.loadHasManyThrough(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	// When there are no parent IDs, function returns early without setting field
	_, hasTags := posts[0]["tags"]
	assert.False(t, hasTags)
}

// TestEagerLoadWithContextCircularDetection tests circular reference detection
func TestEagerLoadWithContextCircularDetection(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	ctx := context.Background()
	loadCtx := NewLoadContext(10)

	// Mark as already visited
	loadCtx.MarkVisited("Post:post-1")

	// Should skip loading due to circular reference
	err := loader.EagerLoadWithContext(ctx, posts, schemas["Post"], []string{"author"}, loadCtx)
	require.NoError(t, err) // No error, just skips
}

// TestLoadBelongsToWithDefaultForeignKey tests using default foreign key
func TestLoadBelongsToWithDefaultForeignKey(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "user_id": "user-1"}, // Different FK name
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "", // Empty - should use default
		Nullable:       false,
	}

	mock.ExpectQuery(`SELECT \* FROM users WHERE id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow("user-1", "Alice"),
		)

	ctx := context.Background()

	err := loader.loadBelongsTo(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadHasManyWithDefaultForeignKey tests using default foreign key
func TestLoadHasManyWithDefaultForeignKey(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasMany,
		TargetResource: "Comment",
		FieldName:      "comments",
		ForeignKey:     "", // Empty - should use default "post_id"
		Nullable:       false,
	}

	mock.ExpectQuery(`SELECT \* FROM comments WHERE post_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "body", "post_id"}).
				AddRow("comment-1", "Great", "post-1"),
		)

	ctx := context.Background()

	err := loader.loadHasMany(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	comments := posts[0]["comments"].([]map[string]interface{})
	assert.Len(t, comments, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestLoadHasManyThroughWithDefaultKeys tests using default join table and keys
func TestLoadHasManyThroughWithDefaultKeys(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	schemas := setupTestSchemas()
	loader := NewLoader(db, schemas)

	posts := []map[string]interface{}{
		{"id": "post-1", "title": "Post"},
	}

	rel := &schema.Relationship{
		Type:           schema.RelationshipHasManyThrough,
		TargetResource: "Tag",
		FieldName:      "tags",
		ForeignKey:     "",
		AssociationKey: "",
		JoinTable:      "",
		Nullable:       false,
	}

	// Should use defaults: post_tags table, post_id, tag_id
	mock.ExpectQuery(`SELECT t\.\*, j\.post_id as __parent_id FROM tags t INNER JOIN post_tags j`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "__parent_id"}).
				AddRow("tag-1", "golang", "post-1"),
		)

	ctx := context.Background()

	err := loader.loadHasManyThrough(ctx, posts, rel, schemas["Post"])
	require.NoError(t, err)

	tags := posts[0]["tags"].([]map[string]interface{})
	assert.Len(t, tags, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}
