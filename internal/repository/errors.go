package repository

import (
	"errors"

	"github.com/mattn/go-sqlite3"
)

// IsUniqueConstraintError reports whether err is a SQLite UNIQUE constraint
// violation (e.g. a duplicate email or username). This matters for races
// that slip past an application-level existence check — two requests can
// both pass an EmailExists check before either has inserted, so the INSERT
// itself is the real source of truth. Callers should check this rather than
// treating every insert failure as an unexpected 500.
func IsUniqueConstraintError(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code == sqlite3.ErrConstraint
	}
	return false
}
