package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/observability/logging"
)

// Database wraps the PostgreSQL connection pool and logger.
type Database struct {
	Pool   *pgxpool.Pool
	logger *zerolog.Logger
}

// multiTracer allows chaining multiple pgx tracers together.
// This enables us to run OpenTelemetry distributed tracing and Zerolog SQL logging simultaneously.
type multiTracer struct {
	tracers []any
}

func (mt *multiTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryStart(context.Context, *pgx.Conn, pgx.TraceQueryStartData) context.Context
		}); ok {
			ctx = t.TraceQueryStart(ctx, conn, data)
		}
	}
	return ctx
}

func (mt *multiTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData)
		}); ok {
			t.TraceQueryEnd(ctx, conn, data)
		}
	}
}

const DatabasePingTimeout = 10 * time.Second

// New is a constructor that sets up a PostgreSQL connection pool with connection pooling limits,
// OpenTelemetry distributed tracing, and Zerolog query logging.
func New(cfg *config.Config, logger *zerolog.Logger) (*Database, error) {
	// 1. Combine host and port safely using net.JoinHostPort
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	// 2. URL-encode the password to prevent crashing if the password contains special characters (like '@', '#', etc.)
	encodedPassword := url.QueryEscape(cfg.Database.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		encodedPassword,
		hostPort,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// 3. Parse DSN into pgxpool configuration struct
	pgxPoolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config: %w", err)
	}

	// 4. Map Connection Pool sizing configurations from your config files
	pgxPoolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	pgxPoolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	pgxPoolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	pgxPoolConfig.MaxConnIdleTime = cfg.Database.ConnMaxIdleTime

	// 5. Setup OpenTelemetry Tracer
	// otelpgx automatically references Trace ID, Parent Span ID, and creates a new Span ID, and also add label automatically
	// otelpgx automatically creates child spans (SpanKindClient) named after the operation (e.g., "SELECT users") and attaches standard OpenTelemetry attributes (db.system, db.name, db.statement, db.user, net.peer.name, net.peer.port) to the active context.
	otelTracer := otelpgx.NewTracer()

	// 6. Setup Zerolog SQL query tracer
	// Intercepts pgx driver connection events, ping failures, and individual query logs.
	// Passes them into your custom PgxLogger, which stamps them with the active trace_id and formats them into JSON.
	// Handled by Zerolog. If LogsEnabled is true, Zerolog ships them over OTLP. Otherwise, they print to stdout for Fluentbit to pick up.
	pgxLogger := logging.NewPgxLogger(logger, zerolog.DebugLevel)
	localTracer := &tracelog.TraceLog{
		Logger:   pgxLogger,
		LogLevel: logging.GetZerologLevelToPgx(zerolog.InfoLevel),
	}

	// 7. Chain both tracers together so queries emit both OTel spans and Zerolog log entries
	pgxPoolConfig.ConnConfig.Tracer = &multiTracer{
		tracers: []any{otelTracer, localTracer},
	}

	// 8. Initialize connection pool with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, pgxPoolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// 9. Register Metrics ONLY if enabled
	// Captures the physical health of your PostgreSQL connection pool.
	// Exports gauges and counters: pgx.pool.max_conns, pgx.pool.acquired_conns, pgx.pool.idle_conns, and pgx.pool.wait_duration.
	// Sends them continuously in the background to your OTel Collector so you can build Grafana dashboards showing if your database pool is choking.
	if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.MetricsEnabled {
		if err := otelpgx.RecordStats(pool); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to record database pool stats: %w", err)
		}
	}

	// 10. Perform a fail-fast health check (Ping)
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Str("host", cfg.Database.Host).
		Str("database", cfg.Database.Name).
		Msg("Successfully connected to PostgreSQL database and verified pool health")

	return &Database{
		Pool:   pool,
		logger: logger,
	}, nil
}

// Close gracefully shuts down the database connection pool.
func (db *Database) Close() error {
	db.logger.Info().Msg("Closing PostgreSQL connection pool gracefully...")
	db.Pool.Close()
	return nil
}
