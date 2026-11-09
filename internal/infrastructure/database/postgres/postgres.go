package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

// database wraps the PostgreSQL connection pool and logger.
type Database struct {
	Pool   *pgxpool.Pool
	logger *zerolog.Logger
	config *config.Config
}

const DatabasePingTimeout = 10 * time.Second

// New is a constructor that sets up a PostgreSQL connection pool with connection pooling limits
func NewClient(cfg *config.Config, logger *zerolog.Logger) (*Database, error) {
	// 1. combine host and port safely using net.JoinHostPort
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

	// 3. parse DSN into pgxpool configuration struct
	pgxPoolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config: %w", err)
	}

	// 4. map connection pool sizing configurations from your config files
	pgxPoolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	pgxPoolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	pgxPoolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	pgxPoolConfig.MaxConnIdleTime = cfg.Database.ConnMaxIdleTime

	// 5. attach OpenTelemetry Tracer
	// This silently handles all Trace ID linking, Span creation, and error recording.
	if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.TracesEnabled {
		pgxPoolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
	}

	// 6. initialize the connection pool
	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, pgxPoolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// 7. register Metrics (Pool Health)
	if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.MetricsEnabled {
		if err := otelpgx.RecordStats(pool); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to record database pool stats: %w", err)
		}
	}

	// 8. verify Connection
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Str("host", cfg.Database.Host).
		Str("database", cfg.Database.Name).
		Msg("successfully connected to PostgreSQL database")

	return &Database{
		Pool:   pool,
		logger: logger,
		config: cfg,
	}, nil
}

// Close gracefully shuts down the database connection pool.
func (db *Database) Close() error {
	db.Pool.Close()
	db.logger.Info().
		Str("host", db.config.Database.Host).
		Str("database", db.config.Database.Name).
		Msg("database connection closed successfully")
	return nil
}
