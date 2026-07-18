package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// pageTemplates holds one *template.Template per content page, each built
// from layout.html plus that page's own file. These are kept as separate
// template sets (rather than one shared template.ParseGlob) because every
// page defines a block named "content" — parsed together into one shared
// namespace, each file's "content" definition would silently overwrite the
// last one parsed. Parsing each page with its own copy of layout.html avoids
// that collision entirely.
var pageTemplates map[string]*template.Template

// errorTemplate renders standalone, outside the shared layout. RespondError
// doesn't receive the current *http.Request, so it has no way to know
// whether to show the logged-in or guest nav state — simplest to keep the
// error page self-contained rather than guess.
var errorTemplate *template.Template

// LoadTemplates parses web/templates/layout.html together with each
// individual page template, plus the standalone error page. Must be called
// once at startup, before any request is served.
func LoadTemplates(dir string) error {
	layout := filepath.Join(dir, "layout.html")

	pages := []string{"index.html", "post.html", "create_post.html", "login.html", "register.html"}
	loaded := make(map[string]*template.Template, len(pages))

	for _, page := range pages {
		t, err := template.New("layout.html").ParseFiles(layout, filepath.Join(dir, page))
		if err != nil {
			return err
		}
		loaded[page] = t
	}

	errTmpl, err := template.ParseFiles(filepath.Join(dir, "error.html"))
	if err != nil {
		return err
	}

	pageTemplates = loaded
	errorTemplate = errTmpl
	return nil
}

// RenderPage executes the named page (e.g. "index.html") within the shared
// layout. data must include a User field for the layout's nav to render
// the correct logged-in/guest state.
func RenderPage(w http.ResponseWriter, page string, data any) {
	t, ok := pageTemplates[page]
	if !ok {
		log.Printf("render: no such page template %q", page)
		RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render: error executing %q: %v", page, err)
	}
}

// RespondError writes a status code and renders web/templates/error.html
// with the given message. If templates aren't loaded yet or rendering fails
// for some reason, it falls back to a plain-text response so an error page
// never itself causes an unhandled failure.
func RespondError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)

	if errorTemplate == nil {
		w.Write([]byte(message))
		return
	}

	data := struct {
		Status  int
		Message string
	}{Status: status, Message: message}

	if err := errorTemplate.Execute(w, data); err != nil {
		log.Printf("error rendering error.html: %v", err)
		w.Write([]byte(message))
	}
}
