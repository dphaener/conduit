# Conduit: A Guide for LLMs

This document provides guidance for Large Language Models (LLMs) working with the Conduit programming language and codebase.

## What is Conduit?

Conduit is an LLM-first programming language for building web applications. It compiles to Go and provides explicit, unambiguous syntax designed to minimize hallucination and maximize correctness when AI assists with development.

## Learning Path for LLMs

When learning Conduit or assisting users with Conduit development, follow this progressive learning path:

### Level 1: Quick Start with Examples

**Start here** to understand Conduit syntax and capabilities through working code:

1. **Minimal Example** (`examples/minimal/`)
   - The simplest possible Conduit application
   - 3 fields, automatic REST API generation
   - Demonstrates: resource definition, primary keys, required fields, auto-generated values
   - Time to understand: 2 minutes
   - Builds successfully with `conduit build`

2. **Todo App Example** (`examples/todo-app/`)
   - Basic CRUD application with validation
   - Demonstrates: field constraints, defaults, optional fields, lifecycle hooks
   - Time to understand: 5 minutes
   - Builds successfully with `conduit build`

**Key takeaway**: From minimal resource definitions, Conduit generates complete REST APIs with database schemas, validation, and type safety.

### Level 2: Core Documentation

After understanding the examples, read these documents in order:

1. **README.md** - Project overview, current status, what works today
2. **GETTING-STARTED.md** - Detailed guide with working examples only
3. **ROADMAP.md** - What's implemented vs planned (critical for accuracy)
4. **LANGUAGE-SPEC.md** - Complete language reference (aspirational - check ROADMAP.md for actual status)

### Level 3: Implementation Understanding

For deeper work on the compiler or runtime:

1. **Architecture** (`docs/ARCHITECTURE.md` if exists)
2. **Compiler internals** (`internal/compiler/`)
3. **Code generation** (`internal/compiler/codegen/`)
4. **Runtime system** (`runtime/`)

## Key Principles for LLMs

### 1. Explicit Nullability

**Every type MUST specify nullability:**
```conduit
name: string!     // Required (never null)
bio: string?      // Optional (can be null)
```

Never suggest or generate code with bare types like `name: string` - this will fail compilation.

### 2. Namespaced Standard Library

**All built-in functions are namespaced:**
```conduit
// Correct
self.slug = String.slugify(self.title)
timestamp = Time.now()
id = UUID.generate()

// Wrong - will cause errors
self.slug = slugify(self.title)    // ❌ Function not found
timestamp = now()                   // ❌ Function not found
```

This prevents function hallucination - if it's not in the namespace, it doesn't exist.

### 3. Check Current Implementation Status

**Before suggesting features, check ROADMAP.md:**

The LANGUAGE-SPEC.md is aspirational. Many documented features don't work yet. Always verify implementation status:

- ✅ **Works today**: Basic resources, REST API, belongs_to relationships, lifecycle hooks, field constraints
- ⚠️ **Partially works**: Constraints (parsed but not executed)
- ❌ **Not implemented**: has_many relationships, scopes, computed fields, most stdlib functions

### 4. Progressive Disclosure

Start simple, add complexity only when needed:

```conduit
// Simple (3 lines = complete API)
resource Item {
  id: uuid! @primary @auto
  name: string!
  created_at: timestamp! @auto
}

// Complex (when needed)
resource Post {
  // ... fields ...

  @before create {
    self.slug = String.slugify(self.title)
  }

  @constraint published_requires_content {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts must have at least 500 characters"
  }
}
```

### 5. Type System Guarantees

Conduit's type system prevents entire classes of bugs:

- No null pointer exceptions (explicit `!` vs `?`)
- No type coercion surprises (explicit conversions)
- No undefined function calls (namespaced stdlib)
- No SQL injection (parameterized queries generated)

## Common Patterns

### Resource Definition
```conduit
resource Name {
  id: uuid! @primary @auto
  field: type! @constraints
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
```

### Relationships
```conduit
// Belongs-to relationship (works today)
author: User! {
  foreign_key: "author_id"
  on_delete: restrict
}

// Has-many (not yet implemented - check ROADMAP.md)
// posts: has_many Post as "author"
```

### Lifecycle Hooks
```conduit
@before create {
  self.slug = String.slugify(self.title)
}

@after create {
  Logger.info("Created", context: { id: self.id })
}
```

### Validation
```conduit
// Declarative constraints (work today)
title: string! @min(5) @max(200)
email: email! @unique

// Constraint blocks (parsed but not executed yet)
@constraint name {
  on: [create, update]
  condition: expression
  error: "message"
}
```

## What to Avoid

### Don't Hallucinate Features

If you're unsure whether a feature exists:
1. Check the examples (`examples/`)
2. Check ROADMAP.md for implementation status
3. Check GETTING-STARTED.md for working examples
4. Ask the user rather than guessing

### Don't Suggest Unimplemented Features

Common features that **don't work yet**:
- `@has_many` relationships
- `@scope` query scopes
- `@computed` fields
- `@function` custom functions
- Expression language features (if/match/rescue in hooks)
- Most stdlib namespaces (Logger, Cache, Crypto, etc.)

### Don't Use Relative Ambiguity

Conduit is designed to eliminate ambiguity:
```conduit
// Good - explicit
created_at: timestamp! @auto

// Bad - ambiguous (when is it set?)
created_at: timestamp!
```

## Building and Testing

### Build an Example
```bash
cd examples/minimal
export CONDUIT_ROOT=/path/to/conduit
conduit build
```

### Verify It Works
```bash
./build/app
# Server starts on :3000
```

### Run Tests
```bash
go test ./...
```

## Working with Users

### When a User Asks About a Feature

1. **Check implementation status first** (ROADMAP.md)
2. **If implemented**: Provide working example from `examples/` or docs
3. **If not implemented**: Be honest, explain workaround if available
4. **If unsure**: Acknowledge uncertainty, suggest checking docs

### When Generating Code

1. **Start with examples** as templates
2. **Verify syntax** against GETTING-STARTED.md
3. **Test buildability** mentally (can this compile?)
4. **Explain choices** (why this pattern, why this type)

### When Debugging

1. **Check error messages** - Conduit provides detailed errors with codes
2. **Verify nullability** - most issues are `!` vs `?`
3. **Check function namespaces** - must be `String.X` not `X`
4. **Validate against examples** - does it match working code?

## Resources for LLMs

### Primary Sources (Always Accurate)
- `examples/` - Working, buildable code
- `ROADMAP.md` - Current implementation status
- `README.md` - Project overview and status

### Secondary Sources (Check ROADMAP First)
- `GETTING-STARTED.md` - Tutorial with working examples
- `LANGUAGE-SPEC.md` - Aspirational spec (not all features work)

### Code Sources
- `internal/compiler/` - How Conduit compiles
- `runtime/` - Generated code patterns
- `tests/` - Test cases and fixtures

## Success Criteria

You're successfully working with Conduit when:

1. ✅ Code you suggest builds without errors
2. ✅ You only suggest implemented features
3. ✅ You use explicit nullability everywhere
4. ✅ You namespace all stdlib functions
5. ✅ You provide working examples based on `examples/`
6. ✅ You acknowledge limitations honestly

## Example Interaction Pattern

**User**: "How do I add validation to a field?"

**Good Response**:
```
For string length validation (works today):

title: string! @min(5) @max(200)

This generates database CHECK constraints and application-level validation.

For numeric range validation (@min/@max on integers), there's currently
a codegen bug. Use string validation or implement checks in @before hooks.

See examples/todo-app for a working example.
```

**Bad Response**:
```
Use @validate blocks for custom validation:

@validate {
  if self.price < 0 {
    error("Price must be positive")
  }
}
```
_(This feature is not yet implemented - check ROADMAP.md)_

## Questions?

When in doubt:
1. Check `examples/` for working patterns
2. Check `ROADMAP.md` for implementation status
3. Acknowledge uncertainty rather than guessing
4. Suggest the user check documentation

---

**Remember**: Conduit is designed for LLM success. Use explicit syntax, follow the examples, and check implementation status. You've got this! 🤖
