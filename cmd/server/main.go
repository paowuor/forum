package main

import (
	"log"
	"net/http"

	"forum/internal/database"
	"forum/internal/handlers"
	"forum/internal/repository"
)

const (
	dbPath = "data/forum.db"
	addr   = ":8080"
)

func main() {
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("database ready at", dbPath)

	if err := handlers.LoadTemplates("web/templates"); err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	reactionRepo := repository.NewReactionRepository(db)

	if err := categoryRepo.SeedDefaults(); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	authHandler := handlers.NewAuthHandler(userRepo, sessionRepo)
	postHandler := handlers.NewPostHandler(postRepo, categoryRepo, commentRepo, reactionRepo)
	commentHandler := handlers.NewCommentHandler(commentRepo)
	reactionHandler := handlers.NewReactionHandler(reactionRepo, commentRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", postHandler.List)
	mux.HandleFunc("GET /posts/new", handlers.RequireAuth(postHandler.NewPostForm))
	mux.HandleFunc("POST /posts", handlers.RequireAuth(postHandler.Create))
	mux.HandleFunc("GET /posts/{id}", postHandler.View)
	mux.HandleFunc("POST /posts/{id}/comments", handlers.RequireAuth(commentHandler.Create))
	mux.HandleFunc("POST /posts/{id}/react", handlers.RequireAuth(reactionHandler.ReactToPost))
	mux.HandleFunc("POST /comments/{id}/react", handlers.RequireAuth(reactionHandler.ReactToComment))

	// Login/register pages render through the same shared layout as every
	// other page, so they need a data value with a User field too (even
	// though it'll always be nil for a guest-only page like this) —
	// otherwise the layout's {{if .User}} check has nothing to evaluate.
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, r *http.Request) {
		handlers.RenderPage(w, "register.html", struct{ User any }{User: handlers.UserFromContext(r)})
	})
	mux.HandleFunc("POST /register", authHandler.Register)

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		handlers.RenderPage(w, "login.html", struct{ User any }{User: handlers.UserFromContext(r)})
	})
	mux.HandleFunc("POST /login", authHandler.Login)

	mux.HandleFunc("POST /logout", authHandler.Logout)

	// Static assets (CSS/JS). "GET /static/" is a subtree pattern, more
	// specific than the bare "/" catch-all below, so it always wins for
	// paths under /static/ regardless of registration order.
	staticFiles := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFiles))

	// Catch-all: "/" is a subtree pattern in Go's ServeMux, so it only fires
	// when nothing more specific above matched — i.e. genuinely unknown
	// routes. More specific patterns (like "GET /{$}" for the homepage)
	// always take priority over this one, regardless of registration order.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.RespondError(w, http.StatusNotFound, "page not found")
	})

	// WithUser wraps the whole mux so every route can check
	// handlers.UserFromContext(r) to see who (if anyone) is logged in.
	// Recover wraps that in turn so a panic anywhere below returns a clean
	// 500 instead of dropping the connection.
	withAuth := handlers.WithUser(sessionRepo, userRepo)(mux)
	withRecover := handlers.Recover(withAuth)

	log.Println("listening on", addr)
	if err := http.ListenAndServe(addr, withRecover); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
