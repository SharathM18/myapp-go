#### Main components of Backend

1. PostgresSQL DB with connection pool and index
2. Redis cache
3. OpenTelemetry with trace, metrics, or logs collector
4. Logging
5. Asynq for background tasks and Asynqmon for monitoring
6. Apache Kafka for distributed event streaming
7. Agents/ deamon to collects the logs or metrics

#### Tips

1.

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

## Install and Verify the Taskfile locally

```bash
brew install go-task/tap/go-task
task --version
```
