package handlers

import (
	"context"
	"net/http"

	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/repository"
)

type contextKey string

const userContextKey contextKey = "user"

// WithUser returns a middleware that checks for a session cookie and, if valid,
// attaches the corresponding user to the request context. Requests with no
// cookie or an invalid/expired session are passed through unchanged (as a
// guest) rather than rejected - routes that require login use RequireAuth
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

// RequiredAuth wraps a handler so it only runs for logged-in users. It must be
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