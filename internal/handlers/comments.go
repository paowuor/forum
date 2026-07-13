package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"forum/internal/repository"
	"forum/internal/utils"
)

type CommentHandler struct {
	comments *repository.CommentRepository
}

func NewCommentHandler(comments *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{comments: comments}
}

// Create handles POST /posts/{id}/comments — only reachable by logged-in
// users (see RequireAuth in main.go).
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "you must be logged in to comment")
		return
	}

	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	exists, err := h.comments.PostExists(postID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !exists {
		utils.RespondError(w, http.StatusNotFound, "post not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		utils.RespondError(w, http.StatusBadRequest, "comment cannot be empty")
		return
	}

	if _, err := h.comments.Create(postID, user.ID, content); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not create comment")
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
