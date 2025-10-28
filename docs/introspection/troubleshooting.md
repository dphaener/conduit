# Troubleshooting Guide

This guide covers common issues you might encounter when using the Conduit introspection system and how to resolve them.

## Table of Contents

- [Registry Issues](#registry-issues)
- [CLI Issues](#cli-issues)
- [API Issues](#api-issues)
- [Performance Issues](#performance-issues)
- [Build Issues](#build-issues)
- [Pattern Discovery Issues](#pattern-discovery-issues)
- [Dependency Issues](#dependency-issues)

## Registry Issues

### Error: "Registry not initialized"

**Symptom:**
```
Error: registry not initialized - run 'conduit build' first to generate metadata
```

**Cause:** The application hasn't been compiled, or metadata wasn't generated during compilation.

**Solutions:**

1. **Build your application first:**
   ```bash
   conduit build
   ```

2. **Verify metadata file exists:**
   ```bash
   ls -la build/app.meta.json
   ```

3. **Check for build errors:**
   ```bash
   conduit build --verbose
   ```

4. **Ensure you're in the project directory:**
   ```bash
   pwd  # Should be in your Conduit project root
   ls   # Should see resources/ directory
   ```

### Metadata File Missing

**Symptom:** `build/app.meta.json` doesn't exist after building.

**Cause:** Compilation failed or metadata generation was skipped.

**Solutions:**

1. **Check for compilation errors:**
   ```bash
   conduit build 2>&1 | grep -i error
   ```

2. **Verify resources are defined:**
   ```bash
   ls resources/*.cdt
   ```

3. **Try a clean build:**
   ```bash
   rm -rf build/
   conduit build
   ```

### Stale Metadata

**Symptom:** Introspection shows old data after making changes.

**Cause:** Metadata wasn't regenerated after code changes.

**Solutions:**

1. **Rebuild the application:**
   ```bash
   conduit build
   ```

2. **Clear build cache:**
   ```bash
   rm -rf build/
   conduit build
   ```

3. **Use watch mode during development:**
   ```bash
   conduit watch
   ```

## CLI Issues

### Resource Not Found

**Symptom:**
```
Error: resource not found: post

Did you mean?
  - Post
  - Product
```

**Cause:** Resource names are case-sensitive or the resource doesn't exist.

**Solutions:**

1. **Check exact spelling and capitalization:**
   ```bash
   conduit introspect resources  # List all available resources
   conduit introspect resource Post  # Use correct name
   ```

2. **Verify resource is defined:**
   ```bash
   grep -r "resource Post" resources/
   ```

### Invalid Flags or Arguments

**Symptom:**
```
Error: depth must be between 1 and 5, got: 10
```

**Cause:** Invalid flag value or wrong number of arguments.

**Solutions:**

1. **Check command help:**
   ```bash
   conduit introspect deps --help
   ```

2. **Verify flag values:**
   ```bash
   # Valid depth range: 1-5
   conduit introspect deps Post --depth 2

   # Valid formats: json, table
   conduit introspect resources --format json

   # Valid types: resource, middleware, function
   conduit introspect deps Post --type resource
   ```

3. **Check argument count:**
   ```bash
   # ✓ Correct: deps requires resource name
   conduit introspect deps Post

   # ✗ Wrong: missing resource name
   conduit introspect deps
   ```

### Color Output Issues

**Symptom:** Terminal shows escape codes like `^[[32m` instead of colors.

**Cause:** Terminal doesn't support ANSI colors or color output is broken.

**Solutions:**

1. **Disable color output:**
   ```bash
   conduit introspect resources --no-color
   ```

2. **Check terminal support:**
   ```bash
   echo $TERM  # Should be something like "xterm-256color"
   ```

3. **Set environment variable:**
   ```bash
   export NO_COLOR=1
   conduit introspect resources
   ```

## API Issues

### Nil Pointer Errors

**Symptom:**
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**Cause:** Querying data before registry initialization or handling nil returns incorrectly.

**Solutions:**

1. **Check for nil before using:**
   ```go
   resource, err := registry.Resource("Post")
   if err != nil {
       log.Fatalf("Error: %v", err)
   }
   // Now safe to use resource
   ```

2. **Verify registry is initialized:**
   ```go
   registry := metadata.GetRegistry()
   schema := registry.GetSchema()
   if schema == nil {
       log.Fatal("Registry not initialized")
   }
   ```

3. **Check return values:**
   ```go
   resources := registry.Resources()
   if resources == nil {
       log.Fatal("No resources found")
   }
   ```

### Empty Results

**Symptom:** Queries return empty slices or maps when data should exist.

**Cause:** Incorrect filters, registry not initialized, or no matching data.

**Solutions:**

1. **Remove filters to see all data:**
   ```go
   // Get ALL routes first
   routes := registry.Routes(metadata.RouteFilter{})
   fmt.Printf("Total routes: %d\n", len(routes))

   // Then add filters
   routes = registry.Routes(metadata.RouteFilter{
       Method: "GET",
   })
   ```

2. **Check filter values:**
   ```go
   // Filter values are case-sensitive and exact match
   routes := registry.Routes(metadata.RouteFilter{
       Method: "GET",  // Not "get" or "Get"
   })
   ```

3. **Verify registry state:**
   ```go
   schema := registry.GetSchema()
   if schema == nil {
       log.Fatal("Registry not initialized")
   }
   fmt.Printf("Resources: %d\n", len(schema.Resources))
   ```

### Type Assertion Errors

**Symptom:**
```go
panic: interface conversion: interface {} is nil, not *metadata.DependencyGraph
```

**Cause:** Incorrect type assertion or cache contains unexpected type.

**Solutions:**

1. **Use type assertions with checks:**
   ```go
   if cached := registry.getCached(key); cached != nil {
       if graph, ok := cached.(*metadata.DependencyGraph); ok {
           return graph, nil
       }
   }
   ```

2. **Don't use internal cache directly:**
   ```go
   // ✗ Wrong: accessing internals
   cached := globalRegistry.getCached(key)

   // ✓ Correct: use public API
   graph, err := registry.Dependencies("Post", opts)
   ```

## Performance Issues

### Slow Query Performance

**Symptom:** Introspection queries take several seconds.

**Cause:** Large application, deep traversals, or inefficient queries.

**Solutions:**

1. **Limit traversal depth:**
   ```bash
   # Fast: only direct dependencies
   conduit introspect deps Post --depth 1

   # Slow: deep traversal
   conduit introspect deps Post --depth 5
   ```

2. **Use specific filters:**
   ```go
   // Faster: filter by resource
   routes := registry.Routes(metadata.RouteFilter{
       Resource: "Post",
   })

   // Slower: get all then filter in code
   routes := registry.Routes(metadata.RouteFilter{})
   ```

3. **Cache results in your application:**
   ```go
   var cachedResources []metadata.ResourceMetadata

   func GetResources() []metadata.ResourceMetadata {
       if cachedResources == nil {
           registry := metadata.GetRegistry()
           cachedResources = registry.Resources()
       }
       return cachedResources
   }
   ```

### Memory Usage

**Symptom:** High memory usage when working with introspection.

**Cause:** Loading entire schema, deep dependency graphs, or memory leaks.

**Solutions:**

1. **Query specific data instead of full schema:**
   ```go
   // ✓ Better: query specific resource
   post, err := registry.Resource("Post")

   // ✗ Worse: load everything
   schema := registry.GetSchema()
   post := findResource(schema.Resources, "Post")
   ```

2. **Limit dependency depth:**
   ```go
   // Use smaller depth values
   graph, err := registry.Dependencies("Post", metadata.DependencyOptions{
       Depth: 1,  // Not 0 (unlimited)
   })
   ```

3. **Clear unused data:**
   ```go
   // After processing, let GC reclaim memory
   resources := nil
   runtime.GC()
   ```

## Build Issues

### Metadata Generation Failures

**Symptom:** Build succeeds but `app.meta.json` is incomplete or malformed.

**Cause:** Errors during metadata extraction or serialization.

**Solutions:**

1. **Check build logs:**
   ```bash
   conduit build --verbose 2>&1 | tee build.log
   ```

2. **Validate metadata JSON:**
   ```bash
   cat build/app.meta.json | jq . > /dev/null
   ```

3. **Look for syntax errors in resources:**
   ```bash
   conduit check
   ```

### Compilation Errors

**Symptom:** `conduit build` fails with compiler errors.

**Cause:** Syntax errors, type errors, or invalid resource definitions.

**Solutions:**

1. **Check error messages carefully:**
   ```bash
   conduit build 2>&1 | grep "error:"
   ```

2. **Validate resource files:**
   ```bash
   conduit check resources/*.cdt
   ```

3. **Start with simple resources:**
   ```conduit
   resource Test {
     id: uuid! @primary @auto
     name: string!
   }
   ```

## Pattern Discovery Issues

### No Patterns Found

**Symptom:**
```
No patterns found.
```

**Cause:** Not enough pattern occurrences or min-frequency too high.

**Solutions:**

1. **Lower minimum frequency:**
   ```bash
   # Default: min-frequency=3
   conduit introspect patterns --min-frequency 1
   ```

2. **Check that resources use middleware:**
   ```bash
   conduit introspect resources --verbose
   ```

3. **Verify patterns exist in code:**
   ```bash
   grep -r "@on" resources/
   ```

### Low Confidence Patterns

**Symptom:** Patterns have confidence scores < 0.3.

**Cause:** Pattern appears infrequently (< 3 times).

**Solutions:**

1. **This is expected for emerging patterns:**
   - Confidence 0.1-0.3: Emerging (1-2 uses)
   - Confidence 0.3-0.5: Establishing (3-5 uses)
   - Confidence 0.5+: Standard (5+ uses)

2. **Review pattern to decide if it should be standard:**
   ```bash
   conduit introspect patterns --min-frequency 1
   ```

3. **Apply pattern more consistently to increase confidence:**
   ```conduit
   // Apply the pattern to more resources
   @on create: [auth_required]
   ```

### Incorrect Pattern Detection

**Symptom:** Patterns don't match actual code patterns.

**Cause:** Pattern extraction heuristics are imperfect.

**Solutions:**

1. **Review pattern examples:**
   ```bash
   conduit introspect patterns --format json | jq '.patterns[].examples'
   ```

2. **Report issues:** Pattern detection can be improved based on feedback.

3. **Use custom pattern validation:**
   ```go
   // Write custom logic to validate patterns
   patterns := registry.Patterns("")
   for _, p := range patterns {
       if !isValidPattern(p) {
           fmt.Printf("Suspicious pattern: %s\n", p.Name)
       }
   }
   ```

## Dependency Issues

### Circular Dependencies Detected

**Symptom:** Warning messages about circular dependencies.

**Cause:** Resources have circular relationships.

**Example:**
```
WARNING: Detected 1 circular dependencies in resource graph
  Cycle 1: Post -> User -> Profile -> Post
```

**Solutions:**

1. **Review the cycle:**
   ```bash
   conduit introspect deps Post --depth 3
   ```

2. **Break the cycle by making relationship optional:**
   ```conduit
   // Before: Post requires User (circular)
   author: User! { ... }

   // After: Make optional to break cycle
   author: User? { ... }
   ```

3. **Use through tables for many-to-many:**
   ```conduit
   // Instead of direct circular relationships
   resource PostCategory {
       post: Post!
       category: Category!
   }
   ```

4. **Accept the cycle if it's intentional:**
   - Some cycles are valid (e.g., User <-> Profile)
   - Just be aware when deleting records

### Dependency Depth Errors

**Symptom:**
```
Error: depth must be between 1 and 5, got: 10
```

**Cause:** Requested depth exceeds maximum (5).

**Solutions:**

1. **Use allowed depth range (1-5):**
   ```bash
   conduit introspect deps Post --depth 5  # Maximum
   ```

2. **Use depth 0 for unlimited (programmatically):**
   ```go
   graph, err := registry.Dependencies("Post", metadata.DependencyOptions{
       Depth: 0,  // Unlimited (use with caution)
   })
   ```

3. **Consider if you really need deep traversal:**
   - Depth 1: Direct dependencies (usually sufficient)
   - Depth 2: Dependencies of dependencies
   - Depth 3+: Rarely needed

## Getting Help

If you're still stuck after trying these solutions:

1. **Check the documentation:**
   - [User Guide](user-guide.md)
   - [API Reference](api-reference.md)
   - [Architecture Guide](architecture.md)

2. **Search existing issues:**
   - GitHub Issues: https://github.com/conduit-lang/conduit/issues
   - Look for similar problems

3. **Ask for help:**
   - Include error messages
   - Provide minimal reproduction
   - Share your Conduit version: `conduit --version`

4. **Report bugs:**
   - File an issue on GitHub
   - Include full error output
   - Describe expected vs actual behavior

## Common Error Messages Reference

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `registry not initialized` | Metadata not generated | Run `conduit build` |
| `resource not found: X` | Invalid resource name | Check spelling and case |
| `depth must be between 1 and 5` | Invalid depth value | Use 1-5 range |
| `invalid type filter: X` | Invalid type filter | Use: resource, middleware, function |
| `unsupported format: X` | Invalid output format | Use: json, table |
| `nil pointer dereference` | Using nil value | Check err before using value |
| `interface conversion` | Type assertion failed | Check types, use type switch |
| `no such file or directory` | File/directory missing | Verify path, run `conduit build` |

## Performance Benchmarks

Expected performance for typical applications:

| Operation | Small App (10 resources) | Medium App (50 resources) | Large App (200 resources) |
|-----------|-------------------------|---------------------------|---------------------------|
| List resources | < 1ms | < 1ms | < 5ms |
| Get resource | < 1ms | < 1ms | < 1ms |
| List routes | < 1ms | < 5ms | < 10ms |
| Dependencies (depth 1) | < 5ms | < 10ms | < 20ms |
| Dependencies (depth 3) | < 10ms | < 50ms | < 100ms |
| Pattern discovery | < 10ms | < 50ms | < 200ms |

If your performance is significantly worse, see [Performance Issues](#performance-issues).

## Debug Mode

Enable debug output for troubleshooting:

```bash
# Set debug environment variable
export CONDUIT_DEBUG=1
conduit introspect resources

# Or use verbose flag
conduit introspect resources --verbose
```

## Version Compatibility

Make sure your CLI and runtime versions match:

```bash
# Check CLI version
conduit --version

# Check generated metadata version
cat build/app.meta.json | jq .version
```

If versions don't match, rebuild:

```bash
conduit build --clean
```

---

[← Back to Overview](README.md)
