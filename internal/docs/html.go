package docs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// HTMLGenerator generates interactive HTML documentation
type HTMLGenerator struct {
	config    *Config
	templates *template.Template
}

// NewHTMLGenerator creates a new HTML generator
func NewHTMLGenerator(config *Config) *HTMLGenerator {
	return &HTMLGenerator{
		config: config,
	}
}

// Generate generates HTML documentation
func (g *HTMLGenerator) Generate(doc *Documentation) error {
	outputDir := filepath.Join(g.config.OutputDir, "html")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Load templates
	if err := g.loadTemplates(); err != nil {
		return err
	}

	// Generate index page
	if err := g.generateIndex(doc, outputDir); err != nil {
		return err
	}

	// Generate resource pages
	for _, resource := range doc.Resources {
		if err := g.generateResourcePage(resource, outputDir); err != nil {
			return err
		}
	}

	// Copy static assets
	if err := g.copyStaticAssets(outputDir); err != nil {
		return err
	}

	return nil
}

// loadTemplates loads HTML templates
func (g *HTMLGenerator) loadTemplates() error {
	funcMap := template.FuncMap{
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"json":       g.toJSON,
		"jsonPretty": g.toJSONPretty,
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse embedded templates
	var err error
	tmpl, err = tmpl.Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	tmpl, err = tmpl.Parse(resourceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse resource template: %w", err)
	}

	g.templates = tmpl
	return nil
}

// generateIndex generates the index.html page
func (g *HTMLGenerator) generateIndex(doc *Documentation, outputDir string) error {
	data := map[string]interface{}{
		"ProjectInfo": doc.ProjectInfo,
		"Resources":   doc.Resources,
		"BaseURL":     g.config.BaseURL,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "index", data); err != nil {
		return fmt.Errorf("failed to execute index template: %w", err)
	}

	outputPath := filepath.Join(outputDir, "index.html")
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// generateResourcePage generates a resource detail page
func (g *HTMLGenerator) generateResourcePage(resource *ResourceDoc, outputDir string) error {
	data := map[string]interface{}{
		"Resource": resource,
		"BaseURL":  g.config.BaseURL,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "resource", data); err != nil {
		return fmt.Errorf("failed to execute resource template: %w", err)
	}

	filename := strings.ToLower(resource.Name) + ".html"
	outputPath := filepath.Join(outputDir, filename)
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// copyStaticAssets copies CSS and JS files
func (g *HTMLGenerator) copyStaticAssets(outputDir string) error {
	// Write CSS
	cssPath := filepath.Join(outputDir, "styles.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("failed to write CSS: %w", err)
	}

	// Write JS
	jsPath := filepath.Join(outputDir, "script.js")
	if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("failed to write JS: %w", err)
	}

	return nil
}

// Helper functions

func (g *HTMLGenerator) toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func (g *HTMLGenerator) toJSONPretty(v interface{}) template.HTML {
	data, _ := json.MarshalIndent(v, "", "  ")
	return template.HTML(data)
}

// Template definitions

const indexTemplate = `{{define "index"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.ProjectInfo.Name}} - API Documentation</title>
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <div class="container">
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2>{{.ProjectInfo.Name}}</h2>
                <p class="version">v{{.ProjectInfo.Version}}</p>
            </div>
            <div class="nav-section">
                <h3>Resources</h3>
                <ul class="nav-list">
                    {{range .Resources}}
                    <li><a href="{{lower .Name}}.html">{{.Name}}</a></li>
                    {{end}}
                </ul>
            </div>
        </nav>
        <main class="content">
<div class="page-header">
    <h1>{{.ProjectInfo.Name}} API Documentation</h1>
    <p class="description">{{.ProjectInfo.Description}}</p>
</div>

<div class="section">
    <h2>Getting Started</h2>
    <p>Welcome to the {{.ProjectInfo.Name}} API documentation. This API provides access to all resources in your application.</p>

    <h3>Base URL</h3>
    <pre><code>{{if .BaseURL}}{{.BaseURL}}{{else}}http://localhost:3000{{end}}</code></pre>
</div>

<div class="section">
    <h2>Resources</h2>
    <div class="resource-grid">
        {{range .Resources}}
        <div class="resource-card">
            <h3><a href="{{lower .Name}}.html">{{.Name}}</a></h3>
            <p>{{.Documentation}}</p>
            <div class="resource-stats">
                <span>{{len .Fields}} fields</span>
                <span>{{len .Endpoints}} endpoints</span>
            </div>
        </div>
        {{end}}
    </div>
</div>

<div class="section">
    <h2>Search</h2>
    <input type="text" id="search" class="search-input" placeholder="Search resources and endpoints...">
    <div id="search-results"></div>
</div>
        </main>
    </div>
    <script src="script.js"></script>
</body>
</html>
{{end}}`

const resourceTemplate = `{{define "resource"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Resource.Name}} - API Documentation</title>
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <div class="container">
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2>API Documentation</h2>
            </div>
            <div class="nav-section">
                <h3>Resources</h3>
            </div>
        </nav>
        <main class="content">
<div class="page-header">
    <h1>{{.Resource.Name}}</h1>
    {{if .Resource.Documentation}}
    <p class="description">{{.Resource.Documentation}}</p>
    {{end}}
</div>

<div class="section">
    <h2>Fields</h2>
    <table class="fields-table">
        <thead>
            <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Required</th>
                <th>Constraints</th>
                <th>Example</th>
            </tr>
        </thead>
        <tbody>
            {{range .Resource.Fields}}
            <tr>
                <td><code>{{.Name}}</code></td>
                <td><code>{{.Type}}</code></td>
                <td>{{if .Required}}Yes{{else}}No{{end}}</td>
                <td>{{range .Constraints}}<span class="constraint">{{.}}</span>{{end}}</td>
                <td><code>{{jsonPretty .Example}}</code></td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

{{if .Resource.Relationships}}
<div class="section">
    <h2>Relationships</h2>
    {{range .Resource.Relationships}}
    <div class="relationship">
        <h3>{{.Name}}</h3>
        <ul>
            <li><strong>Type:</strong> <code>{{.Type}}</code></li>
            <li><strong>Kind:</strong> <code>{{.Kind}}</code></li>
            <li><strong>Foreign Key:</strong> <code>{{.ForeignKey}}</code></li>
        </ul>
    </div>
    {{end}}
</div>
{{end}}

<div class="section">
    <h2>Endpoints</h2>
    {{range .Resource.Endpoints}}
    <div class="endpoint">
        <div class="endpoint-header">
            <span class="method method-{{lower .Method}}">{{.Method}}</span>
            <span class="path">{{.Path}}</span>
        </div>
        <p>{{.Summary}}</p>

        {{if .Parameters}}
        <h4>Parameters</h4>
        <table class="params-table">
            <thead>
                <tr>
                    <th>Name</th>
                    <th>In</th>
                    <th>Type</th>
                    <th>Required</th>
                    <th>Description</th>
                </tr>
            </thead>
            <tbody>
                {{range .Parameters}}
                <tr>
                    <td><code>{{.Name}}</code></td>
                    <td>{{.In}}</td>
                    <td><code>{{.Type}}</code></td>
                    <td>{{if .Required}}Yes{{else}}No{{end}}</td>
                    <td>{{.Description}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{end}}

        {{if .RequestBody}}
        <h4>Request Body</h4>
        <pre><code class="json">{{jsonPretty .RequestBody.Example}}</code></pre>
        {{end}}

        <h4>Responses</h4>
        {{range $code, $response := .Responses}}
        <div class="response">
            <strong>{{$code}} {{$response.Description}}</strong>
            {{if $response.Example}}
            <pre><code class="json">{{jsonPretty $response.Example}}</code></pre>
            {{end}}
        </div>
        {{end}}
    </div>
    {{end}}
</div>

{{if .Resource.Hooks}}
<div class="section">
    <h2>Lifecycle Hooks</h2>
    <table class="hooks-table">
        <thead>
            <tr>
                <th>Timing</th>
                <th>Event</th>
                <th>Async</th>
                <th>Transaction</th>
            </tr>
        </thead>
        <tbody>
            {{range .Resource.Hooks}}
            <tr>
                <td>{{.Timing}}</td>
                <td>{{.Event}}</td>
                <td>{{if .IsAsync}}Yes{{else}}No{{end}}</td>
                <td>{{if .IsTransaction}}Yes{{else}}No{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}

{{if .Resource.Validations}}
<div class="section">
    <h2>Validations</h2>
    {{range .Resource.Validations}}
    <div class="validation">
        <h3>{{.Name}}</h3>
        <p><strong>Error:</strong> {{.ErrorMessage}}</p>
    </div>
    {{end}}
</div>
{{end}}
        </main>
    </div>
    <script src="script.js"></script>
</body>
</html>
{{end}}`

const cssContent = `
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    line-height: 1.6;
    color: #333;
    background: #f5f5f5;
}

.container {
    display: flex;
    min-height: 100vh;
}

.sidebar {
    width: 250px;
    background: #2c3e50;
    color: #ecf0f1;
    padding: 20px;
    position: fixed;
    height: 100vh;
    overflow-y: auto;
}

.sidebar-header {
    margin-bottom: 30px;
    border-bottom: 1px solid #34495e;
    padding-bottom: 15px;
}

.sidebar-header h2 {
    font-size: 20px;
    margin-bottom: 5px;
}

.version {
    font-size: 12px;
    color: #95a5a6;
}

.nav-section h3 {
    font-size: 14px;
    text-transform: uppercase;
    color: #95a5a6;
    margin-bottom: 10px;
}

.nav-list {
    list-style: none;
}

.nav-list li {
    margin-bottom: 5px;
}

.nav-list a {
    color: #ecf0f1;
    text-decoration: none;
    display: block;
    padding: 8px 12px;
    border-radius: 4px;
    transition: background 0.2s;
}

.nav-list a:hover {
    background: #34495e;
}

.content {
    margin-left: 250px;
    padding: 40px;
    flex: 1;
    background: white;
}

.page-header {
    margin-bottom: 40px;
    border-bottom: 2px solid #3498db;
    padding-bottom: 20px;
}

.page-header h1 {
    font-size: 32px;
    color: #2c3e50;
    margin-bottom: 10px;
}

.description {
    font-size: 16px;
    color: #7f8c8d;
}

.section {
    margin-bottom: 40px;
}

.section h2 {
    font-size: 24px;
    color: #2c3e50;
    margin-bottom: 20px;
    border-bottom: 1px solid #ecf0f1;
    padding-bottom: 10px;
}

.section h3 {
    font-size: 18px;
    color: #34495e;
    margin-bottom: 10px;
}

.resource-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
    margin-top: 20px;
}

.resource-card {
    background: #f8f9fa;
    border: 1px solid #dee2e6;
    border-radius: 8px;
    padding: 20px;
    transition: box-shadow 0.2s;
}

.resource-card:hover {
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.resource-card h3 a {
    color: #3498db;
    text-decoration: none;
}

.resource-stats {
    margin-top: 15px;
    font-size: 14px;
    color: #7f8c8d;
}

.resource-stats span {
    margin-right: 15px;
}

.fields-table, .params-table, .hooks-table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 15px;
}

.fields-table th, .params-table th, .hooks-table th {
    background: #ecf0f1;
    padding: 12px;
    text-align: left;
    font-weight: 600;
    border-bottom: 2px solid #bdc3c7;
}

.fields-table td, .params-table td, .hooks-table td {
    padding: 12px;
    border-bottom: 1px solid #ecf0f1;
}

.constraint {
    display: inline-block;
    background: #3498db;
    color: white;
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 12px;
    margin-right: 5px;
}

.endpoint {
    background: #f8f9fa;
    border: 1px solid #dee2e6;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 20px;
}

.endpoint-header {
    margin-bottom: 10px;
}

.method {
    display: inline-block;
    padding: 4px 12px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 14px;
    margin-right: 10px;
}

.method-get { background: #27ae60; color: white; }
.method-post { background: #3498db; color: white; }
.method-put { background: #f39c12; color: white; }
.method-delete { background: #e74c3c; color: white; }
.method-patch { background: #9b59b6; color: white; }

.path {
    font-family: 'Courier New', monospace;
    color: #34495e;
    font-size: 16px;
}

pre {
    background: #2c3e50;
    color: #ecf0f1;
    padding: 15px;
    border-radius: 4px;
    overflow-x: auto;
    margin: 10px 0;
}

code {
    font-family: 'Courier New', monospace;
    font-size: 14px;
}

.search-input {
    width: 100%;
    padding: 12px 20px;
    font-size: 16px;
    border: 2px solid #bdc3c7;
    border-radius: 8px;
    margin-top: 10px;
}

.search-input:focus {
    outline: none;
    border-color: #3498db;
}

#search-results {
    margin-top: 20px;
}
`

const jsContent = `
// Simple search functionality
document.addEventListener('DOMContentLoaded', function() {
    const searchInput = document.getElementById('search');
    if (!searchInput) return;

    searchInput.addEventListener('input', function(e) {
        const query = e.target.value.toLowerCase();
        const results = document.getElementById('search-results');

        if (query.length < 2) {
            results.innerHTML = '';
            return;
        }

        // Simple search through navigation items
        const navLinks = document.querySelectorAll('.nav-list a');
        const matches = [];

        navLinks.forEach(link => {
            if (link.textContent.toLowerCase().includes(query)) {
                matches.push({
                    text: link.textContent,
                    href: link.getAttribute('href')
                });
            }
        });

        if (matches.length > 0) {
            results.innerHTML = '<div class="search-matches">' +
                matches.map(m => '<div><a href="' + m.href + '">' + m.text + '</a></div>').join('') +
                '</div>';
        } else {
            results.innerHTML = '<p>No results found</p>';
        }
    });
});
`
