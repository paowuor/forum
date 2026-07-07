package repository

import (
	"database/sql"
	"errors"

	"forum/internal/models"
)

// ErrNotFound is returned when a lookup query finds no matching row.
var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user and returns their generated ID.
func (r *UserRepository) Create(u *models.User) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO users (uuid, email, username, password_hash) VALUES (?, ?, ?, ?)`,
		u.UUID, u.Email, u.Username, u.PasswordHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetByEmail looks up a user by email. Returns ErrNotFound if no match exists.
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	row := r.db.QueryRow(
		`SELECT id, uuid, email, username, password_hash, created_at FROM users WHERE email = ?`,
		email,
	)
	return scanUser(row)
}

// GetByID looks up a user by their primary key. Returns ErrNotFound if no match exists.
func (r *UserRepository) GetByID(id int64) (*models.User, error) {
	row := r.db.QueryRow(
		`SELECT id, uuid, email, username, password_hash, created_at FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

// EmailExists reports whether a user with the given email is already registered.
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`, email).Scan(&exists)
	return exists, err
}

// UsernameExists reports whether a user with the given username is already registered.
func (r *UserRepository) UsernameExists(username string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`, username).Scan(&exists)
	return exists, err
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.UUID, &u.Email, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
