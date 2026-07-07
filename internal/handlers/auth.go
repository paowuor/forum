package handlers

import (
	"errors"
	"net/http"
	"time"

	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/repository"
	"forum/internal/utils"
)

type AuthHandler struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
}

func NewAuthHandler(users *repository.UserRepository, sessions *repository.SessionRepository) *AuthHandler {
	return &AuthHandler{users: users, sessions: sessions}
}

//Register handles POST /register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if err := utils.ValidateEmail(email); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := utils.ValidateUsername(username); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := utils.ValidatePassword(password); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	emailTaken, err := h.users.EmailExists(email)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if emailTaken {
		utils.RespondError(w, http.StatusConflict, "email is already registered")
		return
	}

	usernameTaken, err := h.users.UsernameExists(username)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if usernameTaken {
		utils.RespondError(w, http.StatusConflict, "username is already taken")
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	userUUID, err := auth.NewSessionID()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_, err = h.users.Create(&models.User{
		UUID:         userUUID,
		Email:        email,
		Username:     username,
		PasswordHash: hash,
	})
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Login handles POST /login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := h.users.GetByEmail(email)
	if errors.Is(err, repository.ErrNotFound) {
		utils.RespondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		utils.RespondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	sessionID, err := auth.NewSessionID()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	expiresAt := time.Now().Add(auth.SessionDuration)

	if err := h.sessions.Create(&models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    sessionID,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles POST /logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.CookieName)
	if err == nil {
		h.sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:	 auth.CookieName,
		Value:	 "",
		Path:	 "/",
		Expires: time.Unix(0, 0),
		HttpOnly: true,
		MaxAge: -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}