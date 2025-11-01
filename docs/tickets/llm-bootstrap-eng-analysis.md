# Technical Implementation Plan: LLM Bootstrap Experience

## Executive Summary

**Goal**: Eliminate LLM bootstrap paradox through documentation, tooling, and error messaging improvements.

**Approach**: Three-phase rollout prioritizing documentation changes (immediate impact, low cost) before new CLI commands (higher impact, moderate cost).

**Total Effort**: 9-12 engineering days across 3 phases
**Risk Level**: Low (mostly additive changes, no breaking modifications)

## Technical Feasibility Assessment

### Solution 1: Enhanced CLAUDE.md Bootstrap Section
**Complexity**: Low
**Effort**: 2-3 hours
**Risk**: Minimal (documentation only)

**Implementation**: Template modification in `/Users/darinhaener/code/conduit/internal/templates/claude_md.go`

**Changes Required**:
```go
// Add bootstrap section after Quick Context (line ~44)
// Insert before "Discovery Mechanisms" section (line ~104)
```

**Feasibility**: ✅ Straightforward - just template string modification
**Blocker**: None

---

### Solution 2: Quick Reference Card in CLAUDE.md
**Complexity**: Low
**Effort**: 1-2 hours
**Risk**: Minimal (documentation only)

**Implementation**: Template modification in same file

**Data Source**:
- Types: hardcoded list from LANGUAGE-SPEC.md primitives
- Directives: hardcoded from compiler's recognized directives
- Functions: could reference stdlib registry (future: make dynamic)

**Feasibility**: ✅ Straightforward - static content initially, can enhance later
**Blocker**: None

---

### Solution 3: Improved Error Messages
**Complexity**: Low-Medium
**Effort**: 3-4 hours
**Risk**: Low (enhancement, not replacement)

**Implementation Locations**:
```
internal/compiler/compiler.go (build errors)
internal/cli/commands/introspect.go (introspection errors)
internal/runtime/registry.go (registry errors)
```

**Specific Errors to Enhance**:
1. "registry not initialized" → add bootstrap guidance
2. "no resources found" → show minimal resource example
3. "invalid syntax" → reference Quick Reference card

**Approach**:
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

**Feasibility**: ✅ Low risk - enhanced errors don't break existing handling
**Blocker**: None

---

### Solution 4: `conduit introspect stdlib` Command
**Complexity**: Medium
**Effort**: 6-8 hours
**Risk**: Low (new command, no existing behavior affected)

**Implementation**:

1. **Create Command**: `internal/cli/commands/introspect_stdlib.go`

2. **Data Source Options**:

   **Option A (Recommended)**: Static registry from compiler
   ```go
   // internal/compiler/stdlib/registry.go
   var StdlibFunctions = map[string][]FunctionDef{
       "String": {
           {Name: "slugify", Signature: "(text: string!) -> string!", Desc: "Create URL-friendly slug"},
           {Name: "length", Signature: "(text: string!) -> int!", Desc: "Get string length"},
           // ...
       },
       "Time": {
           {Name: "now", Signature: "() -> timestamp!", Desc: "Current timestamp"},
           // ...
       },
   }
   ```

   **Pros**: Works before build, simple to implement, easy to maintain
   **Cons**: Manual updates when stdlib changes (mitigated by tests)

   **Option B**: Dynamic from runtime
   ```go
   // Query actual stdlib implementation via reflection
   ```
   **Pros**: Always in sync
   **Cons**: Complex, requires runtime initialization, doesn't work pre-build

   **Decision**: Use Option A (static registry) for MVP

3. **Output Format**:
   ```
   String Functions:
     String.slugify(text: string!) -> string!
       Create URL-friendly slug from text

     String.length(text: string!) -> int!
       Get the length of a string

   Time Functions:
     Time.now() -> timestamp!
       Get current timestamp
   ```

4. **JSON Output**:
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

**Testing**:
- Unit test: verify all stdlib functions listed
- Integration test: command runs without build
- CI check: ensure registry stays in sync with actual stdlib

**Feasibility**: ✅ Straightforward implementation, clear data source
**Blocker**: None (can implement independently)

---

### Solution 5: `conduit scaffold` Command
**Complexity**: Medium
**Effort**: 1-2 days
**Risk**: Medium (template management, file generation)

**Implementation**:

1. **Command Structure**: `internal/cli/commands/scaffold.go`

2. **Template System**:
   ```
   internal/templates/scaffolds/
   ├── todo.go          // Minimal CRUD template
   ├── blog.go          // Relationship template
   └── api.go           // Auth template
   ```

3. **Template Definition**:
   ```go
   type ScaffoldTemplate struct {
       Name        string
       Description string
       Files       []FileTemplate
   }

   type FileTemplate struct {
       Path    string  // e.g., "app/resources/todo.cdt"
       Content string  // Template content
   }

   var TodoScaffold = ScaffoldTemplate{
       Name: "todo",
       Description: "Simple CRUD resource",
       Files: []FileTemplate{
           {
               Path: "app/resources/todo.cdt",
               Content: `// Generated by conduit scaffold todo
   resource Todo {
     id: uuid! @primary @auto
     title: string! @min(1) @max(200)
     done: bool! @default(false)
     created_at: timestamp! @auto
   }`,
           },
       },
   }
   ```

4. **Generation Logic**:
   ```go
   func (s *ScaffoldCommand) Execute(template string) error {
       // 1. Look up template
       tmpl, err := templates.GetScaffold(template)
       if err != nil {
           return fmt.Errorf("unknown template: %s", template)
       }

       // 2. Check for existing files (prevent overwrite)
       for _, file := range tmpl.Files {
           if fileExists(file.Path) {
               return fmt.Errorf("file already exists: %s", file.Path)
           }
       }

       // 3. Create directories if needed
       for _, file := range tmpl.Files {
           if err := os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
               return err
           }
       }

       // 4. Write files
       for _, file := range tmpl.Files {
           if err := os.WriteFile(file.Path, []byte(file.Content), 0644); err != nil {
               return err
           }
           fmt.Printf("Created: %s\n", file.Path)
       }

       // 5. Show next steps
       fmt.Println("\nNext steps:")
       fmt.Println("  conduit build")
       fmt.Println("  conduit introspect schema")

       return nil
   }
   ```

**Edge Cases**:
- Existing files (prevent overwrite, show error)
- Missing parent directories (create automatically)
- Invalid template name (show available templates)
- Project not initialized (check for conduit.yaml)

**Testing**:
- Unit test: template parsing and generation
- Integration test: generate scaffold, verify buildable
- CI: automated build test for all scaffolds

**Feasibility**: ✅ Moderate complexity but well-scoped
**Blocker**: None, uses existing template infrastructure

---

### Solution 6: Examples Directory
**Complexity**: Low (content creation, not code)
**Effort**: 4-6 hours
**Risk**: Low (maintenance burden mitigated by CI)

**Structure**:
```
examples/
├── minimal/
│   ├── README.md
│   ├── app/resources/item.cdt
│   └── conduit.yaml
├── todo-app/
│   ├── README.md
│   ├── app/resources/todo.cdt
│   └── conduit.yaml
├── blog/
│   ├── README.md
│   ├── app/resources/
│   │   ├── post.cdt
│   │   ├── user.cdt
│   │   └── comment.cdt
│   └── conduit.yaml
└── api-with-auth/
    └── [similar structure]
```

**CI Integration**:
```yaml
# .github/workflows/examples.yml
name: Validate Examples

on: [push, pull_request]

jobs:
  build-examples:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        example: [minimal, todo-app, blog, api-with-auth]
    steps:
      - uses: actions/checkout@v3
      - name: Build example
        run: |
          cd examples/${{ matrix.example }}
          ../../conduit build
          ../../conduit introspect schema
```

**Feasibility**: ✅ Straightforward, low risk
**Maintenance**: Automated via CI, fails on incompatibility

---

### Solution 7: GETTING-STARTED.md Updates
**Complexity**: Low
**Effort**: 2-3 hours
**Risk**: Minimal (documentation only)

**Changes Required**:
1. Simplify "Your First Resource" example (remove optional elements)
2. Add annotations: "Required", "Optional - add if needed"
3. Insert "For LLMs" callout boxes
4. Link to LANGUAGE-SPEC.md for complete reference
5. Show bootstrap → discovery progression

**Location**: `/Users/darinhaener/code/conduit/GETTING-STARTED.md`

**Feasibility**: ✅ Straightforward documentation work
**Blocker**: None

---

### Solution 8: `conduit validate` Command
**Complexity**: Medium-High
**Effort**: 1.5-2 days
**Risk**: Medium (requires partial compilation without full build)

**Approach**:
```go
// Parse and validate syntax without code generation
func ValidateResource(filepath string) error {
    // 1. Lex and parse .cdt file
    lexer := NewLexer(content)
    parser := NewParser(lexer)
    ast, err := parser.Parse()
    if err != nil {
        return fmt.Errorf("syntax error: %w", err)
    }

    // 2. Run type checker (without code gen)
    checker := NewTypeChecker()
    if err := checker.Validate(ast); err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    return nil  // Valid syntax
}
```

**Value**: Faster feedback loop (no need for full build)

**Priority**: P2 (nice-to-have, not critical for bootstrap)

**Feasibility**: ✅ Feasible but lower priority
**Decision**: Defer to Phase 3 or later

---

### Solution 9: Interactive `conduit new` Mode
**Complexity**: Medium
**Effort**: 1 day
**Risk**: Low (enhancement to existing command)

**Implementation**:
```go
// Use survey/promptui library for interactive prompts
import "github.com/AlecAivazis/survey/v2"

func InteractiveNew(projectName string) error {
    var template string
    prompt := &survey.Select{
        Message: "Choose a template:",
        Options: []string{
            "empty (start from scratch)",
            "todo (simple CRUD)",
            "blog (with relationships)",
            "api (REST API with auth)",
        },
    }
    survey.AskOne(prompt, &template)

    // Generate based on selection
    // ...
}
```

**Priority**: P2 (nice-to-have, not critical for bootstrap)

**Feasibility**: ✅ Feasible but lower priority
**Decision**: Defer to Phase 3 or later

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2) ⭐ Priority
**Goal**: Eliminate immediate bootstrap blockers with zero-risk changes

**Scope**:
1. ✅ Update CLAUDE.md template (Bootstrap + Quick Reference sections)
2. ✅ Improve error messages (3-4 key errors)
3. ✅ Create examples/minimal/ and examples/todo-app/
4. ✅ Update GETTING-STARTED.md

**Effort Estimate**: 3-4 days
- CLAUDE.md changes: 4 hours
- Error messages: 4 hours
- Examples creation: 6 hours
- GETTING-STARTED.md: 3 hours
- Testing and iteration: 1 day

**Files Modified**:
- `internal/templates/claude_md.go` (add sections)
- `internal/compiler/compiler.go` (error messages)
- `internal/cli/commands/introspect.go` (error messages)
- `examples/minimal/*` (new)
- `examples/todo-app/*` (new)
- `GETTING-STARTED.md` (updates)
- `.github/workflows/examples.yml` (new CI job)

**Testing Strategy**:
- Manual: fresh project walkthrough with Claude Code
- Automated: CI builds all examples
- Validation: docs review for clarity

**Success Criteria**:
- LLM creates first resource in <10 minutes (baseline)
- Error messages tested and helpful
- Examples build successfully in CI
- Zero breaking changes to existing projects

**Risk**: ⬇️ Minimal (all additive changes)

---

### Phase 2: Discovery Tools (Week 3-4)
**Goal**: Provide pre-build discovery mechanisms

**Scope**:
1. ✅ Implement `conduit introspect stdlib` command
2. ✅ Implement `conduit scaffold` command (3 templates)
3. ✅ Add help text and examples to CLI

**Effort Estimate**: 4-5 days
- `introspect stdlib`: 1 day (includes stdlib registry creation)
- `scaffold` command: 2 days (includes template system)
- Testing and integration: 1.5 days
- Documentation: 0.5 days

**Files Created/Modified**:
- `internal/cli/commands/introspect_stdlib.go` (new)
- `internal/compiler/stdlib/registry.go` (new)
- `internal/cli/commands/scaffold.go` (new)
- `internal/templates/scaffolds/*.go` (new)
- `cmd/conduit/main.go` (register commands)

**Testing Strategy**:
- Unit: stdlib registry completeness
- Unit: scaffold template generation
- Integration: commands work in fresh project
- CI: scaffold → build → introspect pipeline
- Manual: LLM usage validation

**Success Criteria**:
- `conduit introspect stdlib` lists all stdlib functions
- `conduit scaffold todo` generates buildable project
- Command help text is clear and actionable
- LLM creates first resource in <5 minutes (target)

**Risk**: ⬇️ Low (new commands, no breaking changes)

**Dependencies**:
- Phase 1 completion (error messages reference scaffold)
- Stdlib registry needs initial population (one-time effort)

---

### Phase 3: Polish & Additional Examples (Week 5-6)
**Goal**: Refinement based on usage feedback

**Scope**:
1. ✅ Create examples/blog/ and examples/api-with-auth/
2. ✅ Refine error messages based on user feedback
3. ⚠️ Optional: Implement `conduit validate` if bandwidth allows
4. ⚠️ Optional: Interactive `conduit new` if user demand

**Effort Estimate**: 2-3 days
- Additional examples: 1 day
- Error message refinement: 0.5 days
- Optional features: 1-1.5 days (if pursued)

**Success Criteria**:
- >80% LLM success rate on first try
- Complete example suite covering common patterns
- Optional features validated with users before implementation

**Risk**: ⬇️ Low (polish phase, no critical path items)

**Decision Gates**:
- Validate feature (only if faster feedback demonstrated valuable)
- Interactive new (only if user research shows demand)

---

## Detailed Implementation: Critical Components

### Component 1: CLAUDE.md Bootstrap Section

**File**: `internal/templates/claude_md.go`

**Insertion Point**: After "Quick Context" (line ~44), before "Discovery Mechanisms" (line ~104)

**Implementation**:
```go
func GetCLAUDEMDContent() string {
    return `# CLAUDE.md

## Quick Context (30-Second Orientation)
[existing content...]

## Quick Reference (Before First Build)

**When to use**: Can't introspect yet? Use this minimal guide to get started.

### Minimal Resource Template
` + "```conduit" + `
resource Name {
  id: uuid! @primary @auto          // Primary key (always include)
  your_field: type!                 // Your actual fields here
}
` + "```" + `

### Common Field Types
• string!    - Text (e.g., "hello")
• text!      - Long text
• int!       - Integer (e.g., 42)
• bool!      - True/false
• uuid!      - Unique identifier
• timestamp! - Date and time

Add ? for optional: string?, int?, etc.

### Common Directives
• @primary        - Mark as primary key
• @auto           - Auto-generate (for uuid, timestamps)
• @default(val)   - Set default value
• @min(n)         - Minimum length/value
• @max(n)         - Maximum length/value
• @unique         - Must be unique across records

### Common Functions (Always Namespaced!)
• String.slugify(text)  - Create URL slug
• String.length(text)   - Get length
• Time.now()            - Current time
• Array.length(arr)     - Array size

**Need more?** Run: conduit introspect stdlib
**Complete reference:** See LANGUAGE-SPEC.md

## Bootstrap (Your First Resource)

**Goal**: Create your first resource and unlock introspection.

The absolute minimum working resource:

**app/resources/todo.cdt**:
` + "```conduit" + `
resource Todo {
  id: uuid! @primary @auto      // Required: primary key
  title: string!                // Required: your content field
  done: bool! @default(false)   // Optional: add more as needed
}
` + "```" + `

**What's required?**
• id: uuid! @primary @auto - Every resource needs this
• At least one custom field (like title above)

**What's optional?**
• Additional fields (add as many as you want)
• Validations like @min/@max (add when needed)
• Relationships (add after understanding basics)

**Create the file**:
` + "```bash" + `
cat > app/resources/todo.cdt << 'EOF'
resource Todo {
  id: uuid! @primary @auto
  title: string!
  done: bool! @default(false)
}
EOF
` + "```" + `

**Build and verify**:
` + "```bash" + `
conduit build           # Should succeed! ✓
conduit introspect schema # Should show Todo ✓
` + "```" + `

**✓ Success! Now Use Discovery**

After your first build succeeds, you can use the discovery-first workflow below.
All introspection commands now work - use them for everything!

[rest of existing content...]
`
}
```

**Testing**: Generate CLAUDE.md, manual review for clarity

---

### Component 2: Stdlib Registry

**File**: `internal/compiler/stdlib/registry.go` (new)

**Implementation**:
```go
package stdlib

type FunctionDef struct {
    Name        string
    Signature   string
    Description string
}

// StdlibRegistry contains all stdlib functions organized by namespace
var StdlibRegistry = map[string][]FunctionDef{
    "String": {
        {
            Name:        "slugify",
            Signature:   "(text: string!) -> string!",
            Description: "Create URL-friendly slug from text",
        },
        {
            Name:        "length",
            Signature:   "(text: string!) -> int!",
            Description: "Get the length of a string",
        },
        {
            Name:        "contains",
            Signature:   "(text: string!, substring: string!) -> bool!",
            Description: "Check if string contains substring",
        },
        {
            Name:        "uppercase",
            Signature:   "(text: string!) -> string!",
            Description: "Convert string to uppercase",
        },
        {
            Name:        "lowercase",
            Signature:   "(text: string!) -> string!",
            Description: "Convert string to lowercase",
        },
    },
    "Time": {
        {
            Name:        "now",
            Signature:   "() -> timestamp!",
            Description: "Get current timestamp",
        },
        {
            Name:        "format",
            Signature:   "(time: timestamp!, format: string!) -> string!",
            Description: "Format timestamp as string",
        },
    },
    "Array": {
        {
            Name:        "length",
            Signature:   "(arr: array<T>!) -> int!",
            Description: "Get array length",
        },
        {
            Name:        "contains",
            Signature:   "(arr: array<T>!, item: T) -> bool!",
            Description: "Check if array contains item",
        },
    },
}

// GetNamespace returns all functions in a namespace
func GetNamespace(namespace string) ([]FunctionDef, bool) {
    fns, ok := StdlibRegistry[namespace]
    return fns, ok
}

// GetAllNamespaces returns all namespace names
func GetAllNamespaces() []string {
    namespaces := make([]string, 0, len(StdlibRegistry))
    for ns := range StdlibRegistry {
        namespaces = append(namespaces, ns)
    }
    return namespaces
}
```

**Testing**:
```go
// internal/compiler/stdlib/registry_test.go
func TestStdlibRegistry(t *testing.T) {
    // Ensure all namespaces have functions
    for ns, fns := range StdlibRegistry {
        if len(fns) == 0 {
            t.Errorf("namespace %s has no functions", ns)
        }
    }

    // Ensure all functions have required fields
    for ns, fns := range StdlibRegistry {
        for _, fn := range fns {
            if fn.Name == "" {
                t.Errorf("function in %s has no name", ns)
            }
            if fn.Signature == "" {
                t.Errorf("function %s.%s has no signature", ns, fn.Name)
            }
        }
    }
}
```

**CI Sync Check**: Add test that fails if stdlib implementation has functions not in registry

---

### Component 3: Scaffold Command

**File**: `internal/cli/commands/scaffold.go` (new)

**Implementation**:
```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/conduit-lang/conduit/internal/templates"
    "github.com/spf13/cobra"
)

func NewScaffoldCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "scaffold <template>",
        Short: "Generate a working resource from a template",
        Long: `Generate working Conduit resources from templates.

Available templates:
  todo     Simple CRUD resource (recommended for first project)
  blog     Blog with relationships (Post, User, Comment)
  api      REST API with authentication patterns

Examples:
  conduit scaffold todo    # Creates app/resources/todo.cdt
  conduit scaffold blog    # Creates multiple related resources

After scaffolding:
  conduit build
  conduit introspect schema
`,
        Args: cobra.ExactArgs(1),
        RunE: runScaffold,
    }

    return cmd
}

func runScaffold(cmd *cobra.Command, args []string) error {
    templateName := args[0]

    // Get template
    scaffold, err := templates.GetScaffold(templateName)
    if err != nil {
        return fmt.Errorf("unknown template: %s\n\nAvailable templates: todo, blog, api\nRun 'conduit scaffold --help' for more info", templateName)
    }

    // Check project initialized (conduit.yaml exists)
    if _, err := os.Stat("conduit.yaml"); os.IsNotExist(err) {
        return fmt.Errorf("not in a Conduit project directory (conduit.yaml not found)\n\nRun 'conduit new <project-name>' first")
    }

    // Check for file conflicts
    for _, file := range scaffold.Files {
        if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
            return fmt.Errorf("file already exists: %s\n\nScaffold cannot overwrite existing files", file.Path)
        }
    }

    // Create directories
    for _, file := range scaffold.Files {
        dir := filepath.Dir(file.Path)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    // Write files
    for _, file := range scaffold.Files {
        if err := os.WriteFile(file.Path, []byte(file.Content), 0644); err != nil {
            return fmt.Errorf("failed to write %s: %w", file.Path, err)
        }
        fmt.Printf("Created: %s\n", file.Path)
    }

    // Show next steps
    fmt.Println("\n✓ Scaffold generated successfully!")
    fmt.Println("\nNext steps:")
    fmt.Println("  conduit build")
    fmt.Println("  conduit introspect schema")

    return nil
}
```

**Testing**:
```go
func TestScaffoldCommand(t *testing.T) {
    // Create temp project
    tmpDir := t.TempDir()
    os.Chdir(tmpDir)

    // Create conduit.yaml
    os.WriteFile("conduit.yaml", []byte("project:\n  name: test"), 0644)

    // Run scaffold
    cmd := NewScaffoldCommand()
    cmd.SetArgs([]string{"todo"})
    err := cmd.Execute()
    require.NoError(t, err)

    // Verify file created
    _, err = os.Stat("app/resources/todo.cdt")
    require.NoError(t, err)

    // Verify buildable (requires conduit build in PATH)
    // This would be integration test, not unit test
}
```

---

## Risks & Mitigation

### Risk 1: Stdlib Registry Drift
**Description**: Manual stdlib registry gets out of sync with actual implementation

**Likelihood**: Medium
**Impact**: Medium (LLMs learn wrong functions)

**Mitigation**:
1. CI test that parses actual stdlib implementation
2. Fails if registry missing functions found in code
3. Automated PR when stdlib changes detected
4. Quarterly manual review

**Code**:
```go
// internal/compiler/stdlib/registry_sync_test.go
func TestRegistrySyncWithImplementation(t *testing.T) {
    // Parse actual stdlib implementation
    actualFunctions := parseStdlibImplementation()

    // Compare with registry
    for ns, fns := range actualFunctions {
        registryFns, ok := StdlibRegistry[ns]
        if !ok {
            t.Errorf("namespace %s exists in implementation but not registry", ns)
            continue
        }

        for _, fn := range fns {
            found := false
            for _, regFn := range registryFns {
                if regFn.Name == fn.Name {
                    found = true
                    break
                }
            }
            if !found {
                t.Errorf("function %s.%s exists in implementation but not registry", ns, fn.Name)
            }
        }
    }
}
```

---

### Risk 2: Example Maintenance Burden
**Description**: Examples break as language evolves

**Likelihood**: Medium
**Impact**: High (broken examples = bad first impression)

**Mitigation**:
1. CI builds all examples on every commit
2. Automated test suite for examples
3. Examples kept minimal (less to break)
4. Clear ownership (docs team)

**CI Job**:
```yaml
# .github/workflows/examples.yml
name: Validate Examples

on: [push, pull_request]

jobs:
  test-examples:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        example: [minimal, todo-app, blog, api-with-auth]

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Build Conduit
        run: go build -o conduit ./cmd/conduit

      - name: Build Example
        run: |
          cd examples/${{ matrix.example }}
          ../../conduit build

      - name: Test Introspection
        run: |
          cd examples/${{ matrix.example }}
          ../../conduit introspect schema

      - name: Fail if not buildable
        if: failure()
        run: exit 1
```

---

### Risk 3: Template Maintenance
**Description**: Scaffold templates become outdated

**Likelihood**: Low
**Impact**: Medium

**Mitigation**:
1. Keep templates minimal
2. Test scaffolds in CI (generate + build)
3. Version templates if breaking changes needed
4. Templates stored as code (easy to review in PRs)

---

### Risk 4: Documentation Drift
**Description**: CLAUDE.md diverges from actual behavior

**Likelihood**: Low
**Impact**: Medium

**Mitigation**:
1. CLAUDE.md generated from template (single source of truth)
2. Examples validate documentation claims
3. LLM testing validates effectiveness
4. Docs team reviews on language changes

---

## Testing Strategy

### Unit Tests
- Stdlib registry completeness
- Scaffold template validation
- Error message formatting
- Template generation logic

### Integration Tests
- `conduit scaffold` → build → introspect pipeline
- Error messages in actual failure scenarios
- Examples build successfully
- Commands work in fresh project

### Manual LLM Testing
- Fresh project walkthrough with Claude Code
- Measure time to first successful build
- Track error messages encountered
- Validate introspection usage

### CI Automation
- Build all examples on every commit
- Generate and build all scaffolds
- Stdlib registry sync check
- Documentation link validation

---

## Rollback Plan

### Documentation Changes (Phase 1)
**Rollback**: Git revert of template changes
**Risk**: Minimal (no code behavior change)
**Recovery Time**: <5 minutes

### CLI Commands (Phase 2)
**Rollback**: Feature flag to disable commands
```go
// config.yaml
features:
  scaffold_command: false
  stdlib_introspection: false
```
**Risk**: Low (commands are additive)
**Recovery Time**: <1 hour (deploy config change)

### Examples (All Phases)
**Rollback**: Remove examples directory
**Risk**: None (doesn't affect functionality)
**Recovery Time**: <5 minutes

---

## Success Metrics & Monitoring

### Development Metrics
- Time to implement each phase (track actual vs estimate)
- Test coverage for new code (target: >80%)
- CI build times (ensure not degraded)

### User Metrics (Post-Release)
- Time from `conduit new` to first successful build (target: <5 min)
- Error message display frequency (which errors hit most)
- Scaffold command usage (which templates most popular)
- LLM success rate (% completing first resource on first try)

### Monitoring Plan
```go
// Optional telemetry (opt-in only)
type BootstrapMetrics struct {
    ProjectCreatedAt     time.Time
    FirstBuildSuccess    time.Time
    FirstIntrospection   time.Time
    ScaffoldUsed         bool
    ScaffoldTemplate     string
    ErrorsEncountered    []string
}
```

---

## Dependencies

### External Dependencies
- None (all changes internal to Conduit)

### Internal Dependencies
- Phase 2 depends on Phase 1 (error messages reference scaffold)
- Examples CI depends on examples existence
- Stdlib introspection depends on registry creation

### Backward Compatibility
- ✅ All changes are additive
- ✅ Existing projects unaffected
- ✅ Existing workflows continue working
- ✅ No breaking changes to CLI or compiler

---

## Alternative Approaches Considered

### Alternative 1: Pre-generate Sample Resource on `conduit new`
**Approach**: Always create a sample resource when project initialized

**Pros**:
- Immediate working introspection
- No documentation needed

**Cons**:
- Violates "empty project" expectation
- User might not want sample resource
- Still need documentation for second resource
- Doesn't teach bootstrap process

**Decision**: ❌ Rejected - too opinionated, doesn't solve learning problem

---

### Alternative 2: Dynamic Stdlib Introspection from Runtime
**Approach**: Query actual stdlib implementation via reflection

**Pros**:
- Always in sync
- No manual maintenance

**Cons**:
- Complex implementation
- Requires runtime initialization
- Doesn't work pre-build (defeats purpose)
- Performance overhead

**Decision**: ❌ Rejected for MVP - static registry simpler and works pre-build

---

### Alternative 3: Interactive Tutorial Mode
**Approach**: `conduit tutorial` command that walks through bootstrap step-by-step

**Pros**:
- Highly guided experience
- Educational

**Cons**:
- High development effort
- LLMs prefer text documentation over interactive
- Maintenance burden
- Doesn't solve "quick reference" need

**Decision**: ❌ Rejected - docs + examples are more LLM-friendly

---

### Alternative 4: AI-Powered Error Suggestions
**Approach**: Use LLM to suggest fixes for build errors

**Pros**:
- Potentially very helpful
- Could handle novel errors

**Cons**:
- Requires external API (privacy/reliability concerns)
- High complexity
- Not needed if errors are already actionable
- Slow response time

**Decision**: ❌ Rejected - simpler to make errors actionable directly

---

## Final Recommendation

**Implement in 3 Phases**:

**Phase 1** (3-4 days): Documentation + Examples
- Immediate impact
- Zero risk
- Fast to implement
- 50% improvement expected

**Phase 2** (4-5 days): CLI Commands
- High impact
- Low risk
- Moderate effort
- Reaches 80% target

**Phase 3** (2-3 days): Polish
- Final refinements
- Optional features evaluated
- Based on real usage data

**Total Effort**: 9-12 engineering days
**Risk Level**: Low
**Expected Outcome**: >80% LLM success rate, <5 minute bootstrap time

---

**Document Owner**: Senior Software Engineer
**Last Updated**: 2025-10-31
**Status**: Ready for Implementation
**Next Step**: PM prioritization and eng assignment
