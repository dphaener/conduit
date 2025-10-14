# Linting Guide for Conduit

## Setup

Linting is configured using **golangci-lint** v1.64.8 with settings optimized for compiler development.

Configuration file: `.golangci.yml`

## Running the Linter

### Check all code
```bash
golangci-lint run
```

### Check specific package
```bash
golangci-lint run ./internal/compiler/lexer/...
```

### Check specific file
```bash
golangci-lint run internal/compiler/lexer/lexer.go
```

### Auto-fix issues (where possible)
```bash
golangci-lint run --fix
```

### Show all enabled linters
```bash
golangci-lint linters
```

## Enabled Linters

### Essential (always enabled)
- `errcheck` - Unchecked errors
- `gosimple` - Code simplification
- `govet` - Go vet checks
- `ineffassign` - Unused assignments
- `staticcheck` - Advanced static analysis
- `unused` - Unused code

### Additional linters
- `gofmt` / `goimports` - Code and import formatting
- `gocritic` - Performance and style issues
- `revive` - Enhanced golint replacement
- `misspell` - Spelling errors
- `errorlint` - Error wrapping best practices
- `gocyclo` / `cyclop` - Cyclomatic complexity
- `funlen` - Function length
- `lll` - Line length (140 chars)

### Bug detection
- `bodyclose` - HTTP response body closure
- `nilerr` - nil error issues
- `nilnil` - Returning nil error and nil value
- `rowserrcheck` - SQL Rows.Err checking
- `sqlclosecheck` - SQL resource closure

## Configuration Highlights

### Compiler-specific adjustments
- **Higher complexity limits** (25 instead of 15) for lexer/parser code
- **ALL_CAPS allowed** for token constants (`TOKEN_STRING`, not `TokenString`)
- **Field alignment disabled** - readability over micro-optimizations
- **if-else chains allowed** - often clearer than switches in lexers

### Exceptions
- Test files: Relaxed complexity and function length limits
- `token.go`: ALL_CAPS naming and group comments allowed
- Benchmark files: No function length limits

## Common Issues & Fixes

### 1. Unchecked errors (errcheck)
```go
// ❌ Bad
file.Close()

// ✅ Good
if err := file.Close(); err != nil {
    return err
}

// ✅ Also good (when intentionally ignoring)
_ = file.Close()
```

### 2. Unused variables (unused)
```go
// ❌ Bad
func parse() {
    unused := 42
    // ...
}

// ✅ Good - remove it
func parse() {
    // ...
}
```

### 3. Package comments (revive)
```go
// ✅ Add package-level comment
// Package lexer tokenizes Conduit source code into tokens.
package lexer
```

### 4. Empty string test (gocritic)
```go
// ❌ Less idiomatic
if len(s) == 0 { }

// ✅ More idiomatic
if s == "" { }
```

### 5. Unnecessary nil initialization (revive)
```go
// ❌ Redundant
var literal interface{} = nil

// ✅ Cleaner
var literal interface{}
```

## Integration with CI/CD

### GitHub Actions
```yaml
- name: Run linter
  run: golangci-lint run --timeout 5m
```

### Pre-commit hook
```bash
#!/bin/sh
# .git/hooks/pre-commit
golangci-lint run --new-from-rev=HEAD~1
```

### Make target
```makefile
.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix
```

## When to Ignore Linter Warnings

Use `//nolint` comments sparingly and only when:
1. The linter is provably wrong
2. The suggested fix would make code less clear
3. You're working with external constraints

```go
//nolint:errcheck // Intentionally ignoring write to stderr
fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
```

**Always add a reason** after `//nolint` explaining why.

## Performance Notes

Typical run time for current codebase:
- Lexer package: < 1 second
- Full project (when larger): < 10 seconds

Parallel execution is enabled by default for faster checks.

## Resources

- [golangci-lint documentation](https://golangci-lint.run/)
- [Enabled linters reference](https://golangci-lint.run/usage/linters/)
- Config file: `.golangci.yml`
