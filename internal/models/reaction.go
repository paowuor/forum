package models

// TargetType identifies whether a reaction applies to a post or a comment.
// Mirrors repository.TargetType's underlying values ("post"/"comment"); kept
// as a plain string here since models has no dependency on repository.
type TargetType string

const (
	TargetPost    TargetType = "post"
	TargetComment TargetType = "comment"
)

// Reaction mirrors a single row of the reactions table: one user's like
// (Value = 1) or dislike (Value = -1) on a post or comment.
//
// The repository layer mostly works with aggregates instead of individual
// rows — see repository.Counts for like/dislike totals, and the
// LikeCount/DislikeCount/UserReaction fields already attached to Post and
// Comment for what handlers and templates actually use. This struct exists
// for the cases that do need the row shape itself (e.g. debugging a query,
// or a future admin/audit view over raw reactions).
type Reaction struct {
	ID         int64
	UserID     int64
	TargetType TargetType
	TargetID   int64
	Value      int
}
