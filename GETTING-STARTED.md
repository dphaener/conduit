# Getting Started with Conduit

**Version:** 1.0
**Status:** Quick Start Guide
**Updated:** 2025-10-13

This guide will walk you through installing Conduit, creating your first application, and understanding the core concepts.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Your First Project](#your-first-project)
4. [Your First Resource](#your-first-resource)
5. [Running the Server](#running-the-server)
6. [Making API Requests](#making-api-requests)
7. [Adding Relationships](#adding-relationships)
8. [Adding Lifecycle Hooks](#adding-lifecycle-hooks)
9. [Adding Validations](#adding-validations)
10. [Development Workflow](#development-workflow)
11. [Next Steps](#next-steps)

---

## Prerequisites

Before installing Conduit, ensure you have:

**Required:**
- **Go 1.23+** - [Download](https://go.dev/dl/)
- **PostgreSQL 15+** - [Download](https://www.postgresql.org/download/)

**Recommended:**
- **VS Code** with Conduit extension
- **Docker** (optional, for running PostgreSQL)
- **Git** for version control

**Check installations:**

```bash
go version        # Should show go1.23 or later
psql --version    # Should show PostgreSQL 15 or later
```

---

## Installation

### 1. Install Conduit CLI

```bash
# Install via Go
go install github.com/conduit-lang/conduit@latest

# Verify installation
conduit --version
# Output: Conduit v1.0.0
```

### 2. Add to PATH

Ensure `$GOPATH/bin` is in your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$PATH:$(go env GOPATH)/bin"

# Reload shell
source ~/.bashrc  # or source ~/.zshrc
```

### 3. Set up PostgreSQL

**Option A: Local Installation**

```bash
# macOS (Homebrew)
brew install postgresql@15
brew services start postgresql@15

# Ubuntu/Debian
sudo apt install postgresql-15
sudo systemctl start postgresql
```

**Option B: Docker**

```bash
docker run --name conduit-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=conduit_dev \
  -p 5432:5432 \
  -d postgres:15
```

**Create development database:**

```bash
createdb blog_dev
# Or via Docker:
# docker exec conduit-postgres createdb -U postgres blog_dev
```

---

## Your First Project

### 1. Create a new project

```bash
conduit new blog
cd blog
```

**Generated structure:**

```
blog/
├── src/
│   ├── resources/     # Your Conduit resources
│   └── config/        # Configuration files
├── migrations/        # Database migrations
├── tests/            # Test files
├── conduit.yaml      # Project configuration
└── README.md
```

### 2. Configure your project

Edit `conduit.yaml`:

```yaml
project:
  name: blog
  version: 1.0.0

database:
  driver: postgres
  host: localhost
  port: 5432
  name: blog_dev
  user: postgres
  password: postgres
  pool:
    min: 5
    max: 100

server:
  host: 0.0.0.0
  port: 3000
  cors:
    origins: ["*"]
    credentials: true

introspection:
  enabled: true
  auth_required: false  # Set to true in production
```

### 3. Verify setup

```bash
conduit doctor
# ✓ Go installation (1.23.2)
# ✓ Database connection (blog_dev)
# ✓ Configuration valid
# ✓ Ready to build!
```

---

## Your First Resource

### 1. Create a Post resource

Create `src/resources/post.cdt`:

```conduit
@strict nullability

/// Blog post with title and content
resource Post {
  // Primary key (auto-generated)
  id: uuid! @primary @auto

  // Basic fields
  title: string! @min(5) @max(200)
  content: text! @min(100)

  // Status
  status: enum ["draft", "published"]! @default("draft")

  // Timestamps (auto-managed)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
```

**What's happening here:**
- `@strict nullability` requires `!` or `?` on every type
- `!` means required (never null), `?` means optional (can be null)
- `@primary` marks the primary key
- `@auto` generates value automatically (UUID, timestamps)
- `@min`, `@max` are field constraints
- `@default` provides default values

### 2. Build and run migrations

```bash
# Generate migration from resource
conduit migrate generate

# Output:
# Created migrations/001_create_posts.sql

# Apply migration
conduit migrate up

# Output:
# Running migration 001_create_posts.sql... ✓
# Database schema up to date
```

**Generated SQL (migrations/001_create_posts.sql):**

```sql
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title VARCHAR(200) NOT NULL CHECK (length(title) >= 5),
  content TEXT NOT NULL CHECK (length(content) >= 100),
  status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
```

---

## Running the Server

### 1. Start development server

```bash
conduit run --watch
```

**Output:**

```
Conduit v1.0.0

Compiling...
  ✓ Parsing resources... (1 file)
  ✓ Type checking...
  ✓ Generating Go code...
  ✓ Building binary...

Compiled in 1.2s

Starting server...
  ✓ Database connected (blog_dev)
  ✓ Routes registered (5 endpoints)

Server ready at http://localhost:3000

  REST API:
    GET    /posts           List posts
    POST   /posts           Create post
    GET    /posts/:id       Get post by ID
    PUT    /posts/:id       Update post
    DELETE /posts/:id       Delete post

  Introspection:
    POST   /introspect      Query schema and patterns

  Watching for changes in src/...
```

### 2. Verify server is running

```bash
curl http://localhost:3000/health
# {"status":"ok","version":"1.0.0"}
```

---

## Making API Requests

### 1. Create a post

```bash
curl -X POST http://localhost:3000/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Hello Conduit",
    "content": "This is my first post using Conduit! It is a language designed for LLMs and humans to collaborate on building web applications. The explicit syntax makes it easy for AI to generate correct code."
  }'
```

**Response:**

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "title": "Hello Conduit",
  "content": "This is my first post using Conduit! ...",
  "status": "draft",
  "created_at": "2025-10-13T10:30:00Z",
  "updated_at": "2025-10-13T10:30:00Z"
}
```

### 2. List posts

```bash
curl http://localhost:3000/posts
```

**Response:**

```json
{
  "data": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Hello Conduit",
      "content": "This is my first post...",
      "status": "draft",
      "created_at": "2025-10-13T10:30:00Z",
      "updated_at": "2025-10-13T10:30:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20
  }
}
```

### 3. Get a specific post

```bash
curl http://localhost:3000/posts/123e4567-e89b-12d3-a456-426614174000
```

### 4. Update a post

```bash
curl -X PUT http://localhost:3000/posts/123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "published"
  }'
```

### 5. Delete a post

```bash
curl -X DELETE http://localhost:3000/posts/123e4567-e89b-12d3-a456-426614174000
```

---

## Adding Relationships

### 1. Create a User resource

Create `src/resources/user.cdt`:

```conduit
@strict nullability

/// User who can create posts
resource User {
  id: uuid! @primary @auto

  // Profile
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  full_name: string!

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationships
  @has_many Post as "posts" {
    foreign_key: "author_id"
    on_delete: cascade
  }
}
```

### 2. Update Post with relationship

Update `src/resources/post.cdt`:

```conduit
@strict nullability

/// Blog post with title and content
resource Post {
  id: uuid! @primary @auto

  title: string! @min(5) @max(200)
  content: text! @min(100)
  status: enum ["draft", "published"]! @default("draft")

  // Add author relationship
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
```

### 3. Generate and apply migration

```bash
conduit migrate generate
conduit migrate up
```

### 4. Create user and post with relationship

**Create user:**

```bash
curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "full_name": "Alice Smith"
  }'

# Response includes: "id": "abc123..."
```

**Create post with author:**

```bash
curl -X POST http://localhost:3000/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Second Post",
    "content": "This post has an author relationship...",
    "author_id": "abc123..."
  }'
```

### 5. Query with relationships

**Include author in post query:**

```bash
curl 'http://localhost:3000/posts?include=author'
```

**Response:**

```json
{
  "data": [
    {
      "id": "...",
      "title": "My Second Post",
      "author": {
        "id": "abc123...",
        "username": "alice",
        "email": "alice@example.com",
        "full_name": "Alice Smith"
      }
    }
  ]
}
```

---

## Adding Lifecycle Hooks

Lifecycle hooks let you execute logic before or after operations.

### 1. Add slug generation

Update `src/resources/post.cdt`:

```conduit
@strict nullability

resource Post {
  id: uuid! @primary @auto

  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  status: enum ["draft", "published"]! @default("draft")

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Generate slug from title before creating
  @before create {
    self.slug = String.slugify(self.title)
  }

  // Update slug if title changes
  @before update {
    if self.title_changed? {
      self.slug = String.slugify(self.title)
    }
  }
}
```

**What's happening:**
- `@before create` runs before inserting into database
- `String.slugify()` is a built-in function (namespaced to prevent ambiguity)
- `self.title_changed?` checks if the field was modified

### 2. Add published_at timestamp

```conduit
resource Post {
  // ... other fields ...

  published_at: timestamp?

  // Set published_at when status changes to published
  @after update {
    if self.status_changed_to?("published") && self.published_at == nil {
      self.published_at = Time.now()
    }
  }
}
```

### 3. Add async notifications

```conduit
resource Post {
  // ... other fields ...

  @after create @transaction {
    // Run in transaction
    Logger.info("Post created", context: { post_id: self.id })

    // Queue async job (won't block response)
    @async {
      Notify.send(
        to: self.author.email,
        template: "new_post",
        data: { title: self.title }
      ) rescue |err| {
        Logger.error("Failed to send notification", error: err)
      }
    }
  }
}
```

**What's happening:**
- `@transaction` ensures hook runs in database transaction
- `@async` queues work to run after response is sent
- `rescue |err|` catches errors without failing the request

---

## Adding Validations

### 1. Field constraints (declarative)

Already covered! These are on fields:

```conduit
title: string! @min(5) @max(200)
email: email! @unique
age: int! @min(18) @max(120)
```

### 2. Procedural validation

For complex logic that can't be expressed declaratively:

```conduit
resource Post {
  // ... fields ...

  @validate {
    // Custom validation logic
    if self.status == "published" && String.length(self.content) < 500 {
      error("Published posts must have at least 500 characters")
    }

    // Multi-field validation
    if self.scheduled_for != nil && self.scheduled_for! <= Time.now() {
      error("Scheduled date must be in the future")
    }
  }
}
```

### 3. Declarative constraints (reusable)

```conduit
resource Post {
  // ... fields ...

  @constraint published_requires_content {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts must have at least 500 characters"
  }

  @constraint unique_slug_when_published {
    on: [create, update]
    when: self.status == "published"
    condition: !Post.exists?(slug: self.slug, status: "published", id_not: self.id)
    error: "A published post with this slug already exists"
  }
}
```

### 4. Runtime invariants (always checked)

```conduit
resource Post {
  view_count: int! @default(0)
  like_count: int! @default(0)

  @invariant metrics_non_negative {
    condition: self.view_count >= 0 && self.like_count >= 0
    error: "Metrics cannot be negative"
  }
}
```

---

## Development Workflow

### Hot Reload

The development server watches for file changes and automatically rebuilds:

```bash
# Start with watch mode (default)
conduit run --watch
```

**Make a change to any `.cdt` file:**
- Server detects change
- Recompiles in < 1 second
- Restarts server automatically
- Browser refreshes (if WebSocket connected)

### Using the LSP in VS Code

**1. Install Conduit extension:**
- Open VS Code
- Search extensions for "Conduit"
- Install and reload

**2. LSP features:**
- **Hover:** See type information
- **Autocomplete:** Completion for fields, functions, keywords
- **Go to Definition:** Jump to resource definition
- **Find References:** Find all uses of a field
- **Diagnostics:** Real-time errors and warnings
- **Formatting:** Format on save

**3. Settings (`.vscode/settings.json`):**

```json
{
  "conduit.lsp.enabled": true,
  "conduit.format.onSave": true,
  "conduit.diagnostics.realTime": true
}
```

### Debugging

**1. Start with debugger:**

```bash
conduit run --debug
```

**2. Attach VS Code debugger:**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Attach to Conduit",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}/build",
      "port": 2345,
      "host": "localhost"
    }
  ]
}
```

**3. Set breakpoints:**
- Open generated Go code in `build/generated/`
- Set breakpoints
- Make request to trigger breakpoint

### Running Tests

**1. Create test file (`tests/post_test.cdt`):**

```conduit
@test "Creating a post" {
  let user = User.create!({
    username: "testuser",
    email: "test@example.com",
    full_name: "Test User"
  })

  let post = Post.create!({
    title: "Test Post",
    content: "This is a test post with enough content to pass validation...",
    author_id: user.id
  })

  assert_not_nil(post.id)
  assert_equal(post.slug, "test-post")
  assert_equal(post.status, "draft")
}

@test "Publishing a post sets published_at" {
  let post = create_test_post()

  Post.update!(post.id, { status: "published" })

  let updated = Post.find!(post.id)
  assert_not_nil(updated.published_at)
}
```

**2. Run tests:**

```bash
conduit test

# Run specific test file
conduit test tests/post_test.cdt

# Run with coverage
conduit test --coverage
```

---

## Next Steps

### 1. Learn More Concepts

**Read the documentation:**
- **LANGUAGE-SPEC.md** - Complete language reference
- **ARCHITECTURE.md** - System architecture overview
- **IMPLEMENTATION-*.md** - Deep dives into each subsystem

**Key concepts to explore:**
- Query scopes (reusable queries)
- Computed fields (derived values)
- Nested resources (RESTful nesting)
- Middleware (auth, rate limiting, caching)
- Custom functions (reusable logic)

### 2. Build a Real Application

**Example projects to try:**

**Blog (beginner):**
- Users, Posts, Comments, Tags
- Authentication and authorization
- Rich text editor integration
- Image uploads

**E-commerce (intermediate):**
- Products, Orders, Cart, Payments
- Inventory management
- Order processing workflow
- Email notifications

**SaaS App (advanced):**
- Multi-tenancy
- Subscription billing
- Admin dashboard
- Background jobs
- Real-time updates

### 3. Explore Advanced Features

**Introspection API:**

Query your schema programmatically:

```bash
curl -X POST http://localhost:3000/introspect \
  -H "Content-Type: application/json" \
  -d '{
    "query": "what resources exist?"
  }'
```

**Pattern Discovery:**

```bash
curl -X POST http://localhost:3000/introspect \
  -H "Content-Type: application/json" \
  -d '{
    "query": "how do I add authentication to a handler?"
  }'
```

**GraphQL API (v1.1+):**

```conduit
@graphql
resource Post {
  // Automatically generates GraphQL schema
}
```

**Background Jobs (v1.2+):**

```conduit
@job process_payment(order_id: uuid) {
  let order = Order.find!(order_id)
  // Process payment...
}

// Queue job
@after create {
  Jobs.enqueue(:process_payment, order_id: self.id)
}
```

### 4. Deploy to Production

**Build for production:**

```bash
# Build optimized binary
conduit build --release

# Output: build/app (single binary)
```

**Deploy options:**

**1. Simple VPS (DigitalOcean, Linode):**

```bash
# Copy binary to server
scp build/app user@server:/opt/blog/app

# Run with systemd
sudo systemctl start blog
```

**2. Docker:**

```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY build/app /app
EXPOSE 3000
CMD ["/app"]
```

**3. Kubernetes:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blog
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: app
        image: blog:latest
        ports:
        - containerPort: 3000
```

**4. Cloud platforms:**
- AWS (Elastic Beanstalk, ECS, Lambda)
- Google Cloud (Cloud Run, GKE)
- Heroku, Render, Fly.io

### 5. Join the Community

**Get help:**
- **Documentation:** https://docs.conduit-lang.org
- **Discord:** https://discord.gg/conduit
- **GitHub:** https://github.com/conduit-lang/conduit
- **Forum:** https://forum.conduit-lang.org

**Contribute:**
- Report bugs and request features
- Submit pull requests
- Write tutorials and guides
- Help others in Discord/Forum

---

## Common Issues

### Database Connection Failed

**Error:** `Failed to connect to database`

**Solution:**

1. Check PostgreSQL is running:
   ```bash
   pg_isready
   # Should show: /tmp:5432 - accepting connections
   ```

2. Verify database exists:
   ```bash
   psql -l | grep blog_dev
   ```

3. Check connection string in `conduit.yaml`

4. Test connection manually:
   ```bash
   psql -h localhost -U postgres -d blog_dev
   ```

### Compilation Errors

**Error:** `Type mismatch: expected string!, got string?`

**Solution:** Every type must explicitly specify nullability (`!` or `?`).

```conduit
# Wrong:
title: string

# Correct:
title: string!       # Required
bio: string?         # Optional
```

**Error:** `Function 'slugify' not found`

**Solution:** Built-in functions are namespaced.

```conduit
# Wrong:
self.slug = slugify(self.title)

# Correct:
self.slug = String.slugify(self.title)
```

### Port Already in Use

**Error:** `Port 3000 already in use`

**Solution:**

1. Find and kill process:
   ```bash
   lsof -ti:3000 | xargs kill -9
   ```

2. Or change port in `conduit.yaml`:
   ```yaml
   server:
     port: 3001
   ```

---

## Quick Reference

### Conduit CLI Commands

```bash
conduit new <name>              # Create new project
conduit run [--watch]           # Start dev server
conduit build [--release]       # Build binary
conduit format [files...]       # Format code
conduit migrate generate        # Generate migration
conduit migrate up              # Apply migrations
conduit migrate down            # Rollback migration
conduit test [files...]         # Run tests
conduit introspect              # Query schema
conduit docs generate           # Generate documentation
conduit --version               # Show version
conduit --help                  # Show help
```

### Type System Quick Reference

```conduit
// Primitives
string!         string?         // Text
int!            int?            // Integer
float!          float?          // Float
bool!           bool?           // Boolean
uuid!           uuid?           // UUID
timestamp!      timestamp?      // Date & time
email!          email?          // Email (validated)
url!            url?            // URL (validated)
text!           text?           // Long text
json!           json?           // JSON data

// Structural
array<T>!       array<T>?       // Arrays
hash<K,V>!      hash<K,V>?      // Maps/hashes
enum [...]!     enum [...]?     // Enums

// Inline structs
field: {
  subfield: type!
}!

// Required: !
// Optional: ?
```

### Common Annotations

```conduit
// Field
@primary        // Primary key
@auto           // Auto-generate value
@auto_update    // Auto-update timestamp
@unique         // Unique constraint
@min(n)         // Minimum value/length
@max(n)         // Maximum value/length
@default(val)   // Default value
@pattern(regex) // Regex pattern

// Resource
@has_many Resource as "field"
@nested under Parent as "collection"

// Hooks
@before create/update/delete/save
@after create/update/delete/save
@transaction    // Run in transaction
@async          // Run asynchronously

// Validation
@validate { }   // Procedural validation
@constraint name { }  // Declarative constraint
@invariant name { }   // Runtime invariant

// Middleware
@on operation: [middleware_list]

// Scopes
@scope name { }  // Reusable query scope

// Functions
@function name(params) -> return_type { }

// Computed fields
@computed name: type { }
```

### Standard Library Namespaces

```conduit
String.*        // String operations
Time.*          // Time/date operations
Array.*         // Array operations
Hash.*          // Hash operations
Number.*        // Number operations
UUID.*          // UUID operations
Random.*        // Random generation
Crypto.*        // Cryptography
HTML.*          // HTML utilities
JSON.*          // JSON operations
Regex.*         // Regex operations
Logger.*        // Logging
Cache.*         // Caching
Context.*       // Request context
Env.*           // Environment variables
```

---

## Congratulations!

You've completed the Conduit getting started guide. You now know:

✅ How to install and set up Conduit
✅ How to create projects and resources
✅ How to define relationships between resources
✅ How to add lifecycle hooks and validations
✅ How to use the development server and hot reload
✅ How to make API requests

**Ready to build something amazing? Start coding!**

---

**Related Documentation:**
- **LANGUAGE-SPEC.md** - Complete language reference
- **ARCHITECTURE.md** - System architecture
- **IMPLEMENTATION-COMPILER.md** - Compiler details
- **IMPLEMENTATION-RUNTIME.md** - Runtime details
- **IMPLEMENTATION-ORM.md** - ORM details
- **IMPLEMENTATION-WEB.md** - Web framework details
- **IMPLEMENTATION-TOOLING.md** - Tooling details

**Support:**
- Discord: https://discord.gg/conduit
- Forum: https://forum.conduit-lang.org
- GitHub: https://github.com/conduit-lang/conduit

**Happy Building! 🚀**
