package repository

import (
	"database/sql"
	"errors"
	"time"

	"forum/internal/models"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create deletes any existing sessions for the user (so each user only ever
// has one active session) and inserts a new one.
func (r *SessionRepository) Create(s *models.Session) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, s.UserID); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		s.ID, s.UserID, s.ExpiresAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID looks up a session by its ID. Returns ErrNotFound if no match exists,
// regardless of whether it's missing or has already expired.
func (r *SessionRepository) GetByID(id string) (*models.Session, error) {
	var s models.Session
	err := r.db.QueryRow(
		`SELECT id, user_id, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.ExpiresAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(s.ExpiresAt) {
		return nil, ErrNotFound
	}
	return &s, nil
}

// Delete removes a session by ID (used on logout).
func (r *SessionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}
