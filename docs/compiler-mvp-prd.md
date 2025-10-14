# Conduit Compiler MVP - Product Requirements Document

**Version:** 1.0
**Date:** 2025-10-13
**Status:** Ready for Implementation

---

## Context & Why Now

The Conduit language specification is complete, and the project has reached an inflection point where implementation must begin. The compiler is the foundational component that validates the entire language concept—without it, Conduit remains theoretical. Early implementation enables:

- **Validation of language design** through real-world compilation scenarios
- **Rapid feedback cycles** with early adopters (LLMs and developers)
- **Foundation for ecosystem growth** (tooling, runtime, framework layers)
- **Market timing** to capture growing LLM-assisted development momentum

Source — Design docs complete (30KB spec, 100KB+ implementation guides)
Source — LLM adoption accelerating with 92% of developers using AI tools (GitHub Copilot Survey 2024)

---

## Users & JTBD

### Primary Users

1. **LLMs (Claude, GPT-4, Codex)**
   - JTBD: Generate syntactically correct code without ambiguity
   - JTBD: Self-correct errors based on structured compiler feedback
   - JTBD: Learn patterns from existing codebases

2. **Human Developers**
   - JTBD: Write web applications with minimal boilerplate
   - JTBD: Get immediate feedback on type/nullability errors
   - JTBD: Understand compilation errors quickly

3. **IDE/Editor Tools**
   - JTBD: Provide real-time syntax validation
   - JTBD: Enable code completion and navigation
   - JTBD: Display helpful error diagnostics

### User Priority for MVP
Focus on LLM success first, human developer experience second, IDE integration third.

---

## Business Goals & Success Metrics

### Business Goals
1. Prove Conduit can compile real applications to working Go code
2. Achieve 95%+ LLM first-attempt compilation success rate
3. Enable building a production-ready blog application
4. Establish foundation for community adoption

### Success Metrics

**Leading Indicators:**
- Compilation speed: < 500ms for 1000 LOC
- LLM error self-correction rate: > 90%
- Parser error recovery: > 80% partial AST generation
- Test coverage: > 90% for core paths

**Lagging Indicators:**
- Successfully compile example blog application
- Generate working CRUD operations for 10+ resources
- Process 50+ real-world `.cdt` files without crashes
- 5+ external developers successfully compile projects

---

## Functional Requirements

### 1. Lexical Analysis (Tokenizer)

**Requirement:** Convert Conduit source text to token stream

**Acceptance Criteria:**
- Tokenize all Conduit keywords (`resource`, `@before`, `@after`, etc.)
- Recognize nullability markers (`!` and `?`) correctly
- Handle multi-line strings and escape sequences
- Track source position (line, column) for each token
- Process 10,000 lines in < 100ms

### 2. Syntax Analysis (Parser)

**Requirement:** Build Abstract Syntax Tree from tokens

**Acceptance Criteria:**
- Parse complete resource definitions with fields, hooks, validations
- Support nested structures (inline objects, arrays)
- Implement panic-mode error recovery to continue after syntax errors
- Generate partial AST even with errors (for IDE support)
- Handle 1000 LOC in < 50ms

### 3. Type System

**Requirement:** Enforce explicit nullability and type safety

**Acceptance Criteria:**
- Validate every field has `!` or `?` nullability marker
- Check type compatibility in assignments and expressions
- Resolve resource references in relationships
- Report clear type mismatch errors with fix suggestions
- Support primitive, structural, and enum types

### 4. Go Code Generation

**Requirement:** Transform AST to compilable Go code

**Acceptance Criteria:**
- Generate Go structs with proper tags (`db`, `json`)
- Create CRUD methods (Create, Read, Update, Delete)
- Implement lifecycle hooks as Go methods
- Produce `go fmt` compatible output
- Generate valid Go that compiles without errors

### 5. Error Reporting

**Requirement:** Provide LLM-optimized error messages

**Acceptance Criteria:**
- Return JSON-formatted errors with structured fields
- Include error code, type, file, line, column
- Provide fix suggestions for common errors
- Support both LLM (JSON) and human (terminal) formats
- Batch multiple errors in single response

### 6. Introspection Metadata

**Requirement:** Generate runtime metadata for pattern discovery

**Acceptance Criteria:**
- Output `.meta.json` with complete schema information
- Include all resources, fields, types, relationships
- Document patterns found in codebase
- Generate in < 50ms for typical project

### 7. Standard Library

**Requirement:** Implement namespaced built-in functions

**Acceptance Criteria:**
- Support core namespaces: String, Time, Array, Hash
- Validate function calls have correct arity and types
- Generate appropriate Go stdlib calls
- Prevent non-namespaced function calls

### 8. Expression Language

**Requirement:** Parse and compile expressions in hooks/validations

**Acceptance Criteria:**
- Support arithmetic, comparison, logical operators
- Handle safe navigation (`?.`) and null coalescing (`??`)
- Parse string interpolation (`#{...}`)
- Generate corresponding Go expressions

### 9. CLI Interface

**Requirement:** Provide command-line compilation interface

**Acceptance Criteria:**
- `conduit compile <file>` produces Go output
- `conduit check <file>` validates without generating code
- Return appropriate exit codes (0 = success, 1 = errors)
- Support `--format=json` for LLM consumption

### 10. Basic Lifecycle Hooks

**Requirement:** Compile `@before` and `@after` hooks

**Acceptance Criteria:**
- Parse hook timing and operation type
- Generate Go methods with correct signatures
- Handle `@transaction` and `@async` modifiers
- Support field assignments in hooks

---

## Non-Functional Requirements

### Performance
- **Compilation Speed:** < 500ms for 1000 LOC
- **Memory Usage:** < 100MB for typical project
- **Startup Time:** < 50ms to initialize compiler
- **Incremental Compilation:** Not required for MVP

### Scale
- **File Size:** Support files up to 10,000 lines
- **Project Size:** Handle 50+ resource files
- **Error Limit:** Report up to 100 errors per compilation

### SLOs/SLAs
- **Availability:** N/A (compile-time tool)
- **Reliability:** Zero panics on valid or invalid input
- **Error Recovery:** Continue parsing after first error

### Privacy
- **No telemetry** in MVP
- **Local compilation only**
- **No network calls**

### Security
- **Path traversal protection** in file operations
- **Resource limits** to prevent DoS
- **No code execution** during compilation

### Observability
- **Compilation timing** breakdown (lex, parse, typecheck, codegen)
- **Error categorization** by type
- **Memory profiling** capability
- **Debug mode** with verbose AST output

---

## Scope

### In Scope - Phase 1 (MVP)

**Core Compilation Pipeline:**
1. Lexer for complete Conduit syntax
2. Parser with error recovery
3. Type checker with nullability validation
4. Basic Go code generator
5. JSON error format for LLMs
6. CLI with compile/check commands

**Language Features:**
1. Resource definitions with fields
2. Explicit nullability (`!` and `?`)
3. Basic types (primitives, arrays, hashes)
4. Simple lifecycle hooks (@before/@after create/update/delete)
5. Field-level constraints (@min, @max, @unique)
6. Relationships (belongs_to, has_many)

**Standard Library (Minimal):**
1. String.slugify, String.upcase, String.downcase
2. Time.now, Time.format
3. Array.first, Array.last, Array.length

### Out of Scope - Phase 1

**Deferred to Phase 2:**
- Incremental compilation
- Source maps
- Watch mode
- Hot reload
- Advanced expressions (match statements)
- Custom functions (@function)
- Query scopes (@scope)
- Computed fields (@computed)
- Complex validations (@validate blocks)
- Migration generation
- Full standard library

**Deferred to Phase 3:**
- LSP implementation
- IDE plugins
- Debug adapter protocol
- Documentation generation
- Pattern extraction
- Introspection API

**Not Planned:**
- Alternative compilation targets (WASM, LLVM)
- Alternative databases (MySQL, SQLite)
- GraphQL generation
- Frontend framework

---

## Rollout Plan

### Phase 1: Foundation (Weeks 1-4)

**Week 1-2: Lexer & Parser**
- Ship: Working tokenizer and recursive descent parser
- Validation: Parse example resources without errors
- Success Gate: Parse 10 example files correctly

**Week 3: Type System**
- Ship: Type representation and basic checking
- Validation: Catch nullability violations
- Success Gate: Detect 100% of type errors in test suite

**Week 4: Code Generation**
- Ship: Basic Go struct and method generation
- Validation: Generated code compiles with `go build`
- Success Gate: Generate working CRUD for User resource

**Milestone:** Compile "Hello World" blog to running server

**Kill Switch:** If lexer/parser takes > 2 weeks, reduce language scope

### Phase 2: Completeness (Weeks 5-8)

**Week 5-6: Lifecycle Hooks**
- Ship: Hook parsing and Go method generation
- Validation: Before/after hooks execute correctly
- Success Gate: Blog example hooks work end-to-end

**Week 7: Standard Library**
- Ship: Core namespace functions
- Validation: Function calls generate correct Go code
- Success Gate: String.slugify works in hooks

**Week 8: Error Handling**
- Ship: JSON error format, fix suggestions
- Validation: LLMs can parse and correct errors
- Success Gate: 90% self-correction rate

**Milestone:** Compile complex blog with all features

**Kill Switch:** If hooks fail, ship without advanced features

### Phase 3: Developer Experience (Weeks 9-12)

**Week 9-10: CLI Polish**
- Ship: Improved CLI with progress, colors
- Validation: External developers can use successfully
- Success Gate: 5 developers compile projects

**Week 11-12: Testing & Hardening**
- Ship: Comprehensive test suite
- Validation: No panics on fuzzing
- Success Gate: 95% code coverage

**Milestone:** Public alpha release

---

## Risks & Mitigation

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Expression language complexity | HIGH | HIGH | Start with subset, expand gradually |
| Type system unsoundness | HIGH | MEDIUM | Formal specification, property testing |
| Go code generation bugs | HIGH | MEDIUM | Extensive test suite, review generated code |
| Poor error messages | MEDIUM | HIGH | User testing with LLMs early |
| Performance regression | MEDIUM | LOW | Benchmark from day 1 |

### Market Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| LLMs can't adapt to syntax | HIGH | LOW | Test with multiple LLMs throughout |
| Developers reject verbosity | MEDIUM | MEDIUM | Show productivity gains |
| Go ecosystem limitations | LOW | LOW | Already validated |

---

## Open Questions

### Technical Questions
1. Should we use parser generator (ANTLR) or hand-written?
   - **Recommendation:** Hand-written for better errors and control
2. How to handle partial compilation for IDE support?
   - **Recommendation:** Best-effort AST with error nodes
3. Should introspection be embedded or separate file?
   - **Recommendation:** Separate `.meta.json` for flexibility

### Product Questions
1. What's minimum viable standard library size?
   - **Recommendation:** 10-15 most common functions
2. Should MVP include migration generation?
   - **Recommendation:** No, defer to Phase 2
3. How much backwards compatibility to guarantee?
   - **Recommendation:** None until v1.0 release

### Resourcing Questions
1. Single developer or small team?
   - **Recommendation:** Single developer for consistency
2. How much time for documentation?
   - **Recommendation:** 20% of development time
3. Community feedback during development?
   - **Recommendation:** Private beta with 5-10 developers

---

## Appendix: MVP Validation Criteria

### Minimum Viable Compiler Must:

✅ **Parse and compile this example:**

```conduit
resource User {
  id: uuid! @primary @auto
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  created_at: timestamp! @auto
}

resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text!
  author: User!

  @before create {
    self.slug = String.slugify(self.title)
  }
}
```

✅ **Generate working Go code that:**
- Compiles without errors
- Handles CRUD operations
- Executes lifecycle hooks
- Validates constraints

✅ **Provide error messages that:**
- LLMs can parse (JSON)
- Include fix suggestions
- Reference source location

✅ **Complete in:**
- < 500ms for typical project
- < 100MB memory usage
- Zero panics on any input

---

## Summary

The Conduit Compiler MVP focuses on proving the core concept: that an LLM-optimized language can compile to production-ready code. By targeting a 4-week foundation phase, we can quickly validate the approach and iterate based on real usage.

Success is defined by LLMs achieving 95%+ compilation success rate and developers being able to build real applications. The phased approach allows for early validation while maintaining flexibility to adjust scope based on learnings.

The explicit focus on LLM success (structured errors, zero ambiguity, namespace enforcement) differentiates Conduit from traditional language implementations and validates the core thesis of LLM-first language design.

---

**Next Steps:**
1. Approve PRD and development timeline
2. Set up development environment and CI/CD
3. Begin Week 1 implementation (Lexer)
4. Recruit 5-10 private beta developers
5. Establish LLM testing harness