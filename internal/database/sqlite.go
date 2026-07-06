package database

import (
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

// migrationFiles embeds every .sql file in the migrations directory into the
// compiled binary, so the app doesn't depend on the filesystem layout at
// runtime.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Open creates (or opens, if it already exists) the SQLite database at the
// given path, enables foreign key enforcement, and applies all migrations.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

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