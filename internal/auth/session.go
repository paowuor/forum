package auth

import (
	"time"

	"github.com/gofrs/uuid"
)

// CookieName is the name of the cookie that stores the session ID in the browser.
const CookieName = "forum_session"

// SessionDuration controls how long a login session stays valid.
const SessionDuration = 24 * time.Hour

// NewSessionID generates a new random UUID to use as a session identifier.
func NewSessionID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
