package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/repository"
	"forum/internal/utils"
)

type PostHandler struct {
	posts      *repository.PostRepository
	categories *repository.CategoryRepository
	templates  *template.Template
}

func NewPostHandler(posts *repository.PostRepository, categories *repository.CategoryRepository, templates *template.Template) *PostHandler {
	return &PostHandler{posts: posts, categories: categories, templates: templates}
}

// List handles GET / - shows every post to everyone, registered or not.
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts.GetAll()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not load posts")
		return
	}

	data := struct {
		User  any
		Posts any
	}{
		User:  UserFromContext(r),
		Posts: posts,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not render page")
	}
}

// View handles GET /posts/{id} - shows a single post, visible to everyone.
func (h *PostHandler) View(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt("id", 10, 64)
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

	data := struct {
		User any
		Post any
	}{
		User: UserFromContext(r),
		Post: post,
	}

	if err := h.templates.ExecuteTemplate(w, "post.html", data); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not render page")
	}
}

// NewPostForm handles GET /posts/new - only reachable by logged-in users.
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

// Create handles POST /posts - only reachable by logged-in users.
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		// Shouldn't happen since RequireAuth guards this route, but just in case.
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