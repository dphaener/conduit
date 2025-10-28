# Schema Explorer Example

An interactive terminal UI for exploring the Conduit schema.

## Usage

```bash
go build -o schema-explorer
./schema-explorer
```

## Features

- Interactive REPL interface
- Browse resources
- View resource details
- Explore dependencies
- Query routes
- Tab completion
- Command history

## Commands

```
Commands:
  list                 List all resources
  show <resource>      Show resource details
  routes [resource]    List routes (optionally filtered)
  deps <resource>      Show dependencies
  patterns [category]  Show patterns
  help                 Show help
  exit                 Exit the explorer
```

## Example Session

```
Conduit Schema Explorer
Type 'help' for available commands

> list
Found 3 resources:
  - Post (5 fields, 1 relationship)
  - User (4 fields)
  - Comment (2 fields, 2 relationships)

> show Post
Resource: Post
Fields: 5
  - id: uuid (required)
  - title: string (required)
  - slug: string (required)
  - content: text (required)
  - published: boolean (required)

Relationships: 1
  - author: belongs_to User

Hooks: 1
  - before_create

> deps Post
Dependencies for Post:
  Direct: 1 (User)
  Reverse: 1 (Comment)

> exit
Goodbye!
```
