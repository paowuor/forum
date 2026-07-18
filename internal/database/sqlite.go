package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

// migrationFiles embeds every .sql file in the migrations directory into the
// compiled binary, so the app doesn't depend on the filesystem layout at
// runtime (important once this runs inside a Docker container).
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Open creates (or opens, if it already exists) the SQLite database at the
// given path, enables foreign key enforcement, and applies all migrations.
func Open(dbPath string) (*sql.DB, error) {
	// The SQLite driver can't create a missing parent directory on its own —
	// it'll fail with "unable to open database file". data/ ships empty (the
	// .db file itself is gitignored and created at runtime), and an empty
	// directory isn't guaranteed to survive every packaging/VCS path, so
	// create it explicitly rather than relying on it already being there.
	// dbPath == ":memory:" (used by tests) has no real directory component;
	// filepath.Dir returns "." in that case, and MkdirAll(".", ...) is a
	// harmless no-op.
	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}

	// _busy_timeout tells SQLite to retry for up to 5s instead of failing
	// immediately when another connection holds the write lock.
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite allows only one writer at a time. Rather than juggling that
	// across a pool of connections (and hitting "database is locked" under
	// concurrent requests even with a busy_timeout), we serialize all
	// access through a single connection. For a forum-scale app this is
	// not a real bottleneck, and it removes an entire class of concurrency
	// bugs outright.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// runMigrations executes every embedded .sql file in filename order.
// Each file uses "CREATE TABLE IF NOT EXISTS", so this is safe to run
// every time the server starts.
func runMigrations(db *sql.DB) error {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // filenames are prefixed 001_, 002_, ... so this orders them correctly

	for _, name := range names {
		contents, err := migrationFiles.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		if _, err := db.Exec(string(contents)); err != nil {
			return fmt.Errorf("executing migration %s: %w", name, err)
		}
	}

	return nil
}
