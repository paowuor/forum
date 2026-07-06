package main

import (
	"log"
	"net/http"

	"forum/internal/database"
)

const (
	dbPath = "data/forum.db"
	addr   = ":8080"
)

func main() {
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	log.Println("database ready at", dbPath)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("forum server is running"))
	})

	log.Println("listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}