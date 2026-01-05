package app

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver
	dbdbdb "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // sqlite3 driver

	"github.com/device-management-toolkit/console/config"
)

const (
	_defaultAttempts     = 20
	_defaultTimeout      = time.Second
	_directoryPermission = 0o755
)

//go:embed all:migrations
var content embed.FS

var errMigrate = errors.New("migrate error")

func MigrationError(op string) error {
	return fmt.Errorf("%w: %s", errMigrate, op)
}

func Init(cfg *config.Config) error {
	databaseURL := cfg.DB.URL
	if databaseURL == "" {
		log.Printf("migrate: environment variable not declared: DB_URL -- using embedded database")
	}

	migrationsSource, err := iofs.New(content, "migrations")
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasPrefix(databaseURL, "postgres://") {
		err := setupHostedDB(migrationsSource, databaseURL)
		if err != nil {
			return err
		}
	} else {
		// make sure the directory exists
		err := setupLocalDB(migrationsSource)
		if err != nil {
			return err
		}
	}

	return nil
}

func setupLocalDB(migrationsSource source.Driver) error {
	dirname, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	consoleDir := filepath.Join(dirname, "device-management-toolkit")

	if _, err = os.Stat(consoleDir); os.IsNotExist(err) {
		if err1 := os.Mkdir(consoleDir, _directoryPermission); err1 != nil {
			return fmt.Errorf("failed to create config directory: %w", err1)
		}
	}

	log.Printf("DB path : %s\n", filepath.Join(consoleDir, "console.db"))

	db, err := sql.Open("sqlite", filepath.Join(consoleDir, "console.db"))
	if err != nil {
		return err
	}

	defer func() {
		if err1 := db.Close(); err1 != nil {
			log.Printf("error closing local db: %v", err1)
		}
	}()

	driver, err := dbdbdb.WithInstance(db, &dbdbdb.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", migrationsSource, "console", driver)
	if err != nil {
		return err
	}
	defer m.Close()

	versions, latestVersion, err := collectMigrationVersions(migrationsSource)
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	currentVersion, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		currentVersion = 0
		err = nil
	}
	if err != nil {
		return fmt.Errorf("failed to read current migration version: %w", err)
	}

	skipMigrations := false
	if !dirty {
		if currentVersion > latestVersion {
			log.Printf("Database version %d is newer than bundled migrations (%d); skipping migration (likely downgrade).", currentVersion, latestVersion)
			skipMigrations = true
		} else if currentVersion > 0 {
			if _, ok := versions[currentVersion]; !ok {
				log.Printf("Database version %d not found in bundled migrations (latest %d); skipping migration to avoid downgrading.", currentVersion, latestVersion)
				skipMigrations = true
			}
		}
	}

	if !skipMigrations {
		err = m.Up()
		if err != nil {
			var dirtyErr migrate.ErrDirty
			if errors.As(err, &dirtyErr) {
				log.Printf("Dirty database version found. Forcing to previous version and retrying.")

				if dirtyErr.Version > 0 {
					if forceErr := m.Force(dirtyErr.Version - 1); forceErr != nil {
						return fmt.Errorf("failed to force database version: %w", forceErr)
					}
					if upErr := m.Up(); upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
						return fmt.Errorf("failed to migrate up after forcing: %w", upErr)
					}
				} else {
					return err
				}
			} else if !errors.Is(err, migrate.ErrNoChange) {
				return err
			}
		}
	}

	_, err = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return nil
}

func setupHostedDB(migrationsSource source.Driver, databaseURL string) error {
	databaseURL += "?sslmode=disable"

	var (
		attempts = _defaultAttempts
		err      error
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.NewWithSourceInstance("iofs", migrationsSource, databaseURL)
		if err == nil {
			break
		}

		log.Printf("Migrate: postgres is trying to connect, attempts left: %d", attempts)
		time.Sleep(_defaultTimeout)

		attempts--
	}

	if err != nil {
		return MigrationError(fmt.Sprintf("postgres connect error: %s", err))
	}
	defer m.Close()

	versions, latestVersion, err := collectMigrationVersions(migrationsSource)
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	currentVersion, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		currentVersion = 0
		err = nil
	}
	if err != nil {
		return fmt.Errorf("failed to read current migration version: %w", err)
	}

	skipMigrations := false
	if !dirty {
		if currentVersion > latestVersion {
			log.Printf("Database version %d is newer than bundled migrations (%d); skipping migration (likely downgrade).", currentVersion, latestVersion)
			skipMigrations = true
		} else if currentVersion > 0 {
			if _, ok := versions[currentVersion]; !ok {
				log.Printf("Database version %d not found in bundled migrations (latest %d); skipping migration to avoid downgrading.", currentVersion, latestVersion)
				skipMigrations = true
			}
		}
	}

	if !skipMigrations {
		err = m.Up()
		if err != nil {
			var dirtyErr migrate.ErrDirty
			if errors.As(err, &dirtyErr) {
				log.Printf("Dirty database version found. Forcing to previous version and retrying.")

				if dirtyErr.Version > 0 {
					if forceErr := m.Force(dirtyErr.Version - 1); forceErr != nil {
						return fmt.Errorf("failed to force database version: %w", forceErr)
					}
					if upErr := m.Up(); upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
						return fmt.Errorf("failed to migrate up after forcing: %w", upErr)
					}
				} else {
					return err
				}
			} else if !errors.Is(err, migrate.ErrNoChange) {
				return MigrationError(fmt.Sprintf("up error: %s", err))
			}
		}
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Printf("Migrate: no change")

		return nil
	}

	log.Printf("Migrate: up success")

	return nil
}

func collectMigrationVersions(migrationsSource source.Driver) (map[uint]struct{}, uint, error) {
	first, err := migrationsSource.First()
	if err != nil {
		return nil, 0, err
	}

	versions := map[uint]struct{}{first: {}}
	latest := first

	for {
		next, err := migrationsSource.Next(latest)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return versions, latest, nil
			}

			return nil, 0, err
		}

		versions[next] = struct{}{}
		latest = next
	}
}
