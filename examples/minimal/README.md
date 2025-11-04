# Minimal Conduit Example

**Difficulty:** Beginner
**Time:** 5 minutes

The simplest possible Conduit application. This example demonstrates the absolute minimum code needed to create a working REST API.

## What You'll Learn

- Basic resource definition syntax
- Required field types with `!` (non-nullable)
- Primary key with `@primary` and `@auto`
- Automatic timestamps with `@auto`
- How to build and run a Conduit application

## Quick Start

```bash
# Navigate to this directory
cd examples/minimal

# Build the application
conduit build

# The build creates a binary and generates:
# - REST API endpoints (GET, POST, PUT, DELETE /items)
# - Database schema
# - Type-safe validation
```

## What's Inside

### `app/resources/item.cdt`

A single resource with just 3 fields - the minimum for a working application:

```conduit
resource Item {
  id: uuid! @primary @auto
  name: string!
  created_at: timestamp! @auto
}
```

**Breaking it down:**

- `resource Item { }` - Defines a new resource (like a database table + API endpoints)
- `id: uuid! @primary @auto` - Primary key, auto-generated UUID
  - `uuid!` means "required UUID" (the `!` means it can never be null)
  - `@primary` marks this as the primary key
  - `@auto` means Conduit generates the value automatically
- `name: string!` - A required text field (must be provided, cannot be null)
- `created_at: timestamp! @auto` - Automatically set to current time when created

### `conduit.yaml`

Minimal configuration file specifying the project name.

## Generated API

Once built, you get these REST endpoints automatically:

- `GET /items` - List all items
- `POST /items` - Create a new item
- `GET /items/:id` - Get a specific item
- `PUT /items/:id` - Update an item
- `DELETE /items/:id` - Delete an item

## Try It Out

You can test the generated API (note: this requires setting up a database connection in `conduit.yaml`):

```bash
# Create an item
curl -X POST http://localhost:3000/items \
  -H "Content-Type: application/json" \
  -d '{"name": "My First Item"}'

# List items
curl http://localhost:3000/items
```

## Key Concepts

### Explicit Nullability

Every type in Conduit must specify whether it can be null:
- `string!` - Required (never null)
- `string?` - Optional (can be null)

This eliminates an entire class of null pointer bugs.

### Namespaced Built-ins

All built-in functions are namespaced to prevent ambiguity:
- `String.slugify()` not `slugify()`
- `Time.now()` not `now()`
- `UUID.generate()` not `uuid()`

This makes it clear where functions come from and prevents LLMs from hallucinating function names.

> **Note:** Only functions listed in ROADMAP.md under "What Works Today" are currently implemented. Most String, Time, Array, and other stdlib functions are planned but not yet available. Check ROADMAP.md for the complete list of working functions.

### Automatic Code Generation

From this simple resource definition, Conduit generates:
- Database schema with proper types and constraints
- REST API handlers with proper routing
- Input validation
- JSON serialization/deserialization
- Error handling

## Next Steps

Ready for more? Try these examples:

1. **Add more fields** - Add `description: string?` (optional field)
2. **Add validation** - Add `@min(3) @max(100)` to the name field
3. **Add a default** - Add `status: string! @default("pending")`
4. **Try the todo-app example** - See `examples/todo-app/` for CRUD with validation

## Learn More

- **GETTING-STARTED.md** - Full getting started guide
- **LANGUAGE-SPEC.md** - Complete language reference
- **examples/todo-app/** - Next level example with validation
