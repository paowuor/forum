package main

import (
	"html/template"
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

var templates = template.Must(template.ParseGlob("web/templates/*.html"))

func main() {
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("database ready at", dbPath)

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	if err := categoryRepo.SeedDefaults(); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	authHandler := handlers.NewAuthHandler(userRepo, sessionRepo)
	postHandler := handlers.NewPostHandler(postRepo, categoryRepo, commentRepo, templates)
	commentHandler := handlers.NewCommentHandler(commentRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", postHandler.List)
	mux.HandleFunc("GET /posts/new", handlers.RequireAuth(postHandler.NewPostForm))
	mux.HandleFunc("POST /posts", handlers.RequireAuth(postHandler.Create))
	mux.HandleFunc("GET /posts/{id}", postHandler.View)
	mux.HandleFunc("POST /posts/{id}/comments", handlers.RequireAuth(commentHandler.Create))

	mux.HandleFunc("GET /register", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "register.html", nil)
	})
	mux.HandleFunc("POST /register", authHandler.Register)

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "login.html", nil)
	})
	mux.HandleFunc("POST /login", authHandler.Login)

	mux.HandleFunc("POST /logout", authHandler.Logout)

	// WithUser wraps the whole mux so every route can check
	// handlers.UserFromContext(r) to see who (if anyone) is logged in.
	withAuth := handlers.WithUser(sessionRepo, userRepo)(mux)

	log.Println("listening on", addr)
	if err := http.ListenAndServe(addr, withAuth); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
