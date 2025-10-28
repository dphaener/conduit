package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

type Explorer struct {
	registry *metadata.RegistryAPI
	scanner  *bufio.Scanner
}

func main() {
	registry := metadata.GetRegistry()

	if registry.GetSchema() == nil {
		fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
		fmt.Fprintln(os.Stderr, "Run 'conduit build' first to generate metadata")
		os.Exit(1)
	}

	explorer := &Explorer{
		registry: registry,
		scanner:  bufio.NewScanner(os.Stdin),
	}

	explorer.Run()
}

func (e *Explorer) Run() {
	fmt.Println("Conduit Schema Explorer")
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	for {
		fmt.Print("> ")

		if !e.scanner.Scan() {
			break
		}

		line := strings.TrimSpace(e.scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		command := parts[0]
		args := parts[1:]

		switch command {
		case "help":
			e.showHelp()
		case "list":
			e.listResources()
		case "show":
			if len(args) < 1 {
				fmt.Println("Usage: show <resource>")
				continue
			}
			e.showResource(args[0])
		case "routes":
			if len(args) > 0 {
				e.listRoutes(args[0])
			} else {
				e.listRoutes("")
			}
		case "deps":
			if len(args) < 1 {
				fmt.Println("Usage: deps <resource>")
				continue
			}
			e.showDependencies(args[0])
		case "patterns":
			if len(args) > 0 {
				e.showPatterns(args[0])
			} else {
				e.showPatterns("")
			}
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Type 'help' for available commands")
		}

		fmt.Println()
	}
}

func (e *Explorer) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  list                 List all resources")
	fmt.Println("  show <resource>      Show resource details")
	fmt.Println("  routes [resource]    List routes (optionally filtered)")
	fmt.Println("  deps <resource>      Show dependencies")
	fmt.Println("  patterns [category]  Show patterns")
	fmt.Println("  help                 Show this help")
	fmt.Println("  exit                 Exit the explorer")
}

func (e *Explorer) listResources() {
	resources := e.registry.Resources()

	fmt.Printf("Found %d resources:\n", len(resources))

	// Sort by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	for _, res := range resources {
		details := []string{}
		if len(res.Fields) > 0 {
			details = append(details, fmt.Sprintf("%d fields", len(res.Fields)))
		}
		if len(res.Relationships) > 0 {
			details = append(details, fmt.Sprintf("%d relationships", len(res.Relationships)))
		}

		detailStr := ""
		if len(details) > 0 {
			detailStr = " (" + strings.Join(details, ", ") + ")"
		}

		fmt.Printf("  - %s%s\n", res.Name, detailStr)
	}
}

func (e *Explorer) showResource(name string) {
	res, err := e.registry.Resource(name)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Resource: %s\n", res.Name)

	if res.Documentation != "" {
		fmt.Printf("Description: %s\n", res.Documentation)
	}

	fmt.Printf("Fields: %d\n", len(res.Fields))
	for _, field := range res.Fields {
		required := ""
		if field.Required {
			required = " (required)"
		}
		fmt.Printf("  - %s: %s%s\n", field.Name, field.Type, required)
	}

	if len(res.Relationships) > 0 {
		fmt.Printf("\nRelationships: %d\n", len(res.Relationships))
		for _, rel := range res.Relationships {
			fmt.Printf("  - %s: %s %s\n", rel.Name, rel.Type, rel.TargetResource)
		}
	}

	if len(res.Hooks) > 0 {
		fmt.Printf("\nHooks: %d\n", len(res.Hooks))
		for _, hook := range res.Hooks {
			fmt.Printf("  - %s\n", hook.Type)
		}
	}

	if len(res.Middleware) > 0 {
		fmt.Println("\nMiddleware:")
		for op, mw := range res.Middleware {
			fmt.Printf("  %s: %s\n", op, strings.Join(mw, ", "))
		}
	}
}

func (e *Explorer) listRoutes(resourceFilter string) {
	filter := metadata.RouteFilter{}
	if resourceFilter != "" {
		filter.Resource = resourceFilter
	}

	routes := e.registry.Routes(filter)

	if len(routes) == 0 {
		fmt.Println("No routes found")
		return
	}

	fmt.Printf("Found %d routes:\n", len(routes))

	// Sort by path
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	for _, route := range routes {
		mw := ""
		if len(route.Middleware) > 0 {
			mw = fmt.Sprintf(" [%s]", strings.Join(route.Middleware, ", "))
		}

		fmt.Printf("  %-6s %-30s -> %s%s\n",
			route.Method, route.Path, route.Operation, mw)
	}
}

func (e *Explorer) showDependencies(name string) {
	// Get forward dependencies
	forwardGraph, err := e.registry.Dependencies(name, metadata.DependencyOptions{
		Depth:   1,
		Reverse: false,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get reverse dependencies
	reverseGraph, err := e.registry.Dependencies(name, metadata.DependencyOptions{
		Depth:   1,
		Reverse: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Dependencies for %s:\n", name)

	// Count dependencies (exclude self)
	directCount := 0
	for _, edge := range forwardGraph.Edges {
		if edge.From == name {
			directCount++
		}
	}

	reverseCount := 0
	for _, edge := range reverseGraph.Edges {
		if edge.To == name {
			reverseCount++
		}
	}

	fmt.Printf("  Direct: %d", directCount)
	if directCount > 0 {
		deps := []string{}
		for _, edge := range forwardGraph.Edges {
			if edge.From == name {
				node := forwardGraph.Nodes[edge.To]
				deps = append(deps, node.Name)
			}
		}
		fmt.Printf(" (%s)", strings.Join(deps, ", "))
	}
	fmt.Println()

	fmt.Printf("  Reverse: %d", reverseCount)
	if reverseCount > 0 {
		deps := []string{}
		for _, edge := range reverseGraph.Edges {
			node := reverseGraph.Nodes[edge.From]
			deps = append(deps, node.Name)
		}
		fmt.Printf(" (%s)", strings.Join(deps, ", "))
	}
	fmt.Println()
}

func (e *Explorer) showPatterns(category string) {
	patterns := e.registry.Patterns(category)

	if len(patterns) == 0 {
		fmt.Println("No patterns found")
		return
	}

	fmt.Printf("Found %d patterns:\n", len(patterns))

	// Sort by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	for _, p := range patterns {
		fmt.Printf("  - %s (used %d times, confidence %.1f)\n",
			p.Name, p.Frequency, p.Confidence)

		if p.Description != "" {
			fmt.Printf("    %s\n", p.Description)
		}
	}
}
