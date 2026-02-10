package domain

import "time"

// Rant represents a single developer rant from the timeline.
type Rant struct {
	ID        string
	Author    string
	Content   string // Plain text, HTML stripped
	CreatedAt time.Time
	URL       string // Original post URL
	IsOwn     bool   // True if this rant belongs to the authenticated user
}
