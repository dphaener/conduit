package tooling

import (
	"strings"
)

// CompletionContext describes the context at a completion position
type CompletionContext struct {
	// Kind of completion requested
	Kind CompletionContextKind

	// For field type completions: nothing
	// For namespace completions: the namespace name (e.g., "String")
	Namespace string

	// For pattern completions: the category
	Category string

	// Token before cursor
	PrecedingToken string
}

// CompletionContextKind categorizes the completion context
type CompletionContextKind int

const (
	// CompletionContextUnknown represents an unknown context
	CompletionContextUnknown CompletionContextKind = iota
	// CompletionContextType represents a type completion context
	CompletionContextType
	// CompletionContextAnnotation represents an annotation completion context
	CompletionContextAnnotation
	// CompletionContextKeyword represents a keyword completion context
	CompletionContextKeyword
	// CompletionContextFieldName represents a field name completion context
	CompletionContextFieldName
	// CompletionContextNamespace represents a namespace method completion context
	CompletionContextNamespace
)

// getCompletionContext determines the completion context at a position
func (a *API) getCompletionContext(doc *Document, pos Position) *CompletionContext {
	// Get the line content up to the cursor
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return &CompletionContext{Kind: CompletionContextUnknown}
	}

	line := lines[pos.Line]
	if pos.Character > len(line) {
		pos.Character = len(line)
	}

	prefix := line[:pos.Character]
	trimmed := strings.TrimSpace(prefix)

	// Check for annotation context (@)
	if strings.HasSuffix(trimmed, "@") {
		return &CompletionContext{
			Kind: CompletionContextAnnotation,
		}
	}

	// Check for type context (after colon)
	if strings.Contains(trimmed, ":") {
		parts := strings.Split(trimmed, ":")
		if len(parts) >= 2 {
			afterColon := strings.TrimSpace(parts[len(parts)-1])
			if afterColon == "" || !strings.Contains(afterColon, " ") {
				return &CompletionContext{
					Kind: CompletionContextType,
				}
			}
		}
	}

	// Check for namespace method completion (e.g., "String.")
	if strings.Contains(trimmed, ".") {
		parts := strings.Split(trimmed, ".")
		if len(parts) >= 2 {
			namespace := strings.TrimSpace(parts[len(parts)-2])
			// Check if it's a valid namespace
			if isNamespace(namespace) {
				return &CompletionContext{
					Kind:      CompletionContextNamespace,
					Namespace: namespace,
				}
			}
		}
	}

	// Default to keyword completion
	return &CompletionContext{
		Kind: CompletionContextKeyword,
	}
}

// isNamespace checks if a string is a valid namespace
func isNamespace(s string) bool {
	namespaces := []string{
		"String", "Time", "Array", "Hash", "Math", "JSON",
	}
	for _, ns := range namespaces {
		if s == ns {
			return true
		}
	}
	return false
}

// buildCompletions builds completion items based on context
func (a *API) buildCompletions(context *CompletionContext) []CompletionItem {
	switch context.Kind {
	case CompletionContextType:
		return a.getTypeCompletions()

	case CompletionContextAnnotation:
		return a.getAnnotationCompletions()

	case CompletionContextKeyword:
		return a.getKeywordCompletions()

	case CompletionContextNamespace:
		return a.getNamespaceCompletions(context.Namespace)

	default:
		return nil
	}
}

// getTypeCompletions returns type completions
func (a *API) getTypeCompletions() []CompletionItem {
	types := []struct {
		name   string
		detail string
	}{
		{"string", "Text string (variable length)"},
		{"text", "Long text (unlimited length)"},
		{"markdown", "Markdown formatted text"},
		{"int", "Integer number"},
		{"float", "Floating point number"},
		{"decimal", "Precise decimal number"},
		{"bool", "Boolean (true/false)"},
		{"timestamp", "Date and time"},
		{"date", "Date only"},
		{"time", "Time only"},
		{"uuid", "UUID identifier"},
		{"ulid", "ULID identifier"},
		{"email", "Email address"},
		{"url", "URL"},
		{"phone", "Phone number"},
		{"json", "JSON data"},
		{"array", "Array collection"},
		{"hash", "Key-value map"},
		{"enum", "Enumeration"},
	}

	items := make([]CompletionItem, len(types))
	for i, t := range types {
		items[i] = CompletionItem{
			Label:         t.name,
			Kind:          CompletionKindType,
			Detail:        t.detail,
			Documentation: t.detail,
			InsertText:    t.name,
		}
	}

	return items
}

// getAnnotationCompletions returns annotation completions
func (a *API) getAnnotationCompletions() []CompletionItem {
	annotations := []struct {
		name   string
		detail string
		insert string
	}{
		{"@before create", "Lifecycle hook before creating", "@before create {\n  $0\n}"},
		{"@before update", "Lifecycle hook before updating", "@before update {\n  $0\n}"},
		{"@before delete", "Lifecycle hook before deleting", "@before delete {\n  $0\n}"},
		{"@before save", "Lifecycle hook before saving", "@before save {\n  $0\n}"},
		{"@after create", "Lifecycle hook after creating", "@after create {\n  $0\n}"},
		{"@after update", "Lifecycle hook after updating", "@after update {\n  $0\n}"},
		{"@after delete", "Lifecycle hook after deleting", "@after delete {\n  $0\n}"},
		{"@after save", "Lifecycle hook after saving", "@after save {\n  $0\n}"},
		{"@primary", "Primary key constraint", "@primary"},
		{"@auto", "Auto-generated value", "@auto"},
		{"@auto_update", "Auto-update on save", "@auto_update"},
		{"@unique", "Unique constraint", "@unique"},
		{"@default", "Default value", "@default($0)"},
		{"@min", "Minimum value constraint", "@min($0)"},
		{"@max", "Maximum value constraint", "@max($0)"},
		{"@pattern", "Regular expression pattern", "@pattern($0)"},
		{"@validate", "Validation block", "@validate ${1:name} {\n  condition: $0\n  error: \"\"\n}"},
		{"@constraint", "Constraint block", "@constraint ${1:name} {\n  on: [create, update]\n  condition: $0\n  error: \"\"\n}"},
		{"@scope", "Named scope", "@scope ${1:name} {\n  $0\n}"},
		{"@computed", "Computed field", "@computed ${1:name}: ${2:type} {\n  $0\n}"},
		{"@transaction", "Transaction modifier", "@transaction"},
		{"@async", "Async modifier", "@async"},
	}

	items := make([]CompletionItem, len(annotations))
	for i, a := range annotations {
		items[i] = CompletionItem{
			Label:         a.name,
			Kind:          CompletionKindKeyword,
			Detail:        a.detail,
			Documentation: a.detail,
			InsertText:    a.insert,
		}
	}

	return items
}

// getKeywordCompletions returns keyword completions
func (a *API) getKeywordCompletions() []CompletionItem {
	keywords := []struct {
		name   string
		detail string
	}{
		{"resource", "Define a new resource"},
		{"if", "Conditional statement"},
		{"elsif", "Else-if clause"},
		{"else", "Else clause"},
		{"match", "Pattern matching"},
		{"when", "Match case"},
		{"let", "Variable declaration"},
		{"return", "Return statement"},
		{"self", "Current resource instance"},
	}

	items := make([]CompletionItem, len(keywords))
	for i, k := range keywords {
		items[i] = CompletionItem{
			Label:         k.name,
			Kind:          CompletionKindKeyword,
			Detail:        k.detail,
			Documentation: k.detail,
			InsertText:    k.name,
		}
	}

	return items
}

// getNamespaceCompletions returns completions for namespace methods
func (a *API) getNamespaceCompletions(namespace string) []CompletionItem {
	switch namespace {
	case "String":
		return getStringNamespaceCompletions()
	case "Time":
		return getTimeNamespaceCompletions()
	case "Array":
		return getArrayNamespaceCompletions()
	case "Hash":
		return getHashNamespaceCompletions()
	case "Math":
		return getMathNamespaceCompletions()
	case "JSON":
		return getJSONNamespaceCompletions()
	default:
		return nil
	}
}

func getStringNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"length", "length(s: string!): int!", "Returns the length of a string"},
		{"slugify", "slugify(s: string!): string!", "Converts string to URL-friendly slug"},
		{"uppercase", "uppercase(s: string!): string!", "Converts string to uppercase"},
		{"lowercase", "lowercase(s: string!): string!", "Converts string to lowercase"},
		{"trim", "trim(s: string!): string!", "Removes leading/trailing whitespace"},
		{"contains", "contains(s: string!, substr: string!): bool!", "Checks if string contains substring"},
		{"starts_with", "starts_with(s: string!, prefix: string!): bool!", "Checks if string starts with prefix"},
		{"ends_with", "ends_with(s: string!, suffix: string!): bool!", "Checks if string ends with suffix"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}

func getTimeNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"now", "now(): timestamp!", "Returns current timestamp"},
		{"today", "today(): date!", "Returns today's date"},
		{"parse", "parse(s: string!): timestamp!", "Parses timestamp from string"},
		{"format", "format(t: timestamp!, fmt: string!): string!", "Formats timestamp as string"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}

func getArrayNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"length", "length<T>(arr: array<T>!): int!", "Returns array length"},
		{"contains", "contains<T>(arr: array<T>!, item: T!): bool!", "Checks if array contains item"},
		{"map", "map<T, U>(arr: array<T>!, fn: function): array<U>!", "Transforms array elements"},
		{"filter", "filter<T>(arr: array<T>!, fn: function): array<T>!", "Filters array elements"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}

func getHashNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"keys", "keys<K, V>(h: hash<K, V>!): array<K>!", "Returns hash keys"},
		{"values", "values<K, V>(h: hash<K, V>!): array<V>!", "Returns hash values"},
		{"has_key", "has_key<K, V>(h: hash<K, V>!, key: K!): bool!", "Checks if key exists"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}

func getMathNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"abs", "abs(n: int!): int!", "Returns absolute value"},
		{"min", "min(a: int!, b: int!): int!", "Returns minimum value"},
		{"max", "max(a: int!, b: int!): int!", "Returns maximum value"},
		{"round", "round(n: float!): int!", "Rounds to nearest integer"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}

func getJSONNamespaceCompletions() []CompletionItem {
	funcs := []struct {
		name      string
		signature string
		detail    string
	}{
		{"parse", "parse(s: string!): json!", "Parses JSON from string"},
		{"stringify", "stringify(j: json!): string!", "Converts JSON to string"},
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		items[i] = CompletionItem{
			Label:         f.name,
			Kind:          CompletionKindFunction,
			Detail:        f.signature,
			Documentation: f.detail,
			InsertText:    f.name + "($0)",
		}
	}

	return items
}
