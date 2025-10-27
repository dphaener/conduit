# LLM Validation Test Harness

This package provides an automated test harness for validating that LLMs can correctly use extracted patterns from the Conduit introspection system to generate code. This is critical for validating the introspection system's primary mission: enabling LLMs to learn and apply codebase patterns.

## Overview

The test harness:

1. Takes pattern metadata extracted from Conduit resources
2. Generates prompts for LLMs asking them to apply these patterns
3. Provides patterns via a simulated introspection API
4. Calls real LLM APIs (Claude, OpenAI) with the prompts
5. Parses the generated code from LLM responses
6. Validates pattern adherence (exact or semantic matching)
7. Generates comprehensive reports with success rates per LLM and per category

## Success Targets

- **Claude Opus**: 80%+ exact match rate
- **GPT-4**: 70%+ exact match rate
- **GPT-3.5**: 60%+ exact match rate

## Architecture

### Core Components

#### 1. `config.go` - Configuration

Manages LLM provider configuration:

```go
config := llm.NewDefaultConfig()
// Loads from environment variables:
// - ANTHROPIC_API_KEY for Claude
// - OPENAI_API_KEY for OpenAI
```

#### 2. `client.go` - LLM API Client

Unified interface for calling different LLM providers:

```go
client, err := llm.NewClient(providerConfig)
response, err := client.Generate(ctx, prompt)
```

Supports:
- Anthropic Claude API (messages endpoint)
- OpenAI API (chat completions)
- Automatic retries with exponential backoff
- Proper timeout handling

#### 3. `test_case.go` - Test Case Definition

Defines and generates test cases:

```go
generator := llm.NewTestCaseGenerator("Comment", "create")
testCase := generator.GenerateFromPattern(pattern)
```

Test cases include:
- Descriptive name
- Category (authentication, caching, etc.)
- Prompt for the LLM
- Available patterns via introspection
- Expected output pattern
- Validation mode (exact or semantic)

#### 4. `introspection_mock.go` - Simulated Introspection API

Provides patterns to test cases, simulating the runtime introspection API:

```go
mock := llm.NewDefaultIntrospectionMock()
authPatterns := mock.Patterns("authentication")
```

#### 5. `parser.go` - Code Parser

Extracts code from LLM responses:

```go
parser := llm.NewParser()
parsed, err := parser.Parse(response)
// Extracts:
// - Code blocks (```conduit ... ```)
// - Middleware declarations (@on create: [auth])
```

Handles various response formats:
- Plain text responses
- Markdown-formatted responses with code blocks
- Responses with explanatory text

#### 6. `validator.go` - Pattern Validator

Validates generated code against expected patterns:

```go
validator := llm.NewValidator()
result := validator.Validate(response, expected, "semantic")
// Returns:
// - Passed (bool)
// - Confidence (0.0-1.0)
// - Differences (if any)
```

Validation modes:
- **Exact**: Exact string match (normalized whitespace)
- **Semantic**: Same semantic meaning (allows formatting variations)

#### 7. `reporter.go` - Results Reporter

Generates comprehensive reports:

```go
reporter := llm.NewReporter()
report := reporter.GenerateReport(results, executionTime)
reporter.PrintReport(report)
```

Reports include:
- Overall success rate
- Per-provider statistics
- Per-category statistics
- Detailed failure information
- JSON export for CI integration

#### 8. `harness.go` - Main Orchestrator

Coordinates all components:

```go
harness, err := llm.NewHarness(config)
report, err := harness.Run(ctx, testCases)
```

Features:
- Parallel execution with controlled concurrency
- Rate limiting to respect API limits
- Progress reporting
- Graceful error handling

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/conduit-lang/conduit/internal/testing/llm"
)

func main() {
    // Create configuration (loads API keys from env)
    config := llm.NewDefaultConfig()

    // Create harness
    harness, err := llm.NewHarness(config)
    if err != nil {
        panic(err)
    }

    // Use predefined test cases
    testCases := llm.PredefinedTestCases()

    // Run tests
    ctx := context.Background()
    report, err := harness.Run(ctx, testCases)
    if err != nil {
        panic(err)
    }

    // Print report
    reporter := llm.NewReporter()
    reporter.PrintReport(report)
}
```

### Testing with Mock LLM

For unit tests, use the mock LLM functionality:

```go
mockFn := func(prompt string) string {
    // Return appropriate response based on prompt
    if strings.Contains(prompt, "authentication") {
        return "@on create: [auth]"
    }
    return "@on create: [cache]"
}

report, err := harness.RunWithMockLLM(ctx, testCases, mockFn)
```

### Creating Custom Test Cases

```go
// Method 1: Generate from pattern metadata
generator := llm.NewTestCaseGenerator("Post", "create")
testCase := generator.GenerateFromPattern(pattern)

// Method 2: Create manually
testCase := llm.TestCase{
    Name:     "Add authentication to Post.create",
    Category: "authentication",
    Prompt: `Add authentication middleware to Post.create.
Available pattern: @on <operation>: [auth]
Generate the middleware declaration.`,
    ExpectedPattern: "@on create: [auth]",
    ValidationMode:  "semantic",
}
```

### Running Tests

#### Prerequisites

Set environment variables for API keys:

```bash
export ANTHROPIC_API_KEY="your-claude-api-key"
export OPENAI_API_KEY="your-openai-api-key"
```

#### Run Unit Tests

```bash
cd internal/testing/llm
go test -v
```

#### Run Integration Tests

```bash
go test -v -run TestHarnessIntegration
```

#### Run Against Real LLM APIs

```bash
# Create a test program
cat > test_llm.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "github.com/conduit-lang/conduit/internal/testing/llm"
)

func main() {
    config := llm.NewDefaultConfig()
    harness, err := llm.NewHarness(config)
    if err != nil {
        panic(err)
    }

    harness.SetVerbose(true)

    testCases := llm.PredefinedTestCases()
    report, err := harness.Run(context.Background(), testCases)
    if err != nil {
        panic(err)
    }

    reporter := llm.NewReporter()
    reporter.PrintReport(report)
}
EOF

go run test_llm.go
```

## Test Fixtures

Pattern fixtures are provided in `fixtures/`:

- `auth_pattern.json` - Authentication middleware pattern
- `cache_pattern.json` - Caching pattern
- `rate_limit_pattern.json` - Rate limiting pattern
- `auth_rate_limit_pattern.json` - Combined auth + rate limit
- `validation_pattern.json` - Field validation constraints
- `hook_pattern.json` - Lifecycle hooks
- `relationship_pattern.json` - Resource relationships

### Loading Fixtures

```go
import "encoding/json"
import "os"

func loadPattern(path string) (metadata.PatternMetadata, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return metadata.PatternMetadata{}, err
    }

    var pattern metadata.PatternMetadata
    err = json.Unmarshal(data, &pattern)
    return pattern, err
}

pattern, err := loadPattern("fixtures/auth_pattern.json")
```

## Report Format

### Console Output

```
======================================
LLM Validation Test Harness Report
======================================

Execution Time: 2m30s
Timestamp: 2025-10-27T10:00:00Z

Total Tests: 30
Passed: 24
Failed: 5
Errors: 1
Success Rate: 80.0%

Results by Provider:
--------------------

claude:claude-opus-4:
  Tests: 10
  Passed: 9
  Success Rate: 90.0%
  Avg Duration: 2.5s
  Avg Confidence: 0.95

openai:gpt-4:
  Tests: 10
  Passed: 8
  Success Rate: 80.0%
  Avg Duration: 3.2s
  Avg Confidence: 0.88

Results by Category:
--------------------

authentication:
  Tests: 15
  Passed: 13
  Success Rate: 86.7%

caching:
  Tests: 10
  Passed: 8
  Success Rate: 80.0%

Failed Tests:
-------------

1. Add rate limiting to Post.create [openai:gpt-3.5]
   Expected: @on create: [rate_limit(10/hour)]
   Actual: @on create: [rate_limit(5/minute)]
   Message: middleware mismatch
   Differences:
     - middleware at position 0: expected rate_limit(10/hour), got rate_limit(5/minute)
```

### JSON Export

```go
reporter := llm.NewReporter()
json, err := reporter.ExportJSON(report)
// Save to file or send to CI system
```

## Performance

- Full test suite (<30 tests): <5 minutes
- Parallel execution with configurable concurrency
- Rate limiting to respect API limits
- Efficient parsing and validation

## Best Practices

### 1. Writing Test Cases

- **Be specific**: Clearly state what the LLM should generate
- **Provide context**: Include pattern templates and examples
- **Use semantic validation**: Unless exact format is critical
- **Test variations**: Test different operations and resources

### 2. Interpreting Results

- **85%+ is excellent**: LLM consistently applies patterns
- **70-85% is good**: Most patterns are learned correctly
- **<70% needs investigation**: Patterns may be ambiguous or prompts unclear

### 3. Debugging Failures

1. Check the actual vs expected output in the report
2. Review the prompt - is it clear and specific?
3. Check if the pattern template is ambiguous
4. Try the prompt manually with the LLM to understand its reasoning
5. Adjust prompts or use exact validation if needed

### 4. CI Integration

```yaml
# .github/workflows/llm-validation.yml
name: LLM Pattern Validation

on:
  push:
    paths:
      - 'runtime/metadata/**'
      - 'internal/testing/llm/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run LLM validation
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          cd internal/testing/llm
          go test -v -timeout 10m
```

## Extending the Harness

### Adding a New LLM Provider

1. Update `ProviderType` in `config.go`
2. Implement the API client in `client.go`
3. Add configuration in `defaultProviders()`

### Adding New Test Cases

```go
// In your test file
func TestMyCustomScenario(t *testing.T) {
    testCases := []llm.TestCase{
        {
            Name:     "My custom test",
            Category: "authentication",
            Prompt:   "...",
            ExpectedPattern: "@on create: [auth]",
            ValidationMode:  "semantic",
        },
    }

    // Run tests...
}
```

### Custom Validation Logic

```go
// Extend the validator
type CustomValidator struct {
    *llm.Validator
}

func (v *CustomValidator) CustomMatch(response, expected string) llm.ValidationResult {
    // Your custom validation logic
}
```

## Troubleshooting

### API Key Not Found

**Error**: `at least one provider must be enabled`

**Solution**: Set environment variables:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
```

### Rate Limit Errors

**Error**: `API returned status 429: rate limit exceeded`

**Solution**: Increase `RateLimitDelay` in config:
```go
config.RateLimitDelay = 5 * time.Second
```

### Timeout Errors

**Error**: `context deadline exceeded`

**Solution**: Increase timeout:
```go
config.DefaultTimeout = 120 * time.Second
```

### Parsing Failures

**Issue**: Parser can't extract middleware declarations

**Solution**: Check the response format. The parser expects:
- Format: `@on <operation>: [<middleware>]`
- Example: `@on create: [auth, cache(300)]`

## License

This package is part of the Conduit project. See the main repository LICENSE file.
