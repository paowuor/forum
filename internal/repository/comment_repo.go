package repository

import (
	"database/sql"

	"forum/internal/models"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create inserts a new comment on a post and returns its generated ID.
func (r *CommentRepository) Create(postID, userID int64, content string) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)`,
		postID, userID, content,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetByPostID returns every comment on a post, oldest first, with each
// commenter's username attached.
func (r *CommentRepository) GetByPostID(postID int64) ([]models.Comment, error) {
	rows, err := r.db.Query(`
		SELECT comments.id, comments.post_id, comments.user_id, comments.content, comments.created_at, users.username
		FROM comments
		JOIN users ON users.id = comments.user_id
		WHERE comments.post_id = ?
		ORDER BY comments.created_at ASC
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// PostExists reports whether a post with the given ID exists — used to
// validate comment submissions before inserting them.
func (r *CommentRepository) PostExists(postID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)`, postID).Scan(&exists)
	return exists, err
}

// GetPostID returns the post_id a comment belongs to — used to redirect
// back to the right post after reacting to a comment.
func (r *CommentRepository) GetPostID(commentID int64) (int64, error) {
	var postID int64
	err := r.db.QueryRow(`SELECT post_id FROM comments WHERE id = ?`, commentID).Scan(&postID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	return postID, err
}
