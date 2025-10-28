package ui

import (
	"sort"
	"strings"
)

const (
	// DefaultMaxDistance is the default maximum edit distance to consider for fuzzy matching
	DefaultMaxDistance = 3
	// DefaultMaxSuggestions is the default maximum number of suggestions to return
	DefaultMaxSuggestions = 3
)

// FuzzyMatchOptions configures fuzzy matching behavior
type FuzzyMatchOptions struct {
	MaxDistance    int  // Maximum Levenshtein distance to consider (default: 3)
	MaxSuggestions int  // Maximum number of suggestions to return (default: 3)
	CaseSensitive  bool // Whether matching is case-sensitive (default: false)
}

// suggestion represents a fuzzy match result with its edit distance
type suggestion struct {
	value    string
	distance int
}

// FindSimilar finds strings similar to the target using Levenshtein distance
//
// Example:
//
//	candidates := []string{"Post", "User", "Product", "Comment"}
//	suggestions := FindSimilar("Pst", candidates, nil)
//	// Returns: ["Post"]
func FindSimilar(target string, candidates []string, opts *FuzzyMatchOptions) []string {
	if opts == nil {
		opts = &FuzzyMatchOptions{
			MaxDistance:    DefaultMaxDistance,
			MaxSuggestions: DefaultMaxSuggestions,
			CaseSensitive:  false,
		}
	}

	if opts.MaxDistance == 0 {
		opts.MaxDistance = DefaultMaxDistance
	}
	if opts.MaxSuggestions == 0 {
		opts.MaxSuggestions = DefaultMaxSuggestions
	}

	var suggestions []suggestion

	for _, candidate := range candidates {
		targetCmp := target
		candidateCmp := candidate

		if !opts.CaseSensitive {
			targetCmp = strings.ToLower(target)
			candidateCmp = strings.ToLower(candidate)
		}

		dist := LevenshteinDistance(targetCmp, candidateCmp)
		if dist <= opts.MaxDistance {
			suggestions = append(suggestions, suggestion{
				value:    candidate,
				distance: dist,
			})
		}
	}

	// Sort by distance (closest first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].distance < suggestions[j].distance
	})

	// Return top suggestions
	result := make([]string, 0, opts.MaxSuggestions)
	for i := 0; i < len(suggestions) && i < opts.MaxSuggestions; i++ {
		result = append(result, suggestions[i].value)
	}

	return result
}

// LevenshteinDistance calculates the Levenshtein distance between two strings
// This is the minimum number of single-character edits (insertions, deletions, or substitutions)
// required to change one string into the other.
//
// Example:
//
//	LevenshteinDistance("kitten", "sitting") // Returns: 3
//	LevenshteinDistance("saturday", "sunday") // Returns: 3
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// FindBestMatch returns the single best match for a target string
// Returns an empty string if no match is found within the max distance
func FindBestMatch(target string, candidates []string, opts *FuzzyMatchOptions) string {
	matches := FindSimilar(target, candidates, opts)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// HasCloseMatch checks if there's at least one match within the max distance
func HasCloseMatch(target string, candidates []string, opts *FuzzyMatchOptions) bool {
	return FindBestMatch(target, candidates, opts) != ""
}
