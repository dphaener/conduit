# Blog Example

**Difficulty:** Intermediate
**Time:** 15 minutes

A complete blog application demonstrating relationships between resources. This example shows how to build a multi-resource application with proper foreign key relationships.

## What You'll Learn

- `belongs_to` relationships with foreign keys
- Multiple relationships in a single resource
- Different `on_delete` behaviors (cascade vs restrict)
- Email validation and uniqueness constraints
- Slug generation from titles
- Optional vs required fields (`?` vs `!`)
- Lifecycle hooks for data normalization

## Quick Start

```bash
# Navigate to this directory
cd examples/blog

# Build the application
conduit build

# The build creates a complete blog API with:
# - User management (authors)
# - Post creation and publishing
# - Comment system
# - Proper relationship handling
```

## What's Inside

This example contains three interconnected resources:

### 1. User Resource (`app/resources/user.cdt`)

Represents blog authors with authentication capabilities:

```conduit
resource User {
  id: uuid! @primary @auto
  email: string! @unique @min(5) @max(255)
  password_hash: string! @min(60) @max(255)
  name: string! @min(2) @max(100)
  bio: text?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.email = String.downcase(self.email)
  }
}
```

**Key features:**
- `@unique` on email prevents duplicate accounts
- `bio: text?` is optional (can be null)
- Email normalized to lowercase for case-insensitive uniqueness
- Password stored as hash (bcrypt in production)

### 2. Post Resource (`app/resources/post.cdt`)

Blog posts written by users:

```conduit
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  published: bool! @default(false)
  author_id: uuid!
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }
}
```

**Key features:**
- `belongs_to User` via `author` relationship
- `on_delete: restrict` prevents deleting users with posts
- `slug` auto-generated from title for SEO-friendly URLs
- `published` defaults to false (draft state)
- `text!` type for long-form content

### 3. Comment Resource (`app/resources/comment.cdt`)

Comments on blog posts:

```conduit
resource Comment {
  id: uuid! @primary @auto
  content: text! @min(1) @max(2000)
  post_id: uuid!
  author_id: uuid!
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }
}
```

**Key features:**
- **Two relationships** in one resource (post and author)
- `on_delete: cascade` for post (delete comments when post deleted)
- `on_delete: restrict` for author (prevent deleting users with comments)
- Whitespace trimming with `String.trim()`

## Generated API Endpoints

Once built, you get these REST endpoints:

### Users
- `GET /users` - List all users
- `POST /users` - Create a new user
- `GET /users/:id` - Get a specific user
- `PUT /users/:id` - Update a user
- `DELETE /users/:id` - Delete a user (fails if user has posts/comments)

### Posts
- `GET /posts` - List all posts
- `POST /posts` - Create a new post
- `GET /posts/:id` - Get a specific post
- `PUT /posts/:id` - Update a post
- `DELETE /posts/:id` - Delete a post (cascades to comments)

### Comments
- `GET /comments` - List all comments
- `POST /comments` - Create a new comment
- `GET /comments/:id` - Get a specific comment
- `PUT /comments/:id` - Update a comment
- `DELETE /comments/:id` - Delete a comment

## Understanding Relationships

### belongs_to Relationships

In Conduit, relationships are defined inline with metadata:

```conduit
// The foreign key field (stores the UUID)
author_id: uuid!

// The relationship definition
author: User! {
  foreign_key: "author_id"
  on_delete: restrict
}
```

**This generates:**
1. Database foreign key constraint
2. Validation to ensure the referenced user exists
3. Proper JOIN queries for loading related data

### on_delete Behaviors

**restrict** - Prevents deletion if referenced:
```conduit
author: User! {
  foreign_key: "author_id"
  on_delete: restrict
}
```
Cannot delete a User if they have Posts or Comments.

**cascade** - Deletes related records:
```conduit
post: Post! {
  foreign_key: "post_id"
  on_delete: cascade
}
```
Deleting a Post automatically deletes all its Comments.

### has_many Relationships

**Note:** `has_many` relationships are **not yet implemented** in Conduit (see ROADMAP.md).

Instead, query related records via foreign keys:
```
# To get a user's posts (in application code)
# Post.where(author_id: user.id)

# To get a post's comments (in application code)
# Comment.where(post_id: post.id)
```

## Try It Out

Example workflow for creating a blog post:

```bash
# 1. Create a user
curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "author@example.com",
    "password_hash": "$2a$10$...",  # bcrypt hash
    "name": "Jane Author",
    "bio": "Technical writer and blogger"
  }'
# Returns: {"id": "550e8400-e29b-41d4-a716-446655440000", ...}

# 2. Create a post
curl -X POST http://localhost:3000/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Getting Started with Conduit",
    "content": "Conduit is an LLM-first programming language...",
    "author_id": "550e8400-e29b-41d4-a716-446655440000",
    "published": true
  }'
# Returns: {"id": "...", "slug": "getting-started-with-conduit", ...}

# 3. Add a comment
curl -X POST http://localhost:3000/comments \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Great introduction!",
    "post_id": "...",
    "author_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

## Key Concepts Demonstrated

### 1. Explicit Foreign Keys

Unlike ORMs that hide foreign keys, Conduit makes them explicit:

```conduit
author_id: uuid!  // The actual database column

author: User! {   // The relationship definition
  foreign_key: "author_id"
  on_delete: restrict
}
```

This prevents confusion and makes the database schema clear.

### 2. Relationship Constraints

The `on_delete` behavior is enforced at both database and application level:

- **Database:** Foreign key constraints
- **Application:** Validation before deletion

### 3. Lifecycle Hooks for Data Integrity

```conduit
@before create {
  self.slug = String.slugify(self.title)
}
```

Ensures data consistency without requiring manual intervention.

### 4. Conservative Feature Use

This example **only** uses features that work today:
- ✅ `belongs_to` relationships (inline metadata form)
- ✅ String functions: `slugify()`, `downcase()`, `trim()`
- ✅ Field constraints: `@unique`, `@min`, `@max`, `@default`
- ✅ Lifecycle hooks: `@before create/update`

**Not used** (not yet implemented):
- ❌ `has_many` relationships
- ❌ `@scope` query scopes
- ❌ `@computed` fields
- ❌ Query methods like `Post.find()` or `Post.where()`

See ROADMAP.md for implementation status.

## Database Schema

The generated database schema includes:

```sql
-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(100) NOT NULL,
  bio TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Posts table with foreign key
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title VARCHAR(200) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  content TEXT NOT NULL,
  published BOOLEAN NOT NULL DEFAULT false,
  author_id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Comments table with two foreign keys
CREATE TABLE comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  content TEXT NOT NULL,
  post_id UUID NOT NULL,
  author_id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
);
```

## Common Patterns

### Email Normalization

```conduit
@before create {
  self.email = String.downcase(self.email)
}

@before update {
  self.email = String.downcase(self.email)
}
```

Ensures "User@Example.com" and "user@example.com" are treated as the same.

### Slug Generation

```conduit
@before create {
  self.slug = String.slugify(self.title)
}
```

Converts "My First Post!" to "my-first-post" for URLs.

### Content Cleanup

```conduit
@before create {
  self.content = String.trim(self.content)
}
```

Removes unwanted whitespace from user input.

## Next Steps

1. **Add more fields** - Try adding `published_at: timestamp?` to Post
2. **Add validation** - Add `@min(500)` to require longer posts
3. **Add more lifecycle hooks** - Use `@after create` to send notifications
4. **Explore api-with-auth** - See `examples/api-with-auth/` for authentication patterns

## Learn More

- **GETTING-STARTED.md** - Full tutorial with all features
- **LANGUAGE-SPEC.md** - Complete language reference
- **ROADMAP.md** - What's implemented vs planned
- **examples/minimal/** - Start here if you're new to Conduit
- **examples/todo-app/** - Simpler example with validation
