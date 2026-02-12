package domain

import "time"

// Rant represents a single developer rant from the timeline.
type Rant struct {
	ID           string
	AccountID    string // Author account ID
	Author       string // Display Name
	Username     string // @handle
	Content      string // Plain text, HTML stripped
	CreatedAt    time.Time
	URL          string // Original post URL
	IsOwn        bool   // True if this rant belongs to the authenticated user
	Liked        bool   // True if the current user has liked this rant
	LikesCount   int
	RepliesCount int
	InReplyToID  string
}
