# Pattern Quality Checklist

Use this checklist when manually reviewing extracted patterns to ensure they meet quality standards and are usable by AI assistants (LLMs).

## Pattern Naming

A good pattern name is clear, descriptive, and follows consistent conventions.

### Checklist

- [ ] Name clearly describes the pattern's purpose without ambiguity
- [ ] Name follows snake_case convention (e.g., `authenticated_handler`, not `AuthenticatedHandler`)
- [ ] Name includes operation context if pattern is operation-specific (e.g., `create_with_auth` not just `handler`)
- [ ] Name avoids abbreviations that might confuse LLMs (use `authenticated` not `authed`)
- [ ] Name is understandable without additional context or documentation
- [ ] Name doesn't conflict with similar patterns (e.g., `auth_handler` vs `authenticated_handler`)

### Examples

✅ **Good Names:**
- `authenticated_handler` - Clear, explicit purpose
- `cached_list_handler` - Specifies operation (list)
- `rate_limited_create` - Operation + middleware clear
- `cors_enabled_api_endpoint` - Verbose but unambiguous

❌ **Poor Names:**
- `handler` - Too generic, no information
- `auth` - Unclear if it's a middleware, pattern, or function
- `h_auth` - Abbreviation, unclear
- `create` - Missing middleware context
- `middleware_pattern_1` - No semantic meaning

### Naming Guidelines by Category

**Authentication Patterns:**
- Use `authenticated_` prefix for patterns requiring auth
- Specify auth level if applicable: `admin_authenticated_handler`
- Don't use: `protected`, `secure`, `private` (ambiguous)

**Caching Patterns:**
- Use `cached_` prefix
- Include operation if specific: `cached_list_handler`
- Optionally include duration: `cached_handler_300s`
- Don't use: `memoized`, `stored` (less common terms)

**Rate Limiting Patterns:**
- Use `rate_limited_` prefix
- Include rate if relevant: `rate_limited_10_per_hour`
- Don't use: `throttled`, `limited` alone

## Template Quality

The template is the core pattern structure that LLMs will learn and reproduce.

### Checklist

- [ ] Template uses clear, explicit Conduit syntax
- [ ] Template includes all required middleware in correct order
- [ ] Template shows parameter examples where applicable (e.g., `cache(300)` not `cache(<ttl>)`)
- [ ] Template uses `<operation>` placeholder for operations if generic
- [ ] Template matches actual code structure from examples exactly
- [ ] Template is copy-pasteable and syntactically valid
- [ ] Template doesn't include optional elements that might confuse (use separate patterns)

### Examples

✅ **Good Templates:**
```conduit
@on <operation>: [auth]
```
- Clear middleware declaration
- Generic operation placeholder
- No ambiguity

```conduit
@on create: [auth, rate_limit(10/hour)]
```
- Specific operation
- Multiple middleware with parameters
- Exact syntax

```conduit
@on list: [cache(300)]
@on show: [cache(300)]
```
- Multiple related operations
- Consistent caching pattern

❌ **Poor Templates:**
```conduit
@on <operation>: [<middleware>]
```
- Too generic, no useful information

```conduit
@on create: [auth(required=true, roles=["admin"])]
```
- Too specific, unlikely to match other code
- Complex parameters reduce reusability

```conduit
// Apply authentication
@on create: [auth]
```
- Includes comments that won't be in actual code
- LLM might reproduce comments incorrectly

### Template Guidelines by Pattern Type

**Simple Middleware:**
```conduit
@on <operation>: [middleware_name]
```

**Middleware with Parameters:**
```conduit
@on <operation>: [middleware_name(param)]
```
- Use actual parameter values from most common usage
- Don't use generic placeholders like `<param>`

**Multiple Middleware:**
```conduit
@on <operation>: [middleware1, middleware2, middleware3]
```
- Preserve execution order (critical for correctness)
- Don't reorder for aesthetics

**Operation-Specific:**
```conduit
@on specific_operation: [middleware]
```
- Use actual operation names, not placeholders
- Group similar operations in separate patterns if needed

## Examples

Examples are crucial for LLM learning. Quality examples show diverse usage and provide context.

### Checklist

- [ ] At least 3 real examples from actual codebase
- [ ] Examples show diverse usage scenarios (different resources/operations)
- [ ] Examples include file paths for traceability
- [ ] Examples are recent and not from deprecated code
- [ ] Examples represent current best practices (not legacy code)
- [ ] Examples show the pattern in realistic context
- [ ] All examples are syntactically identical in structure

### Example Quality Assessment

✅ **Good Example Set:**
```json
{
  "examples": [
    {
      "resource": "Post",
      "file_path": "resources/post.cdt",
      "line_number": 12,
      "code": "@on create: [auth]"
    },
    {
      "resource": "Comment",
      "file_path": "resources/comment.cdt",
      "line_number": 8,
      "code": "@on create: [auth]"
    },
    {
      "resource": "Article",
      "file_path": "resources/article.cdt",
      "line_number": 15,
      "code": "@on create: [auth]"
    }
  ]
}
```
- 3+ examples ✓
- Different resources ✓
- Consistent structure ✓
- File paths included ✓

❌ **Poor Example Set:**
```json
{
  "examples": [
    {
      "resource": "Post",
      "file_path": "unknown",
      "line_number": 0,
      "code": "@on create: [auth]"
    }
  ]
}
```
- Only 1 example ✗
- Missing file path ✗
- No line number ✗
- No diversity ✗

### Example Diversity Guidelines

**Resource Diversity:**
- Include examples from at least 3 different resources
- Prefer core resources over edge cases
- Include both simple and complex resources

**Operation Diversity:**
- Show pattern on different operations where applicable
- Don't just use `create` - include `update`, `delete`, `list`, etc.

**Context Diversity:**
- Show pattern in different contexts (authenticated vs public resources)
- Include patterns from different subsystems if applicable

## Description

A good description explains when to use the pattern and provides context.

### Checklist

- [ ] Description explains when to use this pattern (use cases)
- [ ] Description mentions key benefits of following this pattern
- [ ] Description notes any important caveats or limitations
- [ ] Description is concise (1-2 sentences, max 3)
- [ ] Description avoids jargon and uses plain language
- [ ] Description provides value beyond what the name conveys
- [ ] Description is written for someone unfamiliar with the codebase

### Examples

✅ **Good Descriptions:**

```
"Use for operations that require user authentication. Validates JWT token and loads user context before handler execution."
```
- Clear use case ✓
- Mentions behavior ✓
- Concise ✓

```
"Apply to list and show operations to cache responses for 5 minutes, reducing database load for frequently accessed resources."
```
- Specific operations ✓
- Benefit stated ✓
- Duration specified ✓

```
"Rate limit create operations to 10 per hour per IP to prevent abuse. Use for resource-intensive operations like file uploads."
```
- Clear purpose ✓
- Specific rate ✓
- Example use case ✓

❌ **Poor Descriptions:**

```
"Handler with middleware"
```
- Too vague, no useful information

```
"This pattern is used to add authentication middleware to the create operation on the Post resource using the auth middleware which checks if the user is logged in and has valid credentials."
```
- Too long and rambling
- Too specific to one resource
- Redundant information

```
"Use for auth"
```
- Incomplete, unhelpful
- Doesn't explain when or why

### Description Guidelines

**Structure:**
1. **When to use**: "Use for..." or "Apply to..."
2. **What it does**: Brief behavior description
3. **Why/benefit**: Optional performance, security, or UX note

**Length:**
- Aim for 15-30 words
- Maximum 50 words
- Split complex patterns into separate patterns if description too long

## Category

Categories group related patterns and help with discovery.

### Checklist

- [ ] Category is accurate and specific enough to be useful
- [ ] Category matches existing taxonomy (don't invent new categories unnecessarily)
- [ ] Category helps developers discover related patterns
- [ ] Category groups functionally similar patterns together
- [ ] Category name is lowercase with underscores (e.g., `rate_limiting` not `Rate Limiting`)

### Standard Categories

**Security:**
- `authentication` - User identity verification
- `authorization` - Permission checking
- `encryption` - Data encryption patterns
- `validation` - Input validation patterns

**Performance:**
- `caching` - Response/data caching
- `lazy_loading` - Deferred data loading
- `pagination` - Result pagination patterns

**Reliability:**
- `rate_limiting` - Request rate limiting
- `retry` - Retry logic patterns
- `circuit_breaker` - Circuit breaker patterns
- `timeout` - Timeout handling

**API:**
- `cors` - Cross-origin resource sharing
- `versioning` - API versioning patterns
- `serialization` - Data serialization patterns

**General:**
- `logging` - Logging patterns
- `monitoring` - Monitoring/metrics patterns
- `general` - Uncategorized patterns

### Examples

✅ **Good Categorization:**
```json
{
  "name": "authenticated_handler",
  "category": "authentication"
}
```
- Accurate ✓
- Standard category ✓

```json
{
  "name": "cached_list_handler",
  "category": "caching"
}
```
- Specific ✓
- Matches function ✓

❌ **Poor Categorization:**
```json
{
  "name": "authenticated_handler",
  "category": "security_authentication_jwt"
}
```
- Too specific ✗
- Non-standard ✗

```json
{
  "name": "cached_handler",
  "category": "performance_optimization"
}
```
- Use `caching` not `performance_optimization` ✗

## Frequency & Confidence

These metrics indicate pattern quality and reliability.

### Checklist

- [ ] Frequency >= 3 (appears multiple times in codebase)
- [ ] Confidence >= 0.3 (sufficiently common)
- [ ] Pattern represents actual convention, not one-off code
- [ ] Pattern isn't an anti-pattern that should be refactored
- [ ] High-frequency patterns (10+) reviewed for overgeneralization
- [ ] Low-confidence patterns (< 0.5) reviewed for utility

### Frequency Guidelines

| Frequency | Interpretation | Action |
|-----------|---------------|---------|
| 1-2 | One-off or rare | Likely not a pattern, exclude |
| 3-5 | Emerging pattern | Include if confidence high |
| 6-10 | Established pattern | Include if not anti-pattern |
| 11+ | Core pattern | Review for overgeneralization |

### Confidence Guidelines

| Confidence | Interpretation | Action |
|------------|---------------|---------|
| < 0.3 | Rare/specific | Exclude unless critical |
| 0.3-0.5 | Moderate | Include with good examples |
| 0.5-0.8 | Common | Good pattern candidate |
| > 0.8 | Very common | Verify not overgeneralized |

### Examples

✅ **Good Metrics:**
```json
{
  "name": "authenticated_handler",
  "frequency": 12,
  "confidence": 0.6
}
```
- High frequency ✓
- Good confidence ✓
- Likely a true pattern ✓

```json
{
  "name": "specialized_import_handler",
  "frequency": 3,
  "confidence": 0.3
}
```
- At minimum threshold ✓
- Edge case but legitimate ✓

❌ **Questionable Metrics:**
```json
{
  "name": "custom_handler",
  "frequency": 1,
  "confidence": 0.1
}
```
- Too infrequent ✗
- Very low confidence ✗
- Not a real pattern ✗

```json
{
  "name": "handler",
  "frequency": 50,
  "confidence": 1.0
}
```
- Too generic ✗
- Likely overgeneralized ✗
- Not useful for LLMs ✗

## LLM Usability

The ultimate test: can an LLM understand and reproduce this pattern?

### Checklist

- [ ] LLM can understand pattern purpose from template alone
- [ ] Pattern name is unambiguous (LLM won't confuse with similar concepts)
- [ ] Examples clarify any edge cases or variations
- [ ] Pattern doesn't conflict with other patterns (no ambiguity)
- [ ] Pattern has been tested with validation harness
- [ ] LLM generates syntactically correct code when using pattern
- [ ] LLM applies pattern in appropriate contexts

### Testing LLM Usability

**Step 1: Run Validation Harness**
```bash
conduit test-patterns --verbose
```

**Step 2: Check Success Rate**
- Claude Opus: Should be >= 80%
- GPT-4: Should be >= 70%
- Any pattern < 50%: Review and improve

**Step 3: Review Failures**
```bash
# Check failure reasons
grep "name_unclear" pattern-report.txt
grep "template_ambiguous" pattern-report.txt
```

**Step 4: Iterate**
- If `name_unclear` high: Rename pattern or enable VerboseNames
- If `template_ambiguous` high: Clarify template syntax
- If `llm_hallucination` high: Review prompt clarity

### Common LLM Usability Issues

**Issue: LLM uses synonyms**
```
Pattern: authenticated_handler
Template: @on create: [auth]
LLM Output: @on create: [login]  ❌
```
**Fix:** Make pattern name more explicit or add description clarifying exact middleware name.

**Issue: LLM adds extra parameters**
```
Pattern: cached_handler
Template: @on list: [cache(300)]
LLM Output: @on list: [cache(ttl=300, strategy="lru")]  ❌
```
**Fix:** Simplify template or add note that no extra parameters should be added.

**Issue: LLM confuses similar patterns**
```
Pattern: authenticated_handler, authorized_handler
LLM confuses which to use
```
**Fix:** Make pattern names more distinct or merge into one pattern with variants.

## Complete Example: High-Quality Pattern

Here's an example of a pattern that passes all quality checks:

```json
{
  "id": "pattern-auth-001",
  "name": "authenticated_handler",
  "category": "authentication",
  "description": "Use for operations that require user authentication. Validates JWT token and loads user context before handler execution.",
  "template": "@on <operation>: [auth]",
  "examples": [
    {
      "resource": "Post",
      "file_path": "resources/post.cdt",
      "line_number": 12,
      "code": "@on create: [auth]"
    },
    {
      "resource": "Comment",
      "file_path": "resources/comment.cdt",
      "line_number": 8,
      "code": "@on update: [auth]"
    },
    {
      "resource": "Article",
      "file_path": "resources/article.cdt",
      "line_number": 15,
      "code": "@on delete: [auth]"
    },
    {
      "resource": "User",
      "file_path": "resources/user.cdt",
      "line_number": 20,
      "code": "@on update: [auth]"
    },
    {
      "resource": "Profile",
      "file_path": "resources/profile.cdt",
      "line_number": 10,
      "code": "@on create: [auth]"
    }
  ],
  "frequency": 15,
  "confidence": 0.75,
  "validation_results": {
    "claude_opus_success_rate": 0.95,
    "gpt4_success_rate": 0.88,
    "last_tested": "2025-10-27T14:00:00Z"
  }
}
```

**Why This Pattern is High-Quality:**

✅ **Name**: Clear, descriptive, follows convention
✅ **Category**: Accurate and standard
✅ **Description**: Concise, explains when/why to use
✅ **Template**: Simple, explicit, copy-pasteable
✅ **Examples**: 5 examples, diverse resources and operations
✅ **Frequency**: High (15 occurrences)
✅ **Confidence**: Good (0.75)
✅ **LLM Usability**: >80% success rate on both major LLMs

## Pattern Review Workflow

### Initial Review (When Pattern is Extracted)

1. **Automated Checks** (Run by system):
   - [ ] Frequency >= MinFrequency threshold
   - [ ] Confidence >= MinConfidence threshold
   - [ ] Examples >= 3
   - [ ] Template syntactically valid

2. **Manual Review** (Developer):
   - [ ] Review name against checklist
   - [ ] Verify template accuracy
   - [ ] Check example quality
   - [ ] Write or improve description
   - [ ] Confirm category

3. **LLM Validation** (Automated):
   - [ ] Run pattern through validation harness
   - [ ] Check success rates per provider
   - [ ] Review failure reasons
   - [ ] Iterate if needed

### Periodic Review (Monthly or After Major Changes)

1. **Metrics Review**:
   - [ ] Check if pattern still appears in codebase
   - [ ] Verify frequency hasn't dropped significantly
   - [ ] Review recent validation results

2. **Quality Review**:
   - [ ] Examples still reflect current code
   - [ ] Description still accurate
   - [ ] No better patterns emerged

3. **Retirement Decision**:
   - [ ] If frequency < 2: Consider removing
   - [ ] If confidence < 0.2: Remove or improve
   - [ ] If validation fails consistently: Review or remove

## Quick Reference Card

Print or save this quick reference for pattern reviews:

```
┌─────────────────────────────────────────────────────────┐
│           PATTERN QUALITY QUICK CHECK                    │
├─────────────────────────────────────────────────────────┤
│ NAME:                                                    │
│  □ Clear and descriptive                                │
│  □ snake_case convention                                │
│  □ No ambiguous abbreviations                           │
│                                                          │
│ TEMPLATE:                                               │
│  □ Explicit Conduit syntax                              │
│  □ Copy-pasteable                                       │
│  □ Matches examples exactly                             │
│                                                          │
│ EXAMPLES:                                               │
│  □ At least 3 examples                                  │
│  □ Diverse resources/operations                         │
│  □ Include file paths                                   │
│                                                          │
│ DESCRIPTION:                                            │
│  □ Explains when to use                                 │
│  □ 1-2 sentences, concise                               │
│  □ Plain language                                       │
│                                                          │
│ CATEGORY:                                               │
│  □ Accurate and specific                                │
│  □ Standard category name                               │
│                                                          │
│ METRICS:                                                │
│  □ Frequency >= 3                                       │
│  □ Confidence >= 0.3                                    │
│                                                          │
│ LLM VALIDATION:                                         │
│  □ Tested with validation harness                       │
│  □ Success rate >= 50% per pattern                      │
│  □ No major failure categories                          │
└─────────────────────────────────────────────────────────┘
```

## Related Documentation

- [Pattern Validation Guide](./PATTERN-VALIDATION-GUIDE.md) - Complete validation system documentation
- [Architecture Overview](../ARCHITECTURE.md) - System architecture
- [Language Specification](../LANGUAGE-SPEC.md) - Conduit syntax reference
