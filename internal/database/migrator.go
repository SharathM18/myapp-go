package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

const MigrationTimeout = 60 * time.Second

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate automatically applies pending database schema updates on startup.
func Migrate(ctx context.Context, logger *zerolog.Logger, cfg config.DatabaseConfig) error {
	// 1. Enforce a timeout budget so startup never hangs indefinitely
	ctx, cancel := context.WithTimeout(ctx, MigrationTimeout)
	defer cancel()

	// Combine host and port safely using net.JoinHostPort
	hostPort := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	// URL-encode the password to prevent crashing if the password contains special characters (like '@', '#', etc.)
	encodedPassword := url.QueryEscape(cfg.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.User,
		encodedPassword,
		hostPort,
		cfg.Name,
		cfg.SSLMode,
	)

	// 2. Open a single, exclusive connection for schema migrations
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database for migration: %w", err)
	}
	defer conn.Close(ctx)

	// 2. Initialize Tern migrator tracking the "schema_version" table
	m, err := tern.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("constructing database migrator: %w", err)
	}

	// 3. Extract the embedded SQL files
	subtree, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("retrieving database migrations subtree: %w", err)
	}

	if err := m.LoadMigrations(subtree); err != nil {
		return fmt.Errorf("loading database migrations: %w", err)
	}

	// 4. Determine current state
	from, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("retrieving current database migration version: %w", err)
	}

	// 5. Execute pending migrations
	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("executing database migrations: %w", err)
	}

	// 6. Log results
	if from == int32(len(m.Migrations)) {
		logger.Info().Msgf("Database schema is up to date (version %d)", len(m.Migrations))
	} else {
		logger.Info().Msgf("Migrated database schema from version %d to %d", from, len(m.Migrations))
	}

	return nil
}
