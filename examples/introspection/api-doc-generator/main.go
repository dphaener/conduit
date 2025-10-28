package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
	format := flag.String("format", "markdown", "Output format: markdown or html")
	title := flag.String("title", "API Documentation", "Documentation title")
	flag.Parse()

	registry := metadata.GetRegistry()

	if registry.GetSchema() == nil {
		fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
		os.Exit(1)
	}

	if *format == "html" {
		generateHTML(registry, *title)
	} else {
		generateMarkdown(registry, *title)
	}
}

func generateMarkdown(registry *metadata.RegistryAPI, title string) {
	routes := registry.Routes(metadata.RouteFilter{})
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	// Group by resource
	byResource := make(map[string][]metadata.RouteMetadata)
	for _, route := range routes {
		byResource[route.Resource] = append(byResource[route.Resource], route)
	}

	// Generate markdown
	fmt.Printf("# %s\n\n", title)
	fmt.Println("Auto-generated from Conduit introspection.\n")
	fmt.Printf("**Generated:** %s\n\n", "now")

	// Table of contents
	fmt.Println("## Table of Contents\n")
	resourceNames := make([]string, 0, len(byResource))
	for name := range byResource {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	for _, name := range resourceNames {
		fmt.Printf("- [%s](#%s)\n", name, strings.ToLower(name))
	}
	fmt.Println()

	// Document each resource
	for _, resourceName := range resourceNames {
		routes := byResource[resourceName]

		fmt.Printf("## %s\n\n", resourceName)

		// Get resource metadata
		resource, err := registry.Resource(resourceName)
		if err == nil && resource.Documentation != "" {
			fmt.Printf("%s\n\n", resource.Documentation)
		}

		// Document each route
		for _, route := range routes {
			fmt.Printf("### %s `%s`\n\n", route.Method, route.Path)
			fmt.Printf("**Operation:** `%s`\n\n", route.Operation)

			if len(route.Middleware) > 0 {
				fmt.Printf("**Middleware:** %s\n\n", strings.Join(route.Middleware, ", "))
			}

			// Describe operation
			switch route.Operation {
			case "list":
				fmt.Println("Lists all records with optional filtering and pagination.\n")
				fmt.Printf("**Response:** `200 OK` - Array of %s objects\n\n", resourceName)
			case "show":
				fmt.Println("Retrieves a single record by ID.\n")
				fmt.Println("**Parameters:**")
				fmt.Println("- `id` (path, required): Resource identifier\n")
				fmt.Printf("**Response:** `200 OK` - Single %s object\n\n", resourceName)
			case "create":
				fmt.Println("Creates a new record.\n")
				fmt.Printf("**Request Body:** %s object (see schema below)\n\n", resourceName)
				fmt.Printf("**Response:** `201 Created` - Created %s object\n\n", resourceName)
			case "update":
				fmt.Println("Updates an existing record.\n")
				fmt.Println("**Parameters:**")
				fmt.Println("- `id` (path, required): Resource identifier\n")
				fmt.Printf("**Request Body:** Partial %s object\n\n", resourceName)
				fmt.Printf("**Response:** `200 OK` - Updated %s object\n\n", resourceName)
			case "delete":
				fmt.Println("Deletes a record.\n")
				fmt.Println("**Parameters:**")
				fmt.Println("- `id` (path, required): Resource identifier\n")
				fmt.Println("**Response:** `204 No Content`\n\n")
			}

			// Show schema
			if resource != nil && (route.Operation == "create" || route.Operation == "update" || route.Operation == "show" || route.Operation == "list") {
				fmt.Printf("**%s Schema:**\n\n", resourceName)
				fmt.Println("```json")
				fmt.Println("{")
				for i, field := range resource.Fields {
					comma := ","
					if i == len(resource.Fields)-1 {
						comma = ""
					}
					required := ""
					if field.Required {
						required = " (required)"
					}
					fmt.Printf("  \"%s\": \"%s\"%s%s\n", field.Name, field.Type, required, comma)
				}
				fmt.Println("}")
				fmt.Println("```\n")
			}
		}
	}
}

func generateHTML(registry *metadata.RegistryAPI, title string) {
	fmt.Println("<!DOCTYPE html>")
	fmt.Println("<html>")
	fmt.Println("<head>")
	fmt.Printf("<title>%s</title>\n", title)
	fmt.Println("<style>")
	fmt.Println("body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }")
	fmt.Println("h1 { color: #333; }")
	fmt.Println("h2 { color: #555; border-bottom: 2px solid #ddd; padding-bottom: 10px; }")
	fmt.Println("h3 { color: #666; }")
	fmt.Println(".method { display: inline-block; padding: 4px 8px; border-radius: 3px; font-weight: bold; }")
	fmt.Println(".get { background: #61affe; color: white; }")
	fmt.Println(".post { background: #49cc90; color: white; }")
	fmt.Println(".put { background: #fca130; color: white; }")
	fmt.Println(".delete { background: #f93e3e; color: white; }")
	fmt.Println("pre { background: #f5f5f5; padding: 10px; border-radius: 4px; }")
	fmt.Println("</style>")
	fmt.Println("</head>")
	fmt.Println("<body>")
	fmt.Printf("<h1>%s</h1>\n", title)
	fmt.Println("<p>Auto-generated from Conduit introspection.</p>")

	routes := registry.Routes(metadata.RouteFilter{})
	byResource := make(map[string][]metadata.RouteMetadata)
	for _, route := range routes {
		byResource[route.Resource] = append(byResource[route.Resource], route)
	}

	for resourceName, routes := range byResource {
		fmt.Printf("<h2>%s</h2>\n", resourceName)

		for _, route := range routes {
			methodClass := strings.ToLower(route.Method)
			fmt.Printf("<h3><span class=\"method %s\">%s</span> <code>%s</code></h3>\n",
				methodClass, route.Method, route.Path)

			fmt.Printf("<p><strong>Operation:</strong> %s</p>\n", route.Operation)

			if len(route.Middleware) > 0 {
				fmt.Printf("<p><strong>Middleware:</strong> %s</p>\n",
					strings.Join(route.Middleware, ", "))
			}
		}
	}

	fmt.Println("</body>")
	fmt.Println("</html>")
}
