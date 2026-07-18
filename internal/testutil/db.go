// Package testutil provides shared helpers for repository unit tests.
package testutil

import (
	"database/sql"
	"testing"

	"forum/internal/database"
)

// NewTestDB returns an in-memory SQLite database with all migrations applied,
// automatically closed when the test finishes.
//
// MaxOpenConns is pinned to 1: without this, database/sql may open a second
// connection under the hood, and since ":memory:" databases are private to
// the connection that created them, that would silently produce a second,
// empty database instead of reusing the first.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
