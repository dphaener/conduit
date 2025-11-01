# Product Requirements: LLM Bootstrap Experience Improvements

## Context & Why Now

**Current State**: Conduit is marketed as an "LLM-first programming language" with a discovery-first philosophy centered on runtime introspection. However, LLM coding agents (Claude Code, Cursor, etc.) struggle significantly when bootstrapping new projects after `conduit new`.

**The Problem**: Bootstrap paradox - LLMs need to build the project to use introspection for discovering patterns/syntax, but they need syntax knowledge to write their first resource to build. This breaks the entire discovery-first value proposition at the most critical moment: first contact.

**Why Now**:
- Conduit is in alpha with active user acquisition
- First impressions determine adoption - bootstrap failure = immediate churn
- The introspection system is mature and working well, but only accessible after bootstrap
- Real-world testing with Claude Code revealed this critical gap

**Evidence**: Direct observation of Claude Code struggling for extended periods to create a simple todo app in a fresh Conduit project.

## Users & Jobs to Be Done

**Primary User**: LLM coding agents (Claude Code, Cursor AI, Copilot, etc.)

**Job to Be Done**: "When I'm starting a new Conduit project, I need to quickly create my first working resource and reach the discovery-first workflow, so that I can iteratively build the application using introspection."

**Current Painful Workflow**:
1. User runs `conduit new my-project` (works fine)
2. LLM reads CLAUDE.md for guidance
3. CLAUDE.md says "use introspection to discover patterns"
4. LLM tries `conduit introspect schema` → ERROR: no resources exist
5. LLM tries to guess syntax from incomplete examples
6. Build fails with cryptic errors
7. LLM searches for documentation, finds scattered information
8. Repeated trial-and-error, potential hallucination of syntax
9. Eventually succeeds or gives up

**Desired Workflow**:
1. User runs `conduit new my-project`
2. LLM reads CLAUDE.md, sees clear "Bootstrap" section
3. LLM copies minimal working example OR runs `conduit scaffold todo`
4. First build succeeds
5. `conduit introspect schema` now works
6. LLM enters discovery-first workflow → iterative success

## Business Goals & Success Metrics

**Business Goal**: Make Conduit the easiest framework for LLM-assisted development by eliminating bootstrap friction.

**Leading Indicators** (measure during alpha):
- Time from `conduit new` to first successful `conduit build` (Target: <5 minutes)
- Number of build attempts before first success (Target: 1-2 attempts)
- LLM success rate creating first resource without external docs (Target: >80%)
- Error rate decrease in CLI introspection commands (Target: -50%)

**Lagging Indicators** (measure over time):
- Project completion rate (LLMs finish what they start)
- Community feedback sentiment on "getting started"
- Reduced support requests for basic syntax questions
- Increased GitHub stars/adoption rate

**Success Definition**: An LLM with zero Conduit experience can create a working CRUD resource and successfully introspect it within 5 minutes of project creation.

## Functional Requirements

### FR1: CLAUDE.md Bootstrap Section
**Priority**: P0 (Must Have)

Add explicit bootstrap guidance before the existing "Discovery Mechanisms" section that assumes working introspection.

**Acceptance Criteria**:
- [ ] New "Bootstrap (First Resource)" section appears before "Discovery Mechanisms"
- [ ] Shows absolute minimal working resource (3-4 fields max, no optional features)
- [ ] Explicitly states what's required vs optional (`id`, `@primary`, `@auto` explained)
- [ ] Provides copy-pasteable commands to create first resource
- [ ] Shows exact file path: `app/resources/todo.cdt`
- [ ] Includes verification step: `conduit build && conduit introspect schema`
- [ ] Contains explicit transition: "After first build succeeds, use discovery-first workflow below"

**Source**: Analysis of claude_md.go template (lines 36-103) shows current flow assumes resources exist

### FR2: Quick Reference Card in CLAUDE.md
**Priority**: P0 (Must Have)

Provide a non-introspection-dependent syntax reference for bootstrap phase.

**Acceptance Criteria**:
- [ ] New "Quick Reference" section after "Quick Context"
- [ ] Lists minimal resource syntax template with annotations
- [ ] Shows 6-8 most common field types (string!, int!, bool!, uuid!, timestamp!, text!)
- [ ] Shows 6-8 most common directives (@primary, @auto, @default, @min, @max, @unique, @auto_update)
- [ ] Shows 4-6 most common stdlib functions (String.slugify, String.length, Time.now, Array.length)
- [ ] Explicitly states: "For complete reference: See LANGUAGE-SPEC.md"
- [ ] Formatted for quick scanning (table or compact list)

**Source**: LANGUAGE-SPEC.md exists (lines 99-200) but not referenced in bootstrap flow

### FR3: Improved Error Messages
**Priority**: P0 (Must Have)

Transform blocking errors into actionable guidance during bootstrap.

**Acceptance Criteria**:
- [ ] Error "registry not initialized" includes example minimal resource
- [ ] Error includes step-by-step commands to create first resource
- [ ] Error mentions `conduit scaffold` as easier alternative (if implemented)
- [ ] Error includes exact file path: `mkdir -p app/resources && cat > app/resources/...`
- [ ] Error is helpful but concise (≤10 lines)
- [ ] User can copy-paste commands directly from error output

**Source**: Current error message observed during testing provides no guidance

### FR4: `conduit introspect stdlib` Command
**Priority**: P1 (Should Have)

Make stdlib discoverable BEFORE first build.

**Acceptance Criteria**:
- [ ] Command works without requiring successful build
- [ ] Lists all stdlib functions organized by namespace (String, Time, Array, etc.)
- [ ] Shows function signatures: `String.slugify(text: string!) -> string!`
- [ ] Includes brief descriptions (1 line per function)
- [ ] Supports filtering: `conduit introspect stdlib String`
- [ ] Output is LLM-friendly (clear structure, easy to parse)
- [ ] JSON format available: `--format json`

**Rationale**: Prevents hallucination of function names (e.g., `slugify()` vs `String.slugify()`)

### FR5: `conduit scaffold` Command
**Priority**: P1 (Should Have)

Provide one-command bootstrap with working examples.

**Acceptance Criteria**:
- [ ] `conduit scaffold todo` generates minimal CRUD resource
- [ ] `conduit scaffold blog` generates resource with relationships
- [ ] Generated code includes helpful comments explaining syntax
- [ ] Creates proper directory structure if missing
- [ ] Immediately buildable after generation
- [ ] Includes README with next steps
- [ ] Help text lists available scaffolds: `conduit scaffold --help`

**Variants**:
- `todo`: Single resource, 4-5 fields, basic CRUD
- `blog`: Post + User, demonstrates relationships
- `api`: Authentication patterns, middleware examples

### FR6: Examples Directory
**Priority**: P1 (Should Have)

Provide reference implementations at multiple complexity levels.

**Acceptance Criteria**:
- [ ] `examples/minimal/` exists with simplest possible resource
- [ ] `examples/todo-app/` exists with basic CRUD
- [ ] `examples/blog/` exists with relationships
- [ ] Each example has README with "What you'll learn"
- [ ] Each example is immediately buildable
- [ ] Code includes inline comments explaining syntax choices
- [ ] CLAUDE.md references examples as learning path

**Structure**:
```
examples/
├── minimal/          # 1 resource, 3 fields (id, name, created_at)
├── todo-app/         # CRUD with basic validation
├── blog/             # Relationships (Post, User, Comment)
└── api-with-auth/    # Auth patterns, middleware
```

### FR7: GETTING-STARTED.md Updates
**Priority**: P1 (Should Have)

Align getting started guide with bootstrap-first approach.

**Acceptance Criteria**:
- [ ] "Your First Resource" section shows minimal syntax (not kitchen sink)
- [ ] Explicitly calls out required vs optional elements
- [ ] Added "For LLMs" callout boxes highlighting discovery workflow
- [ ] Bootstrap → Discovery transition made explicit
- [ ] Links to LANGUAGE-SPEC.md for complete syntax reference
- [ ] Shows progression: minimal → intermediate → advanced

**Source**: Current GETTING-STARTED.md (lines 180-199) shows complete example without explaining what's minimal

## Non-Functional Requirements

### Performance
- `conduit scaffold` generation: <500ms
- `conduit introspect stdlib`: <100ms (no build required)
- Error message generation: no perceptible delay

### Scale
- Stdlib introspection supports 100+ functions without performance degradation
- Examples directory can scale to 10+ examples without organizational complexity

### Observability
- Track which scaffold templates are most used (telemetry opt-in)
- Track error message display frequency (which errors hit most often)
- Track time-to-first-build metric (anonymous)

### Security
- Scaffold templates must not include secrets or credentials
- Generated code must follow security best practices (no SQL injection patterns, etc.)

### Privacy
- No user code or project details sent to telemetry
- Only command usage patterns collected (if telemetry enabled)

## Scope In/Out

### In Scope
✅ Documentation improvements (CLAUDE.md, GETTING-STARTED.md)
✅ New CLI commands (`introspect stdlib`, `scaffold`)
✅ Error message improvements
✅ Example projects (minimal, todo, blog)
✅ Quick reference card

### Out of Scope (Future Work)
❌ Interactive `conduit new` with template selection (P2 priority)
❌ `conduit validate` for pre-build syntax checking (P2 priority)
❌ LSP integration for real-time syntax validation (separate project)
❌ Video tutorials or interactive walkthroughs (content team)
❌ Changes to core introspection system (already works well)

## Rollout Plan

### Phase 1: Foundation (Week 1-2, ~3-4 days eng effort)
**Goal**: Eliminate immediate bootstrap blockers with documentation and error messages

1. Update CLAUDE.md template with Bootstrap + Quick Reference sections
2. Improve error messages for common bootstrap failures
3. Create `examples/minimal/` and `examples/todo-app/`
4. Update GETTING-STARTED.md with explicit bootstrap guidance

**Guardrails**:
- Test with fresh Conduit install on clean machine
- Validate with actual LLM (Claude Code session)
- Ensure backward compatibility (existing projects unaffected)

**Success Metric**: LLM can create first resource in <10 minutes (50% improvement)

### Phase 2: Discovery Tools (Week 3-4, ~4-5 days eng effort)
**Goal**: Provide pre-build discovery mechanisms

1. Implement `conduit introspect stdlib` command
2. Implement `conduit scaffold` command with 3 templates (todo, blog, api)
3. Add help text and examples to CLI

**Guardrails**:
- Command help text includes examples
- Generated code is immediately buildable (automated test)
- Stdlib introspection matches actual stdlib (automated sync check)

**Success Metric**: LLM can create first resource in <5 minutes (target achieved)

### Phase 3: Polish (Week 5-6, ~2-3 days eng effort)
**Goal**: Refinement based on usage data

1. Add remaining examples (`examples/api-with-auth/`, `examples/full-stack/`)
2. Refine error messages based on user feedback
3. Add telemetry for bootstrap success tracking (opt-in)

**Success Metric**: >80% LLM success rate on first try

### Kill-Switch Plan
- Documentation changes: revert via git
- New CLI commands: feature flag controlled, can disable via config
- Error messages: can revert to original via feature flag
- Examples: removal doesn't break existing functionality

## Risks & Mitigation

### Risk 1: Documentation Drift
**Impact**: High - Examples/docs become outdated as language evolves

**Mitigation**:
- Automated tests that build all examples on every commit
- CI fails if examples don't compile
- Add "last verified" date to each example's README
- Quarterly review cycle

### Risk 2: Scaffold Template Maintenance
**Impact**: Medium - Templates need updates as best practices change

**Mitigation**:
- Keep scaffold templates minimal (less surface area)
- Use template variables for framework-specific values
- Version templates if breaking changes needed
- Automated tests build scaffolded projects

### Risk 3: Stdlib Introspection Sync
**Impact**: Medium - `introspect stdlib` shows outdated functions

**Mitigation**:
- Generate stdlib introspection data from code (not manual list)
- CI check ensures introspection data is up to date
- Single source of truth for stdlib definitions

### Risk 4: Over-Engineering Bootstrap
**Impact**: Low - Too many options confuse rather than help

**Mitigation**:
- Start with minimal viable guidance
- Iterate based on actual LLM usage patterns
- Prefer simple documentation over complex tooling
- Follow "progressive disclosure" principle

## Open Questions

1. **Should `conduit new` offer template selection interactively?**
   - Decision: Phase 4 (nice-to-have), not blocking bootstrap improvements
   - Rationale: Can add later without breaking existing flow

2. **Should scaffold templates be user-extensible?**
   - Decision: No for MVP, reconsider in Phase 4
   - Rationale: Complexity vs benefit unclear, wait for user demand

3. **Should we include a "validate syntax" command for faster feedback?**
   - Decision: P2 priority, evaluate after Phase 1-2
   - Rationale: Nice-to-have but not critical for bootstrap success

4. **How much telemetry is acceptable for measuring success?**
   - Decision: Opt-in only, anonymous metrics (command usage, not code)
   - Rationale: Privacy-first, but need data to validate improvements

5. **Should CLAUDE.md be generated differently for empty vs populated projects?**
   - Decision: No, single template with bootstrap section that applies to all
   - Rationale: Simpler maintenance, always relevant

## Success Definition (Revisited)

**Primary Success Criterion**: An LLM with zero Conduit experience can create a working CRUD resource and successfully use `conduit introspect schema` within 5 minutes of running `conduit new`.

**Measurement Approach**:
1. Fresh install of Conduit on clean machine
2. Start Claude Code in new project directory
3. Run `conduit new test-project`
4. Task: "Create a Todo resource with title and done fields"
5. Measure: time to first successful `conduit build && conduit introspect schema`
6. Target: <5 minutes, >80% success rate across 10 trials

**Secondary Success Criteria**:
- Zero external documentation lookups needed (everything in CLAUDE.md)
- No hallucinated function names (thanks to stdlib introspection)
- Error messages guide toward solution, not frustration

---

**Document Owner**: Product Manager
**Last Updated**: 2025-10-31
**Status**: Ready for Engineering Review
