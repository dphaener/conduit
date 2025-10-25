# Conduit Language Support for VS Code

This extension provides language support for the Conduit programming language in Visual Studio Code.

## Features

- **Syntax Highlighting**: Full syntax highlighting for `.cdt` files
- **Code Completion**: Intelligent code completion for types, annotations, keywords, and namespace methods
- **Diagnostics**: Real-time syntax and type error detection
- **Go to Definition**: Navigate to resource and field definitions
- **Hover Information**: View type information and documentation on hover
- **Find References**: Find all references to a symbol
- **Document Symbols**: Navigate symbols within a document
- **Signature Help**: Parameter hints for function calls

## Requirements

This extension requires the `conduit` compiler to be installed and available in your PATH.

You can install Conduit from [github.com/conduit-lang/conduit](https://github.com/conduit-lang/conduit).

## Extension Settings

This extension contributes the following settings:

- `conduit.lsp.enabled`: Enable/disable the Conduit Language Server (default: `true`)
- `conduit.lsp.path`: Path to the conduit executable (default: `conduit`)
- `conduit.lsp.trace.server`: Trace communication with the language server for debugging

## Usage

1. Install the extension
2. Open a folder containing `.cdt` files
3. The language server will start automatically
4. Start writing Conduit code and enjoy IDE features!

## Example

```conduit
/// A blog post resource
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  content: text! @min(100)
  published_at: timestamp?

  author: User! {
    foreign_key: "author_id"
  }

  @before create {
    self.slug = String.slugify(self.title)
  }
}
```

## Known Issues

This is an early release. Please report issues at [github.com/conduit-lang/conduit/issues](https://github.com/conduit-lang/conduit/issues).

## Release Notes

### 0.1.0

Initial release with:
- Syntax highlighting
- LSP integration
- Code completion
- Diagnostics
- Navigation features
