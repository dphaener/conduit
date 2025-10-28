package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

type ValidationRule struct {
	Name        string
	Description string
	Level       string // "error" or "warning"
	Check       func(res metadata.ResourceMetadata) []string
}

func main() {
	strict := flag.Bool("strict", false, "Fail on warnings")
	flag.Parse()

	registry := metadata.GetRegistry()

	if registry.GetSchema() == nil {
		fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
		os.Exit(1)
	}

	rules := defineRules()
	resources := registry.Resources()

	fmt.Println("PATTERN VALIDATION")
	fmt.Println()

	totalWarnings := 0
	totalErrors := 0

	for _, res := range resources {
		violations := []string{}

		for _, rule := range rules {
			issues := rule.Check(res)
			for _, issue := range issues {
				prefix := "⚠️ "
				if rule.Level == "error" {
					prefix = "✗"
					totalErrors++
				} else {
					totalWarnings++
				}
				violations = append(violations, fmt.Sprintf("%s %s", prefix, issue))
			}
		}

		if len(violations) > 0 {
			fmt.Printf("%s:\n", res.Name)
			for _, v := range violations {
				fmt.Printf("  %s\n", v)
			}
			fmt.Println()
		} else {
			fmt.Printf("✓ %s follows all patterns\n", res.Name)
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d warnings, %d errors\n", totalWarnings, totalErrors)

	if totalErrors > 0 || (*strict && totalWarnings > 0) {
		os.Exit(1)
	}
}

func defineRules() []ValidationRule {
	return []ValidationRule{
		{
			Name:        "auth_on_mutations",
			Description: "Mutation operations should require authentication",
			Level:       "warning",
			Check: func(res metadata.ResourceMetadata) []string {
				var issues []string
				for _, op := range []string{"create", "update", "delete"} {
					mw, exists := res.Middleware[op]
					if !exists {
						continue
					}

					hasAuth := false
					for _, m := range mw {
						if strings.Contains(m, "auth") {
							hasAuth = true
							break
						}
					}

					if !hasAuth {
						issues = append(issues, fmt.Sprintf("%s operation should have auth", op))
					}
				}
				return issues
			},
		},
		{
			Name:        "rate_limit_on_creates",
			Description: "Create operations should have rate limiting",
			Level:       "warning",
			Check: func(res metadata.ResourceMetadata) []string {
				var issues []string
				createMW, exists := res.Middleware["create"]
				if !exists {
					return issues
				}

				hasRateLimit := false
				for _, mw := range createMW {
					if strings.Contains(mw, "rate_limit") {
						hasRateLimit = true
						break
					}
				}

				if !hasRateLimit {
					issues = append(issues, "create operation should have rate_limit")
				}

				return issues
			},
		},
		{
			Name:        "slug_for_title",
			Description: "Resources with title should have slug",
			Level:       "warning",
			Check: func(res metadata.ResourceMetadata) []string {
				var issues []string

				hasTitle := false
				hasSlug := false

				for _, field := range res.Fields {
					if field.Name == "title" {
						hasTitle = true
					}
					if field.Name == "slug" {
						hasSlug = true
					}
				}

				if hasTitle && !hasSlug {
					issues = append(issues, "has 'title' but missing 'slug' field")
				}

				return issues
			},
		},
	}
}
