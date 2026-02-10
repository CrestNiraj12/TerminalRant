package app

import (
	"context"

	"terminalrant/domain"
)

// PostService publishes, edits, and deletes rants on a social backend.
type PostService interface {
	// Post publishes a new rant with the given content and hashtag.
	Post(ctx context.Context, content string, hashtag string) (domain.Rant, error)

	// Edit updates an existing rant's content.
	Edit(ctx context.Context, id string, content string, hashtag string) (domain.Rant, error)

	// Delete removes a rant by ID.
	Delete(ctx context.Context, id string) error
}
