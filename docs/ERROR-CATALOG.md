# Conduit Error Catalog

This document provides a comprehensive catalog of all compiler errors, warnings, and informational messages in the Conduit language compiler. Each error has a unique code, detailed explanation, and examples of how to fix it.

## Error Code Ranges

- **SYN001-099**: Syntax errors (parser/lexer)
- **TYP100-199**: Type errors (type checker)
- **SEM200-299**: Semantic errors (undefined references, scope issues)
- **REL300-399**: Relationship errors (foreign keys, associations)
- **PAT400-499**: Pattern warnings (conventions, best practices)
- **VAL500-599**: Validation errors (constraints, enum values)
- **GEN600-699**: Code generation errors
- **OPT700-799**: Optimization hints (performance suggestions)

## Severity Levels

- **Error**: Prevents compilation, must be fixed
- **Warning**: Suggests potential issues, compilation continues
- **Info**: Informational messages, optimization hints

---

## Syntax Errors (SYN001-099)

### SYN001: Unexpected Token
**Severity**: Error

An unexpected token was encountered while parsing.

**Example**:
```conduit
resource Post {
  title: string!
  } // Extra closing brace
}
```

**Fix**: Remove the unexpected token or check your syntax.

---

### SYN002: Expected Token
**Severity**: Error

A specific token was expected but not found.

**Example**:
```conduit
resource Post {
  title: string! // Missing closing brace
```

**Fix**: Add the missing token (e.g., closing brace, semicolon).

---

### SYN003: Invalid Resource Name
**Severity**: Error

Resource names must start with an uppercase letter and use PascalCase.

**Example**:
```conduit
resource blogPost { // Should be BlogPost
  title: string!
}
```

**Fix**:
```conduit
resource BlogPost {
  title: string!
}
```

---

### SYN004: Invalid Field Name
**Severity**: Error

Field names must start with a lowercase letter and use snake_case.

**Example**:
```conduit
resource Post {
  PostTitle: string! // Should be post_title
}
```

**Fix**:
```conduit
resource Post {
  post_title: string!
}
```

---

### SYN005: Missing Nullability
**Severity**: Error

All types must specify nullability with `!` (required) or `?` (optional).

**Example**:
```conduit
resource Post {
  title: string // Missing nullability
}
```

**Fix**:
```conduit
resource Post {
  title: string! // Required
  bio: string?   // Optional
}
```

---

### SYN006: Invalid Type Specification
**Severity**: Error

The type specification is malformed or uses invalid syntax.

**Example**:
```conduit
resource Post {
  tags: array<> // Missing element type
}
```

**Fix**:
```conduit
resource Post {
  tags: array<string>!
}
```

---

### SYN007-017: Other Syntax Errors

See inline code documentation for:
- SYN007: Unterminated String
- SYN008: Invalid Number
- SYN009: Invalid Escape Sequence
- SYN010: Unexpected End of File
- SYN011: Mismatched Brace
- SYN012: Invalid Annotation
- SYN013: Duplicate Field
- SYN014: Invalid Hook Timing
- SYN015: Invalid Hook Event
- SYN016: Invalid Constraint Name
- SYN017: Missing Block Body

---

## Type Errors (TYP100-199)

### TYP101: Nullability Violation
**Severity**: Error

Cannot assign a nullable type to a required field without unwrapping or providing a default value.

**Example**:
```conduit
resource Post {
  title: string!
  bio: text?

  @before create {
    self.title = self.bio // Error: bio is nullable
  }
}
```

**Fix**:
```conduit
@before create {
  // Option 1: Unwrap (panics if nil)
  self.title = self.bio!

  // Option 2: Nil coalescing
  self.title = self.bio ?? "No bio"
}
```

---

### TYP102: Type Mismatch
**Severity**: Error

The type of a value doesn't match the expected type.

**Example**:
```conduit
resource Post {
  views: int!

  @before create {
    self.views = "123" // Error: string assigned to int
  }
}
```

**Fix**:
```conduit
@before create {
  self.views = 123 // Use integer literal
}
```

---

### TYP103: Unnecessary Unwrap
**Severity**: Warning

The unwrap operator `!` is used on a type that's already required.

**Example**:
```conduit
resource Post {
  title: string!

  @before create {
    let t = self.title! // Warning: title is already required
  }
}
```

**Fix**:
```conduit
@before create {
  let t = self.title // Remove the !
}
```

---

### TYP120-141: Other Type Errors

See inline code documentation for:
- TYP120: Invalid Binary Operation
- TYP121: Invalid Unary Operation
- TYP122: Invalid Index Operation
- TYP130: Invalid Argument Count
- TYP131: Invalid Argument Type
- TYP140: Invalid Constraint Type
- TYP141: Constraint Type Mismatch

---

## Semantic Errors (SEM200-299)

### SEM200: Undefined Variable
**Severity**: Error

A variable was used before being declared.

**Example**:
```conduit
@before create {
  self.slug = slugified // Error: slugified not defined
}
```

**Fix**:
```conduit
@before create {
  let slugified = String.slugify(self.title)
  self.slug = slugified
}
```

---

### SEM201: Undefined Function
**Severity**: Error

A function was called that doesn't exist or isn't imported.

**Example**:
```conduit
@before create {
  self.slug = slugify(self.title) // Error: use String.slugify()
}
```

**Fix**:
```conduit
@before create {
  self.slug = String.slugify(self.title)
}
```

---

### SEM202-218: Other Semantic Errors

See inline code documentation for:
- SEM202: Undefined Type
- SEM203: Undefined Field
- SEM204: Undefined Resource
- SEM205: Circular Dependency
- SEM206: Redeclared Variable
- SEM207: Redeclared Resource
- SEM208: Invalid Self Reference
- SEM209: Invalid Return Context
- SEM210: Missing Return
- SEM211: Invalid Break Context
- SEM212: Invalid Continue Context
- SEM213: Unreachable Code
- SEM214: Invalid Assignment Target
- SEM215: Constant Reassignment
- SEM216: Invalid Hook Context
- SEM217: Invalid Async Context
- SEM218: Invalid Transaction Context

---

## Relationship Errors (REL300-399)

### REL300: Invalid Relationship Type
**Severity**: Error

The relationship type is not recognized.

**Example**:
```conduit
resource Post {
  author: User! @belongs // Should specify foreign_key
}
```

**Fix**:
```conduit
resource Post {
  author: User! {
    foreign_key: "author_id"
  }
}
```

---

### REL301: Missing Foreign Key
**Severity**: Error

A relationship is missing the required `foreign_key` specification.

**Example**:
```conduit
resource Post {
  author: User! { } // Missing foreign_key
}
```

**Fix**:
```conduit
resource Post {
  author: User! {
    foreign_key: "author_id"
  }
}
```

---

### REL302-309: Other Relationship Errors

See inline code documentation for:
- REL302: Invalid Foreign Key
- REL303: Invalid On Delete Action
- REL304: Invalid Through Table
- REL305: Self-Referential Relationship
- REL306: Conflicting Relationships
- REL307: Missing Inverse Relationship
- REL308: Invalid Relationship Nullability
- REL309: Polymorphic Not Supported

---

## Pattern Warnings (PAT400-499)

### PAT400: Unconventional Naming
**Severity**: Warning

Naming doesn't follow Conduit conventions.

**Example**:
```conduit
resource user { // Should be PascalCase
  UserName: string! // Should be snake_case
}
```

**Fix**:
```conduit
resource User {
  user_name: string!
}
```

---

### PAT401: Missing Documentation
**Severity**: Info

A resource or field is missing documentation.

**Example**:
```conduit
resource Post {
  title: string!
}
```

**Fix**:
```conduit
/// Blog post with title and content
resource Post {
  /// Post title (max 200 characters)
  title: string! @max(200)
}
```

---

### PAT402-410: Other Pattern Warnings

See inline code documentation for:
- PAT402: Unused Field
- PAT403: Unused Variable
- PAT404: Missing Primary Key
- PAT405: Missing Timestamps
- PAT406: Complex Hook
- PAT407: Magic Number
- PAT408: Deep Nesting
- PAT409: Long Function
- PAT410: Inconsistent Nullability

---

## Validation Errors (VAL500-599)

### VAL500: Invalid Constraint Value
**Severity**: Error

A constraint was given an invalid argument value.

**Example**:
```conduit
resource Post {
  title: string! @min(-5) // Min can't be negative
}
```

**Fix**:
```conduit
resource Post {
  title: string! @min(5)
}
```

---

### VAL501: Conflicting Constraints
**Severity**: Error

Two constraints conflict with each other.

**Example**:
```conduit
resource Post {
  id: uuid! @auto @unique // @auto implies @unique
}
```

**Fix**:
```conduit
resource Post {
  id: uuid! @auto // @unique is implied
}
```

---

### VAL502-509: Other Validation Errors

See inline code documentation for:
- VAL502: Invalid Pattern Regex
- VAL503: Invalid Min/Max Range
- VAL504: Invalid Enum Value
- VAL505: Empty Enum Definition
- VAL506: Duplicate Enum Value
- VAL507: Invalid Default Value
- VAL508: Required Field With Default
- VAL509: Invalid Constraint Combination

---

## Code Generation Errors (GEN600-699)

### GEN600: Code Generation Failed
**Severity**: Error

General code generation failure.

**Fix**: This is likely a compiler bug. Please report it with a minimal reproduction case.

---

### GEN601: Invalid Go Identifier
**Severity**: Error

A name cannot be converted to a valid Go identifier.

**Example**:
```conduit
resource Post {
  123invalid: string! // Can't start with number
}
```

**Fix**:
```conduit
resource Post {
  field_123: string!
}
```

---

### GEN602-608: Other Code Generation Errors

See inline code documentation for:
- GEN602: Unsupported Feature
- GEN603: Migration Conflict
- GEN604: Unsafe Migration
- GEN605: Invalid SQL Generation
- GEN606: Type Conversion Failed
- GEN607: Invalid Expression Context
- GEN608: Go Reserved Word

---

## Optimization Hints (OPT700-799)

### OPT700: Missing Index
**Severity**: Info

A field could benefit from a database index.

**Example**:
```conduit
resource User {
  email: string! @unique // Already has index via @unique
  username: string! // Might benefit from @index
}
```

**Suggestion**:
```conduit
resource User {
  email: string! @unique
  username: string! @index // If frequently queried
}
```

---

### OPT701: Ineffective Query
**Severity**: Info

A query could be optimized.

**Suggestion**: Use field selection, filters, or eager loading to improve query performance.

---

### OPT702: N+1 Query Problem
**Severity**: Warning

Potential N+1 query detected when accessing relationships.

**Example**:
```conduit
// In a loop, accessing post.author causes N+1
for post in Post.query().all() {
  print(post.author.name) // Separate query for each post
}
```

**Fix**:
```conduit
// Eager load the relationship
for post in Post.query().include("author").all() {
  print(post.author.name) // Single query with JOIN
}
```

---

### OPT703-708: Other Optimization Hints

See inline code documentation for:
- OPT703: Large Payload
- OPT704: Unused Eager Loading
- OPT705: Missing Caching
- OPT706: Ineffective Index
- OPT707: Slow Function
- OPT708: Memory Intensive

---

## Error Format

### JSON Format (for LLMs)

All errors can be serialized to JSON for machine parsing:

```json
{
  "code": "TYP101",
  "type": "nullability_violation",
  "category": "type",
  "severity": "error",
  "message": "Cannot assign nullable type to required type",
  "location": {
    "line": 6,
    "column": 12
  },
  "file": "post.cdt",
  "context": {
    "current": "self.title = self.bio",
    "source_lines": [
      "  5 |   @before create {",
      "  6 |     self.title = self.bio",
      "  7 |   }"
    ]
  },
  "expected": "string!",
  "actual": "string?",
  "suggestion": "Use unwrap (!) or nil coalescing (??)",
  "examples": [
    "self.title = self.bio!",
    "self.title = self.bio ?? \"Default\""
  ],
  "documentation": "https://docs.conduit-lang.org/errors/TYP101"
}
```

### Terminal Format (for humans)

```
❌ Type Error in post.cdt
Line 6, Column 12:
  5 |   @before create {
  6 |     self.title = self.bio ← Cannot assign nullable type to required type
  7 |   }

  Expected: string!
  Actual:   string?

💡 Use unwrap (!) or nil coalescing (??)

Quick Fixes:
  1. self.title = self.bio!
  2. self.title = self.bio ?? "Default"

Learn more: https://docs.conduit-lang.org/errors/TYP101
```

---

## Getting Help

For more information about any error:

1. **Check the error code** in this catalog
2. **Visit the documentation URL** shown in the error message
3. **Search examples** in the Conduit repository
4. **Ask in the community** forum or Discord

If you believe an error message is incorrect or could be improved, please open an issue on GitHub.
