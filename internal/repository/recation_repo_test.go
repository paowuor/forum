package repository_test

import (
	"testing"

	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/testutil"
)

func mustCreateUserAndPost(t *testing.T, users *repository.UserRepository, posts *repository.PostRepository, email, username string) (userID, postID int64) {
	t.Helper()
	userID, err := users.Create(&models.User{
		UUID: email + "-uuid", Email: email, Username: username, PasswordHash: "hashed",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	postID, err = posts.Create(userID, "Test Post", "content", nil)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}
	return userID, postID
}

func TestReactionRepository_LikeThenLikeAgain_TogglesOff(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	userID, postID := mustCreateUserAndPost(t, users, posts, "alice@example.com", "alice")

	if err := reactions.SetReaction(userID, repository.TargetPost, postID, 1); err != nil {
		t.Fatalf("first SetReaction (like) returned error: %v", err)
	}
	likes, dislikes, err := reactions.GetCounts(repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetCounts returned error: %v", err)
	}
	if likes != 1 || dislikes != 0 {
		t.Fatalf("after liking, expected 1 like / 0 dislikes, got %d/%d", likes, dislikes)
	}

	if err := reactions.SetReaction(userID, repository.TargetPost, postID, 1); err != nil {
		t.Fatalf("second SetReaction (like again) returned error: %v", err)
	}
	likes, dislikes, err = reactions.GetCounts(repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetCounts returned error: %v", err)
	}
	if likes != 0 || dislikes != 0 {
		t.Fatalf("after liking twice, expected the reaction to toggle off (0/0), got %d/%d", likes, dislikes)
	}
}

func TestReactionRepository_LikeThenDislike_Switches(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	userID, postID := mustCreateUserAndPost(t, users, posts, "bob@example.com", "bob")

	if err := reactions.SetReaction(userID, repository.TargetPost, postID, 1); err != nil {
		t.Fatalf("SetReaction (like) returned error: %v", err)
	}
	if err := reactions.SetReaction(userID, repository.TargetPost, postID, -1); err != nil {
		t.Fatalf("SetReaction (dislike) returned error: %v", err)
	}

	likes, dislikes, err := reactions.GetCounts(repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetCounts returned error: %v", err)
	}
	// The switch must replace the reaction, not stack it: exactly one
	// dislike, zero likes — never both counted.
	if likes != 0 || dislikes != 1 {
		t.Fatalf("after switching like->dislike, expected 0 likes / 1 dislike, got %d/%d", likes, dislikes)
	}
}

func TestReactionRepository_DifferentUsers_CountIndependently(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	aliceID, postID := mustCreateUserAndPost(t, users, posts, "carol@example.com", "carol")
	bobID, err := users.Create(&models.User{UUID: "dave-uuid", Email: "dave@example.com", Username: "dave", PasswordHash: "hashed"})
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}

	if err := reactions.SetReaction(aliceID, repository.TargetPost, postID, 1); err != nil {
		t.Fatalf("alice's SetReaction returned error: %v", err)
	}
	if err := reactions.SetReaction(bobID, repository.TargetPost, postID, -1); err != nil {
		t.Fatalf("bob's SetReaction returned error: %v", err)
	}

	likes, dislikes, err := reactions.GetCounts(repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetCounts returned error: %v", err)
	}
	if likes != 1 || dislikes != 1 {
		t.Fatalf("expected 1 like / 1 dislike from two different users, got %d/%d", likes, dislikes)
	}

	aliceReaction, err := reactions.GetUserReaction(aliceID, repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetUserReaction (alice) returned error: %v", err)
	}
	if aliceReaction != 1 {
		t.Errorf("expected alice's own reaction to be 1 (like), got %d", aliceReaction)
	}

	bobReaction, err := reactions.GetUserReaction(bobID, repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetUserReaction (bob) returned error: %v", err)
	}
	if bobReaction != -1 {
		t.Errorf("expected bob's own reaction to be -1 (dislike), got %d", bobReaction)
	}
}

func TestReactionRepository_GetUserReaction_NoneReturnsZero(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	userID, postID := mustCreateUserAndPost(t, users, posts, "erin@example.com", "erin")

	value, err := reactions.GetUserReaction(userID, repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("GetUserReaction returned error: %v", err)
	}
	if value != 0 {
		t.Errorf("expected 0 for a user with no reaction, got %d", value)
	}
}

func TestReactionRepository_TargetExists(t *testing.T) {
	db := testutil.NewTestDB(t)
	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	_, postID := mustCreateUserAndPost(t, users, posts, "frank@example.com", "frank")

	exists, err := reactions.TargetExists(repository.TargetPost, postID)
	if err != nil {
		t.Fatalf("TargetExists returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected TargetExists to return true for a real post")
	}

	exists, err = reactions.TargetExists(repository.TargetPost, 999999)
	if err != nil {
		t.Fatalf("TargetExists returned error: %v", err)
	}
	if exists {
		t.Errorf("expected TargetExists to return false for a nonexistent post")
	}
}
