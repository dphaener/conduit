# Pattern Validation Guide

## Overview

The LLM Pattern Validation System is a comprehensive framework for validating that extracted code patterns from the Conduit codebase are learnable and usable by AI assistants (LLMs). This system enables:

- **Automated pattern extraction** from resources and middleware declarations
- **Multi-provider LLM testing** to validate pattern quality
- **Intelligent failure analysis** to categorize why patterns fail
- **Iterative improvement** through parameter tuning
- **Comprehensive reporting** with actionable insights

## Quick Start

### Basic Usage

Run pattern validation with mock LLM (no API keys required):

```bash
conduit test-patterns --mock
```

Run with real LLM providers:

```bash
# Set API keys
export ANTHROPIC_API_KEY=your_key_here
export OPENAI_API_KEY=your_key_here

# Run validation
conduit test-patterns
```

### CLI Options

```bash
conduit test-patterns [flags]

Flags:
  --mock                    Use mock LLM for testing (no API keys required)
  --max-iterations int      Maximum number of iterations (default: 5)
  --format string           Output format: text, json (default: text)
  --output string           Write report to file instead of stdout
  --verbose                 Enable verbose output
  --providers strings       Comma-separated list of providers (default: all enabled)
```

## How It Works

The pattern validation system operates in six phases across multiple iterations:

### Phase 1: Pattern Extraction

The system analyzes your Conduit codebase to identify common patterns:

```
Resources → Pattern Extractor → Pattern Metadata
```

**What Gets Extracted:**
- Middleware chains (e.g., `@on create: [auth]`)
- Frequency of occurrence across resources
- Example usage with file paths and line numbers
- Confidence scores based on usage patterns

**Example Pattern:**
```json
{
  "name": "authenticated_handler",
  "category": "authentication",
  "template": "@on <operation>: [auth]",
  "examples": [
    {
      "resource": "Post",
      "code": "@on create: [auth]",
      "file_path": "resources/post.cdt"
    }
  ],
  "frequency": 5,
  "confidence": 0.5
}
```

### Phase 2: Test Case Generation

For each pattern, the system generates targeted test cases:

```
Pattern → Test Case Generator → Test Cases
```

**Test Case Structure:**
```
Prompt: "Generate a middleware declaration for {pattern.category}
         on the {operation} operation for the {resource} resource.
         Use this pattern: {pattern.template}"

Expected: "@on create: [auth]"
Validation: semantic (understands intent, not just exact match)
```

### Phase 3: LLM Validation

Test cases are executed against configured LLM providers:

```
Test Cases → LLM Providers → Responses → Validator → Results
```

**Supported Providers:**
- Claude (Anthropic): claude-opus, claude-sonnet
- GPT (OpenAI): gpt-4, gpt-3.5-turbo

**Validation Modes:**
- `exact`: Response must match expected pattern exactly
- `semantic`: Response must match the intent (normalized comparison)

### Phase 4: Failure Analysis

Failed test cases are analyzed to determine root causes:

```
Test Results + Patterns → Failure Analyzer → Categorized Failures
```

**Six Failure Categories:**

1. **pattern_too_specific** - Pattern confidence too low (< 0.5)
   - Occurs when pattern appears infrequently
   - Recommendation: Lower MinFrequency threshold

2. **pattern_too_generic** - Pattern is too vague
   - Occurs when pattern lacks specificity
   - Recommendation: Raise MinConfidence threshold

3. **name_unclear** - LLM uses different naming
   - Example: LLM outputs `[login]` instead of `[auth]`
   - Recommendation: Enable VerboseNames for clarity

4. **template_ambiguous** - Syntax unclear
   - Example: LLM outputs `[auth(required)]` instead of `[auth]`
   - Recommendation: Improve template documentation

5. **insufficient_examples** - Too few examples (< 3)
   - Pattern lacks diverse usage examples
   - Recommendation: Require more examples per pattern

6. **llm_hallucination** - Completely wrong output
   - LLM generates unrelated code
   - Recommendation: Review prompt clarity

### Phase 5: Success Criteria Evaluation

Each iteration is evaluated against success criteria:

```
Results → Success Checker → Met/Not Met + Recommendations
```

**Default Success Criteria:**

| Provider | Target Success Rate |
|----------|-------------------|
| Claude Opus | 80% |
| GPT-4 | 70% |
| Minimum Pattern Success | 50% |

**Evaluation:**
- ✅ **Pass**: All providers meet targets AND no pattern below minimum
- ❌ **Fail**: Any provider below target OR any pattern below minimum

### Phase 6: Parameter Tuning

Based on failure analysis, extraction parameters are adjusted:

```
Recommendations → Parameter Adjuster → Updated Params → Next Iteration
```

**Tunable Parameters:**

| Parameter | Default | Purpose | When to Adjust |
|-----------|---------|---------|----------------|
| `MinFrequency` | 3 | Min occurrences | Lower if patterns too specific |
| `MinConfidence` | 0.3 | Quality threshold | Raise if patterns too generic |
| `MaxExamples` | 5 | Example limit | Increase for complex patterns |
| `IncludeDescriptions` | true | Add descriptions | Disable for concise output |
| `VerboseNames` | false | Detailed names | Enable if names unclear |

## Success Criteria

### Per-Provider Targets

Success criteria define the minimum acceptable pattern adherence for each LLM provider:

**Claude Opus (80% target):**
- Premium model with strong pattern recognition
- Should reliably follow established patterns
- Validates production-ready pattern quality

**GPT-4 (70% target):**
- High-capability model with good understanding
- Slightly lower target accounts for different training
- Ensures cross-provider compatibility

**Minimum Pattern Success (50% threshold):**
- No individual pattern should fail more than half the time
- Patterns below 50% should be removed or improved
- Ensures pattern library quality

### Why These Thresholds?

1. **80% for Claude Opus**: Premium models should exhibit strong pattern adherence for production use
2. **70% for GPT-4**: Accounts for model differences while maintaining quality bar
3. **50% minimum**: Prevents inclusion of unreliable or confusing patterns
4. **Graduated targets**: Recognizes different model capabilities without lowering quality

### When to Adjust

Consider adjusting criteria if:
- **New pattern categories** introduced (may need learning period)
- **Complex domain patterns** that are inherently harder (lower temporarily)
- **Production deployment** (raise targets for mission-critical usage)
- **Different LLM versions** with improved capabilities (raise targets)

## Iteration Strategy

The system runs multiple iterations to converge on optimal patterns:

### Iteration 1: Baseline

**Goal:** Establish initial performance metrics

```
Default Parameters:
  MinFrequency: 3
  MinConfidence: 0.3
  MaxExamples: 5
  VerboseNames: false
```

**Expected Outcomes:**
- Extract 5-15 patterns from typical codebase
- Identify 2-3 dominant failure categories
- Baseline success rate: 40-60%

### Iteration 2: Address Top Failure

**Goal:** Fix most common failure category

**Common Scenarios:**

**If name_unclear > 30% of failures:**
```
Adjustment: Enable VerboseNames
Rationale: Make pattern names more descriptive
Expected Improvement: +15-20% success rate
```

**If pattern_too_specific > 30%:**
```
Adjustment: Lower MinFrequency to 2
Rationale: Include less common but valid patterns
Expected Improvement: +10-15% success rate
```

**If pattern_too_generic > 30%:**
```
Adjustment: Raise MinConfidence to 0.4
Rationale: Filter out vague patterns
Expected Improvement: +10-15% success rate
```

### Iteration 3: Secondary Adjustments

**Goal:** Fine-tune parameters based on remaining failures

**Typical Adjustments:**
- Adjust MaxExamples if examples insufficient
- Enable/disable descriptions based on prompt length
- Further tune confidence thresholds

### Iteration 4-5: Convergence

**Goal:** Reach success criteria or identify problematic patterns

**Decision Points:**
- **Success achieved**: Stop iteration, patterns validated
- **Plateau reached**: Review specific failing patterns manually
- **Failure persists**: Consider pattern removal or prompt redesign

## Failure Analysis Deep Dive

### Pattern Too Specific

**Symptoms:**
- Low frequency (appears 2-3 times only)
- Low confidence score (< 0.5)
- LLM often fails to reproduce

**Root Cause:**
Pattern is an edge case or one-off implementation, not a true convention.

**Example:**
```conduit
// Only appears in one resource
@on bulk_import: [auth, rate_limit(1/day), validate_csv]
```

**Resolution:**
1. Lower MinFrequency to include it (if actually useful)
2. Or accept it as too specialized and exclude it

### Pattern Too Generic

**Symptoms:**
- Vague middleware names
- Overly broad category
- LLM outputs vary wildly

**Root Cause:**
Pattern captures incidental similarity, not semantic pattern.

**Example:**
```conduit
// Too generic: any middleware on create
@on create: [middleware]
```

**Resolution:**
1. Raise MinConfidence to filter out
2. Improve pattern naming to be more specific

### Name Unclear

**Symptoms:**
- LLM uses synonyms (login vs auth, memoize vs cache)
- Correct concept, wrong identifier
- High failure rate despite understanding

**Root Cause:**
Pattern name doesn't match LLM's learned vocabulary.

**Example:**
```conduit
// Pattern name: "authenticated_handler"
// LLM outputs: @on create: [login]  ❌
// Expected: @on create: [auth]  ✓
```

**Resolution:**
1. Enable VerboseNames for clarity
2. Add descriptions explaining the exact middleware names
3. Consider renaming middleware to match common terms

### Template Ambiguous

**Symptoms:**
- LLM adds extra parameters
- LLM uses slightly different syntax
- Functionally correct but doesn't validate

**Root Cause:**
Template doesn't specify parameter constraints clearly.

**Example:**
```conduit
// Template: "@on <operation>: [cache(300)]"
// LLM outputs: @on list: [cache(ttl=300)]  ❌
// Expected: @on list: [cache(300)]  ✓
```

**Resolution:**
1. Make templates more explicit
2. Add parameter format examples
3. Include parameter constraints in description

### Insufficient Examples

**Symptoms:**
- Pattern has 1-2 examples only
- High variability in LLM responses
- Unclear usage context

**Root Cause:**
Not enough diverse examples for LLM to learn pattern.

**Example:**
```json
{
  "name": "admin_only_handler",
  "examples": [
    {"resource": "AdminPanel", "code": "@on *: [admin_auth]"}
  ]
}
```

**Resolution:**
1. Require minimum 3 examples
2. Find more usage or exclude pattern
3. Document specific use cases better

### LLM Hallucination

**Symptoms:**
- LLM generates completely unrelated code
- Wrong language/syntax
- Ignores prompt instructions

**Root Cause:**
- Prompt unclear or confusing
- Pattern template malformed
- LLM context confusion

**Example:**
```conduit
// Expected: @on create: [auth]
// LLM outputs: function requireAuth() { ... }  ❌
```

**Resolution:**
1. Review and clarify prompts
2. Add explicit format instructions
3. Provide more context in examples
4. Consider if pattern is learnable by LLMs

## Reporting Formats

### Text Format (Console)

Human-readable output with visual progress indicators:

```
═══════════════════════════════════════════════════════════
  PATTERN VALIDATION ITERATION SYSTEM
═══════════════════════════════════════════════════════════

PROGRESS OVERVIEW

claude-opus (target: 80%)
  Iter 1: ██████████░░░░░░░░░░░░░░░░░░░░ 65.0%
  Iter 2: ████████████████░░░░░░░░░░░░░░ 75.0%
  Iter 3: ██████████████████████░░░░░░░░ 85.0% ✓

Overall Improvement: +20.0%

━━━ ITERATION 3: Tuned Parameters ━━━━━━━━━━━━━━━━━━━━━━━━

Extracted 8 patterns from resources
Generated 16 test cases (8 patterns × 2 providers)

Results:
  claude-opus:  14/16  (87.5%)  ██████████████░░  ✓ Target: 80%
  gpt-4:        12/16  (75.0%)  ████████████░░░░  ✓ Target: 70%

✅ SUCCESS CRITERIA MET!

Final Report:
  - Total iterations: 3
  - Patterns validated: 8
  - Overall improvement: +20.0%
  - All patterns above 50% threshold

Provider Success Rates:
  ✓ claude-opus:  87.5% (target: 80%)
  ✓ gpt-4:        75.0% (target: 70%)

═══════════════════════════════════════════════════════════
```

### JSON Format (Machine-Readable)

Structured data for CI/CD integration and analysis:

```json
{
  "timestamp": "2025-10-27T14:30:00Z",
  "total_iterations": 3,
  "final_success": true,
  "improvement": 0.20,
  "success_criteria": {
    "claude-opus": 0.80,
    "gpt-4": 0.70
  },
  "iterations": [
    {
      "iteration_number": 1,
      "patterns": [
        {
          "name": "authenticated_handler",
          "frequency": 5,
          "confidence": 0.5
        }
      ],
      "report": {
        "summary": {
          "total_tests": 16,
          "passed_tests": 10,
          "failed_tests": 6,
          "success_rate": 0.625,
          "by_provider": {
            "claude-opus": {
              "total_tests": 8,
              "passed_tests": 5,
              "success_rate": 0.625
            },
            "gpt-4": {
              "total_tests": 8,
              "passed_tests": 5,
              "success_rate": 0.625
            }
          }
        }
      },
      "failure_analysis": {
        "total_failures": 6,
        "by_reason": {
          "name_unclear": 4,
          "template_ambiguous": 2
        },
        "recommendations": [
          "Improve pattern naming to be more descriptive (4 cases)",
          "Clarify pattern templates with more explicit syntax (2 cases)"
        ]
      },
      "met_criteria": false
    }
  ]
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Validate Patterns

on:
  pull_request:
    paths:
      - 'resources/**'
      - 'internal/**'
  schedule:
    - cron: '0 0 * * 0'  # Weekly validation

jobs:
  validate:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Build Conduit
        run: go build -o conduit ./cmd/conduit

      - name: Run Pattern Validation
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          ./conduit test-patterns \
            --format json \
            --output pattern-report.json \
            --max-iterations 5

      - name: Check Success
        run: |
          success=$(jq '.final_success' pattern-report.json)
          if [ "$success" != "true" ]; then
            echo "Pattern validation failed"
            jq '.iterations[-1].failure_analysis.recommendations[]' pattern-report.json
            exit 1
          fi
          echo "Pattern validation passed!"

      - name: Upload Report
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: pattern-validation-report
          path: pattern-report.json

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const report = JSON.parse(fs.readFileSync('pattern-report.json', 'utf8'));

            const success = report.final_success ? '✅' : '❌';
            const improvement = (report.improvement * 100).toFixed(1);

            const comment = `## Pattern Validation ${success}

            - **Status**: ${report.final_success ? 'Passed' : 'Failed'}
            - **Iterations**: ${report.total_iterations}
            - **Improvement**: ${improvement}%

            ${!report.final_success ? '### Recommendations\n' +
              report.iterations[report.iterations.length - 1]
                .failure_analysis.recommendations
                .map(r => `- ${r}`).join('\n') : ''}
            `;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

### GitLab CI Example

```yaml
pattern-validation:
  stage: test
  image: golang:1.23

  script:
    - go build -o conduit ./cmd/conduit
    - |
      ./conduit test-patterns \
        --format json \
        --output pattern-report.json \
        --max-iterations 5
    - |
      if [ "$(jq '.final_success' pattern-report.json)" != "true" ]; then
        echo "Pattern validation failed"
        exit 1
      fi

  artifacts:
    reports:
      junit: pattern-report.json
    paths:
      - pattern-report.json
    expire_in: 30 days

  only:
    - merge_requests
    - main
```

## Troubleshooting

### Common Issues

#### Issue: No patterns extracted

**Symptoms:**
```
Error: no patterns extracted from resources
```

**Causes:**
- No middleware declarations in codebase
- MinFrequency threshold too high
- Resources not loaded correctly

**Resolution:**
```bash
# Check resource files exist
ls -la resources/

# Lower frequency threshold
# Edit internal/testing/llm config to set MinFrequency: 1

# Verify resource metadata generation
conduit introspect resources
```

#### Issue: All tests failing

**Symptoms:**
```
Success rate: 0.0%
All providers: 0/N passed
```

**Causes:**
- LLM API keys invalid or missing
- Network connectivity issues
- Malformed prompts or templates

**Resolution:**
```bash
# Verify API keys
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY

# Test with mock LLM first
conduit test-patterns --mock --verbose

# Check network access
curl https://api.anthropic.com
curl https://api.openai.com
```

#### Issue: Validation stuck on one iteration

**Symptoms:**
```
Iteration 1/5 complete... (no progress)
```

**Causes:**
- Rate limiting from LLM providers
- Timeout too short
- Deadlock in concurrent processing

**Resolution:**
```bash
# Increase timeout and reduce concurrency
# Edit config to set:
#   Timeout: 60 * time.Second
#   MaxConcurrentRequests: 1

# Check provider rate limits
# Anthropic: 50 req/min (Opus), 100 req/min (Sonnet)
# OpenAI: 60 req/min (GPT-4), 200 req/min (GPT-3.5)
```

#### Issue: Inconsistent results between runs

**Symptoms:**
```
Run 1: 75% success
Run 2: 45% success
Run 3: 82% success
```

**Causes:**
- LLM response variability (temperature setting)
- Non-deterministic pattern extraction order
- Race conditions in concurrent processing

**Resolution:**
```bash
# Run with mock LLM to verify determinism
conduit test-patterns --mock

# If still inconsistent, check:
# 1. Pattern extraction uses stable sorting
# 2. Test case generation is deterministic
# 3. Validation logic handles edge cases consistently
```

#### Issue: High cost from API calls

**Symptoms:**
```
Warning: 500+ API calls made
Estimated cost: $X.XX
```

**Causes:**
- Too many patterns extracted
- Too many providers configured
- MaxIterations set too high

**Resolution:**
```bash
# Reduce test load:
conduit test-patterns \
  --providers claude-opus \
  --max-iterations 3

# Or use mock mode for development:
conduit test-patterns --mock

# Raise MinConfidence to extract fewer patterns
# Edit config: MinConfidence: 0.5
```

## Best Practices

### 1. Start with Mock Testing

Always validate your test harness with mock LLM before using real APIs:

```bash
# Develop and debug with mock
conduit test-patterns --mock --verbose

# Switch to real LLMs when ready
conduit test-patterns
```

### 2. Run Regularly in CI/CD

Patterns degrade as codebases evolve. Regular validation catches issues early:

```yaml
# Weekly validation
schedule:
  - cron: '0 0 * * 0'
```

### 3. Version Your Patterns

Track pattern changes over time:

```bash
# Export patterns to version control
conduit introspect patterns > patterns-v1.0.0.json
git add patterns-v1.0.0.json
git commit -m "docs: snapshot pattern library v1.0.0"
```

### 4. Document Pattern Intent

Include rich descriptions explaining when and why to use each pattern:

```json
{
  "name": "authenticated_handler",
  "description": "Use for operations that require user authentication. Validates JWT token and loads user context before handler execution.",
  "when_to_use": "Any operation that accesses user-specific data or performs privileged actions",
  "see_also": ["rate_limited_authenticated_handler", "admin_authenticated_handler"]
}
```

### 5. Monitor Success Rates Over Time

Track pattern validation metrics to identify trends:

```bash
# Store historical results
conduit test-patterns --format json >> pattern-history.jsonl

# Analyze trends
jq -s 'map(.final_success)' pattern-history.jsonl
```

### 6. Involve LLMs in Pattern Design

Use LLM feedback to improve pattern clarity:

```bash
# Run validation
conduit test-patterns > report.txt

# Review failure categories
grep "Failure analysis" report.txt

# Adjust patterns based on feedback
# Example: If name_unclear is high, rename middleware
```

### 7. Set Realistic Targets

Adjust success criteria based on your domain:

```yaml
# Complex domain (database internals, algorithms)
target_success_rates:
  claude-opus: 0.70  # Lower target
  gpt-4: 0.60

# Standard CRUD API
target_success_rates:
  claude-opus: 0.85  # Higher target
  gpt-4: 0.75
```

## Advanced Usage

### Custom Pattern Extractors

Implement domain-specific pattern extraction:

```go
// Custom extractor for validation patterns
type ValidationPatternExtractor struct {
    params metadata.PatternExtractionParams
}

func (vpe *ValidationPatternExtractor) ExtractValidationPatterns(
    resources []metadata.ResourceMetadata,
) []metadata.PatternMetadata {
    // Extract @validate constraints
    // Group by constraint type
    // Generate pattern metadata
    return patterns
}
```

### Provider-Specific Tuning

Adjust parameters per provider:

```go
config := IterationConfig{
    TargetSuccessRate: map[string]float64{
        "claude-opus": 0.85,  // Higher target for premium model
        "gpt-4": 0.75,
        "claude-haiku": 0.60, // Lower target for fast model
    },
}
```

### Custom Validation Logic

Implement domain-specific validation:

```go
validator := &CustomValidator{
    allowSynonyms: map[string][]string{
        "auth": {"login", "authenticate", "require_auth"},
        "cache": {"memoize", "store"},
    },
}

harness.SetValidator(validator)
```

## References

- [Pattern Quality Checklist](./PATTERN-QUALITY-CHECKLIST.md)
- [Architecture Overview](../ARCHITECTURE.md)
- [Implementation Guide: Tooling](../IMPLEMENTATION-TOOLING.md)

## Support

For issues or questions:
1. Check [Troubleshooting](#troubleshooting) section
2. Review [Common Issues](#common-issues)
3. Search existing GitHub issues
4. Open a new issue with validation report attached
