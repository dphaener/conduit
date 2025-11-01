# UX Design Brief: LLM Bootstrap Experience

## User Problem & Objective

**User**: LLM coding agents (Claude Code, Cursor, etc.) working on behalf of developers

**Problem**: LLMs encounter a discovery deadlock when starting new Conduit projects. The documentation assumes resources exist and promotes introspection-based learning, but introspection requires a successful build, and building requires syntax knowledge the LLM doesn't have yet.

**Objective**: Design an information architecture and user flow that guides LLMs from empty project to working introspection in <5 minutes with minimal friction.

## Current User Journey (Pain Points)

```
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: Project Initialization (✓ Works Well)                 │
└─────────────────────────────────────────────────────────────────┘
  Developer runs: conduit new my-project
  ↓
  Project structure created with app/resources/ (empty)
  ↓
  CLAUDE.md exists and is read by LLM

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: Discovery Attempt (❌ FAILS - Bootstrap Paradox)      │
└─────────────────────────────────────────────────────────────────┘
  LLM reads CLAUDE.md: "Use discovery-first workflow"
  ↓
  LLM tries: conduit introspect schema
  ↓
  ❌ ERROR: "registry not initialized - run 'conduit build' first"
  │  └─ No guidance on how to proceed
  │  └─ No example of what to build
  │  └─ Dead end

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: Guessing Syntax (❌ High Friction)                    │
└─────────────────────────────────────────────────────────────────┘
  LLM examines CLAUDE.md example:
  ```
  resource Example {
    id: uuid! @primary @auto
    name: string! @min(2) @max(100)
    created_at: timestamp! @auto
  }
  ```
  ↓
  Questions (unanswered):
  • Is created_at required?
  • Can I omit @auto on id?
  • What's the absolute minimum?
  • Where does this file go?
  ↓
  LLM guesses syntax, creates file
  ↓
  conduit build
  ↓
  ❌ Build error (likely cryptic)
  │  └─ No clear fix guidance
  │  └─ Repeat loop

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: External Search (❌ Friction, Context Switching)      │
└─────────────────────────────────────────────────────────────────┘
  LLM searches for LANGUAGE-SPEC.md or examples
  ↓
  Finds scattered information
  ↓
  Pieces together syntax
  ↓
  Eventually succeeds OR gives up

TIME: 15-30 minutes | FRICTION: High | SUCCESS RATE: ~60%
```

## Proposed User Journey (Improved)

```
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: Project Initialization (✓ No change needed)           │
└─────────────────────────────────────────────────────────────────┘
  Developer runs: conduit new my-project
  ↓
  Project structure created, CLAUDE.md available

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: Bootstrap Guidance (✓ NEW - Clear Path Forward)       │
└─────────────────────────────────────────────────────────────────┘
  LLM reads CLAUDE.md
  ↓
  NEW: "Bootstrap (First Resource)" section comes BEFORE discovery
  ↓
  Shows minimal working resource with clear annotations:
  ```
  // Absolute minimum - everything else is optional
  resource Todo {
    id: uuid! @primary @auto      // Required: primary key
    title: string!                // Required: your field
    done: bool! @default(false)   // Optional: add as needed
  }
  ```
  ↓
  Copy-pasteable commands provided:
  ```
  cat > app/resources/todo.cdt << 'EOF'
  [minimal resource]
  EOF
  ```
  ↓
  ✓ LLM creates file successfully

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: First Build (✓ Success + Verification)                │
└─────────────────────────────────────────────────────────────────┘
  LLM runs: conduit build
  ↓
  ✓ BUILD SUCCESS
  ↓
  LLM runs: conduit introspect schema
  ↓
  ✓ Schema displayed
  ↓
  TRANSITION: "Now use discovery-first workflow" marker

┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: Discovery-First Workflow (✓ Original flow now works)  │
└─────────────────────────────────────────────────────────────────┘
  LLM uses introspection for all future changes
  ↓
  Iterative development with fast feedback

TIME: <5 minutes | FRICTION: Low | SUCCESS RATE: >80%
```

## Information Architecture: CLAUDE.md Restructure

### Current Structure (Problems)
```
CLAUDE.md
├── Quick Context ✓
├── Starting from Scratch
│   ├── Step 1: Understand What Exists ❌ (assumes resources exist)
│   ├── Step 2: Create Your First Resource ❌ (too complex example)
│   └── Step 3-6: Build and introspect ✓
├── Discovery Mechanisms ❌ (presented before bootstrap succeeds)
└── Rest of guide...
```

**Problem**: Discovery-first content comes before bootstrap succeeds

### Proposed Structure (Solution)
```
CLAUDE.md
├── Quick Context ✓
│
├── [NEW] Quick Reference Card ⭐
│   ├── When to use: "Before your first build"
│   ├── Minimal syntax template
│   ├── Common types (6 most used)
│   ├── Common directives (6 most used)
│   ├── Common functions (4 most used)
│   └── Pointer to LANGUAGE-SPEC.md
│
├── [ENHANCED] Bootstrap (First Resource) ⭐
│   ├── Clear goal: "Get to working introspection"
│   ├── Absolute minimal resource (3 fields max)
│   ├── Annotations: "Required", "Optional - add if needed"
│   ├── Copy-pasteable commands
│   ├── Build + verify steps
│   └── Explicit transition: "After first build, use discovery below"
│
├── Discovery Mechanisms ✓
│   └── (Existing content, now accessible after bootstrap)
│
└── Rest of guide... ✓
```

**Key Change**: Bootstrap comes first, discovery comes second (when it's actually usable)

## Design Specifications

### Component 1: Quick Reference Card

**Purpose**: Provide syntax lookup without requiring introspection

**Layout**:
```markdown
## Quick Reference (Before First Build)

**When to use**: Can't introspect yet? Use this minimal guide.

### Minimal Resource Template
resource Name {
  id: uuid! @primary @auto          // Primary key (always include)
  your_field: type!                 // Your actual fields here
}

### Common Types
• string!    - Text (e.g., "hello")
• text!      - Long text (e.g., article body)
• int!       - Integer (e.g., 42)
• bool!      - True/false
• uuid!      - Unique identifier
• timestamp! - Date and time

### Common Directives
• @primary        - Mark as primary key
• @auto           - Auto-generate (for uuid, timestamps)
• @default(val)   - Set default value
• @min(n)         - Minimum length/value
• @max(n)         - Maximum length/value
• @unique         - Must be unique across records

### Common Functions (Always Namespaced)
• String.slugify(text)  - Create URL slug
• String.length(text)   - Get length
• Time.now()            - Current time
• Array.length(arr)     - Array size

**Need more?** See LANGUAGE-SPEC.md for complete syntax reference.
```

**Design Principles**:
- Scannable (bullets, short descriptions)
- Progressive disclosure (most common first)
- Clear escape hatch (link to complete docs)
- Emphasizes patterns (namespacing, nullability)

**States**: None (static reference content)

---

### Component 2: Bootstrap Section

**Purpose**: Guide LLM from empty project to first successful build

**Layout**:
```markdown
## Bootstrap (First Resource)

**Goal**: Create your first resource and unlock introspection.

### Step 1: Create Minimal Resource

The absolute minimum working resource:

app/resources/todo.cdt:
```conduit
resource Todo {
  id: uuid! @primary @auto      // Required: primary key
  title: string!                // Required: your content
  done: bool! @default(false)   // Optional: add more fields as needed
}
```

**What's required?**
• `id: uuid! @primary @auto` - Every resource needs this
• At least one custom field (like `title` above)

**What's optional?**
• Additional fields (add as many as needed)
• Validations like @min/@max (add later)
• Relationships (add after understanding basics)

### Step 2: Create the File

[Copy-pasteable command]

### Step 3: Build and Verify

```bash
conduit build
# Should succeed! ✓

conduit introspect schema
# You should see your Todo resource ✓
```

### ✓ Success! Now Use Discovery

After your first build succeeds, you can use the discovery-first workflow below.
All the introspection commands now work.
```

**Design Principles**:
- Action-oriented (clear steps)
- Progressive disclosure (required first, optional later)
- Explicit success markers (✓ checkmarks)
- Clear transition to next phase

**States**:
- Initial: Full guidance visible
- Success: Checkmarks appear after commands succeed

---

### Component 3: Improved Error Messages

**Current Error (Blocking)**:
```
Error: registry not initialized - run 'conduit build' first
```

**Proposed Error (Guiding)**:
```
Error: No resources found. To get started:

1. Create your first resource:

   cat > app/resources/todo.cdt << 'EOF'
   resource Todo {
     id: uuid! @primary @auto
     title: string!
     done: bool! @default(false)
   }
   EOF

2. Build the project:

   conduit build

3. Try introspection again:

   conduit introspect schema

Or use: conduit scaffold todo
```

**Design Principles**:
- Actionable (commands user can run immediately)
- Progressive (numbered steps)
- Copy-pasteable (no retyping needed)
- Helpful alternative (scaffold command)

**States**:
- Error state: Show full guidance
- Success state: Normal command output

---

### Component 4: Scaffold Command UX

**Purpose**: One-command bootstrap with working examples

**Command Structure**:
```bash
conduit scaffold <template>
```

**Available Templates**:
- `todo` - Minimal CRUD (recommended for beginners)
- `blog` - With relationships (Post, User, Comment)
- `api` - Authentication patterns

**Help Text Design**:
```bash
$ conduit scaffold --help

USAGE:
  conduit scaffold <template>

TEMPLATES:
  todo     Simple CRUD resource (recommended for first project)
  blog     Blog with relationships (Post, User, Comment)
  api      REST API with authentication patterns

EXAMPLES:
  conduit scaffold todo    # Creates app/resources/todo.cdt
  conduit scaffold blog    # Creates app/resources/{post,user,comment}.cdt

After scaffolding, run:
  conduit build
  conduit introspect schema
```

**Generated File Comments**:
```conduit
// Scaffolded by: conduit scaffold todo
// Learn more: conduit introspect schema Todo

/// A simple todo item
resource Todo {
  // Every resource needs a unique identifier
  id: uuid! @primary @auto

  // Your custom fields
  title: string! @min(1) @max(200)    // Required text field
  done: bool! @default(false)          // Optional boolean with default

  // Automatic timestamps
  created_at: timestamp! @auto
}

// Next steps:
// 1. Build: conduit build
// 2. Introspect: conduit introspect schema Todo
// 3. Add more fields or create new resources
```

**Design Principles**:
- Immediate feedback (file created message)
- Self-documenting (comments explain syntax)
- Next steps provided (what to do after scaffolding)
- Reinforces discovery (mentions introspection)

---

### Component 5: Examples Structure

**Directory Layout**:
```
examples/
├── minimal/
│   ├── README.md           "The simplest possible Conduit app"
│   ├── app/resources/
│   │   └── item.cdt        // 3 fields only
│   └── conduit.yaml
│
├── todo-app/
│   ├── README.md           "Basic CRUD with validation"
│   ├── app/resources/
│   │   └── todo.cdt        // Validations, defaults
│   └── migrations/
│
├── blog/
│   ├── README.md           "Resources with relationships"
│   ├── app/resources/
│   │   ├── post.cdt        // belongs_to User
│   │   ├── user.cdt
│   │   └── comment.cdt     // belongs_to Post, User
│   └── migrations/
│
└── api-with-auth/
    ├── README.md           "Authentication & middleware"
    ├── app/resources/
    │   ├── user.cdt        // Auth patterns
    │   └── post.cdt        // Middleware examples
    └── migrations/
```

**Example README Template**:
```markdown
# [Example Name]

## What You'll Learn
- Bullet list of concepts demonstrated
- Specific to this example's complexity level

## Quick Start
```bash
cd examples/minimal
conduit build
conduit introspect schema
```

## Files to Examine
- `app/resources/item.cdt` - Learn: minimal resource syntax
- `conduit.yaml` - Learn: basic configuration

## Next Steps
After understanding this example:
1. [Link to next more complex example]
2. [Or modify this example by...]
```

**Design Principles**:
- Progressive complexity (each example builds on previous)
- Self-contained (immediately buildable)
- Educational (README explains what to learn)
- Clear progression path (suggests next example)

---

## All Possible States & Appearances

### State 1: Fresh Project (Empty Resources)
**Appearance**:
- CLAUDE.md Bootstrap section is most relevant
- Quick Reference card available
- Error messages are helpful and actionable

**User Actions**:
- Read Bootstrap section
- Copy minimal resource example
- Run `conduit build`

---

### State 2: First Build Success
**Appearance**:
- Success message confirms build
- Transition marker visible: "Now use discovery-first workflow"
- Introspection commands now work

**User Actions**:
- Run `conduit introspect schema` to verify
- Read Discovery Mechanisms section
- Begin iterative development

---

### State 3: Build Failure
**Appearance**:
- Error message with syntax problem
- Suggests fix or points to Quick Reference
- Offers scaffold alternative if syntax is complex

**User Actions**:
- Review error message
- Check Quick Reference for correct syntax
- Fix and retry build

---

### State 4: Discovery-First Workflow (Post-Bootstrap)
**Appearance**:
- Introspection commands work
- Bootstrap section no longer needed
- Discovery Mechanisms section is primary guide

**User Actions**:
- Use `conduit introspect` for all discovery
- Iterate on resources
- Refer to Quick Reference only for new syntax

---

## Accessibility Notes

**For LLM "Screen Readers"** (text parsing):
- Clear heading hierarchy (H2 for sections, H3 for subsections)
- Code blocks use explicit language tags (```conduit, ```bash)
- Commands preceded by descriptive text
- Explicit transition markers ("Now use...", "After...")
- Navigation hints ("See Quick Reference above")

**For Human Developers** (reviewing LLM work):
- Visual markers (✓, ❌, ⭐) for scanning
- Consistent formatting for code blocks
- Clear section boundaries
- Progress indicators in multi-step processes

**Keyboard Navigation** (CLI):
- All commands work without mouse
- Copy-paste friendly (no hidden characters)
- Tab completion where possible (future)

---

## User Stories & Acceptance Criteria

### Story 1: First-Time Bootstrap
**As an** LLM agent with no Conduit experience
**I want to** create my first working resource
**So that** I can start using introspection for discovery

**Acceptance Criteria**:
- [ ] CLAUDE.md has Bootstrap section before Discovery section
- [ ] Bootstrap section shows minimal 3-field resource
- [ ] Annotations clarify required vs optional syntax
- [ ] Copy-pasteable commands provided for file creation
- [ ] Build + verify steps included
- [ ] Explicit transition to discovery workflow
- [ ] Time to first successful build <5 minutes

---

### Story 2: Syntax Lookup Without Introspection
**As an** LLM agent before first build
**I want to** look up basic syntax
**So that** I don't have to guess or hallucinate

**Acceptance Criteria**:
- [ ] Quick Reference section exists in CLAUDE.md
- [ ] Shows 6+ most common types
- [ ] Shows 6+ most common directives
- [ ] Shows 4+ most common stdlib functions
- [ ] Explicitly states functions are namespaced
- [ ] Links to LANGUAGE-SPEC.md for complete reference
- [ ] Formatted for quick scanning (table or compact list)

---

### Story 3: Error Recovery
**As an** LLM agent hitting a build error
**I want** actionable guidance in the error message
**So that** I can fix the problem without external research

**Acceptance Criteria**:
- [ ] Error "registry not initialized" includes example resource
- [ ] Error shows exact commands to create first resource
- [ ] Error mentions scaffold alternative
- [ ] Error is copy-pasteable
- [ ] Error is concise (<10 lines)
- [ ] Error doesn't require external documentation lookup

---

### Story 4: Quick Scaffold Bootstrap
**As an** LLM agent wanting to skip manual setup
**I want to** generate a working example with one command
**So that** I can reach working introspection immediately

**Acceptance Criteria**:
- [ ] `conduit scaffold todo` command exists
- [ ] Generates valid .cdt file
- [ ] Generated file includes helpful comments
- [ ] File is immediately buildable
- [ ] Command suggests next steps after generation
- [ ] Help text lists available templates
- [ ] Output confirms file location created

---

### Story 5: Learning from Examples
**As an** LLM agent learning Conduit patterns
**I want** example projects at progressive complexity
**So that** I can understand syntax by examining working code

**Acceptance Criteria**:
- [ ] `examples/minimal/` exists and builds successfully
- [ ] `examples/todo-app/` exists with more complexity
- [ ] Each example has README explaining what it demonstrates
- [ ] Code includes inline comments explaining syntax
- [ ] Examples progress from simple → complex
- [ ] Each example suggests next example to try

---

## Technical Constraints

### Constraint 1: Backward Compatibility
**Description**: Changes must not break existing projects or workflows

**Impact on Design**:
- CLAUDE.md changes are additive (new sections, not removed)
- Error messages enhanced, not replaced
- New commands are optional (existing workflow still works)

---

### Constraint 2: Template Maintenance
**Description**: Generated templates must stay in sync with language changes

**Impact on Design**:
- Keep scaffold templates minimal (less to maintain)
- Automated tests that build all examples
- CI fails if examples don't compile
- Version tracking for template changes

---

### Constraint 3: Information Density
**Description**: CLAUDE.md must remain scannable while adding content

**Impact on Design**:
- Use progressive disclosure (essential first, details later)
- Clear visual hierarchy (headings, bullets)
- Compact formatting for Quick Reference
- Explicit navigation markers ("See above", "See below")

---

## Design Deliverables Summary

### Documentation Designs
1. ✅ CLAUDE.md Bootstrap Section (structure defined)
2. ✅ CLAUDE.md Quick Reference Card (layout defined)
3. ✅ Error message format (template provided)

### CLI Command Designs
4. ✅ `conduit scaffold` UX (command structure, help text, output)
5. ✅ `conduit introspect stdlib` UX (output format, filtering)

### Content Structure Designs
6. ✅ Examples directory layout (hierarchy and progression)
7. ✅ Example README template (consistent structure)
8. ✅ Generated code comment patterns (self-documenting)

### State & Flow Designs
9. ✅ User journey map (current vs improved)
10. ✅ Information architecture (CLAUDE.md restructure)
11. ✅ All state definitions (fresh, success, error, discovery)

---

**Document Owner**: UX Designer
**Last Updated**: 2025-10-31
**Status**: Ready for Engineering Review
**Dependencies**: Requires PM approval on scope/priorities
