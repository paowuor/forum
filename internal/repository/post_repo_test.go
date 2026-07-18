package repository_test

import (
	"testing"

	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/testutil"
)

// postFilterFixture sets up two users, two categories, and three posts
// spread across them, for exercising all three filter types.
type postFilterFixture struct {
	users      *repository.UserRepository
	posts      *repository.PostRepository
	reactions  *repository.ReactionRepository
	aliceID    int64
	bobID      int64
	techCatID  int64
	gameCatID  int64
	aliceTech  int64 // Alice's post, tagged Tech
	bobGame    int64 // Bob's post, tagged Gaming
	bobTech    int64 // Bob's post, tagged Tech
}

func newPostFilterFixture(t *testing.T) postFilterFixture {
	t.Helper()
	db := testutil.NewTestDB(t)

	users := repository.NewUserRepository(db)
	posts := repository.NewPostRepository(db)
	reactions := repository.NewReactionRepository(db)

	aliceID, err := users.Create(&models.User{UUID: "alice-uuid", Email: "alice@example.com", Username: "alice", PasswordHash: "hashed"})
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	bobID, err := users.Create(&models.User{UUID: "bob-uuid", Email: "bob@example.com", Username: "bob", PasswordHash: "hashed"})
	if err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}

	// Categories aren't exposed with a Create method on CategoryRepository
	// (only SeedDefaults), so insert directly for test control over IDs.
	res, err := db.Exec(`INSERT INTO categories (name) VALUES ('Technology')`)
	if err != nil {
		t.Fatalf("failed to insert Technology category: %v", err)
	}
	techCatID, _ := res.LastInsertId()

	res, err = db.Exec(`INSERT INTO categories (name) VALUES ('Gaming')`)
	if err != nil {
		t.Fatalf("failed to insert Gaming category: %v", err)
	}
	gameCatID, _ := res.LastInsertId()

	aliceTech, err := posts.Create(aliceID, "Alice Tech Post", "content", []int64{techCatID})
	if err != nil {
		t.Fatalf("failed to create alice's post: %v", err)
	}
	bobGame, err := posts.Create(bobID, "Bob Gaming Post", "content", []int64{gameCatID})
	if err != nil {
		t.Fatalf("failed to create bob's gaming post: %v", err)
	}
	bobTech, err := posts.Create(bobID, "Bob Tech Post", "content", []int64{techCatID})
	if err != nil {
		t.Fatalf("failed to create bob's tech post: %v", err)
	}

	return postFilterFixture{
		users: users, posts: posts, reactions: reactions,
		aliceID: aliceID, bobID: bobID,
		techCatID: techCatID, gameCatID: gameCatID,
		aliceTech: aliceTech, bobGame: bobGame, bobTech: bobTech,
	}
}

func postIDs(posts []models.Post) map[int64]bool {
	ids := make(map[int64]bool, len(posts))
	for _, p := range posts {
		ids[p.ID] = true
	}
	return ids
}

func TestPostRepository_GetByCategory(t *testing.T) {
	f := newPostFilterFixture(t)

	techPosts, err := f.posts.GetByCategory(f.techCatID)
	if err != nil {
		t.Fatalf("GetByCategory returned error: %v", err)
	}
	ids := postIDs(techPosts)
	if len(techPosts) != 2 || !ids[f.aliceTech] || !ids[f.bobTech] {
		t.Errorf("expected exactly alice's and bob's tech posts, got %v", ids)
	}

	gamePosts, err := f.posts.GetByCategory(f.gameCatID)
	if err != nil {
		t.Fatalf("GetByCategory returned error: %v", err)
	}
	ids = postIDs(gamePosts)
	if len(gamePosts) != 1 || !ids[f.bobGame] {
		t.Errorf("expected exactly bob's gaming post, got %v", ids)
	}
}

func TestPostRepository_GetByCategory_NoMatches(t *testing.T) {
	f := newPostFilterFixture(t)

	posts, err := f.posts.GetByCategory(999999)
	if err != nil {
		t.Fatalf("GetByCategory returned error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected no posts for a nonexistent category, got %d", len(posts))
	}
}

func TestPostRepository_GetByUser(t *testing.T) {
	f := newPostFilterFixture(t)

	alicePosts, err := f.posts.GetByUser(f.aliceID)
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	ids := postIDs(alicePosts)
	if len(alicePosts) != 1 || !ids[f.aliceTech] {
		t.Errorf("expected exactly alice's one post, got %v", ids)
	}

	bobPosts, err := f.posts.GetByUser(f.bobID)
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	ids = postIDs(bobPosts)
	if len(bobPosts) != 2 || !ids[f.bobGame] || !ids[f.bobTech] {
		t.Errorf("expected exactly bob's two posts, got %v", ids)
	}
}

func TestPostRepository_GetLikedByUser(t *testing.T) {
	f := newPostFilterFixture(t)

	// Alice likes Bob's gaming post. Only that reaction should ever
	// surface in her "liked posts" filter.
	if err := f.reactions.SetReaction(f.aliceID, repository.TargetPost, f.bobGame, 1); err != nil {
		t.Fatalf("SetReaction returned error: %v", err)
	}
	// A dislike must NOT show up under "liked" — only value=1 counts.
	if err := f.reactions.SetReaction(f.aliceID, repository.TargetPost, f.bobTech, -1); err != nil {
		t.Fatalf("SetReaction (dislike) returned error: %v", err)
	}

	liked, err := f.posts.GetLikedByUser(f.aliceID)
	if err != nil {
		t.Fatalf("GetLikedByUser returned error: %v", err)
	}
	ids := postIDs(liked)
	if len(liked) != 1 || !ids[f.bobGame] {
		t.Errorf("expected exactly bob's gaming post (the one alice liked), got %v", ids)
	}

	// Bob hasn't liked anything.
	bobLiked, err := f.posts.GetLikedByUser(f.bobID)
	if err != nil {
		t.Fatalf("GetLikedByUser returned error: %v", err)
	}
	if len(bobLiked) != 0 {
		t.Errorf("expected bob to have no liked posts, got %d", len(bobLiked))
	}
}

func TestPostRepository_GetAll_IncludesEveryPost(t *testing.T) {
	f := newPostFilterFixture(t)

	all, err := f.posts.GetAll()
	if err != nil {
		t.Fatalf("GetAll returned error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected all 3 posts, got %d", len(all))
	}
}

func TestPostRepository_Create_AttachesCategoriesCorrectly(t *testing.T) {
	f := newPostFilterFixture(t)

	post, err := f.posts.GetByID(f.aliceTech)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if len(post.Categories) != 1 || post.Categories[0].ID != f.techCatID {
		t.Errorf("expected post to have exactly the Technology category, got %+v", post.Categories)
	}
	if post.Username != "alice" {
		t.Errorf("expected author username 'alice', got %q", post.Username)
	}
}

func TestPostRepository_GetByID_NotFound(t *testing.T) {
	f := newPostFilterFixture(t)

	_, err := f.posts.GetByID(999999)
	if err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
