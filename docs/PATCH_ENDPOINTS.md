# PATCH Endpoints for Partial Resource Updates

## Overview

Conduit now supports PATCH endpoints for all resources, enabling partial updates to reduce payload sizes by 60-90% and follow RESTful best practices.

## Features

### Core Functionality

1. **Partial Field Updates**: Send only the fields you want to update (1 to N fields)
2. **Merge with Existing Data**: PATCH automatically loads the existing resource and merges your changes
3. **Full Validation**: The merged result is validated just like a full PUT update
4. **Complete Response**: Returns the complete updated resource after the operation
5. **PUT Unchanged**: Existing PUT endpoints continue to work as full replacements

### Safety Features

6. **Read-Only Field Protection**: Attempting to update `id`, `created_at`, or `updated_at` returns a 400 error
7. **Unknown Field Rejection**: Sending fields that don't exist returns a 400 error with a list of valid fields
8. **Empty PATCH Rejection**: Sending an empty JSON object (`{}`) returns a 400 error
9. **Type Validation**: Type mismatches return 400 with clear error messages
10. **Required Field Validation**: Required fields are validated in the merged resource
11. **Nullable Field Support**: Nullable fields can be explicitly set to `null`

### Error Handling

12. **404 Not Found**: When the resource doesn't exist
13. **400 Bad Request**: For invalid JSON, type mismatches, read-only fields, unknown fields, or empty PATCH
14. **422 Unprocessable Entity**: For validation failures on the merged resource

## Usage Examples

### Basic PATCH Request

Update only the status of a post:

```bash
# Legacy JSON format
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/json" \
  -d '{"status": "published"}'

# JSON:API format
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/vnd.api+json" \
  -H "Accept: application/vnd.api+json" \
  -d '{
    "data": {
      "type": "posts",
      "id": "abc-123",
      "attributes": {
        "status": "published"
      }
    }
  }'
```

### Update Multiple Fields

Update both title and body:

```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Title",
    "body": "Updated content for the post"
  }'
```

### Set Nullable Field to Null

Explicitly set a nullable field to null:

```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/json" \
  -d '{"published_at": null}'
```

## Comparison: PUT vs PATCH

### PUT (Full Update)

**Request:**
```bash
curl -X PUT http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-456",
    "title": "My Post",
    "slug": "my-post",
    "body": "Full content here...",
    "status": "published",
    "published_at": "2025-11-02T14:00:00Z"
  }'
```

- **Payload Size**: ~200 bytes
- **Behavior**: Replaces the entire resource
- **Requirements**: All required fields must be present

### PATCH (Partial Update)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "published",
    "published_at": "2025-11-02T14:00:00Z"
  }'
```

- **Payload Size**: ~60 bytes (70% reduction)
- **Behavior**: Updates only the specified fields
- **Requirements**: Only the fields you want to change

## Error Examples

### Read-Only Field Error (400)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -d '{"id": "different-id"}'
```

**Response:**
```json
{
  "error": "field 'id' is read-only and cannot be updated"
}
```

### Unknown Field Error (400)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -d '{"invalid_field": "value"}'
```

**Response:**
```json
{
  "error": "unknown field 'invalid_field': valid fields are: [title, body, status, published_at, ...]"
}
```

### Empty PATCH Error (400)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -d '{}'
```

**Response:**
```json
{
  "error": "empty PATCH request: no fields provided"
}
```

### Validation Error (422)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/abc-123 \
  -d '{"title": "abc"}'  # Title too short (min: 5)
```

**Response:**
```json
{
  "error": "validation failed: title must be at least 5 characters"
}
```

### Not Found Error (404)

**Request:**
```bash
curl -X PATCH http://localhost:8080/api/posts/nonexistent-id \
  -d '{"status": "published"}'
```

**Response:**
```json
{
  "error": "Not found"
}
```

## Implementation Details

### Generated Code

For each resource, Conduit generates:

1. **PATCH Handler** (`Patch{Resource}Handler`): HTTP handler that accepts partial JSON
2. **Patch Method** (`{resource}.Patch()`): Model method that performs the merge and validation
3. **Route Registration**: `r.Patch("/{resources}/{id}", Patch{Resource}Handler(db))`

### Validation Flow

1. Load existing resource by ID (404 if not found)
2. Parse partial JSON into map to identify provided fields
3. Reject empty PATCH requests (400)
4. Validate no read-only fields are present (400)
5. Validate no unknown fields are present (400)
6. Merge partial data with existing resource
7. Validate the merged result (422 if validation fails)
8. Update auto_update fields (e.g., `updated_at`)
9. Save to database
10. Return complete updated resource

### Route Registration

Both PUT and PATCH routes are registered for each resource:

```go
r.Put("/posts/{id}", UpdatePostHandler(db))
r.Patch("/posts/{id}", PatchPostHandler(db))
```

## Content Negotiation

PATCH endpoints support both:
- **Legacy JSON** (`application/json`)
- **JSON:API** (`application/vnd.api+json`)

The response format matches the request format based on the `Accept` and `Content-Type` headers.

## Resource Definition Example

```conduit
resource Post {
  id: uuid! @primary @auto
  user_id: uuid!
  title: string! @min(5) @max(200)
  slug: string! @unique
  body: text! @min(100)
  status: enum ["draft", "published"]! @default("draft")
  published_at: timestamp?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
```

This resource automatically gets:
- `PUT /posts/{id}` - Full update
- `PATCH /posts/{id}` - Partial update

## Benefits

1. **Reduced Bandwidth**: 60-90% smaller payloads for partial updates
2. **Better UX**: Update single fields without sending entire resource
3. **RESTful**: Follows HTTP semantic conventions
4. **Type Safe**: Full validation of merged result
5. **Secure**: Read-only field protection built-in
6. **Clear Errors**: Detailed error messages for debugging

## Testing

The implementation includes comprehensive tests in:
- `/internal/compiler/codegen/patch_test.go` - Code generation tests
- Verify PATCH handler generation
- Verify Patch method generation
- Verify route registration

## Migration Guide

Existing code using PUT endpoints continues to work without changes. To adopt PATCH:

1. **Frontend**: Change `PUT` to `PATCH` in your HTTP requests
2. **Payload**: Send only the fields you want to update
3. **Handle Errors**: Update error handling for new 400 cases (read-only, unknown fields, empty PATCH)

## Known Limitations

### Concurrent Update Risk
PATCH operations use read-modify-write without locking. In high-concurrency
scenarios, concurrent PATCH requests to the same resource may result in lost
updates. Consider using PUT for critical updates or implementing optimistic
locking in a future enhancement.

**Example of potential issue:**
- Request A: PATCH {views: 100}
- Request B: PATCH {status: "published"}
- If these happen concurrently, one update may be lost

**Mitigation strategies:**
- Use PUT for critical updates that need strong consistency
- Implement optimistic locking with ETags (future enhancement)
- Use database-level row locking (future enhancement)

## Notes

- PATCH always loads the existing resource first (one SELECT query)
- The merged resource is validated as if it were a full PUT
- Auto-update fields (e.g., `updated_at`) are automatically updated
- Hooks (`@before update`, `@after update`) are triggered for PATCH operations
- PATCH uses the same `UPDATE` SQL query as PUT (updates all fields)
