package app

import (
	"context"

	"terminalrant/domain"
)

// TimelineService fetches rants from a social timeline.
type TimelineService interface {
	// FetchByHashtag returns rants for a given hashtag, newest first.
	FetchByHashtag(ctx context.Context, hashtag string, limit int) ([]domain.Rant, error)

	// FetchThread returns the context of a rant (ancestors and replies).
	FetchThread(ctx context.Context, id string) (ancestors, descendants []domain.Rant, err error)
}
