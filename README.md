### Main components of Backend

- Config management
- Application server startup and shutdown gracefully
- PostgresSQL DB with Connection pool, index, and migrations
- Redis cache (Serialization using MessagePack (msgpack))
- OpenTelemetry with trace, metrics, or logs collector
- Logging
- Asynq for background tasksqueue and Asynqmon for monitoring
- Apache Kafka for distributed event streaming
- Agents/ deamon to collects the logs or metrics

## Configuration

Configuration is managed through environment variables with the `MYAPP_` prefix:

## Development

### Install and Verify the Taskfile locally

```bash
brew install go-task/tap/go-task
task --version
```

### Available Tasks

```bash
$ task help

task: Available tasks for this project:
* build:                    Build both App and Worker binaries
* build-app:                Build the APP binary
* build-worker:             Build the Worker binary
* help:                     Display available tasks
* run-app:                  Run the APP Server
* run-worker:               Run the Worker Server
* test:                     Run all tests
* tidy:                     Format all .go files, and Tidy and verify Go module dependencies
* migrations:down:          Rollback the last migration
* migrations:new:           Create a new database migration file
* migrations:up:            Apply all pending up database migrations
* migrations:version:       Check current database migration version
```

### Go Commands

| Commands                                      | Description                                                                                            |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `go mod init github.com/<your-account>/myapp` | Generates the go.mod                                                                                   |
| `go mod tidy`                                 | Downloads missing dependencies and cleans up unused entries in go.mod and go.sum                       |
| `go fmt ./...`                                | Automatically formats all Go files in your project according to official Go style guidelines.          |
| `go vet ./...`                                | Examines your code and reports suspicious constructs (like printf formatting mistakes or mutex locks). |
| `go run cmd/api/main.go`                      | Compiles your code into a temporary directory in memory and executes it immediately.                   |
| `go build -o bin/api cmd/api/main.go`         | Compiles your source code into a standalone executable binary saved at bin/api.                        |
| `go clean -modcache`                          | Delete all Go modules downloaded on your entire machine                                                |

pending
err
sqlerr
validator

|                                                                           |                                                         |
| ------------------------------------------------------------------------- | ------------------------------------------------------- |
| `curl -i http://localhost:9090/health/live`                               | Liveness Probe (Checks if the Go binary is running)     |
| `curl -i http://localhost:9090/health/ready`                              | Readiness Probe (Checks Postgres and Redis connections) |
| `curl -s http://localhost:9090/info`                                      | Build Info                                              |
| `curl -u admin:secret -s http://localhost:9090/admin/metrics `            | Prometheus Metrics Scraper                              |
| `curl -u admin:secret -s http://localhost:9090/admin/asynqmon/api/queues` | Asynq Dashboard (API Check)                             |
| `curl -i -u admin:secret http://localhost:9090/admin/debug/pprof/`        | Go Profiler (pprof)                                     |
