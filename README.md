## Architecture Overview

This backend follows clean architecture principles with clear separation of concerns:

```
backend/
├── cmd/myapp/                # Application entry point
├── internal/                 # Private application code
│   ├── config/               # Configuration management
│   ├── database/             # Database connections and migrations
│   ├── handler/              # HTTP request handlers
│   ├── service/              # Business logic layer
│   ├── repository/           # Data access layer
│   ├── model/                # Domain models
│   ├── middleware/           # HTTP middleware
│   ├── lib/                  # Shared libraries
│   └── validation/           # Request validation
├── static/                   # Static files (OpenAPI spec)
├── templates/                # Email templates
└── Taskfile.yml              # Task automation
```

## Configuration

Configuration is managed through environment variables with the `MYAPP_` prefix:

## Development

### Available Tasks

```bash
task help                    # Show all available tasks
task run                     # Run the application
task build                   # Build the application
task test                    # Run tests
task tidy                    # Format and tidy dependencies
task migrations:new name=X   # Create new migration
task migrations:up           # Apply migrations
```

# Initial Steps

## Generates the go.mod

```bash
go mod init github.com/<your-account>/myapp
```

## Install and verify the Taskfile pkg

```bash
brew install go-task/tap/go-task
task --version
```

# install deps

```
go get github.com/knadh/koanf/v2
go get github.com/knadh/koanf/providers/env/v2
go get github.com/knadh/koanf/parsers/dotenv
go get github.com/go-playground/validator/v10

go get github.com/rs/zerolog
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/otel/sdk/resource
go get go.opentelemetry.io/otel/semconv/v1.24.0


go get github.com/jackc/pgx/v5


go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/exaring/otelpgx
go get github.com/jackc/pgx-zerolog
```
