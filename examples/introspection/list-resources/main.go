package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// Config holds the application configuration
type Config struct {
	Format   string
	Category string
	Verbose  bool
}

// ResourceSummary contains summary information about a resource
type ResourceSummary struct {
	Name              string   `json:"name"`
	FieldCount        int      `json:"field_count"`
	RelationshipCount int      `json:"relationship_count"`
	HookCount         int      `json:"hook_count"`
	ValidationCount   int      `json:"validation_count"`
	ConstraintCount   int      `json:"constraint_count"`
	Category          string   `json:"category"`
	Relationships     []string `json:"relationships,omitempty"`
	Hooks             []string `json:"hooks,omitempty"`
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Get the registry
	registry := metadata.GetRegistry()

	// Get all resources
	resources := registry.Resources()
	if resources == nil {
		fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
		fmt.Fprintln(os.Stderr, "Run 'conduit build' first to generate metadata")
		os.Exit(1)
	}

	// Build resource summaries
	summaries := buildSummaries(resources, config)

	// Filter by category if specified
	if config.Category != "" {
		filtered := []ResourceSummary{}
		for _, s := range summaries {
			if s.Category == config.Category {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	// Output results
	if config.Format == "json" {
		outputJSON(summaries)
	} else {
		outputTable(summaries, config.Verbose)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.Format, "format", "table",
		"Output format: table or json")
	flag.StringVar(&config.Category, "category", "",
		"Filter by category (e.g., 'Core Resources')")
	flag.BoolVar(&config.Verbose, "verbose", false,
		"Show detailed information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nLists all resources in a Conduit application.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	return config
}

func buildSummaries(resources []metadata.ResourceMetadata, config Config) []ResourceSummary {
	summaries := make([]ResourceSummary, 0, len(resources))

	for _, res := range resources {
		summary := ResourceSummary{
			Name:              res.Name,
			FieldCount:        len(res.Fields),
			RelationshipCount: len(res.Relationships),
			HookCount:         len(res.Hooks),
			ValidationCount:   len(res.Validations),
			ConstraintCount:   len(res.Constraints),
			Category:          categorizeResource(res.Name),
		}

		if config.Verbose {
			// Add relationship details
			for _, rel := range res.Relationships {
				summary.Relationships = append(summary.Relationships,
					fmt.Sprintf("%s %s", rel.Type, rel.TargetResource))
			}

			// Add hook details
			for _, hook := range res.Hooks {
				summary.Hooks = append(summary.Hooks, hook.Type)
			}
		}

		summaries = append(summaries, summary)
	}

	// Sort by category, then by name
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Category == summaries[j].Category {
			return summaries[i].Name < summaries[j].Name
		}
		return summaries[i].Category < summaries[j].Category
	})

	return summaries
}

func categorizeResource(name string) string {
	// Simple categorization based on common patterns
	corePatterns := []string{"User", "Post", "Comment", "Article", "Page", "Product", "Order"}
	adminPatterns := []string{"Category", "Tag", "Setting", "Config"}
	systemPatterns := []string{"Log", "Audit", "Session", "Token", "Job"}

	for _, pattern := range corePatterns {
		if name == pattern {
			return "Core Resources"
		}
	}

	for _, pattern := range adminPatterns {
		if name == pattern {
			return "Administrative"
		}
	}

	for _, pattern := range systemPatterns {
		if name == pattern {
			return "System"
		}
	}

	return "Other"
}

func outputJSON(summaries []ResourceSummary) {
	output := struct {
		TotalCount int               `json:"total_count"`
		Resources  []ResourceSummary `json:"resources"`
	}{
		TotalCount: len(summaries),
		Resources:  summaries,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputTable(summaries []ResourceSummary, verbose bool) {
	fmt.Println("CONDUIT RESOURCES")
	fmt.Println()
	fmt.Printf("Total: %d resources\n", len(summaries))
	fmt.Println()

	// Group by category
	byCategory := make(map[string][]ResourceSummary)
	for _, s := range summaries {
		byCategory[s.Category] = append(byCategory[s.Category], s)
	}

	// Sort categories
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Print each category
	for _, cat := range categories {
		resources := byCategory[cat]
		fmt.Printf("%s (%d):\n", cat, len(resources))

		for _, res := range resources {
			fmt.Printf("  %s\n", res.Name)

			if verbose {
				fmt.Printf("    Fields: %d\n", res.FieldCount)

				if res.RelationshipCount > 0 {
					fmt.Printf("    Relationships: %d", res.RelationshipCount)
					if len(res.Relationships) > 0 {
						fmt.Printf(" (%s)", strings.Join(res.Relationships, ", "))
					}
					fmt.Println()
				}

				if res.HookCount > 0 {
					fmt.Printf("    Hooks: %d", res.HookCount)
					if len(res.Hooks) > 0 {
						fmt.Printf(" (%s)", strings.Join(res.Hooks, ", "))
					}
					fmt.Println()
				}

				if res.ValidationCount > 0 {
					fmt.Printf("    Validations: %d\n", res.ValidationCount)
				}

				if res.ConstraintCount > 0 {
					fmt.Printf("    Constraints: %d\n", res.ConstraintCount)
				}

				fmt.Println()
			} else {
				// Compact format
				details := []string{}
				if res.FieldCount > 0 {
					details = append(details, fmt.Sprintf("%d fields", res.FieldCount))
				}
				if res.RelationshipCount > 0 {
					details = append(details, fmt.Sprintf("%d relationships", res.RelationshipCount))
				}
				if res.HookCount > 0 {
					details = append(details, fmt.Sprintf("%d hooks", res.HookCount))
				}

				if len(details) > 0 {
					fmt.Printf("    (%s)\n", strings.Join(details, ", "))
				}
			}
		}

		fmt.Println()
	}
}
