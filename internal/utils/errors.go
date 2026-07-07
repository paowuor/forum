package utils

import "net/http"

// RespondError writes a plain-text error response with the given HTTP status code.
// Centralizing this means every handler reports errors the same way.
func RespondError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}
