# Phase 3: Compound Documents Implementation Plan

## Status: NOT IMPLEMENTED ⚠️

**Reason:** Requires significant foundational changes to model generation and relationship loading infrastructure.

## What's Been Completed

✅ Query parameter parsing for `?include=`
✅ Sparse fieldsets (`?fields[type]=field1,field2`)
✅ Filtering (`?filter[field]=value`)
✅ Sorting (`?sort=-field1,field2`)
✅ Handler codegen updated to use Phase 3 features

## What's Required for Compound Documents

### 1. Model Generation Changes

**Current State:**
- Structs are generated in `/internal/compiler/codegen/struct.go`
- No relationship fields in generated structs
- Relationships defined in schema but not materialized

**Required Changes:**
```go
type Post struct {
    ID        int64     `jsonapi:"primary,posts" db:"id" json:"id"`
    Title     string    `jsonapi:"attr,title" db:"title" json:"title"`
    AuthorID  int64     `jsonapi:"attr,author_id" db:"author_id" json:"author_id"`

    // ADD: Relationship fields with jsonapi:"relation,..." tags
    Author    *User     `jsonapi:"relation,author"`
    Comments  []*Comment `jsonapi:"relation,comments"`
}
```

**Implementation Steps:**
1. Update `generateStruct()` to analyze relationships from AST
2. Add relationship fields based on relationship type:
   - BelongsTo/HasOne → pointer to target type
   - HasMany → slice of pointers
3. Add proper jsonapi relation tags
4. Import required model types

### 2. Relationship Loading Methods

**Location:** `/internal/orm/codegen/query_methods.go` or new file

**Required Methods:**

```go
// FindAllPostWithIncludes loads posts with specified relationships
func FindAllPostWithIncludes(ctx context.Context, db *sql.DB,
    limit, offset int, includes []string) ([]*Post, error) {

    // Load primary resources
    posts, err := FindAllPost(ctx, db, limit, offset)
    if err != nil {
        return nil, err
    }

    // Load relationships based on includes
    if contains(includes, "author") {
        if err := loadPostAuthors(ctx, db, posts); err != nil {
            return nil, err
        }
    }

    if contains(includes, "comments") {
        if err := loadPostComments(ctx, db, posts); err != nil {
            return nil, err
        }
    }

    return posts, nil
}

// loadPostAuthors loads author for each post (BelongsTo)
func loadPostAuthors(ctx context.Context, db *sql.DB, posts []*Post) error {
    // Extract author IDs
    authorIDs := make([]int64, 0, len(posts))
    for _, post := range posts {
        if post.AuthorID != 0 {
            authorIDs = append(authorIDs, post.AuthorID)
        }
    }

    if len(authorIDs) == 0 {
        return nil
    }

    // Batch load authors (prevents N+1)
    query := "SELECT * FROM users WHERE id IN (?)"
    // Use sqlx or similar for IN clause expansion
    authors, err := FindUsersByIDs(ctx, db, authorIDs)
    if err != nil {
        return err
    }

    // Map authors to posts
    authorMap := make(map[int64]*User)
    for _, author := range authors {
        authorMap[author.ID] = author
    }

    for _, post := range posts {
        if author, ok := authorMap[post.AuthorID]; ok {
            post.Author = author
        }
    }

    return nil
}

// loadPostComments loads comments for each post (HasMany)
func loadPostComments(ctx context.Context, db *sql.DB, posts []*Post) error {
    // Extract post IDs
    postIDs := make([]int64, len(posts))
    for i, post := range posts {
        postIDs[i] = post.ID
    }

    // Batch load comments
    query := "SELECT * FROM comments WHERE post_id IN (?)"
    comments, err := FindCommentsByPostIDs(ctx, db, postIDs)
    if err != nil {
        return err
    }

    // Group comments by post_id
    commentsByPost := make(map[int64][]*Comment)
    for _, comment := range comments {
        commentsByPost[comment.PostID] = append(
            commentsByPost[comment.PostID], comment)
    }

    // Assign to posts
    for _, post := range posts {
        post.Comments = commentsByPost[post.ID]
    }

    return nil
}
```

### 3. Handler Integration

**Location:** `/internal/compiler/codegen/handlers.go`

**Required Changes in `generateListHandler()`:**

```go
// CURRENT (TODO comment exists):
// TODO: Phase 3 - Load relationships if includes is not empty

// REPLACE WITH:
var results interface{}
if len(includes) > 0 {
    // Validate includes against valid relationships
    validRelationships := []string{/* from resource schema */}
    for _, inc := range includes {
        if !contains(validRelationships, inc) {
            respondWithError(w,
                fmt.Sprintf("Invalid include: %s", inc),
                http.StatusBadRequest)
            return
        }
    }

    // Load with relationships
    results, err = models.FindAll{Resource}WithIncludes(ctx, db, limit, offset, includes)
    if err != nil {
        respondWithError(w,
            fmt.Sprintf("Failed to load relationships: %v", err),
            http.StatusInternalServerError)
        return
    }
} else {
    // Load without relationships (existing path)
    results, err = models.FindAll{Resource}(ctx, db, limit, offset)
    if err != nil {
        respondWithError(w,
            fmt.Sprintf("Failed to list: %v", err),
            http.StatusInternalServerError)
        return
    }
}
```

### 4. DataDog/jsonapi Library Usage

**Good News:** Library handles included array automatically!

When relationship fields are populated and tagged with `jsonapi:"relation,..."`:
- Library automatically extracts relationships into `included` array
- Library automatically deduplicates resources
- Library handles circular references

**No additional code needed** once relationships are loaded.

### 5. Testing Requirements

**Unit Tests:**
- `loadPostAuthors()` correctly batches and maps
- `loadPostComments()` correctly groups by post_id
- Multiple includes work together
- Invalid relationship names return errors

**Integration Tests:**
- `GET /posts?include=author` returns author in included
- `GET /posts?include=author,comments` returns both
- `GET /posts/123?include=author` works for single resource
- No N+1 queries (verify with SQL logging)

**Performance Tests:**
- Compound document with 100 posts + authors < 100ms
- Query count for ?include=author,comments <= 3 queries:
  1. SELECT posts
  2. SELECT authors (batch)
  3. SELECT comments (batch)

## Implementation Estimate

**If starting fresh:** 2-3 days

1. Model generation changes: 0.5 days
2. Relationship loading infrastructure: 1 day
3. Handler integration: 0.5 days
4. Testing: 0.5-1 days

## Why Not Implemented Now

1. **Scope:** Requires changes to 3 different code generation systems
2. **Complexity:** Each relationship type needs custom loading logic
3. **Risk:** Changes affect all generated models, not just Phase 3
4. **MVP Approach:** Core Phase 3 features (filtering, sorting, sparse fieldsets) are complete
5. **Uncertainty:** Ticket marked this as "HIGH UNCERTAINTY" at 1.5 days

## Recommendation

**Option 1:** Create separate ticket "CON-73: JSON:API Compound Documents"
- Allows focused implementation
- Can be properly estimated after seeing Phase 3 work
- Reduces risk to existing functionality

**Option 2:** Implement in current ticket
- Requires additional 2-3 days
- Higher risk due to model generation changes
- Should be reviewed carefully before merging

## Usage Example (When Implemented)

```bash
# Request posts with author included
GET /api/posts?include=author

# Response:
{
  "data": [
    {
      "type": "posts",
      "id": "1",
      "attributes": {
        "title": "Hello World"
      },
      "relationships": {
        "author": {
          "data": {"type": "users", "id": "123"}
        }
      }
    }
  ],
  "included": [
    {
      "type": "users",
      "id": "123",
      "attributes": {
        "name": "John Doe",
        "email": "john@example.com"
      }
    }
  ]
}
```

## References

- DataDog/jsonapi relation docs: https://pkg.go.dev/github.com/DataDog/jsonapi#hdr-Relationships
- Existing relationship codegen: `/internal/orm/codegen/relationships.go`
- Schema relationship types: `/internal/orm/schema/relationships.go`
