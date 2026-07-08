package repository

import (
	"database/sql"
	"errors"

	"forum/internal/models"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// Create inserts a new post and links it to the given category IDs, all in a
// single transaction so a post is never left without its categories.
func (r *PostRepository) Create(userID int64, title, content string, categoryIDs []int64) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)`,
		userID, title, content,
	)
	if err != nil {
		return 0, err
	}

	postID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, catID := range categoryIDs {
		if _, err := tx.Exec(
			`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			postID, catID,
		); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return postID, nil
}

// GetAll returns every post, newest first, with the author's username and
// its categories attached.
func (r *PostRepository) GetAll() ([]models.Post, error) {
	rows, err := r.db.Query(`
		SELECT posts.id, posts.user_id, posts.title, posts.content, posts.created_at, users.username
		FROM posts
		JOIN users ON users.id = posts.user_id
		ORDER BY posts.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt, &p.Username); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.attachCategories(posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// GetByID returns a single post with its author's username and categories attached.
// Returns ErrNotFound if no post with that ID exists.
func (r *PostRepository) GetByID(id int64) (*models.Post, error) {
	var p models.Post
	err := r.db.QueryRow(`
		SELECT posts.id, posts.user_id, posts.title, posts.content, posts.created_at, users.username
		FROM posts
		JOIN users ON users.id = posts.user_id
		WHERE posts.id = ?
	`, id).Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt, &p.Username)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	posts := []models.Post{p}
	if err := r.attachCategories(posts); err != nil {
		return nil, err
	}
	return &posts[0], nil
}

// attachCategories fills in the Categories field for each post in place,
// using a single query for all posts rather than one query per post.
func (r *PostRepository) attachCategories(posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}

	indexByPostID := make(map[int64]int, len(posts))
	placeholders := make([]any, len(posts))
	for i, p := range posts {
		indexByPostID[p.ID] = i
		placeholders[i] = p.ID
	}

	query := `
		SELECT post_categories.post_id, categories.id, categories.name
		FROM post_categories
		JOIN categories ON categories.id = post_categories.category_id
		WHERE post_categories.post_id IN (` + placeholdersList(len(posts)) + `)
	`
	rows, err := r.db.Query(query, placeholders...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var postID int64
		var c models.Category
		if err := rows.Scan(&postID, &c.ID, &c.Name); err != nil {
			return err
		}
		idx := indexByPostID[postID]
		posts[idx].Categories = append(posts[idx].Categories, c)
	}
	return rows.Err()
}

// placeholdersList returns "?, ?, ?" repeated n times, for building IN clauses.
func placeholdersList(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ", "
		}
		s += "?"
	}
	return s
}
