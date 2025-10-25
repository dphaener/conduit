package templates

// NewMicroserviceTemplate creates the official microservice template
func NewMicroserviceTemplate() *Template {
	return &Template{
		Name:        "microservice",
		Description: "Microservice with event-driven architecture and message queues",
		Version:     "1.0.0",
		Variables: []*TemplateVariable{
			{
				Name:        "project_name",
				Description: "Name of your microservice",
				Type:        VariableTypeString,
				Required:    true,
				Prompt:      "Microservice name",
			},
			{
				Name:        "port",
				Description: "Server port",
				Type:        VariableTypeInt,
				Default:     8080,
				Prompt:      "Server port",
			},
			{
				Name:        "service_type",
				Description: "Type of microservice",
				Type:        VariableTypeSelect,
				Options:     []string{"api-gateway", "data-service", "worker-service"},
				Default:     "data-service",
				Prompt:      "Service type",
			},
			{
				Name:        "include_metrics",
				Description: "Include Prometheus metrics",
				Type:        VariableTypeConfirm,
				Default:     true,
				Prompt:      "Include metrics?",
			},
			{
				Name:        "include_tracing",
				Description: "Include distributed tracing",
				Type:        VariableTypeConfirm,
				Default:     true,
				Prompt:      "Include tracing?",
			},
			{
				Name:        "database_url",
				Description: "Database connection URL",
				Type:        VariableTypeString,
				Default:     "",
				Prompt:      "Database URL (optional)",
			},
		},
		Directories: []string{
			"app",
			"app/resources",
			"app/events",
			"migrations",
			"build",
			"config",
			"scripts",
		},
		Files: []*TemplateFile{
			{
				TargetPath: "app/resources/event.cdt",
				Template:   true,
				Content: `/// Event resource for event sourcing
resource Event {
  id: uuid! @primary @auto
  aggregate_id: uuid!
  aggregate_type: string! @max(100)
  event_type: string! @max(100)
  payload: json!
  version: int! @default(1)
  created_at: timestamp! @auto

  @constraint valid_aggregate_type {
    on: [create]
    condition: String.length(self.aggregate_type) > 0
    error: "Aggregate type is required"
  }
}
`,
			},
			{
				TargetPath: "app/resources/entity.cdt",
				Template:   true,
				Content: `/// Entity resource for microservice data
resource Entity {
  id: uuid! @primary @auto
  name: string! @min(2) @max(100)
  status: string! @default("active")
  metadata: json?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @constraint valid_status {
    on: [create, update]
    condition: self.status == "active" || self.status == "inactive" || self.status == "archived"
    error: "Status must be active, inactive, or archived"
  }

  @after create @async {
    // Publish entity created event
    Event.publish("entity.created", {
      entity_id: self.id,
      name: self.name,
      timestamp: Time.now()
    })
  }

  @after update @async {
    // Publish entity updated event
    Event.publish("entity.updated", {
      entity_id: self.id,
      name: self.name,
      timestamp: Time.now()
    })
  }
}
`,
			},
			{
				TargetPath: "conduit.yaml",
				Template:   true,
				Content: `# Conduit Microservice Configuration
project:
  name: {{.ProjectName}}
  version: 1.0.0
  type: microservice

server:
  port: {{.Variables.port}}
  host: "0.0.0.0"
  read_timeout: 30
  write_timeout: 30
  graceful_shutdown: 10

database:
  driver: postgresql
  {{- if .Variables.database_url}}
  url: "{{.Variables.database_url}}"
  {{- else}}
  # Set via DATABASE_URL environment variable
  {{- end}}
  pool_size: 20
  max_idle_time: 300
  max_lifetime: 3600

logging:
  level: info
  format: json
  output: stdout

{{- if .Variables.include_metrics}}
metrics:
  enabled: true
  port: 9090
  path: /metrics
{{- end}}

{{- if .Variables.include_tracing}}
tracing:
  enabled: true
  service_name: {{.ProjectName}}
  endpoint: "http://localhost:14268/api/traces"
{{- end}}

messaging:
  driver: nats
  url: "nats://localhost:4222"
  max_reconnects: 10

health:
  enabled: true
  path: /health
  readiness_path: /ready

rate_limiting:
  enabled: true
  requests_per_second: 100
  burst: 200
`,
			},
			{
				TargetPath: "scripts/docker-compose.yml",
				Template:   true,
				Content: `version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: {{.ProjectName}}
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"

{{- if .Variables.include_tracing}}
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "5775:5775/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"
      - "14268:14268"
      - "14250:14250"
      - "9411:9411"
{{- end}}

{{- if .Variables.include_metrics}}
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
{{- end}}

volumes:
  postgres_data:
{{- if .Variables.include_metrics}}
  prometheus_data:
{{- end}}
`,
			},
			{
				TargetPath: "scripts/prometheus.yml",
				Template:   true,
				Condition:  "{{.Variables.include_metrics}}",
				Content: `global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: '{{.ProjectName}}'
    static_configs:
      - targets: ['host.docker.internal:9090']
`,
			},
			{
				TargetPath: ".gitignore",
				Template:   false,
				Content: `# Build output
build/
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test
*.out

# Dependencies
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Database
*.db
*.sqlite

# Docker
.docker/
`,
			},
			{
				TargetPath: ".env.example",
				Template:   true,
				Content: `# Database
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/{{.ProjectName}}

# Server
PORT={{.Variables.port}}

# Messaging
NATS_URL=nats://localhost:4222

# Application
LOG_LEVEL=info
SERVICE_NAME={{.ProjectName}}

{{- if .Variables.include_tracing}}
# Tracing
JAEGER_ENDPOINT=http://localhost:14268/api/traces
{{- end}}

{{- if .Variables.include_metrics}}
# Metrics
METRICS_PORT=9090
{{- end}}
`,
			},
			{
				TargetPath: "README.md",
				Template:   true,
				Content: `# {{.ProjectName}}

A Conduit microservice.

## Service Type

{{.Variables.service_type}}

## Features

- Event-driven architecture
- Asynchronous event publishing
{{- if .Variables.include_metrics}}
- Prometheus metrics
{{- end}}
{{- if .Variables.include_tracing}}
- Distributed tracing with Jaeger
{{- end}}
- Health checks
- Rate limiting
- Graceful shutdown

## Getting Started

### Prerequisites

- Conduit CLI installed
- Docker and Docker Compose (for dependencies)

### Setup

1. Start infrastructure services:
   ` + "```bash" + `
   docker-compose -f scripts/docker-compose.yml up -d
   ` + "```" + `

2. Set up environment:
   ` + "```bash" + `
   cp .env.example .env
   # Edit .env with your configuration
   ` + "```" + `

3. Run migrations:
   ` + "```bash" + `
   conduit migrate up
   ` + "```" + `

4. Build and run the service:
   ` + "```bash" + `
   conduit run
   ` + "```" + `

## API Endpoints

- ` + "`GET /health`" + ` - Health check
- ` + "`GET /ready`" + ` - Readiness check
{{- if .Variables.include_metrics}}
- ` + "`GET /metrics`" + ` - Prometheus metrics
{{- end}}
- ` + "`GET /api/entities`" + ` - List entities
- ` + "`POST /api/entities`" + ` - Create entity
- ` + "`GET /api/events`" + ` - List events

## Architecture

This microservice follows event-driven architecture patterns:

1. **Entities** - Core domain objects
2. **Events** - Event sourcing for audit trail
3. **Async Processing** - Background event publishing

## Monitoring
{{- if .Variables.include_metrics}}
### Metrics

Prometheus metrics available at http://localhost:9090/metrics

Common metrics:
- ` + "`http_requests_total`" + ` - Total HTTP requests
- ` + "`http_request_duration_seconds`" + ` - Request latency
- ` + "`database_queries_total`" + ` - Total database queries
{{- end}}
{{- if .Variables.include_tracing}}
### Tracing

Jaeger UI available at http://localhost:16686

All requests are traced with correlation IDs.
{{- end}}
## Development

Run tests:
` + "```bash" + `
conduit test
` + "```" + `

Run with watch mode:
` + "```bash" + `
conduit watch
` + "```" + `

## Deployment

Build production binary:
` + "```bash" + `
conduit build --release
` + "```" + `

The service can be deployed as:
- Docker container
- Kubernetes pod
- Standalone binary

## Documentation

Learn more at https://conduit-lang.org/docs
`,
			},
		},
		Hooks: &TemplateHooks{
			AfterCreate: []string{
				"echo 'Microservice created successfully!'",
				"echo 'Next steps:'",
				"echo '  1. Start infrastructure: docker-compose -f scripts/docker-compose.yml up -d'",
				"echo '  2. Copy .env.example to .env and configure'",
				"echo '  3. Run: conduit migrate up'",
				"echo '  4. Run: conduit run'",
			},
		},
		Metadata: map[string]interface{}{
			"category": "backend",
			"tags":     []string{"microservice", "events", "distributed"},
		},
	}
}
