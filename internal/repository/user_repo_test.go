package repository_test

import (
	"errors"
	"testing"

	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/testutil"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)

	id, err := repo.Create(&models.User{
		UUID:         "uuid-1",
		Email:        "alice@example.com",
		Username:     "alice",
		PasswordHash: "hashed",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected a non-zero generated ID")
	}

	byEmail, err := repo.GetByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail returned error: %v", err)
	}
	if byEmail.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", byEmail.Username)
	}

	byID, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if byID.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", byID.Email)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)

	_, err := repo.GetByEmail("nobody@example.com")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepository_EmailAndUsernameExists(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)

	if _, err := repo.Create(&models.User{
		UUID: "uuid-1", Email: "bob@example.com", Username: "bob", PasswordHash: "hashed",
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	exists, err := repo.EmailExists("bob@example.com")
	if err != nil {
		t.Fatalf("EmailExists returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected EmailExists to return true for a registered email")
	}

	exists, err = repo.EmailExists("nobody@example.com")
	if err != nil {
		t.Fatalf("EmailExists returned error: %v", err)
	}
	if exists {
		t.Errorf("expected EmailExists to return false for an unregistered email")
	}

	exists, err = repo.UsernameExists("bob")
	if err != nil {
		t.Fatalf("UsernameExists returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected UsernameExists to return true for a registered username")
	}
}

// Duplicate emails must be rejected at the database level (UNIQUE constraint),
// as a last line of defense even if application-level validation is bypassed.
func TestUserRepository_DuplicateEmail_Rejected(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)

	first := &models.User{UUID: "uuid-1", Email: "dup@example.com", Username: "user1", PasswordHash: "hashed"}
	if _, err := repo.Create(first); err != nil {
		t.Fatalf("first Create returned unexpected error: %v", err)
	}

	second := &models.User{UUID: "uuid-2", Email: "dup@example.com", Username: "user2", PasswordHash: "hashed"}
	if _, err := repo.Create(second); err == nil {
		t.Fatalf("expected an error creating a user with a duplicate email, got nil")
	}
}
