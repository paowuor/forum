package repository_test

import (
	"errors"
	"testing"
	"time"

	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/testutil"
)

func mustCreateUser(t *testing.T, repo *repository.UserRepository, email, username string) int64 {
	t.Helper()
	id, err := repo.Create(&models.User{
		UUID: email + "-uuid", Email: email, Username: username, PasswordHash: "hashed",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

func TestSessionRepository_CreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)

	userID := mustCreateUser(t, users, "carol@example.com", "carol")

	expiry := time.Now().Add(24 * time.Hour)
	if err := sessions.Create(&models.Session{ID: "session-1", UserID: userID, ExpiresAt: expiry}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got, err := sessions.GetByID("session-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("expected UserID %d, got %d", userID, got.UserID)
	}
}

func TestSessionRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	sessions := repository.NewSessionRepository(db)

	_, err := sessions.GetByID("does-not-exist")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// A new login must invalidate any previous session for the same user, so
// each user only ever has one active session at a time (per the spec).
func TestSessionRepository_NewLoginInvalidatesOldSession(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)

	userID := mustCreateUser(t, users, "dave@example.com", "dave")
	expiry := time.Now().Add(24 * time.Hour)

	if err := sessions.Create(&models.Session{ID: "session-old", UserID: userID, ExpiresAt: expiry}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}
	if err := sessions.Create(&models.Session{ID: "session-new", UserID: userID, ExpiresAt: expiry}); err != nil {
		t.Fatalf("second Create returned error: %v", err)
	}

	if _, err := sessions.GetByID("session-old"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected the old session to be gone, got err=%v", err)
	}
	if _, err := sessions.GetByID("session-new"); err != nil {
		t.Errorf("expected the new session to still be valid, got err=%v", err)
	}
}

// Expired sessions must be treated as not found, even though the row is
// technically still in the database until cleaned up.
func TestSessionRepository_ExpiredSession_TreatedAsNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)

	userID := mustCreateUser(t, users, "erin@example.com", "erin")
	pastExpiry := time.Now().Add(-1 * time.Hour)

	if err := sessions.Create(&models.Session{ID: "expired-session", UserID: userID, ExpiresAt: pastExpiry}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := sessions.GetByID("expired-session"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound for an expired session, got %v", err)
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)

	userID := mustCreateUser(t, users, "frank@example.com", "frank")
	expiry := time.Now().Add(24 * time.Hour)

	if err := sessions.Create(&models.Session{ID: "session-1", UserID: userID, ExpiresAt: expiry}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := sessions.Delete("session-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := sessions.GetByID("session-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}
