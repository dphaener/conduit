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

// Config holds application configuration
type Config struct {
	Format      string
	CheckCycles bool
	Complexity  bool
	Impact      bool
	Report      bool
	Resource    string
}

// Report structures for JSON output
type Report struct {
	CircularDependencies []Cycle            `json:"circular_dependencies,omitempty"`
	Complexity           []ComplexityMetric `json:"complexity,omitempty"`
	Impact               []ImpactMetric     `json:"impact,omitempty"`
}

type Cycle struct {
	Path []string `json:"path"`
}

type ComplexityMetric struct {
	Resource string `json:"resource"`
	Depth    int    `json:"depth"`
	Level    string `json:"level"` // "low", "medium", "high"
}

type ImpactMetric struct {
	Resource       string   `json:"resource"`
	DependentCount int      `json:"dependent_count"`
	Dependents     []string `json:"dependents"`
}

func main() {
	config := parseFlags()

	// Get the registry
	registry := metadata.GetRegistry()

	// Check registry is initialized
	schema := registry.GetSchema()
	if schema == nil {
		fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
		fmt.Fprintln(os.Stderr, "Run 'conduit build' first to generate metadata")
		os.Exit(1)
	}

	// Handle resource-specific analysis
	if config.Resource != "" {
		analyzeResource(registry, config.Resource)
		return
	}

	// Generate full report
	if config.Report || config.CheckCycles || config.Complexity || config.Impact {
		generateReport(registry, config)
		return
	}

	// Default: show usage
	flag.Usage()
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.Format, "format", "table",
		"Output format: table or json")
	flag.BoolVar(&config.CheckCycles, "check-cycles", false,
		"Check for circular dependencies")
	flag.BoolVar(&config.Complexity, "complexity", false,
		"Analyze dependency complexity")
	flag.BoolVar(&config.Impact, "impact", false,
		"Analyze dependency impact")
	flag.BoolVar(&config.Report, "report", false,
		"Generate full dependency report")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [resource]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAnalyzes dependencies between resources.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s Post              Analyze Post dependencies\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --check-cycles    Check for circular dependencies\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --report          Generate full report\n", os.Args[0])
	}

	flag.Parse()

	// Get resource name if provided
	if flag.NArg() > 0 {
		config.Resource = flag.Arg(0)
	}

	return config
}

func analyzeResource(registry *metadata.RegistryAPI, resourceName string) {
	// Verify resource exists
	res, err := registry.Resource(resourceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("DEPENDENCIES: %s\n", resourceName)
	fmt.Println()

	// Get forward dependencies
	forwardGraph, err := registry.Dependencies(resourceName, metadata.DependencyOptions{
		Depth:   2,
		Reverse: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying dependencies: %v\n", err)
		os.Exit(1)
	}

	// Get reverse dependencies
	reverseGraph, err := registry.Dependencies(resourceName, metadata.DependencyOptions{
		Depth:   1,
		Reverse: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying reverse dependencies: %v\n", err)
		os.Exit(1)
	}

	// Show direct dependencies
	fmt.Println("Direct Dependencies (what " + resourceName + " uses):")
	if len(forwardGraph.Edges) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, edge := range forwardGraph.Edges {
			if edge.From == resourceName {
				toNode := forwardGraph.Nodes[edge.To]
				fmt.Printf("  → %s (%s)\n", toNode.Name, edge.Relationship)

				// Show impact based on relationship
				impact := getImpactDescription(edge, res)
				if impact != "" {
					fmt.Printf("    Impact: %s\n", impact)
				}
			}
		}
	}
	fmt.Println()

	// Show reverse dependencies
	fmt.Println("Reverse Dependencies (what uses " + resourceName + "):")
	if len(reverseGraph.Edges) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, edge := range reverseGraph.Edges {
			fromNode := reverseGraph.Nodes[edge.From]
			fmt.Printf("  ← %s (%s)\n", fromNode.Name, edge.Relationship)

			// Show impact
			impact := getReverseImpactDescription(edge, resourceName)
			if impact != "" {
				fmt.Printf("    Impact: %s\n", impact)
			}
		}
	}
	fmt.Println()

	// Show routes
	routes := registry.Routes(metadata.RouteFilter{
		Resource: resourceName,
	})
	if len(routes) > 0 {
		fmt.Printf("Routes using %s:\n", resourceName)
		for _, route := range routes {
			fmt.Printf("  %-6s %s\n", route.Method, route.Path)
		}
	}
}

func generateReport(registry *metadata.RegistryAPI, config Config) {
	report := Report{}

	if config.CheckCycles || config.Report {
		cycles := checkCircularDependencies(registry)
		report.CircularDependencies = cycles
	}

	if config.Complexity || config.Report {
		complexity := analyzeComplexity(registry)
		report.Complexity = complexity
	}

	if config.Impact || config.Report {
		impact := analyzeImpact(registry)
		report.Impact = impact
	}

	if config.Format == "json" {
		outputReportJSON(report)
	} else {
		outputReportTable(report, config)
	}
}

func checkCircularDependencies(registry *metadata.RegistryAPI) []Cycle {
	resources := registry.Resources()
	var cycles []Cycle

	// Check each resource for circular paths
	for _, res := range resources {
		// Get dependency graph with unlimited depth
		graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
			Depth:   0, // 0 means unlimited depth
			Reverse: false,
		})
		if err != nil {
			continue
		}

		// Look for cycles by checking if any dependency path loops back
		visited := make(map[string]bool)
		path := []string{res.Name}
		if detectCycle(graph, res.Name, res.Name, visited, path, &cycles) {
			// Cycle was detected and added
		}
	}

	return cycles
}

func detectCycle(graph *metadata.DependencyGraph, startNode, currentNode string, visited map[string]bool, path []string, cycles *[]Cycle) bool {
	visited[currentNode] = true

	for _, edge := range graph.Edges {
		if edge.From == currentNode {
			// Found a cycle back to start
			if edge.To == startNode && len(path) > 1 {
				*cycles = append(*cycles, Cycle{Path: append(path, edge.To)})
				return true
			}

			// Continue traversing if not visited
			if !visited[edge.To] {
				newPath := append([]string{}, path...)
				newPath = append(newPath, edge.To)
				if detectCycle(graph, startNode, edge.To, visited, newPath, cycles) {
					return true
				}
			}
		}
	}

	return false
}

func analyzeComplexity(registry *metadata.RegistryAPI) []ComplexityMetric {
	resources := registry.Resources()
	metrics := make([]ComplexityMetric, 0, len(resources))

	for _, res := range resources {
		// Get full dependency graph
		graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
			Depth:   0, // 0 means unlimited depth
			Reverse: false,
		})
		if err != nil {
			continue
		}

		// Calculate depth by counting maximum edge chain length
		depth := calculateGraphDepth(graph, res.Name)

		level := "low"
		if depth >= 4 {
			level = "high"
		} else if depth >= 2 {
			level = "medium"
		}

		metrics = append(metrics, ComplexityMetric{
			Resource: res.Name,
			Depth:    depth,
			Level:    level,
		})
	}

	// Sort by depth (descending)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Depth > metrics[j].Depth
	})

	return metrics
}

func calculateGraphDepth(graph *metadata.DependencyGraph, startNode string) int {
	if len(graph.Edges) == 0 {
		return 0
	}

	// Simple depth calculation: count maximum edge chain length
	maxDepth := 0
	visited := make(map[string]int)

	var dfs func(node string, depth int)
	dfs = func(node string, depth int) {
		if depth > maxDepth {
			maxDepth = depth
		}
		visited[node] = depth

		for _, edge := range graph.Edges {
			if edge.From == node {
				if _, seen := visited[edge.To]; !seen {
					dfs(edge.To, depth+1)
				}
			}
		}
	}

	dfs(startNode, 0)
	return maxDepth
}

func analyzeImpact(registry *metadata.RegistryAPI) []ImpactMetric {
	resources := registry.Resources()
	metrics := make([]ImpactMetric, 0, len(resources))

	for _, res := range resources {
		// Get reverse dependencies
		graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
			Depth:   1,
			Reverse: true,
		})
		if err != nil {
			continue
		}

		// Count unique dependents
		dependents := make(map[string]bool)
		for _, edge := range graph.Edges {
			fromNode := graph.Nodes[edge.From]
			if fromNode.Type == "resource" {
				dependents[fromNode.Name] = true
			}
		}

		dependentList := make([]string, 0, len(dependents))
		for dep := range dependents {
			dependentList = append(dependentList, dep)
		}
		sort.Strings(dependentList)

		metrics = append(metrics, ImpactMetric{
			Resource:       res.Name,
			DependentCount: len(dependentList),
			Dependents:     dependentList,
		})
	}

	// Sort by dependent count (descending)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].DependentCount > metrics[j].DependentCount
	})

	return metrics
}

func outputReportJSON(report Report) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputReportTable(report Report, config Config) {
	fmt.Println("DEPENDENCY ANALYSIS")
	fmt.Println()
	fmt.Println("Analyzing resource dependencies...")
	fmt.Println()

	// Circular dependencies
	if config.CheckCycles || config.Report {
		fmt.Println("=== CIRCULAR DEPENDENCIES ===")
		if len(report.CircularDependencies) == 0 {
			fmt.Println("✓ No circular dependencies detected")
		} else {
			fmt.Printf("⚠️  Found %d circular dependencies:\n", len(report.CircularDependencies))
			for i, cycle := range report.CircularDependencies {
				fmt.Printf("  Cycle %d: %s\n", i+1, strings.Join(cycle.Path, " -> "))
			}
		}
		fmt.Println()
	}

	// Complexity
	if config.Complexity || config.Report {
		fmt.Println("=== DEPENDENCY COMPLEXITY ===")
		fmt.Println()

		// Group by complexity level
		byLevel := make(map[string][]ComplexityMetric)
		for _, m := range report.Complexity {
			byLevel[m.Level] = append(byLevel[m.Level], m)
		}

		if high, ok := byLevel["high"]; ok && len(high) > 0 {
			fmt.Println("High Complexity (depth > 3):")
			for _, m := range high {
				fmt.Printf("  ⚠️  %s (depth: %d)\n", m.Resource, m.Depth)
			}
			fmt.Println()
		}

		if medium, ok := byLevel["medium"]; ok && len(medium) > 0 {
			fmt.Println("Medium Complexity (depth 2-3):")
			for _, m := range medium {
				fmt.Printf("  ➜ %s (depth: %d)\n", m.Resource, m.Depth)
			}
			fmt.Println()
		}

		if low, ok := byLevel["low"]; ok && len(low) > 0 {
			fmt.Println("Low Complexity (depth 0-1):")
			for _, m := range low {
				fmt.Printf("  ✓ %s (depth: %d)\n", m.Resource, m.Depth)
			}
			fmt.Println()
		}
	}

	// Impact
	if config.Impact || config.Report {
		fmt.Println("=== DEPENDENCY IMPACT ===")
		fmt.Println()
		fmt.Println("Most Depended On (high impact if changed):")

		// Show top 5
		count := 5
		if len(report.Impact) < count {
			count = len(report.Impact)
		}

		for i := 0; i < count; i++ {
			m := report.Impact[i]
			if m.DependentCount == 0 {
				break
			}

			fmt.Printf("  %d. %s (%d resources depend on it)\n",
				i+1, m.Resource, m.DependentCount)

			// Show first 3 dependents
			depCount := 3
			if len(m.Dependents) < depCount {
				depCount = len(m.Dependents)
			}
			for j := 0; j < depCount; j++ {
				fmt.Printf("     - %s\n", m.Dependents[j])
			}

			if len(m.Dependents) > depCount {
				fmt.Printf("     ... and %d more\n", len(m.Dependents)-depCount)
			}

			fmt.Println()
		}
	}
}

func getImpactDescription(edge metadata.DependencyEdge, res *metadata.ResourceMetadata) string {
	if edge.Relationship == "belongs_to" {
		// Find the relationship metadata
		for _, rel := range res.Relationships {
			if rel.TargetResource == edge.To && rel.Type == "belongs_to" {
				switch rel.OnDelete {
				case "cascade":
					return fmt.Sprintf("Deleting %s cascades to %s", edge.To, res.Name)
				case "restrict":
					return fmt.Sprintf("Cannot delete %s with existing %s", edge.To, res.Name)
				case "set_null":
					return fmt.Sprintf("Deleting %s nullifies %s.%s", edge.To, res.Name, rel.ForeignKey)
				default:
					return fmt.Sprintf("%s requires %s", res.Name, edge.To)
				}
			}
		}
	}
	return ""
}

func getReverseImpactDescription(edge metadata.DependencyEdge, resourceName string) string {
	if edge.Relationship == "belongs_to" {
		return fmt.Sprintf("Deleting %s affects %s records", resourceName, edge.From)
	}
	return ""
}
