package templates

// GetCLAUDEMDContent generates discovery-first CLAUDE.md content for a project.
// This template teaches Claude HOW to learn Conduit, not WHAT patterns to use.
//
// Required Template Variables:
//   - {{.ProjectName}} - Name of the project
//   - {{.Variables.project_name}} - Project type (api, web, microservice)
//   - {{.Variables.port}} - Server port number
//
// Optional Template Variables:
//   - {{.Variables.include_auth}} - Whether authentication is included
//   - {{.Variables.database_url}} - Database connection URL
//
// The generated file is approximately 12KB and designed for fast LLM parsing.
// It emphasizes runtime introspection as the primary learning mechanism.
func GetCLAUDEMDContent() string {
	return `# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this Conduit project.

## Quick Context (30-Second Orientation)

**Project:** {{.ProjectName}}
**Type:** {{.Variables.project_name}} application
**Port:** {{.Variables.port}}
**Database:** PostgreSQL (connection via DATABASE_URL)

Conduit is an LLM-first language that compiles to Go. The killer feature: **runtime introspection** - query the running application to discover schema, patterns, and available operations.

## Quick Reference Card

**Minimal Resource Template:**
` + "```conduit" + `
resource Name {
  id: uuid! @primary @auto
  name: string! @min(2) @max(100)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
` + "```" + `

**Common Types:** ` + "`string! text! int! bool! uuid! timestamp!`" + `

**Common Directives:** ` + "`@primary @auto @auto_update @unique @min(n) @max(n) @default(val)`" + `

**Common Functions:** ` + "`String.slugify() String.length() String.contains() Time.now() UUID.generate() Array.length()`" + `

**CRITICAL: All functions MUST be namespaced** (` + "`String.slugify()`" + ` not ` + "`slugify()`" + `)

See LANGUAGE-SPEC.md for complete reference

## Bootstrap (First Resource)

**Creating your first working resource (empty project):**

### Step 1: Create Resource File
` + "```bash" + `
mkdir -p app/resources && cat > app/resources/item.cdt << 'EOF'
resource Item {
  id: uuid! @primary @auto
  name: string! @min(2) @max(100)
  created_at: timestamp! @auto
}
EOF
` + "```" + `

### Step 2: Build
` + "```bash" + `
conduit build  # Generates build/app.meta.json
` + "```" + `

### Step 3: Migrate
` + "```bash" + `
conduit migrate generate && conduit migrate up
` + "```" + `

### Step 4: Run
` + "```bash" + `
conduit run --watch  # Server starts on port {{.Variables.port}}
` + "```" + `

### Step 5: Test
` + "```bash" + `
curl http://localhost:{{.Variables.port}}/api/items
` + "```" + `

### Step 6: Discover
` + "```bash" + `
conduit introspect schema  # See what was generated
` + "```" + `

**After first build succeeds, use Discovery Mechanisms below.**

## Discovery Mechanisms (YOUR PRIMARY TOOLS)

**Philosophy:** Don't guess patterns - discover them. Conduit is designed for LLM-first development through introspection.

### 1. Runtime Introspection (PRIMARY TOOL)

Query the running application to discover what's available:

` + "```bash" + `
# View all resources and their fields
conduit introspect schema

# View specific resource with full details
conduit introspect schema ResourceName

# View all relationships
conduit introspect relationships

# View hooks for a resource
conduit introspect hooks ResourceName

# View validators and constraints
conduit introspect validators

# Search for existing patterns
conduit introspect patterns "email"
conduit introspect patterns "validation"
conduit introspect patterns "relationship"
` + "```" + `

**Why introspection matters:** Instead of guessing syntax or memorizing docs, query the actual schema to see how things work.

### 2. Metadata Files (Post-Build Discovery)

After each ` + "`conduit build`" + `, check ` + "`build/app.meta.json`" + `:

` + "```bash" + `
# View all resources
cat build/app.meta.json | jq '.resources'

# View specific resource fields
cat build/app.meta.json | jq '.resources[] | select(.name=="Post") | .fields'

# View hooks
cat build/app.meta.json | jq '.resources[] | select(.name=="Post") | .hooks'

# View relationships
cat build/app.meta.json | jq '.resources[] | select(.name=="Post") | .relationships'

# View validators
cat build/app.meta.json | jq '.resources[] | select(.name=="Post") | .validators'
` + "```" + `

### 3. CLI Discovery

` + "```bash" + `
# See all available commands
conduit help

# View project schema
conduit schema

# View migration status
conduit migrate status

# Check for issues
conduit doctor

# View generated routes
conduit routes
` + "```" + `

### 4. Generated Code Inspection

` + "```bash" + `
# View generated Go structs
cat build/generated/resources.go

# View generated API handlers
cat build/generated/handlers.go

# View generated validation logic
cat build/generated/validators.go
` + "```" + `

## How to Learn Conduit (Discovery-First Workflow)

**DON'T** try to memorize syntax or guess patterns.
**DO** use the DISCOVER → LEARN → APPLY → VERIFY pattern:

### Example: Adding Email Validation to User Resource

**WRONG APPROACH** (guessing):
` + "```conduit" + `
// Guessing the validation syntax - might not compile
email: string! @validate_email
` + "```" + `

**RIGHT APPROACH** (discovery-first):

` + "```bash" + `
# 1. DISCOVER: Search for existing email patterns
conduit introspect patterns "email"

# 2. DISCOVER: Check metadata for validator syntax
cat build/app.meta.json | jq '.patterns[] | select(.name | contains("email"))'

# 3. LEARN: Examine existing resources
conduit introspect schema User

# 4. APPLY: Use the discovered pattern
` + "```conduit" + `
email: string! @min(5) @max(255)

@constraint valid_email {
  on: [create, update]
  condition: String.contains(self.email, "@")
  error: "Email must be valid"
}
` + "```" + `

# 5. VERIFY: Build and introspect
conduit build
conduit introspect validators User
` + "```" + `

### Example: Adding a Relationship

` + "```bash" + `
# 1. DISCOVER: View existing relationships
conduit introspect relationships

# 2. LEARN: See relationship syntax
cat build/app.meta.json | jq '.resources[].relationships'

# 3. APPLY: Follow the discovered pattern
# 4. VERIFY: Build and check
conduit build
conduit introspect relationships
` + "```" + `

### Example: Discovering Available Functions

` + "```bash" + `
# Don't guess function names - discover them
conduit introspect patterns "String"
conduit introspect patterns "Time"
conduit introspect patterns "Array"

# Or check the metadata
cat build/app.meta.json | jq '.stdlib'
` + "```" + `

## Project Structure

` + "```" + `
{{.ProjectName}}/
├── app/
│   └── resources/          # Your .cdt files (version controlled)
│       └── *.cdt          # Resource definitions - EDIT THESE
├── build/                  # Generated (gitignored - DO NOT EDIT)
│   ├── app                # Compiled binary
│   ├── app.meta.json      # ← DISCOVER: Check this for schema
│   └── generated/         # Generated Go code (read for learning)
│       ├── resources.go   # See how resources compile to Go
│       ├── handlers.go    # See generated API endpoints
│       └── validators.go  # See validation logic
├── migrations/             # SQL migrations (version controlled)
│   └── *.sql              # IMMUTABLE after running
├── conduit.yaml           # Project configuration
├── .gitignore             # Ensures build/ is not committed
└── README.md
` + "```" + `

**Key Discovery Files:**
- **` + "`build/app.meta.json`" + `** - Complete schema, hooks, validators, relationships
- **` + "`build/generated/resources.go`" + `** - See how .cdt compiles to Go structs
- **` + "`migrations/*.sql`" + `** - Database schema evolution history

**CRITICAL WARNINGS:**
- ⚠️ **NEVER edit files in ` + "`build/`" + `** - they're auto-generated and gitignored
- ⚠️ **NEVER modify migrations after running** - they're immutable for safety
- ⚠️ **ALWAYS use introspection** before making changes to understand current state

## Development Workflow

**Standard iterative development loop:**

` + "```bash" + `
# 1. DISCOVER: Understand current state
conduit introspect schema

# 2. MODIFY: Edit a resource in app/resources/
# (edit app/resources/post.cdt)

# 3. BUILD: Generate code
conduit build

# 4. DISCOVER: Verify changes
conduit introspect schema Post
cat build/app.meta.json | jq '.resources[] | select(.name=="Post")'

# 5. MIGRATE: If schema changed
conduit migrate generate
conduit migrate up

# 6. RUN: Start the server
conduit run --watch

# 7. TEST: Verify endpoint works
curl http://localhost:{{.Variables.port}}/api/posts

# 8. REPEAT: Introspection guides next steps
` + "```" + `

**Quick Commands:**
` + "```bash" + `
conduit build              # Compile .cdt to Go
conduit run --watch        # Run with auto-reload
conduit migrate generate   # Create migration from schema changes
conduit migrate up         # Apply pending migrations
conduit introspect schema  # View all resources
conduit doctor             # Check for issues
` + "```" + `

## Language Design Principles (Discovered Through Use)

### Explicit Nullability
Every field must specify ` + "`!`" + ` (required) or ` + "`?`" + ` (optional):

` + "```bash" + `
# DISCOVER how existing resources handle nullability
conduit introspect schema | grep -A 5 "fields"
` + "```" + `

### Namespaced Standard Library
All functions use namespaces to prevent LLM hallucination:

` + "```bash" + `
# DISCOVER available functions
conduit introspect patterns "String."
conduit introspect patterns "Time."
conduit introspect patterns "Array."
` + "```" + `

Examples:
- ✓ ` + "`String.slugify(text)`" + ` - Correct
- ✓ ` + "`Time.now()`" + ` - Correct
- ✗ ` + "`slugify(text)`" + ` - Won't compile (no namespace)

### Transaction Boundaries
Hooks have explicit transaction control:

` + "```bash" + `
# DISCOVER existing hook patterns
conduit introspect hooks
` + "```" + `

## Common Tasks (Discovery-First)

### Adding a New Field

` + "```bash" + `
# 1. See how existing fields are defined
conduit introspect schema ResourceName

# 2. Edit the resource file
# (app/resources/resource_name.cdt)

# 3. Build and verify
conduit build
conduit introspect schema ResourceName

# 4. Generate migration
conduit migrate generate
` + "```" + `

### Adding Validation

` + "```bash" + `
# 1. Discover existing validators
conduit introspect validators

# 2. Check metadata for constraint syntax
cat build/app.meta.json | jq '.resources[].validators'

# 3. Apply discovered pattern
# 4. Build and verify
conduit build
conduit introspect validators ResourceName
` + "```" + `

### Adding a Relationship

` + "```bash" + `
# 1. View existing relationships
conduit introspect relationships

# 2. Learn the syntax from metadata
cat build/app.meta.json | jq '.resources[].relationships'

# 3. Add to your resource
# 4. Verify
conduit build
conduit introspect relationships
` + "```" + `

### Adding a Hook (Before/After)

` + "```bash" + `
# 1. See existing hooks
conduit introspect hooks

# 2. Learn hook structure
cat build/app.meta.json | jq '.resources[].hooks'

# 3. Add hook to resource
# 4. Verify
conduit build
conduit introspect hooks ResourceName
` + "```" + `

## Critical Safety Rules

### 1. Gitignored Files (NEVER Commit)
The ` + "`build/`" + ` directory is gitignored for good reason:
- Auto-generated on every build
- Can contain sensitive metadata
- Would create merge conflicts

**DO:**
- Commit ` + "`app/resources/*.cdt`" + `
- Commit ` + "`migrations/*.sql`" + `
- Commit ` + "`conduit.yaml`" + `

**DON'T:**
- Commit ` + "`build/`" + ` directory
- Commit ` + "`.env`" + ` files
- Edit generated Go files directly

### 2. Migration Immutability
Once a migration is run (` + "`conduit migrate up`" + `), it becomes immutable:

**Why?** Other developers may have already run it. Changing it would cause inconsistencies.

**DO:**
- Create a new migration for schema changes
- Use ` + "`conduit migrate generate`" + ` for new changes

**DON'T:**
- Modify existing migration files after running
- Delete migration files
- Reorder migration timestamps

### 3. Introspection Before Changes
Always check current state before making changes:

` + "```bash" + `
# Before adding a field
conduit introspect schema ResourceName

# Before adding a relationship
conduit introspect relationships

# Before adding validation
conduit introspect validators ResourceName
` + "```" + `

## Where to Find More Information

**Primary sources** (in order of preference):
1. **Runtime introspection** - ` + "`conduit introspect <command>`" + `
2. **Metadata file** - ` + "`build/app.meta.json`" + `
3. **Generated code** - ` + "`build/generated/*.go`" + `
4. **Official docs** - https://conduit-lang.org/docs

**When introspection isn't enough:**
- Language syntax: Check LANGUAGE-SPEC.md in Conduit repo
- CLI commands: ` + "`conduit help`" + `
- Architecture: ARCHITECTURE.md in Conduit repo
- Specific subsystems: IMPLEMENTATION-*.md files

## Summary: The Discovery-First Philosophy

Traditional approach:
` + "```" + `
Read docs → Memorize syntax → Write code → Hope it compiles → Debug errors
` + "```" + `

Conduit's discovery-first approach:
` + "```" + `
Introspect → Learn patterns → Apply patterns → Verify with introspection
` + "```" + `

**Remember:** Conduit is designed to teach itself to you through introspection. Use it!

---

*This CLAUDE.md was auto-generated by ` + "`conduit new`" + `. For updates, regenerate the project or manually sync with latest template.*
`
}
