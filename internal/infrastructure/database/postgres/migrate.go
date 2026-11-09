package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	iofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

const MigrationTimeout = 60 * time.Second

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrate automatically applies pending database schema updates using golang-migrate.
func Migrate(ctx context.Context, logger *zerolog.Logger, cfg config.DatabaseConfig) error {
	// 1. enforce a timeout budget so startup never hangs indefinitely
	ctx, cancel := context.WithTimeout(ctx, MigrationTimeout)
	defer cancel()

	// 2. build standard PostgreSQL DSN
	hostPort := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	encodedPassword := url.QueryEscape(cfg.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.User,
		encodedPassword,
		hostPort,
		cfg.Name,
		cfg.SSLMode,
	)

	// 3. wrap embedded migration files into an iofs source driver
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create embedded migration source driver: %w", err)
	}

	// 4. initialize golang-migrate instance pointing to our PostgreSQL DSN
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize golang-migrate instance: %w", err)
	}
	defer m.Close()

	// 5. check version before migration
	versionBefore, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to retrieve current migration version: %w", err)
	}
	if dirty {
		return errors.New("database schema is in a dirty state; manual intervention required")
	}

	// 6. execute pending migrations (Up)
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info().Msg("Database schema is already up to date")
			return nil
		}
		return fmt.Errorf("failed to execute database migrations: %w", err)
	}

	// 7. check version after migration for logging
	versionAfter, _, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to retrieve updated migration version: %w", err)
	}

	if versionBefore == versionAfter {
		logger.Info().Msg("Database schema is up to date")
	} else {
		logger.Info().
			Uint("from_version", versionBefore).
			Uint("to_version", versionAfter).
			Msg("database schema migrated successfully")
	}

	return nil
}
