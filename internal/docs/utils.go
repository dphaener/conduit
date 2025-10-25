package docs

// containsPathTraversal checks if a path contains path traversal sequences
func containsPathTraversal(path string) bool {
	// Check for .. after cleaning
	parts := splitPath(path)
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}

// splitPath splits a path into its components
func splitPath(path string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' || path[i] == '\\' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(path[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
