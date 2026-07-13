package repository

import (
	"database/sql"
	"errors"
)

// TargetType identifies whether a reaction applies to a post or a comment.
type TargetType string

const (
	TargetPost    TargetType = "post"
	TargetComment TargetType = "comment"
)

type ReactionRepository struct {
	db *sql.DB
}

func NewReactionRepository(db *sql.DB) *ReactionRepository {
	return &ReactionRepository{db: db}
}

// SetReaction records a user's like (value=1) or dislike (value=-1) on a
// target. Clicking the same reaction again removes it (toggle off);
// clicking the opposite reaction switches it.
func (r *ReactionRepository) SetReaction(userID int64, targetType TargetType, targetID int64, value int) error {
	var existing int
	err := r.db.QueryRow(
		`SELECT value FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ?`,
		userID, targetType, targetID,
	).Scan(&existing)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = r.db.Exec(
			`INSERT INTO reactions (user_id, target_type, target_id, value) VALUES (?, ?, ?, ?)`,
			userID, targetType, targetID, value,
		)
		return err

	case err != nil:
		return err

	case existing == value:
		// Same reaction clicked again: remove it.
		_, err = r.db.Exec(
			`DELETE FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ?`,
			userID, targetType, targetID,
		)
		return err

	default:
		// Opposite reaction clicked: flip it.
		_, err = r.db.Exec(
			`UPDATE reactions SET value = ? WHERE user_id = ? AND target_type = ? AND target_id = ?`,
			value, userID, targetType, targetID,
		)
		return err
	}
}

// GetCounts returns the like and dislike counts for a single target.
func (r *ReactionRepository) GetCounts(targetType TargetType, targetID int64) (likes, dislikes int, err error) {
	err = r.db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		FROM reactions WHERE target_type = ? AND target_id = ?`,
		targetType, targetID,
	).Scan(&likes, &dislikes)
	return likes, dislikes, err
}

// GetUserReaction returns the current user's reaction to a target: 1 (like),
// -1 (dislike), or 0 (no reaction).
func (r *ReactionRepository) GetUserReaction(userID int64, targetType TargetType, targetID int64) (int, error) {
	var value int
	err := r.db.QueryRow(
		`SELECT value FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ?`,
		userID, targetType, targetID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

// Counts holds like/dislike totals for a single target.
type Counts struct {
	Likes    int
	Dislikes int
}

// GetCountsBatch returns like/dislike counts for many targets of the same
// type in a single query, keyed by target ID. Targets with no reactions at
// all are simply absent from the map (treat a missing entry as 0/0).
func (r *ReactionRepository) GetCountsBatch(targetType TargetType, targetIDs []int64) (map[int64]Counts, error) {
	result := make(map[int64]Counts)
	if len(targetIDs) == 0 {
		return result, nil
	}

	args := make([]any, 0, len(targetIDs)+1)
	args = append(args, targetType)
	for _, id := range targetIDs {
		args = append(args, id)
	}

	query := `
		SELECT target_id,
			SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END)
		FROM reactions
		WHERE target_type = ? AND target_id IN (` + placeholdersList(len(targetIDs)) + `)
		GROUP BY target_id
	`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var c Counts
		if err := rows.Scan(&id, &c.Likes, &c.Dislikes); err != nil {
			return nil, err
		}
		result[id] = c
	}
	return result, rows.Err()
}

// GetUserReactionsBatch returns the given user's reaction (1, -1) to many
// targets of the same type in a single query. Targets the user hasn't
// reacted to are absent from the map (treat a missing entry as 0).
func (r *ReactionRepository) GetUserReactionsBatch(userID int64, targetType TargetType, targetIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(targetIDs) == 0 {
		return result, nil
	}

	args := make([]any, 0, len(targetIDs)+2)
	args = append(args, userID, targetType)
	for _, id := range targetIDs {
		args = append(args, id)
	}

	query := `
		SELECT target_id, value
		FROM reactions
		WHERE user_id = ? AND target_type = ? AND target_id IN (` + placeholdersList(len(targetIDs)) + `)
	`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var value int
		if err := rows.Scan(&id, &value); err != nil {
			return nil, err
		}
		result[id] = value
	}
	return result, rows.Err()
}

// TargetExists reports whether a post or comment with the given ID exists,
// used to validate a reaction before recording it.
func (r *ReactionRepository) TargetExists(targetType TargetType, targetID int64) (bool, error) {
	table := "posts"
	if targetType == TargetComment {
		table = "comments"
	}
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = ?)`, targetID).Scan(&exists)
	return exists, err
}
