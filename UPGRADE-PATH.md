# Conduit Upgrade Path

**Last Updated:** 2025-11-05
**Current Version:** v0.1.0

This document provides migration guides for upgrading between Conduit versions.

---

## Table of Contents

1. [Version Policy](#version-policy)
2. [v0.1.0 (Current)](#v010-current)
3. [v0.2.0 (Planned)](#v020-planned)
4. [v0.3.0 (Planned)](#v030-planned)
5. [v1.0.0 (Planned)](#v100-planned)

---

## Version Policy

Conduit follows [Semantic Versioning](https://semver.org/):

**MAJOR.MINOR.PATCH** (e.g., 1.2.3)

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes - may require code modifications
- **MINOR** (0.1.0 → 0.2.0): New features, backward compatible - no code changes required
- **PATCH** (0.1.0 → 0.1.1): Bug fixes, no new features - no code changes required

### Pre-1.0 Versions

Before v1.0.0, minor version bumps (0.1 → 0.2) may include breaking changes as we refine the language design.

### Feature Stability Levels

- **Alpha:** Experimental, may change without notice
- **Beta:** Stable API, may have bugs
- **Stable:** Production-ready, follows semantic versioning

---

## v0.1.0 (Current)

**Release Date:** November 2025
**Status:** Alpha

### Features

✅ **Implemented:**
- Basic resource definitions
- Explicit nullability (! vs ?)
- Primitive and structural types
- Field constraints (@min, @max, @unique, @primary, @auto, @default)
- Relationships (belongs_to with inline metadata)
- Lifecycle hooks (all 8 types)
- Standard library (15 MVP functions)
- REST API generation
- PostgreSQL migrations

⚠️ **Partially Implemented:**
- @constraint blocks (parsed but not executed)
- Inline enums (work but no validation)

### Breaking Changes

None (initial release)

### Known Issues

- **CON-109:** Some routes return 404 unexpectedly
- **CON-102:** No validation command (must build to check errors)
- Unused imports in generated code (**Fixed in v0.1.1**)
- Hooks execution order (**Fixed in v0.1.1**)

---

## v0.2.0 (Planned)

**Target:** Q1 2026
**Focus:** The Essential 10 - Production essentials

### New Features

📋 **Planned:**
- Named enum support (type-safe status fields)
- Named indexes with unique constraints
- @join_table helper for many-to-many
- Query filtering via URL parameters
- Relationship loading (@include parameter)
- Hot reload dev mode (--watch)
- Routes introspection (conduit routes)
- Seed data system (conduit seed)
- Expanded standard library (30+ functions)
- Performance optimization & benchmarking

### Migration Guide

**⚠️ This section will be updated once v0.2.0 is released.**

#### Named Enums

**Before (v0.1.0):**
```conduit
status: enum ["draft", "published", "archived"]! @default("draft")
```

**After (v0.2.0):**
```conduit
enum PostStatus { draft, published, archived }

resource Post {
  status: PostStatus! @default(draft)
}
```

**Migration:**
1. Define named enums at the top level
2. Replace inline enum fields with named enum types
3. Run `conduit build` to regenerate code
4. No database migrations required (both compile to same SQL ENUM)

#### Composite Unique Constraints

**Before (v0.1.0):**
```conduit
// Not possible - had to add constraints manually in migrations
```

**After (v0.2.0):**
```conduit
resource PostTag {
  post: Post!
  tag: Tag!

  @index {
    columns: [post_id, tag_id]
    unique: true
  }
}
```

**Migration:**
1. Add @index annotations to resources
2. Run `conduit migrate` to generate new migration
3. Apply migration to database

### Breaking Changes

None expected (fully backward compatible)

### Deprecations

None

---

## v0.3.0 (Planned)

**Target:** Q2 2026
**Focus:** Expression Language & Query Builder

### New Features

📋 **Planned:**
- Full expression language for hooks
- Query builder (Post.where, Post.find_by, etc.)
- Query scopes (@scope blocks)
- Aggregations (count, sum, average, etc.)
- Advanced operators (pipeline |>, elvis ?:, safe navigation ?.)
- Control flow (if/else expressions, pattern matching)

### Migration Guide

**⚠️ This section will be updated once v0.3.0 is released.**

#### Expression Language in Hooks

**Before (v0.2.0):**
```conduit
@before create {
  self.slug = String.slugify(self.title)
}
```

**After (v0.3.0):**
```conduit
@before create {
  // Now supports full expression language
  self.slug = self.title |> String.slugify() |> String.downcase()

  // Pattern matching
  self.priority = match self.status {
    "urgent" => 1,
    "high" => 2,
    _ => 3
  }
}
```

**Migration:**
Existing simple hooks continue to work. No changes required unless you want to use new features.

#### Query Builder

**Before (v0.2.0):**
```go
// Had to write raw SQL or use generated Go code
```

**After (v0.3.0):**
```conduit
@function find_published() -> array<Post> {
  return Post.where(status: "published")
              .order_by("created_at DESC")
              .limit(10)
}
```

**Migration:**
1. Replace raw SQL queries with query builder
2. Define @scope blocks for common queries
3. Test thoroughly (query builder may have different behavior)

### Breaking Changes

**⚠️ To be determined** - Will document once v0.3.0 design is finalized

Potential breaking changes:
- Hook execution context may change to support new expression features
- Operator precedence rules may be formalized

### Deprecations

None expected

---

## v1.0.0 (Planned)

**Target:** Q4 2026
**Focus:** Production Stability

### New Features

📋 **Planned:**
- Testing framework built into language
- Advanced resource features (@has_many, @computed, @validate, @function)
- Error handling (rescue blocks, unwrap operator)
- LSP support for editors
- Production-grade tooling
- Performance profiling
- Multi-tenancy support

### Migration Guide

**⚠️ This section will be updated once v1.0.0 is released.**

### Breaking Changes

**⚠️ To be determined**

As part of the 1.0 release, we will:
- Finalize all breaking changes before 1.0
- Provide comprehensive migration tools
- Guarantee backward compatibility after 1.0

### Stability Guarantee

After v1.0.0:
- **No breaking changes** in minor versions (1.x)
- **Semantic versioning** strictly followed
- **Deprecation policy**: Features deprecated for at least one major version before removal

---

## General Upgrade Best Practices

### Before Upgrading

1. **Read the release notes** for the target version
2. **Check breaking changes** in this document
3. **Backup your database** before running migrations
4. **Test in development** environment first
5. **Review generated code** changes after upgrade

### Upgrade Process

```bash
# 1. Update Conduit compiler
brew upgrade conduit  # or download from releases

# 2. Check for breaking changes
conduit version
conduit validate  # Check your code for compatibility

# 3. Rebuild project
conduit build

# 4. Generate new migrations (if needed)
conduit migrate:generate

# 5. Review generated migrations
cat db/migrations/*.sql

# 6. Apply migrations (development first!)
conduit migrate

# 7. Test your application
conduit test  # (when test framework is available)
```

### Rollback Strategy

If you encounter issues after upgrading:

1. **Revert code changes:**
   ```bash
   git checkout main  # or your previous stable branch
   conduit build
   ```

2. **Rollback database migrations:**
   ```bash
   conduit migrate:rollback
   ```

3. **Downgrade Conduit compiler:**
   ```bash
   brew install conduit@0.1.0  # specific version
   ```

---

## Version Compatibility Matrix

| Conduit Version | Go Version | PostgreSQL | Breaking Changes |
|-----------------|------------|------------|------------------|
| 0.1.0           | 1.21+      | 12+        | N/A (initial)    |
| 0.2.0 (planned) | 1.21+      | 12+        | None expected    |
| 0.3.0 (planned) | 1.22+      | 13+        | TBD              |
| 1.0.0 (planned) | 1.23+      | 14+        | TBD              |

---

## Deprecation Policy

### How Deprecation Works

1. **Announcement:** Feature marked as deprecated in release notes
2. **Warning Period:** Feature works but emits warnings (at least 1 minor version)
3. **Removal:** Feature removed in next major version

### Example Timeline

```
v0.9.0: Feature X deprecated (warning added)
v0.10.0: Feature X still works (warning)
v0.11.0: Feature X still works (warning)
v1.0.0: Feature X removed
```

### Deprecation Warnings

When using deprecated features, you'll see:

```
Warning: @old_annotation is deprecated and will be removed in v1.0.0.
Use @new_annotation instead.
See: https://docs.conduit-lang.org/upgrade-path#old-annotation
```

---

## Getting Help with Upgrades

**Documentation:**
- [LANGUAGE-SPEC.md](LANGUAGE-SPEC.md) - Current features
- [FUTURE-VISION.md](FUTURE-VISION.md) - Upcoming features
- [ROADMAP.md](ROADMAP.md) - Implementation status
- [CHANGELOG.md](CHANGELOG.md) - Detailed version history

**Community:**
- GitHub Discussions: https://github.com/dphaener/conduit/discussions
- GitHub Issues: https://github.com/dphaener/conduit/issues
- Linear Project: https://linear.app/haener-dev/team/CON

**Reporting Issues:**

If you encounter problems during an upgrade:

1. Check this document for known breaking changes
2. Search existing GitHub issues
3. Create a new issue with:
   - Your Conduit version (before and after)
   - Steps to reproduce the problem
   - Error messages and logs
   - Minimal code example if possible

---

**Last Updated:** 2025-11-05
**Current Version:** 0.1.0
