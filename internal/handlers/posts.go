package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"forum/internal/models"
	"forum/internal/repository"
)

type PostHandler struct {
	posts      *repository.PostRepository
	categories *repository.CategoryRepository
	comments   *repository.CommentRepository
	reactions  *repository.ReactionRepository
}

func NewPostHandler(posts *repository.PostRepository, categories *repository.CategoryRepository, comments *repository.CommentRepository, reactions *repository.ReactionRepository) *PostHandler {
	return &PostHandler{posts: posts, categories: categories, comments: comments, reactions: reactions}
}

// attachPostReactions fills in LikeCount/DislikeCount/UserReaction on each
// post in place. userID is 0 for guests, meaning UserReaction stays 0.
func (h *PostHandler) attachPostReactions(posts []models.Post, userID int64) error {
	ids := make([]int64, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	countsByID, err := h.reactions.GetCountsBatch(repository.TargetPost, ids)
	if err != nil {
		return err
	}

	var userReactions map[int64]int
	if userID != 0 {
		userReactions, err = h.reactions.GetUserReactionsBatch(userID, repository.TargetPost, ids)
		if err != nil {
			return err
		}
	}

	for i := range posts {
		c := countsByID[posts[i].ID]
		posts[i].LikeCount = c.Likes
		posts[i].DislikeCount = c.Dislikes
		posts[i].UserReaction = userReactions[posts[i].ID]
	}
	return nil
}

// attachCommentReactions does the same as attachPostReactions, for comments.
func (h *PostHandler) attachCommentReactions(comments []models.Comment, userID int64) error {
	ids := make([]int64, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}

	countsByID, err := h.reactions.GetCountsBatch(repository.TargetComment, ids)
	if err != nil {
		return err
	}

	var userReactions map[int64]int
	if userID != 0 {
		userReactions, err = h.reactions.GetUserReactionsBatch(userID, repository.TargetComment, ids)
		if err != nil {
			return err
		}
	}

	for i := range comments {
		c := countsByID[comments[i].ID]
		comments[i].LikeCount = c.Likes
		comments[i].DislikeCount = c.Dislikes
		comments[i].UserReaction = userReactions[comments[i].ID]
	}
	return nil
}

// List handles GET / — shows every post to everyone, registered or not.
// Supports optional filtering via query params:
//   ?category=<id>   — posts tagged with a category (everyone)
//   ?filter=mine      — posts created by the logged-in user (requires login)
//   ?filter=liked      — posts liked by the logged-in user (requires login)
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	var (
		posts        []models.Post
		err          error
		activeFilter string
	)

	switch {
	case r.URL.Query().Has("category"):
		categoryID, parseErr := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
		if parseErr != nil {
			RespondError(w, http.StatusBadRequest, "invalid category id")
			return
		}
		posts, err = h.posts.GetByCategory(categoryID)
		activeFilter = "category:" + r.URL.Query().Get("category")

	case r.URL.Query().Get("filter") == "mine":
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		posts, err = h.posts.GetByUser(userID)
		activeFilter = "mine"

	case r.URL.Query().Get("filter") == "liked":
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		posts, err = h.posts.GetLikedByUser(userID)
		activeFilter = "liked"

	default:
		posts, err = h.posts.GetAll()
	}

	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load posts")
		return
	}

	if err := h.attachPostReactions(posts, userID); err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load reactions")
		return
	}

	categories, err := h.categories.GetAll()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load categories")
		return
	}

	data := struct {
		User         any
		Posts        any
		Categories   any
		ActiveFilter string
	}{
		User:         user,
		Posts:        posts,
		Categories:   categories,
		ActiveFilter: activeFilter,
	}

	RenderPage(w, "index.html", data)
}

// View handles GET /posts/{id} — shows a single post, visible to everyone.
func (h *PostHandler) View(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	post, err := h.posts.GetByID(id)
	if err == repository.ErrNotFound {
		RespondError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load post")
		return
	}

	comments, err := h.comments.GetByPostID(id)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load comments")
		return
	}

	user := UserFromContext(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	postSlice := []models.Post{*post}
	if err := h.attachPostReactions(postSlice, userID); err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load reactions")
		return
	}
	*post = postSlice[0]
	if err := h.attachCommentReactions(comments, userID); err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load reactions")
		return
	}

	data := struct {
		User     any
		Post     any
		Comments any
	}{
		User:     user,
		Post:     post,
		Comments: comments,
	}

	RenderPage(w, "post.html", data)
}

// NewPostForm handles GET /posts/new — only reachable by logged-in users (see RequireAuth in main.go).
func (h *PostHandler) NewPostForm(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.GetAll()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not load categories")
		return
	}

	data := struct {
		User       any
		Categories any
	}{
		User:       UserFromContext(r),
		Categories: categories,
	}

	RenderPage(w, "create_post.html", data)
}

// Create handles POST /posts — only reachable by logged-in users (see RequireAuth in main.go).
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		// Shouldn't happen since RequireAuth guards this route, but guard anyway.
		RespondError(w, http.StatusUnauthorized, "you must be logged in to post")
		return
	}

	if err := r.ParseForm(); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))

	if title == "" || content == "" {
		RespondError(w, http.StatusBadRequest, "title and content are required")
		return
	}

	var categoryIDs []int64
	seen := make(map[int64]bool)
	for _, raw := range r.Form["categories"] {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "invalid category")
			return
		}
		// A crafted request could repeat the same category ID; post_categories
		// has (post_id, category_id) as its primary key, so inserting a
		// duplicate would otherwise fail the whole transaction with a 500.
		if seen[id] {
			continue
		}
		seen[id] = true
		categoryIDs = append(categoryIDs, id)
	}

	if len(categoryIDs) == 0 {
		RespondError(w, http.StatusBadRequest, "select at least one category")
		return
	}

	postID, err := h.posts.Create(user.ID, title, content, categoryIDs)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "could not create post")
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
