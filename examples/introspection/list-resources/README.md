# List Resources Example

A simple command-line tool that lists all resources in a Conduit application with detailed information.

## What It Does

- Lists all resources in the application
- Shows field count, relationship count, and hook count for each
- Supports filtering by category
- Outputs in table or JSON format

## Usage

```bash
# Build the tool
go build -o list-resources

# Run it
./list-resources

# JSON output
./list-resources --format json

# Filter by category
./list-resources --category "Core Resources"

# Verbose output
./list-resources --verbose
```

## Example Output

```
CONDUIT RESOURCES

Total: 5 resources

Core Resources (3):
  Post
    Fields: 5
    Relationships: 1 (belongs_to User)
    Hooks: 1 (before_create)
    Validations: 3

  User
    Fields: 4
    Relationships: 0
    Hooks: 0
    Validations: 1

  Comment
    Fields: 2
    Relationships: 2 (belongs_to Post, belongs_to User)
    Hooks: 0
    Validations: 2
```

## Code Overview

The tool demonstrates:
- Connecting to the metadata registry
- Querying all resources
- Formatting output in multiple formats
- Categorizing resources
- Error handling

## Learning Points

1. **Registry Access**: How to get the metadata registry
2. **Resource Querying**: Iterate over all resources
3. **Data Formatting**: Present data in human-readable format
4. **Error Handling**: Handle missing or invalid metadata

## Next Steps

- See [dependency-analyzer](../dependency-analyzer/) for dependency analysis
- See [api-doc-generator](../api-doc-generator/) for doc generation
