package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"forum/internal/repository"
	"forum/internal/utils"
)

var errInvalidReactionValue = errors.New("value must be 1 or -1")

type ReactionHandler struct {
	reactions *repository.ReactionRepository
	comments  *repository.CommentRepository
}

func NewReactionHandler(reactions *repository.ReactionRepository, comments *repository.CommentRepository) *ReactionHandler {
	return &ReactionHandler{reactions: reactions, comments: comments}
}

// parseReactionValue reads "value" from the submitted form and ensures it's
// exactly 1 (like) or -1 (dislike).
func parseReactionValue(r *http.Request) (int, error) {
	raw := r.FormValue("value")
	value, err := strconv.Atoi(raw)
	if err != nil || (value != 1 && value != -1) {
		return 0, errInvalidReactionValue
	}
	return value, nil
}

// ReactToPost handles POST /posts/{id}/react — only reachable by logged-in
// users (see RequireAuth in main.go).
func (h *ReactionHandler) ReactToPost(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "you must be logged in to react")
		return
	}

	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	value, err := parseReactionValue(r)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "value must be 1 (like) or -1 (dislike)")
		return
	}

	exists, err := h.reactions.TargetExists(repository.TargetPost, postID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !exists {
		utils.RespondError(w, http.StatusNotFound, "post not found")
		return
	}

	if err := h.reactions.SetReaction(user.ID, repository.TargetPost, postID, value); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not save reaction")
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}

// ReactToComment handles POST /comments/{id}/react — only reachable by
// logged-in users (see RequireAuth in main.go).
func (h *ReactionHandler) ReactToComment(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "you must be logged in to react")
		return
	}

	commentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	value, err := parseReactionValue(r)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "value must be 1 (like) or -1 (dislike)")
		return
	}

	postID, err := h.comments.GetPostID(commentID)
	if err == repository.ErrNotFound {
		utils.RespondError(w, http.StatusNotFound, "comment not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.reactions.SetReaction(user.ID, repository.TargetComment, commentID, value); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not save reaction")
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
