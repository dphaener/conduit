# Linear Project: LLM Bootstrap Experience Improvements

## Project Overview

**Type**: Initiative (Multiple Tickets)
**Priority**: P0 (Critical for LLM-first mission)
**Est. Total Effort**: 9-12 engineering days
**Target Completion**: 3-4 weeks

### Executive Summary

Conduit's discovery-first philosophy (runtime introspection) breaks down during initial project setup, creating a bootstrap paradox: LLMs need to build to introspect, but need syntax knowledge to write their first resource to build. This initiative eliminates the paradox through documentation improvements, CLI tooling, and actionable error messages.

**Expected Impact**:
- Time to first working build: 15-30 min → <5 min (83% improvement)
- LLM success rate: ~60% → >80% (33% improvement)
- Bootstrap friction eliminated, enabling the discovery-first workflow

**Implementation Approach**: Three-phase rollout prioritizing low-risk documentation changes before new CLI commands.

---

## Project Tickets Breakdown

This project should be broken into **6 tickets** across 3 phases:

### Phase 1: Foundation (High Priority - P0)
✅ **Ticket 1**: Update CLAUDE.md with Bootstrap Guidance
✅ **Ticket 2**: Improve Bootstrap Error Messages
✅ **Ticket 3**: Create Minimal Examples

### Phase 2: Discovery Tools (High Priority - P1)
✅ **Ticket 4**: Implement `conduit introspect stdlib` Command
✅ **Ticket 5**: Implement `conduit scaffold` Command

### Phase 3: Polish (Medium Priority - P2)
✅ **Ticket 6**: Additional Examples and Refinement

---

## TICKET 1: Update CLAUDE.md with Bootstrap Guidance

### Type
Documentation Enhancement

### Priority
P0 (Blocking - critical for bootstrap success)

### Estimated Effort
4 hours

### Description

**Problem**: Current CLAUDE.md assumes resources already exist and promotes discovery-first workflow before bootstrap succeeds, creating confusion for LLMs starting from empty project.

**Solution**: Restructure CLAUDE.md to provide bootstrap guidance BEFORE discovery mechanisms, with explicit minimal syntax reference and transition point.

**Changes Required**:
1. Add "Quick Reference Card" section after "Quick Context" (non-introspection syntax lookup)
2. Add "Bootstrap (First Resource)" section before "Discovery Mechanisms"
3. Update existing examples to annotate required vs optional elements
4. Add explicit transition: "After first build, use discovery-first workflow"

### Acceptance Criteria

- [ ] New "Quick Reference Card" section exists after "Quick Context"
- [ ] Quick Reference shows:
  - Minimal resource template with annotations
  - 6+ most common field types (string!, int!, bool!, uuid!, timestamp!, text!)
  - 6+ most common directives (@primary, @auto, @default, @min, @max, @unique)
  - 4+ most common stdlib functions (String.slugify, Time.now, etc.)
  - Explicit statement: "Functions are always namespaced"
  - Link to LANGUAGE-SPEC.md for complete reference
- [ ] New "Bootstrap (First Resource)" section exists before "Discovery Mechanisms"
- [ ] Bootstrap section shows:
  - Absolute minimal working resource (3-4 fields max)
  - Clear annotations: "Required", "Optional - add if needed"
  - Copy-pasteable commands to create file
  - Build + verify steps
  - Explicit transition: "After first build succeeds, use discovery below"
- [ ] Updated GETTING-STARTED.md with same approach
- [ ] Manual review by PM and UX confirms clarity
- [ ] Test with fresh Claude Code session confirms effectiveness

### Technical Details

**File to Modify**: `/Users/darinhaener/code/conduit/internal/templates/claude_md.go`

**Insertion Points**:
- Quick Reference: After line ~44 ("Quick Context" section)
- Bootstrap: Before line ~104 ("Discovery Mechanisms" section)

**Also Update**: `/Users/darinhaener/code/conduit/GETTING-STARTED.md`

### Dependencies
None (can start immediately)

### Testing Plan
1. Generate new CLAUDE.md from template
2. Manual review for clarity and scannability
3. Fresh project test with Claude Code
4. Measure time to first successful build

### Definition of Done
- Template updated and generates correct CLAUDE.md
- GETTING-STARTED.md aligned with new approach
- Tested with LLM in fresh project
- Documentation changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR1, FR2, FR7)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Components 1, 2)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 1, 2, 7)

---

## TICKET 2: Improve Bootstrap Error Messages

### Type
Developer Experience Enhancement

### Priority
P0 (Blocking - critical for bootstrap guidance)

### Estimated Effort
4 hours

### Description

**Problem**: When LLMs encounter errors during bootstrap (e.g., "registry not initialized"), the error messages provide no guidance on how to proceed, creating dead-ends.

**Solution**: Transform blocking errors into actionable guidance that shows LLMs exactly what commands to run and what syntax to use.

**Errors to Enhance**:
1. "registry not initialized" → show minimal resource + commands
2. "no resources found" → provide bootstrap guidance
3. Relevant build/syntax errors → reference Quick Reference card

### Acceptance Criteria

- [ ] Error "registry not initialized" shows:
  - Example minimal resource syntax
  - Exact commands to create first resource (copy-pasteable)
  - Reference to `conduit scaffold` as alternative
  - Clear next steps
- [ ] Error message is concise (≤10 lines) but actionable
- [ ] Error includes proper file paths (app/resources/...)
- [ ] Error tested in actual failure scenarios
- [ ] All relevant build errors reference Quick Reference when appropriate
- [ ] No breaking changes to error handling logic
- [ ] Manual test confirms LLM can follow error guidance successfully

### Technical Details

**Files to Modify**:
- `internal/compiler/compiler.go` (build errors)
- `internal/cli/commands/introspect.go` (introspection errors)
- `internal/runtime/registry.go` (registry errors)

**Implementation Pattern**:
```go
// Before
return fmt.Errorf("registry not initialized")

// After
return fmt.Errorf(`registry not initialized - no resources found yet

To create your first resource:

  cat > app/resources/todo.cdt << 'EOF'
  resource Todo {
    id: uuid! @primary @auto
    title: string!
  }
  EOF

Then run: conduit build

Or use: conduit scaffold todo
`)
```

### Dependencies
- Ticket 1 (references Quick Reference and scaffold in errors)
- Should be developed in parallel with Ticket 1

### Testing Plan
1. Unit tests for error message formatting
2. Integration tests triggering actual error scenarios
3. Manual test with LLM following error guidance
4. Verify errors remain helpful but not verbose

### Definition of Done
- Enhanced error messages implemented
- Tests pass for all error scenarios
- Manual LLM test confirms guidance is followable
- No regressions in error handling
- Changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR3)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Component 3)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 3)

---

## TICKET 3: Create Minimal Working Examples

### Type
Content Creation + CI Integration

### Priority
P0 (Blocking - provides working code for learning)

### Estimated Effort
6 hours

### Description

**Problem**: LLMs have no working code to examine and learn from in empty projects. Need reference implementations at progressive complexity levels.

**Solution**: Create examples directory with minimal → intermediate examples that are immediately buildable and demonstrate specific patterns.

**Examples to Create**:
1. `examples/minimal/` - Simplest possible (1 resource, 3 fields)
2. `examples/todo-app/` - Basic CRUD with validation

### Acceptance Criteria

- [ ] `examples/minimal/` exists with:
  - Single resource with 3 fields (id, name, created_at)
  - README explaining "What You'll Learn"
  - Immediately buildable (conduit build succeeds)
  - Commented code explaining syntax choices
- [ ] `examples/todo-app/` exists with:
  - CRUD resource with validation
  - Demonstrates @min/@max, @default
  - README explains concepts
  - Immediately buildable
- [ ] Each example has:
  - Clear README with Quick Start section
  - Inline code comments explaining patterns
  - "Next Steps" section pointing to more complex examples
- [ ] CI workflow created to build all examples on every commit
- [ ] CI fails if examples don't compile
- [ ] Examples included in CLAUDE.md as learning path
- [ ] All examples tested and build successfully

### Technical Details

**New Directory Structure**:
```
examples/
├── minimal/
│   ├── README.md
│   ├── app/resources/item.cdt
│   └── conduit.yaml
└── todo-app/
    ├── README.md
    ├── app/resources/todo.cdt
    └── conduit.yaml
```

**CI Workflow**: `.github/workflows/examples.yml`
```yaml
name: Validate Examples
on: [push, pull_request]
jobs:
  test-examples:
    strategy:
      matrix:
        example: [minimal, todo-app]
    steps:
      - name: Build Example
        run: cd examples/${{ matrix.example }} && ../../conduit build
```

### Dependencies
None (can start immediately, parallel with Ticket 1-2)

### Testing Plan
1. Manual build test for each example
2. CI automation validates on every commit
3. README clarity review
4. LLM test: can it learn from examples?

### Definition of Done
- Both examples created and documented
- All examples build successfully
- CI workflow in place and passing
- READMEs reviewed for clarity
- Examples referenced in CLAUDE.md
- Changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR6)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Component 5)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 6)

---

## TICKET 4: Implement `conduit introspect stdlib` Command

### Type
Feature - New CLI Command

### Priority
P1 (High - enables discovery before build)

### Estimated Effort
1 day (8 hours)

### Description

**Problem**: LLMs cannot discover stdlib functions before first successful build, leading to hallucination of function names (e.g., `slugify()` instead of `String.slugify()`).

**Solution**: New CLI command that lists all stdlib functions organized by namespace, available BEFORE build. Uses static registry that doesn't require runtime initialization.

**Command**: `conduit introspect stdlib [namespace]`

### Acceptance Criteria

- [ ] Command `conduit introspect stdlib` works without requiring build
- [ ] Lists all stdlib functions organized by namespace (String, Time, Array, etc.)
- [ ] Shows function signatures: `String.slugify(text: string!) -> string!`
- [ ] Includes brief descriptions (1 line per function)
- [ ] Supports namespace filtering: `conduit introspect stdlib String`
- [ ] Supports JSON output: `--format json`
- [ ] Help text is clear and includes examples
- [ ] Output is LLM-friendly (structured, easy to parse)
- [ ] Unit tests for registry completeness
- [ ] Integration test verifies command works in fresh project
- [ ] CI check ensures registry stays in sync with actual stdlib

### Technical Details

**New Files**:
- `internal/cli/commands/introspect_stdlib.go` (command implementation)
- `internal/compiler/stdlib/registry.go` (function registry)

**Registry Structure**:
```go
type FunctionDef struct {
    Name        string
    Signature   string
    Description string
}

var StdlibRegistry = map[string][]FunctionDef{
    "String": {...},
    "Time": {...},
    "Array": {...},
}
```

**Output Format** (human-readable):
```
String Functions:
  String.slugify(text: string!) -> string!
    Create URL-friendly slug from text

  String.length(text: string!) -> int!
    Get the length of a string
```

**Output Format** (JSON):
```json
{
  "namespaces": {
    "String": [
      {
        "name": "slugify",
        "signature": "(text: string!) -> string!",
        "description": "Create URL-friendly slug"
      }
    ]
  }
}
```

### Dependencies
- None (standalone feature)
- Should reference this command in Ticket 2 error messages

### Testing Plan
1. Unit test: registry completeness, all functions have required fields
2. Integration test: command runs without build
3. CI sync test: ensures registry matches actual stdlib implementation
4. Manual test: LLM uses output successfully

### Definition of Done
- Command implemented and registered
- Registry populated with all stdlib functions
- Tests passing (unit + integration + CI sync)
- Help text reviewed and clear
- Manual LLM test confirms usefulness
- Changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR4)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Component 4)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 4)

---

## TICKET 5: Implement `conduit scaffold` Command

### Type
Feature - New CLI Command

### Priority
P1 (High - provides one-command bootstrap)

### Estimated Effort
2 days (16 hours)

### Description

**Problem**: Even with documentation, LLMs still need to manually create first resource file. A one-command solution would be faster and less error-prone.

**Solution**: New CLI command that generates working resource files from templates. Provides immediate path to working introspection.

**Command**: `conduit scaffold <template>`
**Templates**: todo, blog, api

### Acceptance Criteria

- [ ] Command `conduit scaffold <template>` implemented
- [ ] Template `todo` generates minimal CRUD resource
- [ ] Template `blog` generates Post + User + Comment (demonstrates relationships)
- [ ] Template `api` generates resource with auth patterns
- [ ] Generated files include helpful comments explaining syntax
- [ ] Command checks for existing files (prevents overwrite)
- [ ] Command verifies project initialized (conduit.yaml exists)
- [ ] Creates directories if missing (app/resources/)
- [ ] Shows clear success message with next steps
- [ ] Help text lists available templates with descriptions
- [ ] Generated code is immediately buildable
- [ ] Unit tests for template generation logic
- [ ] Integration tests: scaffold → build → introspect pipeline
- [ ] CI validates all scaffolds build successfully

### Technical Details

**New Files**:
- `internal/cli/commands/scaffold.go` (command implementation)
- `internal/templates/scaffolds/todo.go` (todo template)
- `internal/templates/scaffolds/blog.go` (blog template)
- `internal/templates/scaffolds/api.go` (api template)

**Template Structure**:
```go
type ScaffoldTemplate struct {
    Name        string
    Description string
    Files       []FileTemplate
}

type FileTemplate struct {
    Path    string  // e.g., "app/resources/todo.cdt"
    Content string  // Template content with comments
}
```

**Generated File Comments**:
```conduit
// Generated by: conduit scaffold todo
// Learn more: conduit introspect schema Todo

resource Todo {
  // Every resource needs a unique identifier
  id: uuid! @primary @auto

  // Your custom fields
  title: string! @min(1) @max(200)
  ...
}
```

**Command Flow**:
1. Verify project initialized (conduit.yaml exists)
2. Check for file conflicts
3. Create directories if needed
4. Write files
5. Show success + next steps

### Dependencies
- Should be referenced in Ticket 2 error messages
- Templates can be refined based on Ticket 3 examples

### Testing Plan
1. Unit test: template validation, file generation
2. Integration test: scaffold → build → introspect pipeline
3. CI test: all scaffolds build successfully
4. Manual test: LLM uses scaffold successfully
5. Edge case tests: existing files, missing directories, not in project

### Definition of Done
- Command implemented with 3 templates
- All templates generate buildable code
- Tests passing (unit + integration + CI)
- Help text clear and actionable
- Manual LLM test confirms usefulness
- Changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR5)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Component 4)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 5)

---

## TICKET 6: Additional Examples and Polish

### Type
Enhancement + Content

### Priority
P2 (Medium - polish and expansion)

### Estimated Effort
1-1.5 days (8-12 hours)

### Description

**Problem**: After Phase 1-2, we have basic bootstrap working but could expand examples to cover more patterns and refine based on usage feedback.

**Solution**: Add more complex examples (blog with relationships, API with auth) and refine error messages based on real-world LLM usage.

**Scope**:
1. Create `examples/blog/` (relationships)
2. Create `examples/api-with-auth/` (auth patterns)
3. Refine error messages based on feedback
4. Optional: Validate command if valuable

### Acceptance Criteria

- [ ] `examples/blog/` exists with:
  - Post, User, Comment resources
  - Demonstrates belongs_to, has_many relationships
  - Shows on_delete behaviors
  - README explains relationship patterns
  - Immediately buildable
- [ ] `examples/api-with-auth/` exists with:
  - User resource with authentication
  - Middleware examples
  - README explains auth patterns
  - Immediately buildable
- [ ] Error messages refined based on actual LLM usage patterns
- [ ] All examples added to CI validation
- [ ] CLAUDE.md updated to reference new examples
- [ ] Success metrics measured:
  - Time to first build <5 minutes
  - LLM success rate >80%
  - Error message effectiveness improved

### Technical Details

**New Examples**:
```
examples/
├── blog/
│   ├── README.md
│   ├── app/resources/
│   │   ├── post.cdt (belongs_to User)
│   │   ├── user.cdt
│   │   └── comment.cdt (belongs_to Post, User)
│   └── conduit.yaml
└── api-with-auth/
    ├── README.md
    ├── app/resources/
    │   ├── user.cdt (auth patterns)
    │   └── post.cdt (middleware)
    └── conduit.yaml
```

**Refinement Areas**:
- Error messages that are still confusing
- Additional stdlib functions in registry
- Scaffold template improvements
- Documentation clarity based on feedback

### Dependencies
- Requires Phase 1-2 completion
- Should gather usage data from real LLM sessions first

### Testing Plan
1. Build all new examples
2. CI validation
3. LLM walkthrough with new examples
4. Measure success metrics
5. Validate improvements meet targets

### Definition of Done
- New examples created and building
- CI passing for all examples
- Error message refinements implemented
- Success metrics validated (>80% success rate, <5 min)
- Optional features evaluated and implemented if valuable
- Changes merged to main

### Links
- PM Analysis: `docs/tickets/llm-bootstrap-pm-analysis.md` (FR6, Phase 3)
- UX Design: `docs/tickets/llm-bootstrap-ux-analysis.md` (Component 5)
- Eng Plan: `docs/tickets/llm-bootstrap-eng-analysis.md` (Solution 6, 8, 9)

---

## Project Success Criteria

### Quantitative Metrics
- ✅ Time to first successful build: <5 minutes (from 15-30 min baseline)
- ✅ LLM success rate on first try: >80% (from ~60% baseline)
- ✅ Number of build attempts before success: 1-2 attempts (from 3-5 baseline)
- ✅ Error message helpfulness rating: >4/5 (from <2/5 baseline)

### Qualitative Metrics
- ✅ LLM can create first resource without external documentation lookup
- ✅ LLM can discover stdlib functions before first build
- ✅ Error messages provide actionable next steps
- ✅ CLAUDE.md clearly delineates bootstrap → discovery transition
- ✅ Examples provide clear learning path at progressive complexity

### Testing Validation
- ✅ Fresh Claude Code session completes bootstrap in <5 min (10 trials, >80% success)
- ✅ All error messages tested and confirmed actionable
- ✅ All examples build successfully in CI
- ✅ Stdlib introspection command works pre-build
- ✅ Scaffold command generates valid, buildable code

---

## Implementation Timeline

### Week 1-2: Phase 1 (Foundation)
- **Days 1-2**: Ticket 1 + 2 (Documentation + Errors) - Parallel development
- **Days 3-4**: Ticket 3 (Examples + CI)
- **Day 5**: Integration testing, refinement

**Milestone**: LLM can bootstrap in <10 min (50% improvement)

### Week 3-4: Phase 2 (Discovery Tools)
- **Days 1-2**: Ticket 4 (introspect stdlib command)
- **Days 3-4**: Ticket 5 (scaffold command)
- **Day 5**: Integration testing, refinement

**Milestone**: LLM can bootstrap in <5 min (target achieved)

### Week 5-6: Phase 3 (Polish)
- **Days 1-2**: Ticket 6 (Additional examples)
- **Days 3-4**: Refinement based on feedback
- **Day 5**: Final validation, metrics collection

**Milestone**: >80% LLM success rate confirmed

---

## Rollback Strategy

### Phase 1 Rollback
- **Trigger**: Documentation confuses rather than helps
- **Action**: Git revert of template changes
- **Recovery Time**: <5 minutes
- **Risk**: Minimal (documentation only)

### Phase 2 Rollback
- **Trigger**: Commands cause issues or confusion
- **Action**: Feature flag to disable commands
- **Recovery Time**: <1 hour (config deploy)
- **Risk**: Low (commands are additive)

### Full Project Rollback
- **Trigger**: Major unforeseen issues
- **Action**: Revert all changes, return to baseline
- **Recovery Time**: <1 day
- **Risk**: Minimal (all changes are additive, no breaking modifications)

---

## Dependencies & Constraints

### External Dependencies
- None (all changes internal to Conduit)

### Internal Dependencies
- Ticket 2 depends on Ticket 1 (references Quick Reference and scaffold)
- Ticket 5 templates can leverage Ticket 3 examples
- Ticket 6 requires data from Phase 1-2 usage

### Technical Constraints
- Must maintain backward compatibility
- No breaking changes to existing projects
- Error messages must remain concise (<10 lines)
- Examples must stay synchronized with language evolution

### Resource Constraints
- 1 senior engineer for implementation
- PM and UX for review and validation
- LLM testing requires Claude Code access

---

## Risk Assessment

### High-Level Risks

**Risk 1: Documentation Doesn't Help**
- **Mitigation**: Test with actual LLM sessions before merging
- **Likelihood**: Low
- **Impact**: High

**Risk 2: Stdlib Registry Drift**
- **Mitigation**: CI sync check, automated tests
- **Likelihood**: Medium
- **Impact**: Medium

**Risk 3: Example Maintenance Burden**
- **Mitigation**: CI builds examples, fails on incompatibility
- **Likelihood**: Medium
- **Impact**: Low

Overall Project Risk: **Low** (mostly additive, well-scoped changes)

---

## Communication Plan

### Stakeholder Updates
- **Weekly**: Progress update on completed tickets
- **Phase Completion**: Demo of improvements to team
- **Project Completion**: Metrics report and retrospective

### User Communication
- **Release Notes**: Highlight bootstrap improvements
- **Blog Post**: "Conduit's New LLM-First Bootstrap Experience"
- **Documentation**: Updated getting started guides

### Team Communication
- **Kickoff**: Project overview, ticket breakdown
- **Daily**: Stand-up updates on progress
- **Blockers**: Immediate escalation to PM

---

## Retrospective Questions

After project completion:
1. Did we achieve <5 min bootstrap time target?
2. Did LLM success rate exceed 80%?
3. Which phase had the most impact?
4. What unexpected issues did we encounter?
5. What would we do differently next time?
6. How well did our effort estimates match reality?

---

**Project Owner**: Product Manager
**Engineering Lead**: Senior Software Engineer
**UX Lead**: UX Designer
**Created**: 2025-10-31
**Status**: Ready for Kickoff
**Next Step**: Create Linear project and tickets, assign engineers

---

## Appendix: Supporting Documents

- **PM Analysis**: `docs/tickets/llm-bootstrap-pm-analysis.md`
- **UX Design**: `docs/tickets/llm-bootstrap-ux-analysis.md`
- **Engineering Plan**: `docs/tickets/llm-bootstrap-eng-analysis.md`
- **Current Templates**: `internal/templates/claude_md.go`
- **Introspection Docs**: `docs/introspection/README.md`
