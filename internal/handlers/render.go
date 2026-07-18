package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// templates is set once at startup via SetTemplates. It's package-level so
// every handler (auth, posts, comments, reactions) can render a consistent
// error page without each one needing its own template reference.
var templates *template.Template

// SetTemplates wires up the shared template set. Must be called once during
// startup, before any request is served.
func SetTemplates(t *template.Template) {
	templates = t
}

// RespondError writes a status code and renders web/templates/error.html
// with the given message. If templates aren't available or rendering fails
// for some reason, it falls back to a plain-text response so an error page
// never itself causes an unhandled failure.
func RespondError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)

	if templates == nil {
		w.Write([]byte(message))
		return
	}

	data := struct {
		Status  int
		Message string
	}{Status: status, Message: message}

	if err := templates.ExecuteTemplate(w, "error.html", data); err != nil {
		log.Printf("error rendering error.html: %v", err)
		w.Write([]byte(message))
	}
}
