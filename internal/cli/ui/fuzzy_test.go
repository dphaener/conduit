package ui

import (
	"reflect"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"Post", "Pst", 1},
		{"User", "Uses", 1},
		{"Product", "Produce", 1},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := LevenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d; want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestFindSimilar(t *testing.T) {
	candidates := []string{"Post", "User", "Product", "Comment", "Category"}

	tests := []struct {
		name     string
		target   string
		opts     *FuzzyMatchOptions
		expected []string
	}{
		{
			name:     "exact match",
			target:   "Post",
			opts:     nil,
			expected: []string{"Post"},
		},
		{
			name:     "one character off",
			target:   "Pst",
			opts:     nil,
			expected: []string{"Post", "User"}, // "User" is also distance 3 from "Pst"
		},
		{
			name:     "case insensitive",
			target:   "post",
			opts:     nil,
			expected: []string{"Post"},
		},
		{
			name:   "case sensitive",
			target: "post",
			opts: &FuzzyMatchOptions{
				MaxDistance:    3,
				MaxSuggestions: 3,
				CaseSensitive:  true,
			},
			expected: []string{"Post"},
		},
		{
			name:     "multiple suggestions",
			target:   "Prod",
			opts:     nil,
			expected: []string{"Post", "Product"}, // Both "Post" and "Product" are within distance 3
		},
		{
			name:     "no match too far",
			target:   "XYZ",
			opts:     nil,
			expected: []string{},
		},
		{
			name:   "max suggestions limit",
			target: "Categor",
			opts: &FuzzyMatchOptions{
				MaxDistance:    3,
				MaxSuggestions: 1,
			},
			expected: []string{"Category"}, // Should be limited to 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindSimilar(tt.target, candidates, tt.opts)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("FindSimilar(%q) returned %d results; want %d\nGot: %v\nWant: %v",
					tt.target, len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Check if results match (order matters due to distance sorting)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FindSimilar(%q) = %v; want %v", tt.target, result, tt.expected)
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	candidates := []string{"Post", "User", "Product", "Comment"}

	tests := []struct {
		target   string
		expected string
	}{
		{"Pst", "Post"},
		{"Usr", "User"},
		{"Prodct", "Product"},
		{"Coment", "Comment"},
		{"XYZ", ""}, // No close match
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := FindBestMatch(tt.target, candidates, nil)
			if result != tt.expected {
				t.Errorf("FindBestMatch(%q) = %q; want %q", tt.target, result, tt.expected)
			}
		})
	}
}

func TestHasCloseMatch(t *testing.T) {
	candidates := []string{"Post", "User", "Product"}

	tests := []struct {
		target   string
		expected bool
	}{
		{"Pst", true},
		{"Post", true},
		{"Usr", true},
		{"XYZ", false},
		{"Zebra", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := HasCloseMatch(tt.target, candidates, nil)
			if result != tt.expected {
				t.Errorf("HasCloseMatch(%q) = %v; want %v", tt.target, result, tt.expected)
			}
		})
	}
}

func TestFuzzyMatchOptions(t *testing.T) {
	candidates := []string{"Post", "User", "Product"}

	// Test with max suggestions = 1
	result := FindSimilar("Pst", candidates, &FuzzyMatchOptions{
		MaxDistance:    3,
		MaxSuggestions: 1,
	})

	if len(result) > 1 {
		t.Errorf("Expected max 1 suggestion, got %d", len(result))
	}

	if len(result) == 0 {
		t.Errorf("Expected at least 1 suggestion")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{5, 5, 5, 5},
		{0, 1, 2, 0},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b, tt.c)
		if result != tt.expected {
			t.Errorf("min(%d, %d, %d) = %d; want %d", tt.a, tt.b, tt.c, result, tt.expected)
		}
	}
}

func TestFindSimilarEmptyCandidates(t *testing.T) {
	result := FindSimilar("test", []string{}, nil)
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty candidates, got %v", result)
	}
}

func TestFindSimilarEmptyTarget(t *testing.T) {
	candidates := []string{"AB", "XY"}
	result := FindSimilar("", candidates, &FuzzyMatchOptions{
		MaxDistance:    2,
		MaxSuggestions: 3,
	})

	// Empty string should have distance of len(candidate) for each
	// With MaxDistance=2, strings <= 2 chars should match
	if len(result) == 0 {
		t.Errorf("Expected some matches for empty target string with short candidates")
	}
}
