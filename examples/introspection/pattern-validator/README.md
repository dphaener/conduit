# Pattern Validator Example

Validates that resources follow discovered patterns and coding standards.

## Usage

```bash
go build -o pattern-validator
./pattern-validator
./pattern-validator --strict  # Fail on any violations
```

## Features

- Validates authentication patterns
- Checks rate limiting on creates
- Validates slug generation patterns
- Enforces coding standards
- Configurable rule sets

## Example Output

```
PATTERN VALIDATION

✓ Post follows authenticated_handler pattern
⚠️  Comment: create operation should have rate_limit
✓ User follows all patterns

Summary: 1 warning, 0 errors
```
