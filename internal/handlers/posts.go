package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/utils"
)

type PostHandler struct {
	posts      *repository.PostRepository
	categories *repository.CategoryRepository
	comments   *repository.CommentRepository
	reactions  *repository.ReactionRepository
	templates  *template.Template
}

func NewPostHandler(posts *repository.PostRepository, categories *repository.CategoryRepository, comments *repository.CommentRepository, reactions *repository.ReactionRepository, templates *template.Template) *PostHandler {
	return &PostHandler{posts: posts, categories: categories, comments: comments, reactions: reactions, templates: templates}
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
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts.GetAll()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load posts")
		return
	}

	user := UserFromContext(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}
	if err := h.attachPostReactions(posts, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load reactions")
		return
	}

	data := struct {
		User  any
		Posts any
	}{
		User:  user,
		Posts: posts,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not render page")
	}
}

// View handles GET /posts/{id} — shows a single post, visible to everyone.
func (h *PostHandler) View(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	post, err := h.posts.GetByID(id)
	if err == repository.ErrNotFound {
		utils.RespondError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load post")
		return
	}

	comments, err := h.comments.GetByPostID(id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load comments")
		return
	}

	user := UserFromContext(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	postSlice := []models.Post{*post}
	if err := h.attachPostReactions(postSlice, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load reactions")
		return
	}
	*post = postSlice[0]
	if err := h.attachCommentReactions(comments, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load reactions")
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

	if err := h.templates.ExecuteTemplate(w, "post.html", data); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not render page")
	}
}

// NewPostForm handles GET /posts/new — only reachable by logged-in users (see RequireAuth in main.go).
func (h *PostHandler) NewPostForm(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.GetAll()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load categories")
		return
	}

	data := struct {
		User       any
		Categories any
	}{
		User:       UserFromContext(r),
		Categories: categories,
	}

	if err := h.templates.ExecuteTemplate(w, "create_post.html", data); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not render page")
	}
}

// Create handles POST /posts — only reachable by logged-in users (see RequireAuth in main.go).
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		// Shouldn't happen since RequireAuth guards this route, but guard anyway.
		utils.RespondError(w, http.StatusUnauthorized, "you must be logged in to post")
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))

	if title == "" || content == "" {
		utils.RespondError(w, http.StatusBadRequest, "title and content are required")
		return
	}

	var categoryIDs []int64
	for _, raw := range r.Form["categories"] {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "invalid category")
			return
		}
		categoryIDs = append(categoryIDs, id)
	}

	postID, err := h.posts.Create(user.ID, title, content, categoryIDs)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not create post")
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
