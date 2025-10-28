# CON-65: Add Tunable Parameters to PatternExtractor

## Summary

Successfully refactored `PatternExtractor` to accept tunable parameters that control pattern extraction quality and behavior. This enables the LLM validation iteration system (CON-65) to adjust parameters based on feedback and improve pattern quality over time.

## Changes Made

### 1. New `PatternExtractionParams` Struct

Added a configuration struct with five tunable parameters:

```go
type PatternExtractionParams struct {
    MinFrequency        int     // Minimum occurrences (default: 3)
    MinConfidence       float64 // Minimum confidence 0.0-1.0 (default: 0.3)
    MaxExamples         int     // Limit examples per pattern (default: 5)
    IncludeDescriptions bool    // Add detailed descriptions (default: true)
    VerboseNames        bool    // Use descriptive names (default: false)
}
```

**Purpose of each parameter:**

- **MinFrequency**: Controls how many times a pattern must appear before extraction. Lower values = more patterns, higher values = only common patterns.
- **MinConfidence**: Filters patterns by confidence score (frequency/10.0 capped at 1.0). Useful for ensuring pattern reliability.
- **MaxExamples**: Limits the number of examples per pattern to reduce output size while maintaining frequency tracking.
- **IncludeDescriptions**: Toggles human-readable descriptions for space savings when not needed.
- **VerboseNames**: Generates detailed names including operation context and parameters (e.g., "cached_handler_with_300_for_list" vs "cached_handler").

### 2. Updated PatternExtractor API

**Backward compatible changes:**

```go
// Existing constructor (unchanged behavior)
func NewPatternExtractor() *PatternExtractor

// New constructor with custom params
func NewPatternExtractorWithParams(params PatternExtractionParams) *PatternExtractor

// Helper for getting defaults
func DefaultParams() PatternExtractionParams
```

All existing code continues to work without changes - `NewPatternExtractor()` uses `DefaultParams()` internally.

### 3. Implementation Updates

**ExtractMiddlewarePatterns**:
- Now filters by both `MinFrequency` and `MinConfidence`
- Passes params through to name generation and pattern creation

**generateMiddlewarePattern**:
- Limits examples to `MaxExamples` (defaults to all if not set)
- Conditionally includes descriptions based on `IncludeDescriptions`
- Passes `usages` to name generator for verbose naming

**generatePatternName**:
- Enhanced to support `VerboseNames` mode
- Extracts parameters from middleware (e.g., "300" from "cache(300)")
- Adds operation context (most common operation) when verbose
- Format: `<middleware>_handler_with_<params>_for_<operation>`

### 4. Comprehensive Test Coverage

Added `TestPatternExtractorWithCustomParams` with 7 subtests:

1. **MinFrequency=2** - Verifies lower threshold includes more patterns
2. **MinConfidence filtering** - Ensures low-confidence patterns are excluded
3. **MaxExamples limiting** - Confirms example count cap works correctly
4. **IncludeDescriptions=false** - Validates descriptions can be omitted
5. **VerboseNames** - Tests detailed naming with operation context
6. **VerboseNames with parameters** - Verifies parameter extraction and formatting
7. **Combined custom params** - Integration test with all params configured

All existing tests updated to work with new API (28 existing tests pass unchanged).

### 5. Documentation Examples

Created `pattern_extractor_example_test.go` with 4 documented examples:

- **ExamplePatternExtractor_defaultParams**: Basic usage with defaults
- **ExamplePatternExtractor_customParams**: Aggressive extraction with custom params
- **ExamplePatternExtractor_verboseNames**: Detailed naming demonstration
- **ExamplePatternExtractor_highQualityPatterns**: High-confidence pattern filtering

## Test Results

```
PASS: TestNewPatternExtractor
PASS: TestPatternExtractorWithCustomParams (all 7 subtests)
PASS: TestDefaultParams
PASS: All 28 existing pattern extractor tests
PASS: All 4 example tests
```

**Total: 100% test pass rate (0 failures)**

## Usage Examples

### Default Parameters (Current Behavior)
```go
pe := metadata.NewPatternExtractor()
patterns := pe.ExtractMiddlewarePatterns(resources)
```

### Custom Parameters for Iteration System
```go
params := metadata.PatternExtractionParams{
    MinFrequency:        2,     // More permissive
    MinConfidence:       0.4,   // Higher quality bar
    MaxExamples:         3,     // Compact output
    IncludeDescriptions: true,  // Keep descriptions
    VerboseNames:        false, // Concise names
}
pe := metadata.NewPatternExtractorWithParams(params)
patterns := pe.ExtractMiddlewarePatterns(resources)
```

### Verbose Naming for LLM Learning
```go
params := metadata.DefaultParams()
params.VerboseNames = true
pe := metadata.NewPatternExtractorWithParams(params)
// Produces: "cached_handler_with_300_for_list"
// Instead of: "cached_handler"
```

## Integration with CON-65

The iteration system can now:

1. **Start with defaults** for initial pattern extraction
2. **Tune MinFrequency** to control pattern volume
3. **Tune MinConfidence** to filter unreliable patterns
4. **Tune MaxExamples** to balance between detail and performance
5. **Toggle VerboseNames** when the LLM needs more context
6. **Toggle IncludeDescriptions** to reduce token usage when not needed

Example iteration loop:
```go
// Iteration 1: Conservative defaults
params := metadata.DefaultParams()

// Iteration 2: LLM requests more patterns
params.MinFrequency = 2

// Iteration 3: Too many low-quality patterns
params.MinConfidence = 0.5

// Iteration 4: Token limit hit
params.MaxExamples = 2
params.IncludeDescriptions = false
```

## Design Decisions

1. **Immutable params after construction**: Prevents accidental state mutation
2. **Struct-based config**: Easy to extend with new parameters
3. **Backward compatible**: No breaking changes to existing code
4. **Sensible defaults**: Current behavior preserved for existing users
5. **Comprehensive validation**: Tests ensure params work in combination
6. **Verbose name format**: Places params at end for readability

## Files Modified

- `/Users/darinhaener/code/conduit/runtime/metadata/pattern_extractor.go` (93 lines added/changed)
- `/Users/darinhaener/code/conduit/runtime/metadata/pattern_extractor_test.go` (259 lines added)

## Files Created

- `/Users/darinhaener/code/conduit/runtime/metadata/pattern_extractor_example_test.go` (149 lines)

## Performance Impact

No performance regression:
- Parameter checks are O(1)
- Example limiting reduces memory when configured
- Confidence filtering is simple float comparison
- All existing benchmarks pass unchanged

## Next Steps for CON-65

1. Integrate `PatternExtractionParams` into LLM validation harness
2. Implement parameter tuning logic based on LLM feedback
3. Track parameter effectiveness across iterations
4. Add parameter recommendations to iteration results
