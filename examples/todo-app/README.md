# Todo App Example

**Difficulty:** Beginner
**Time:** 10 minutes

A basic todo application demonstrating CRUD operations with field validation and constraints. This example shows how to use decorators like `@min`, `@max`, and `@default` to enforce data quality.

## What You'll Learn

- Field validation with `@min` and `@max`
- Default values with `@default`
- Optional fields with `?` (nullable)
- Boolean fields for flags
- Lifecycle hooks with `@before` and `@after`
- Auto-updating timestamps with `@auto_update`

## Quick Start

```bash
# Navigate to this directory
cd examples/todo-app

# Build the application
conduit build

# The build creates:
# - REST API endpoints for todos
# - Database schema with constraints
# - Validation logic
# - Slug generation from title
```

## What's Inside

### `app/resources/todo.cdt`

A todo resource with validation, enums, and lifecycle hooks:

```conduit
resource Todo {
  id: uuid! @primary @auto
  title: string! @min(3) @max(200)
  slug: string! @unique
  description: string?
  status: string! @default("pending")
  priority: int! @default(3)
  completed: bool! @default(false)
  due_date: timestamp?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.slug = String.slugify(self.title)
  }

  @before update {
    self.slug = String.slugify(self.title)
  }
}
```

**Key Features:**

1. **Field Validation**
   - `title: string! @min(3) @max(200)` - Title must be between 3-200 characters
   - Validation works on string fields with character length constraints

2. **Boolean Fields**
   - `completed: bool! @default(false)` - Flag for marking todos as complete
   - Defaults to false for new todos

3. **Default Values**
   - `@default("pending")` - New todos start as "pending"
   - `@default(3)` - Default priority is 3 (medium)

4. **Optional Fields**
   - `description: string?` - Description is optional (can be null)
   - `due_date: timestamp?` - Due date is optional

5. **Lifecycle Hooks**
   - `@before create` - Runs before creating a todo (generates slug)
   - `@before update` - Runs before updating a todo (updates slug if title changed)

6. **Automatic Timestamps**
   - `created_at: timestamp! @auto` - Set once when created
   - `updated_at: timestamp! @auto_update` - Updated every time the record changes

### `conduit.yaml`

Configuration file specifying project name and version.

## Generated API

The following REST endpoints are automatically generated:

### List Todos
```bash
GET /todos
```

Response:
```json
{
  "data": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Write documentation",
      "slug": "write-documentation",
      "description": "Document the API endpoints",
      "status": "pending",
      "priority": 3,
      "completed": false,
      "due_date": null,
      "created_at": "2025-11-03T10:00:00Z",
      "updated_at": "2025-11-03T10:00:00Z"
    }
  ]
}
```

### Create Todo
```bash
POST /todos
Content-Type: application/json

{
  "title": "Write documentation",
  "description": "Document the API endpoints",
  "priority": 4,
  "due_date": "2025-11-10T23:59:59Z"
}
```

The slug is automatically generated from the title, and status defaults to "pending".

### Get Todo
```bash
GET /todos/:id
```

### Update Todo
```bash
PUT /todos/:id
Content-Type: application/json

{
  "status": "completed"
}
```

### Delete Todo
```bash
DELETE /todos/:id
```

## Try It Out

Test the API with these examples:

```bash
# Create a todo (minimal)
curl -X POST http://localhost:3000/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Conduit"}'

# Create a todo (with all fields)
curl -X POST http://localhost:3000/todos \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Build a REST API",
    "description": "Use Conduit to build a production API",
    "priority": 5,
    "due_date": "2025-12-31T23:59:59Z"
  }'

# List all todos
curl http://localhost:3000/todos

# Mark as completed
curl -X PUT http://localhost:3000/todos/YOUR_TODO_ID \
  -H "Content-Type: application/json" \
  -d '{"completed": true, "status": "completed"}'

# These will fail validation (too short):
curl -X POST http://localhost:3000/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Hi"}'
# Error: title must be at least 3 characters

# This will fail validation (missing required title):
curl -X POST http://localhost:3000/todos \
  -H "Content-Type: application/json" \
  -d '{"description": "A todo without a title"}'
# Error: title is required
```

## Validation in Action

Conduit enforces validation at multiple levels:

1. **Type Level** - `string!` vs `string?`
2. **Constraint Level** - `@min(3)`, `@max(200)`
3. **Enum Level** - Must be one of the specified values
4. **Database Level** - CHECK constraints in schema

This means invalid data is caught early and consistently.

## Concepts Explained

### @min and @max

These decorators enforce length constraints:
- For strings: minimum/maximum character count
- Generated as database CHECK constraints
- Note: @min/@max for integer value ranges is not yet fully supported

### @default

Provides a default value when field is not specified:
- Must be a literal value or constant expression
- Used when creating new records
- Becomes database DEFAULT in schema

### @unique

Ensures field value is unique across all records:
- Creates a database UNIQUE index
- Enforces uniqueness at database level
- Returns validation error if violated

### @auto and @auto_update

Automatic timestamp management:
- `@auto` - Set once when created (uses Time.now())
- `@auto_update` - Updated every time record changes
- No manual management needed

### Lifecycle Hooks

Execute code before or after operations:
- `@before create/update/delete` - Runs before database operation
- `@after create/update/delete` - Runs after database operation
- Can modify fields (in @before hooks)
- Can trigger side effects (in @after hooks)

### Boolean Fields

Simple true/false flags:
- Perfect for completion states, feature flags, etc.
- Stored efficiently in database
- Clear and explicit semantics
- Can have defaults with `@default(true)` or `@default(false)`

## Next Steps

Now that you understand validation and CRUD, try:

1. **Add more fields** - Add `completed_at: timestamp?` that's set when status becomes "completed"
2. **Add relationships** - Create a `User` resource and add `owner: User!` to Todo
3. **Add computed fields** - Calculate if todo is overdue
4. **Explore hooks** - Send notifications when todos are completed
5. **Add constraints** - Ensure `due_date` is in the future

## Common Patterns

### Slug Generation
```conduit
@before create {
  self.slug = String.slugify(self.title)
}
```

> **Note on Advanced Hook Patterns:** Change tracking (like `self.field_changed?`) and complex control flow (if/else statements) are not yet available in hook DSL. Check ROADMAP.md for implementation status of advanced features. The simple assignment pattern shown above is fully supported.

## Learn More

- **examples/minimal/** - Start with the simplest example
- **GETTING-STARTED.md** - Full getting started guide
- **LANGUAGE-SPEC.md** - Complete language reference
