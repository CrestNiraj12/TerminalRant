package app

import (
	"context"

	"terminalrant/domain"
)

// TimelineService fetches rants from a social timeline.
type TimelineService interface {
	// FetchByHashtag returns rants for a given hashtag, newest first.
	FetchByHashtag(ctx context.Context, hashtag string, limit int) ([]domain.Rant, error)

	// FetchByHashtagPage returns a page of rants older than maxID (if provided).
	FetchByHashtagPage(ctx context.Context, hashtag string, limit int, maxID string) ([]domain.Rant, error)

	// FetchHomePage returns a page from the authenticated home timeline.
	FetchHomePage(ctx context.Context, limit int, maxID string) ([]domain.Rant, error)

	// FetchPublicPage returns a page from the public timeline.
	FetchPublicPage(ctx context.Context, limit int, maxID string) ([]domain.Rant, error)

	// FetchTrendingPage returns trending posts.
	FetchTrendingPage(ctx context.Context, limit int, maxID string) ([]domain.Rant, error)

	// FetchThread returns the context of a rant (ancestors and replies).
	FetchThread(ctx context.Context, id string) (ancestors, descendants []domain.Rant, err error)
}
