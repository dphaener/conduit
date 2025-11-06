# Conduit Language Specification

**Version:** 0.1.0
**Status:** Current Release
**Updated:** 2025-11-05

✅ **This document describes features that WORK TODAY in Conduit v0.1.0**

For planned features not yet implemented, see [FUTURE-VISION.md](FUTURE-VISION.md).

---

## Table of Contents

1. [Overview & Philosophy](#overview--philosophy)
2. [Type System](#type-system)
3. [Resource Syntax](#resource-syntax)
4. [Standard Library](#standard-library)
5. [Lifecycle Hooks](#lifecycle-hooks)
6. [Complete Examples](#complete-examples)

---

## Overview & Philosophy

### Design Principles

**Explicitness over Brevity**
- LLMs don't experience tedium, they experience ambiguity
- Verbose in service of clarity is GOOD
- Ceremonial verbosity is BAD

**Progressive Disclosure**
- Simple things stay simple (3 lines can be a complete resource)
- Complexity is available when needed
- No forced sophistication

**Correctness over Cleverness**
- Structure carries meaning
- Type safety prevents errors
- Explicit error handling

**Zero Ambiguity**
- Every type must specify nullability (`!` vs `?`)
- All built-in functions are namespaced (`String.slugify()`)
- No implicit behavior

### File Extension

Conduit files use the `.cdt` extension:
```
user.cdt
post.cdt
product.cdt
```

---

## Type System

✅ **Status: Fully Implemented**

### Explicit Nullability

**Every type must specify nullability:**

```conduit
type!     // Required, never null
type?     // Optional, can be null
```

**Examples:**
```conduit
// Strings
username: string!         // Required
bio: text?               // Optional

// Numbers
price: float!            // Required
sale_price: float?       // Optional

// Relationships
author: User!            // Required relationship
category: Category?      // Optional relationship

// Arrays
tags: array<uuid>!       // Required array (can be empty)
images: array<string>?   // Optional array (can be null)
```

### Primitive Types

```conduit
// Text
string!               // Variable-length text
string(50)!          // Max length 50
text!                // Long-form text

// Numbers
int!                 // Integer
float!               // Floating point

// Boolean
bool!                // true/false

// Time
timestamp!           // Date and time
date!                // Date only

// Unique
uuid!                // UUID v4

// Special
email!               // Email address (validated)
url!                 // URL (validated)
json!                // JSON data
```

### Structural Types

**Arrays:**
```conduit
array<T>!            // Typed array
array<uuid>!         // Array of UUIDs
array<string>!       // Array of strings
array<int>!          // Array of integers
```

**Hashes/Maps:**
```conduit
hash<K, V>!          // Typed hash/map
hash<string, int>!   // String keys, int values
hash<uuid, float>!   // UUID keys, float values
```

**Inline Structs:**
```conduit
// Inline object type
seo: {
  title: string? @max(60)
  description: string? @max(160)
  image: url?
}?
```

**Nested Arrays:**
```conduit
// Array of structs
images: array<{
  url: string!
  alt: string?
  order: int! @default(0)
}>!
```

### Enum Types

```conduit
status: enum ["draft", "published", "archived"]! @default("draft")
role: enum ["user", "admin", "moderator"]! @default("user")
```

### Default Values

```conduit
// Primitive defaults
count: int! @default(0)
active: bool! @default(true)
status: enum ["active", "inactive"]! @default("active")

// Structural defaults
tags: array<uuid>! @default([])
metadata: hash<string, string>! @default({})
```

---

## Resource Syntax

✅ **Status: Fully Implemented (basic features)**

⚠️ **Note:** Many advanced annotations are parsed but not functional yet. See [FUTURE-VISION.md](FUTURE-VISION.md) for planned features like `@has_many`, `@scope`, `@computed`, `@validate`, etc.

### Basic Structure

```conduit
resource ResourceName {
  // Primary key
  id: uuid! @primary @auto

  // Fields with constraints
  field_name: type! @constraint

  // Relationships
  related: OtherResource! {
    foreign_key: "field_id"
    on_delete: cascade
  }

  // Lifecycle hooks
  @before create { }
  @after create { }

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto
}
```

### Field Definitions

```conduit
resource User {
  // Primary key (required)
  id: uuid! @primary @auto

  // Simple fields with constraints
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  bio: text?

  // Grouped fields (nested objects)
  profile: {
    full_name: string!
    avatar_url: url?
    location: string?
  }!

  // Timestamps (auto-generated)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto
}
```

### Constraints

✅ **Implemented Constraints:**

```conduit
// Size constraints
@min(3)              // Minimum length/value
@max(100)            // Maximum length/value

// Uniqueness
@unique              // Unique in database

// Automation
@primary             // Primary key
@auto                // Auto-generated (UUIDs, timestamps)

// Defaults
@default(value)      // Default value
```

⚠️ **Note:** `@constraint` blocks are parsed but validation logic is not yet implemented. See CON-102 for roadmap.

### Relationships

✅ **Implemented:** Inline relationship metadata

```conduit
resource Post {
  id: uuid! @primary @auto
  title: string!

  // belongs_to relationship
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  // Optional relationship
  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }
}
```

**On Delete Behaviors:**
- `restrict` - Prevent deletion if references exist (default)
- `cascade` - Delete related records
- `set_null` - Set foreign key to null (only for optional relationships)

❌ **Not Yet Implemented:**
- `@has_many` annotations - See [FUTURE-VISION.md](FUTURE-VISION.md#has_many-relationships)
- `@belongs_to` annotation form - Use inline metadata instead
- Eager loading (`?include=relationship`) - Planned for v0.2.0

---

## Standard Library

⚠️ **Status: Partially Implemented (15 MVP functions)**

All built-in functions are namespaced to prevent ambiguity. Only these 15 functions are currently implemented:

### String Namespace

```conduit
String.length(s: string!) -> int!
// Returns the length of a string
// Example: String.length("hello") // => 5

String.slugify(text: string!) -> string!
// Convert text to URL-friendly slug
// Example: String.slugify("Hello World!") // => "hello-world"

String.upcase(s: string!) -> string!
// Convert to uppercase
// Example: String.upcase("hello") // => "HELLO"

String.downcase(s: string!) -> string!
// Convert to lowercase
// Example: String.downcase("HELLO") // => "hello"

String.trim(s: string!) -> string!
// Remove leading/trailing whitespace
// Example: String.trim("  hello  ") // => "hello"

String.contains(s: string!, substr: string!) -> bool!
// Check if string contains substring
// Example: String.contains("hello world", "world") // => true

String.replace(s: string!, old: string!, new: string!) -> string!
// Replace all occurrences of old with new
// Example: String.replace("hello world", "world", "there") // => "hello there"
```

### Time Namespace

```conduit
Time.now() -> timestamp!
// Get current timestamp
// Example: Time.now()

Time.format(t: timestamp!, format: string!) -> string!
// Format timestamp as string
// Example: Time.format(Time.now(), "2006-01-02") // => "2025-11-05"

Time.parse(s: string!) -> timestamp!
// Parse string to timestamp
// Example: Time.parse("2025-11-05")
```

### Array Namespace

```conduit
Array.length(arr: array<T>!) -> int!
// Get array length
// Example: Array.length([1, 2, 3]) // => 3

Array.contains(arr: array<T>!, item: T!) -> bool!
// Check if array contains item
// Example: Array.contains([1, 2, 3], 2) // => true
```

### UUID Namespace

```conduit
UUID.generate() -> uuid!
// Generate a new UUID v4
// Example: UUID.generate() // => "550e8400-e29b-41d4-a716-446655440000"

UUID.validate(s: string!) -> bool!
// Check if string is valid UUID
// Example: UUID.validate("550e8400-e29b-41d4-a716-446655440000") // => true
```

### Random Namespace

```conduit
Random.int(min: int!, max: int!) -> int!
// Generate random integer between min and max (inclusive)
// Example: Random.int(1, 10) // => 7
```

### ⏳ Planned Functions

For the full list of planned standard library functions (30+ total), see [FUTURE-VISION.md](FUTURE-VISION.md#standard-library-expansion).

---

## Lifecycle Hooks

✅ **Status: Fully Implemented**

Lifecycle hooks allow you to run custom logic at specific points in a resource's lifecycle.

### Hook Types

```conduit
@before create { }     // Before insert
@before update { }     // Before update
@before delete { }     // Before delete
@before save { }       // Before create OR update

@after create { }      // After insert
@after update { }      // After update
@after delete { }      // After delete
@after save { }        // After create OR update
```

### Execution Order

**Create Operation:**
1. Generate @auto fields (UUIDs, timestamps)
2. Execute `@before create` hook
3. Execute `@before save` hook
4. Validate fields and constraints
5. Begin database transaction
6. INSERT record
7. Execute `@after create` hook
8. Execute `@after save` hook
9. Commit transaction

**Update Operation:**
1. Load existing record
2. Apply changes
3. Execute `@before update` hook
4. Execute `@before save` hook
5. Validate fields and constraints
6. Begin database transaction
7. UPDATE record
8. Execute `@after update` hook
9. Execute `@after save` hook
10. Commit transaction

### Hook Examples

**Generate slug from title:**
```conduit
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5)
  slug: string! @unique

  @before create {
    self.slug = String.slugify(self.title)
  }
}
```

**Normalize email:**
```conduit
resource User {
  id: uuid! @primary @auto
  email: email! @unique

  @before create {
    self.email = String.downcase(String.trim(self.email))
  }
}
```

**Set timestamps:**
```conduit
resource Comment {
  id: uuid! @primary @auto
  content: text!
  created_at: timestamp!
  updated_at: timestamp!

  @before create {
    self.created_at = Time.now()
    self.updated_at = Time.now()
  }

  @before update {
    self.updated_at = Time.now()
  }
}
```

### Hook Scope

**Within hooks, you have access to:**
- `self` - The resource instance being created/updated/deleted
- All standard library functions
- Field values via `self.field_name`

**Hooks run:**
- ✅ Before validation (can set required fields)
- ✅ Within transactions (automatic rollback on error)
- ✅ With access to database connection

---

## Complete Examples

### Example 1: Simple Todo Resource

```conduit
resource Todo {
  // Primary key
  id: uuid! @primary @auto

  // Fields
  title: string! @min(1) @max(200)
  completed: bool! @default(false)

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto
}
```

**Generated REST API:**
```
POST   /todos           - Create todo
GET    /todos           - List todos
GET    /todos/:id       - Get todo
PUT    /todos/:id       - Update todo (full)
PATCH  /todos/:id       - Update todo (partial)
DELETE /todos/:id       - Delete todo
```

---

### Example 2: Blog Post with Relationships

```conduit
resource User {
  id: uuid! @primary @auto
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  bio: text?

  created_at: timestamp! @auto
  updated_at: timestamp! @auto

  @before create {
    self.email = String.downcase(String.trim(self.email))
  }
}

resource Post {
  id: uuid! @primary @auto

  // Content
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text!

  // Metadata
  status: enum ["draft", "published", "archived"]! @default("draft")
  published_at: timestamp?

  // Relationship
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto

  // Auto-generate slug from title
  @before create {
    self.slug = String.slugify(self.title)
  }

  // Set published_at when publishing
  @before update {
    if self.status == "published" && !self.published_at {
      self.published_at = Time.now()
    }
  }
}

resource Comment {
  id: uuid! @primary @auto

  content: text! @min(1)

  // Relationships
  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: timestamp! @auto
  updated_at: timestamp! @auto

  // Clean up content
  @before create {
    self.content = String.trim(self.content)
  }
}
```

---

## What's Not Included

This specification focuses on features that work today. For planned features, see:

**[FUTURE-VISION.md](FUTURE-VISION.md)** - Roadmap of planned features:
- Expression Language (v0.3.0)
- Query Language (v0.3.0)
- @has_many relationships (v0.4.0)
- @computed fields (v0.4.0)
- @validate blocks (v0.4.0)
- Custom functions (v0.4.0)
- Error handling (rescue blocks, unwrap operator)
- Testing framework (v1.0.0)
- Advanced resource features

**[ROADMAP.md](ROADMAP.md)** - Detailed feature tracking with implementation status

**[UPGRADE-PATH.md](UPGRADE-PATH.md)** - Migration guides between versions (when features land)

---

## Getting Help

**Documentation:**
- [README.md](README.md) - Project overview
- [GETTING-STARTED.md](GETTING-STARTED.md) - Quick start guide
- [FUTURE-VISION.md](FUTURE-VISION.md) - Planned features
- [ROADMAP.md](ROADMAP.md) - Implementation status

**Community:**
- GitHub Issues: https://github.com/dphaener/conduit/issues
- Linear Project: https://linear.app/haener-dev/team/CON

---

**Last Updated:** 2025-11-05
**Version:** 0.1.0
