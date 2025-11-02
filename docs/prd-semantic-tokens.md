# PRD: Semantic Tokens Support for Conduit LSP

## Context & Why Now

Current state: The Conduit LSP server provides robust IDE features including completion, hover, go-to-definition, and diagnostics. However, syntax highlighting relies exclusively on TextMate grammars—regex-based pattern matching that operates line-by-line without understanding code semantics.

The problem: TextMate grammars cannot distinguish between semantically different but syntactically similar elements. In Conduit, this creates ambiguity in several scenarios:
- Field names vs. resource names (both are identifiers)
- Built-in types (string, uuid) vs. custom resource types (User, Post)
- Annotation parameters vs. field values
- Relationship fields vs. primitive fields
- Hook variables (self) vs. regular variables

Why now:
- LSP 3.16 standardized semantic tokens (2020), now widely adopted across editors (VS Code, Neovim, Emacs, Zed)
- Conduit's AST and type checker already provide rich semantic information—we just need to expose it
- Users expect modern language highlighting quality on par with TypeScript, Rust, and other LSP-enabled languages
- Listed as "Future Enhancement" in internal/tooling/README.md—ready for implementation
- Foundation is solid: parser, type checker, and symbol extraction are mature

Evidence:
- Source: LSP 3.16 specification — "Semantic tokens allow language servers to provide fine-grained token classification beyond regex-based grammars"
- Source: Ruby LSP documentation — "Regular expressions cannot distinguish local variables from method calls; semantic highlighting removes ambiguity"
- Source: Terraform LSP — "TextMate grammars cannot provide accurate highlighting given schema complexity; LSP semantic tokens compensate"

---

## Users & Jobs-to-Be-Done

### Primary Users

**1. Conduit Application Developers**
- JTBD: Quickly distinguish between different types of identifiers when reading unfamiliar Conduit schemas
- JTBD: Reduce cognitive load when scanning resource definitions with many fields and relationships
- JTBD: Identify type errors visually before running diagnostics (e.g., using a field name where a type is expected)

**2. VS Code Users**
- JTBD: Match syntax highlighting quality expectations from other modern languages (TypeScript, Rust, Go)
- JTBD: Leverage custom theme color schemes that differentiate semantic categories beyond basic "keyword" and "string"
- JTBD: Experience consistent highlighting between files (TextMate grammars can be inconsistent across nested contexts)

**3. Team Leads & Code Reviewers**
- JTBD: Quickly scan pull requests and identify resource relationships, constraints, and hooks through visual differentiation
- JTBD: Onboard new team members faster with clearer visual semantics in code

### User Pain Points

Current experience without semantic tokens:
- "I can't tell if `User` is a built-in type or a custom resource without checking definitions"
- "All field names look the same—I have to hover to know if it's a relationship or primitive"
- "The hook body has no semantic highlighting beyond basic string/keyword detection"
- "My custom VS Code theme makes types stand out, but Conduit doesn't use those colors"

---

## Business Goals & Success Metrics

### Business Goals

1. Improve developer experience quality to compete with established schema languages (Prisma, GraphQL)
2. Reduce time-to-productivity for new Conduit users
3. Increase LSP server feature parity with modern language servers
4. Enable richer editor integrations (theme support, accessibility tools)

### Success Metrics

**Leading Indicators (0-3 months):**
- Semantic token provider responds in <50ms for 95th percentile (performance SLO)
- 100% of token types have corresponding TextMate fallback (graceful degradation)
- Zero regression in existing LSP feature response times
- VS Code extension uses semantic tokens with no user configuration required

**Lagging Indicators (3-12 months):**
- User feedback: "Syntax highlighting" mentioned in positive context (qualitative survey)
- Extension activation count increases (proxy for editor satisfaction)
- Support requests about "identifying types" or "reading schemas" decrease by 30%
- Community themes/screenshots showcase semantic token differentiation

---

## Functional Requirements

### FR1: Semantic Token Provider Registration
**Description:** LSP server registers semantic token capability during initialization handshake.

**Acceptance Criteria:**
- Server capabilities include `semanticTokensProvider` with full document and range support
- Token types registered: `namespace`, `type`, `class`, `enum`, `interface`, `struct`, `typeParameter`, `parameter`, `variable`, `property`, `enumMember`, `decorator`, `macro`, `keyword`, `modifier`, `comment`, `string`, `number`, `regexp`, `operator`
- Token modifiers registered: `declaration`, `definition`, `readonly`, `static`, `deprecated`, `abstract`, `async`, `modification`, `documentation`, `defaultLibrary`
- Client (VS Code) receives capabilities and enables semantic highlighting automatically

### FR2: Full Document Token Calculation
**Description:** Compute semantic tokens for an entire Conduit document on request.

**Acceptance Criteria:**
- Handler for `textDocument/semanticTokens/full` returns tokens in delta-encoded format per LSP spec
- All resources, fields, relationships, annotations, and hook bodies are tokenized
- Token calculation uses cached AST from existing tooling API (no re-parsing)
- Returns empty token array for documents with parse errors (fallback to TextMate)
- Handles documents up to 10,000 LOC without exceeding 200ms response time

### FR3: Range Token Calculation
**Description:** Compute semantic tokens for a visible range (viewport optimization).

**Acceptance Criteria:**
- Handler for `textDocument/semanticTokens/range` accepts start/end positions
- Returns only tokens within specified range
- Performance target: <50ms for typical viewport (100 lines)
- Correctly handles partial resources (e.g., viewport shows middle of resource body)

### FR4: Conduit-Specific Token Classification
**Description:** Map Conduit AST nodes to semantic token types with domain-specific accuracy.

**Acceptance Criteria:**
- Resource names → `class` type, `declaration` modifier
- Resource references (in types) → `class` type, no modifier
- Primitive types (string, int, uuid, etc.) → `type` type, `defaultLibrary` modifier
- Field names → `property` type, `declaration` modifier
- Relationship names → `property` type, `declaration` modifier
- Annotation names (@primary, @unique) → `decorator` type
- Annotation arguments → `parameter` type
- Hook keywords (@before, @after) → `decorator` type
- Hook event names (create, update, delete) → `enumMember` type
- Hook body variables (self) → `variable` type, `readonly` modifier
- Nullable markers (!, ?) → `operator` type
- Documentation comments (///) → `comment` type, `documentation` modifier
- Regular comments (#) → `comment` type
- Constraint types in relationships (foreign_key) → `property` type

### FR5: Incremental Token Updates (Future-Proof)
**Description:** Support for `textDocument/semanticTokens/full/delta` to reduce bandwidth on document changes.

**Acceptance Criteria:**
- Not implemented in MVP but capability flag reserved
- LSP server responds with "not supported" error code for delta requests
- Architecture allows future implementation without breaking changes

---

## Non-Functional Requirements

### Performance

**NFR1: Response Time SLOs**
- Full document tokens: <200ms for p95 (files up to 5,000 LOC)
- Range tokens: <50ms for p95 (typical 100-line viewport)
- No impact on existing LSP operations (completion, hover, diagnostics remain <100ms)

**NFR2: Memory Overhead**
- Token cache (if implemented) limited to 50 documents × 10KB = 500KB max
- Reuse existing AST from tooling API document cache (no duplicate parsing)

**NFR3: Scale Limits**
- Graceful degradation: files >10,000 LOC return empty tokens (fallback to TextMate)
- Document with 1,000 resources should complete tokenization in <500ms

### Observability

**NFR4: Logging & Metrics**
- Log semantic token request count, response time (p50, p95, p99) at INFO level
- Log failures with document URI and error details at ERROR level
- Include token count in response metadata for debugging

**NFR5: Graceful Degradation**
- Parse errors → return empty tokens, rely on TextMate grammar
- Type checker errors → still tokenize successfully parsed AST nodes
- Unsupported editor → client ignores semantic tokens, uses TextMate only

### Security & Privacy

**NFR6: Input Validation**
- Validate range parameters (start <= end, within document bounds)
- Reject documents >10,000 LOC to prevent DoS via large file tokenization
- No user data leakage in logs (sanitize URIs to show only filename)

**NFR7: No External Dependencies**
- Semantic token generation uses only Conduit compiler internals
- No network calls or file system access beyond existing document cache

### Compatibility

**NFR8: LSP Specification Compliance**
- Implement LSP 3.16 `SemanticTokens` protocol exactly per specification
- Delta-encoded format: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers]
- Token type/modifier indices match registered legend order

**NFR9: Editor Support**
- Primary: VS Code 1.60+ (semantic tokens introduced in 1.43, stable in 1.60)
- Validated: Neovim 0.5+ with LSP client
- Graceful: Editors without semantic token support ignore capability, use TextMate

**NFR10: Theme Compatibility**
- Default VS Code themes (Dark+, Light+) style all registered token types
- Custom themes degrade gracefully (unmapped tokens use TextMate fallback)
- Document standard token type mappings for theme authors

---

## Scope

### In-Scope for MVP

- Full document semantic token calculation (`textDocument/semanticTokens/full`)
- Range-based token calculation (`textDocument/semanticTokens/range`)
- Token types for: resources, fields, relationships, annotations, types, comments, hooks
- VS Code extension integration (no config changes needed)
- Performance targets: <200ms full document, <50ms range
- Fallback to TextMate on parse errors or large files
- Documentation: token type mapping reference, theming guide

### Out-of-Scope for MVP

- Incremental/delta updates (`textDocument/semanticTokens/full/delta`)—defer to v2
- Cross-file semantic analysis (e.g., highlighting imported resources)—requires workspace indexing
- Custom token types beyond LSP standard set—use existing types creatively
- Semantic tokens in hook body expressions—complex, defer until hook parser matures
- Editor-specific customization (e.g., VS Code settings for token overrides)—not needed
- Automatic theme generation—users leverage existing theme token mappings

### Out-of-Scope Permanently

- Syntax highlighting for non-.cdt files (e.g., embedded Conduit in Markdown)—not LSP's job
- Real-time token streaming during typing—LSP batches updates on didChange
- Semantic tokens for auto-generated code—these files don't exist in LSP workspace

---

## User Stories & Acceptance Scenarios

### Story 1: Developer Distinguishes Types from Resources

**As a** Conduit developer
**I want** built-in types and custom resource types highlighted differently
**So that** I can quickly identify whether a type is primitive or references another resource

**Acceptance:**
- Open `post.cdt` with `author: User!` and `published_at: timestamp?`
- `User` is highlighted with "class" color (e.g., teal in Dark+ theme)
- `timestamp` is highlighted with "default library type" color (e.g., blue-green)
- `uuid`, `string`, `int` all use default library styling

### Story 2: Code Reviewer Scans Relationships Quickly

**As a** code reviewer
**I want** relationship fields visually distinct from primitive fields
**So that** I can understand resource connections without hovering each field

**Acceptance:**
- Open PR with new `Order` resource containing 5 fields and 2 relationships
- All field declarations use "property" color
- Relationship types (e.g., `User!`, `Product!`) use "class" color
- No need to hover to determine which fields are relationships

### Story 3: New User Learns Annotation Syntax

**As a** new Conduit user
**I want** annotations highlighted as decorators
**So that** I recognize them as special metadata, not regular field syntax

**Acceptance:**
- View example `resource User { email: string! @unique @index }`
- `@unique` and `@index` styled as "decorator" (e.g., yellow in most themes)
- Annotation arguments (if present) styled as "parameter"
- Consistent with decorator styling from TypeScript/Python experience

### Story 4: Dark Theme User Enjoys Custom Theme

**As a** developer using a custom VS Code theme (e.g., One Dark Pro)
**I want** Conduit to use semantic token types
**So that** my theme's custom colors apply (without me configuring anything)

**Acceptance:**
- Install One Dark Pro theme
- Open `.cdt` file—types, classes, decorators use theme-defined colors
- Compare to TypeScript file—similar categories use matching colors
- No configuration in VS Code settings required

---

## Priority & Business Value

### Why Build This Now?

**Priority Tier:** P1 (High—enhances core developer experience)

**Value Justification:**

1. **Low implementation cost, high user impact:** Conduit already has AST, type checker, and symbol extraction. Semantic token provider is primarily a "translation layer" from AST to token format. Estimated 2-3 engineering weeks.

2. **Competitive parity:** Prisma, GraphQL, Terraform all have semantic highlighting. Absence signals "unpolished" or "early-stage" tooling, which hurts adoption.

3. **Multiplier effect:** Better syntax highlighting reduces cognitive load for all downstream tasks—reading, writing, reviewing code. Compounds over time.

4. **Ecosystem enabler:** Third-party theme creators and accessibility tools rely on semantic tokens. Enables community contributions (e.g., "Best Conduit VS Code themes" blog posts).

5. **Low risk:** Graceful degradation means zero risk to users without semantic token support. Worst case: feature is invisible. No user-facing config needed.

**ROI Estimate:**
- Engineering cost: 80-120 hours (implementation + testing + docs)
- User value: Saves ~5 seconds per "identify this type" task × 20 tasks/day × 100 active developers = 28 hours/day saved across user base
- Break-even: ~4 days of use

**Sequencing Rationale:**
- Build after: Parser, type checker, LSP foundations (✓ done)
- Build before: Workspace symbols, code actions, rename refactoring (semantic tokens are simpler, higher ROI)

---

## Rollout Plan

### Phase 1: Internal Alpha (Week 1-2)

**Goal:** Validate implementation with internal Conduit development.

**Guardrails:**
- Feature flag: None needed (semantic tokens are opt-in by client)
- Rollback: Remove capability from server initialization if critical bugs found
- Monitoring: Manual testing on 5-10 real Conduit files, log analysis

**Success Criteria:**
- Zero crashes or hangs on internal .cdt files
- All token types appear correctly in VS Code with default theme
- Performance: <200ms for largest internal file (currently ~800 LOC)

### Phase 2: Closed Beta (Week 3)

**Goal:** Gather feedback from 5-10 external early adopters.

**Guardrails:**
- Invite users via GitHub issue comment, Discord DM
- Request specific feedback: accuracy, performance, theme compatibility
- Monitor LSP server logs for errors (ask users to share anonymized logs)

**Success Criteria:**
- 80% of beta users report "highlighting improved" or "no issues"
- Zero critical bugs (crashes, incorrect tokens causing confusion)
- At least 2 themes tested (Dark+, Light+, one custom theme)

### Phase 3: Public Release (Week 4)

**Goal:** Ship to all users via LSP server update + VS Code extension update.

**Guardrails:**
- Release notes clearly explain semantic tokens, link to theming guide
- Changelog includes before/after screenshots
- Kill switch: If >10% error rate in telemetry, document mitigation plan (disable capability)

**Success Criteria:**
- Extension published to VS Code Marketplace
- No spike in GitHub issues tagged "syntax highlighting" or "performance"
- Community shares positive feedback (Discord, Twitter)

### Rollback Procedure

If critical issues arise:
1. Publish LSP server patch removing `semanticTokensProvider` from capabilities
2. Publish VS Code extension patch (can force semantic tokens off client-side if needed)
3. Users fall back to TextMate grammars (same experience as pre-semantic tokens)
4. Timeline: <24 hours from issue report to patch release

---

## Risks & Mitigations

### Risk 1: Performance Regression on Large Files

**Likelihood:** Medium
**Impact:** High (LSP becomes unusable)

**Mitigation:**
- Implement 10,000 LOC hard limit (return empty tokens above threshold)
- Benchmark on generated 5,000 LOC file during development
- Add telemetry for response times >200ms; alert if p95 exceeds SLO

**Contingency:**
- If performance issues arise post-launch, reduce LOC limit or disable for files >1,000 LOC

### Risk 2: Token Misclassification Causes Confusion

**Likelihood:** Medium
**Impact:** Medium (poor user experience, but not broken)

**Mitigation:**
- Create test suite with 20 representative Conduit snippets, manually verify token types
- Beta testing with diverse users (different experience levels)
- Documentation includes token type reference with examples

**Contingency:**
- Hot-fix incorrect token classification (e.g., if relationships tokenized as variables)
- Gather feedback via GitHub issue template specifically for semantic token accuracy

### Risk 3: Editor Incompatibility or Theme Issues

**Likelihood:** Low
**Impact:** Medium (users disable feature or complain)

**Mitigation:**
- Test with 3 editors: VS Code, Neovim, Emacs
- Test with 5 themes: Dark+, Light+, One Dark Pro, Solarized, custom
- Document supported editors and recommended themes

**Contingency:**
- Provide troubleshooting guide: "If colors look wrong, ensure theme supports semantic tokens"
- Allow users to disable semantic tokens in VS Code settings (built-in capability)

### Risk 4: LSP Spec Changes or Deprecation

**Likelihood:** Very Low
**Impact:** High (feature breaks)

**Mitigation:**
- LSP 3.16 (2020) is stable; semantic tokens widely adopted (unlikely to break)
- Monitor LSP spec GitHub repo for changes
- Use well-maintained LSP library (`go.lsp.dev/protocol`)

**Contingency:**
- If spec changes, update implementation; semantic tokens are isolated feature (low blast radius)

---

## Open Questions

### OQ1: Should hook body expressions have semantic tokens?

**Context:** Hook bodies currently captured as raw strings. Full semantic highlighting requires expression parser.

**Decision needed by:** MVP scope freeze (Week 1)

**Options:**
- A) Treat hook body as plain text (no semantic tokens)—simple, defer to future work
- B) Apply basic regex patterns (highlight keywords like `self`, `String`)—moderate effort
- C) Implement full expression parser—high effort, out of scope for MVP

**Recommendation:** Option A for MVP. Hook body tokenization is nice-to-have, not critical. Users rely on diagnostics and hover for hook correctness.

### OQ2: Should we support workspace-wide semantic tokens?

**Context:** Cross-file references (e.g., `author: User!` where `User` defined in `user.cdt`) could use semantic data from other files.

**Decision needed by:** Architecture design (Week 1)

**Options:**
- A) Single-file only—simpler, matches current LSP document cache model
- B) Workspace-aware—requires symbol index lookup, adds complexity

**Recommendation:** Option A for MVP. Workspace symbols are already indexed (FR in tooling API), but semantic token spec is document-scoped. Defer cross-file enhancements until workspace features mature.

### OQ3: How to handle partial AST (incremental parsing)?

**Context:** Future incremental parsing may produce partial ASTs. Semantic tokens need strategy for incomplete nodes.

**Decision needed by:** Post-MVP (when incremental parsing implemented)

**Options:**
- A) Re-tokenize entire document on any change—safe but wasteful
- B) Invalidate only changed ranges—complex but efficient

**Recommendation:** Defer. Current full-document parsing works for typical file sizes. Revisit when incremental parsing lands.

### OQ4: Should we expose custom Conduit-specific token types?

**Context:** LSP allows custom token types (e.g., `conduit.resource`, `conduit.annotation`). Enables richer theming but requires theme author support.

**Decision needed by:** Week 2 (before beta)

**Options:**
- A) Standard types only (`class`, `decorator`, etc.)—works with all themes out of the box
- B) Custom types—richer semantics but requires theme support (most themes won't support)

**Recommendation:** Option A for MVP. Custom types fragment ecosystem. Standard types cover 95% of use cases.

---

## Appendix: Token Type Mapping Reference

| Conduit Element | LSP Token Type | Modifiers | Example |
|-----------------|----------------|-----------|---------|
| Resource name (definition) | `class` | `declaration` | `resource Post` |
| Resource reference (type) | `class` | - | `author: User!` |
| Primitive type | `type` | `defaultLibrary` | `id: uuid!` |
| Field name | `property` | `declaration` | `title: string!` |
| Relationship name | `property` | `declaration` | `author: User!` |
| Annotation | `decorator` | - | `@primary` |
| Annotation argument | `parameter` | - | `@min(5)` |
| Hook keyword | `decorator` | - | `@before` |
| Hook event | `enumMember` | - | `create` |
| Hook variable | `variable` | `readonly` | `self.slug` |
| Nullable operator | `operator` | - | `!`, `?` |
| Doc comment | `comment` | `documentation` | `/// A user` |
| Regular comment | `comment` | - | `# TODO` |
| Constraint key | `property` | - | `foreign_key:` |
| String literal | `string` | - | `"author_id"` |
| Numeric literal | `number` | - | `5`, `200` |
| Keyword | `keyword` | - | `resource` |

---

## Success Definition

This feature is successful when:

1. **Measurable:** 95th percentile response time <200ms for full document tokens on files <5,000 LOC
2. **Observable:** Zero GitHub issues tagged "semantic token bug" within 4 weeks post-launch
3. **Qualitative:** At least 3 community members share screenshots/feedback praising syntax highlighting
4. **Ecosystem:** At least 1 custom VS Code theme author explicitly supports Conduit semantic tokens in release notes
5. **Adoption:** VS Code telemetry shows semantic tokens enabled for 90%+ of Conduit extension users (automatic)

---

**Document Metadata**
Version: 1.0
Author: Product Management
Date: 2025-11-01
Status: Draft for Engineering Review
Stakeholders: Engineering (LSP team), DevRel (documentation), Users (beta testers)
