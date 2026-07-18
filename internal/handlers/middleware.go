package handlers

import (
	"context"
	"log"
	"net/http"

	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/repository"
)

type contextKey string

const userContextKey contextKey = "user"

// Recover wraps a handler so a panic anywhere below it is caught, logged,
// and turned into a clean 500 response instead of an abrupt connection
// drop. Go's server already prevents one panicking request from crashing
// the whole process, but without this the caller just sees the connection
// die with no explanation.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, rec)
				RespondError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// WithUser returns middleware that checks for a session cookie and, if valid,
// attaches the corresponding user to the request context. Requests with no
// cookie or an invalid/expired session are passed through unchanged (as a
// guest) rather than rejected — routes that require login use RequireAuth
// on top of this.
func WithUser(sessions *repository.SessionRepository, users *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			session, err := sessions.GetByID(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, err := users.GetByID(session.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the logged-in user from the request context.
// Returns nil if the request is from a guest.
func UserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// RequireAuth wraps a handler so it only runs for logged-in users. It must be
// used after WithUser in the middleware chain, since it relies on the user
// already being attached to the request context.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}
