package tooling

import (
	"fmt"
	"strings"
)

// buildHover creates hover information for a symbol
func (a *API) buildHover(symbol *Symbol) *Hover {
	var content strings.Builder

	// Write code block with type information
	content.WriteString("```conduit\n")

	switch symbol.Kind {
	case SymbolKindResource:
		content.WriteString(fmt.Sprintf("resource %s", symbol.Name))

	case SymbolKindField:
		content.WriteString(fmt.Sprintf("%s: %s", symbol.Name, symbol.Type))

	case SymbolKindRelationship:
		content.WriteString(fmt.Sprintf("%s: %s", symbol.Name, symbol.Type))

	case SymbolKindComputed:
		content.WriteString(fmt.Sprintf("@computed %s: %s", symbol.Name, symbol.Type))

	case SymbolKindHook:
		content.WriteString(symbol.Detail)

	case SymbolKindScope:
		content.WriteString(fmt.Sprintf("@scope %s", symbol.Name))

	case SymbolKindValidation:
		content.WriteString(fmt.Sprintf("@validate %s", symbol.Name))

	case SymbolKindConstraint:
		content.WriteString(fmt.Sprintf("@constraint %s", symbol.Name))

	case SymbolKindFunction:
		if symbol.Signature != "" {
			content.WriteString(symbol.Signature)
		} else {
			content.WriteString(fmt.Sprintf("function %s", symbol.Name))
		}

	case SymbolKindVariable:
		content.WriteString(fmt.Sprintf("let %s", symbol.Name))
		if symbol.Type != "" {
			content.WriteString(fmt.Sprintf(": %s", symbol.Type))
		}
	}

	content.WriteString("\n```\n\n")

	// Add documentation if available
	if symbol.Documentation != "" {
		content.WriteString(symbol.Documentation)
		content.WriteString("\n\n")
	}

	// Add additional details
	if symbol.ContainerName != "" {
		content.WriteString(fmt.Sprintf("*In resource:* `%s`\n\n", symbol.ContainerName))
	}

	// Add kind-specific information
	switch symbol.Kind {
	case SymbolKindField:
		content.WriteString("---\n\n")
		content.WriteString("**Field**\n\n")
		if strings.HasSuffix(symbol.Type, "!") {
			content.WriteString("*Required* - This field cannot be null\n")
		} else if strings.HasSuffix(symbol.Type, "?") {
			content.WriteString("*Optional* - This field may be null\n")
		}

	case SymbolKindRelationship:
		content.WriteString("---\n\n")
		content.WriteString("**Relationship**\n\n")
		content.WriteString(fmt.Sprintf("Associates this resource with `%s`\n", extractTypeName(symbol.Type)))
	}

	return &Hover{
		Contents: content.String(),
		Range:    symbol.Range,
	}
}

// extractTypeName extracts the type name without nullability marker
func extractTypeName(typeStr string) string {
	typeStr = strings.TrimSuffix(typeStr, "!")
	typeStr = strings.TrimSuffix(typeStr, "?")
	return typeStr
}
