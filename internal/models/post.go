package models

import "time"

type Post struct {
	ID        int64
	UserID    int64
	Title     string
	Content   string
	CreatedAt time.Time

	// Populated by repository queries that join against users/categories.
	// Not stored directly in the posts table.
	Username   string
	Categories []Category
}