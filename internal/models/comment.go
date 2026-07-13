package models

import "time"

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Content   string
	CreatedAt time.Time

	// Populated by repository queries that join against users.
	Username string

	// Populated by reaction queries. UserReaction is 0 if the viewer hasn't
	// reacted (or is a guest), 1 for like, -1 for dislike.
	LikeCount     int
	DislikeCount  int
	UserReaction  int
}