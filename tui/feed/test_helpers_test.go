package feed

import (
	"context"
	"time"

	"terminalrant/app"
	"terminalrant/domain"
)

type stubTimeline struct{}

func (stubTimeline) FetchByHashtag(context.Context, string, int) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchByHashtagPage(context.Context, string, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchHomePage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchPublicPage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchTrendingPage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchThread(context.Context, string) ([]domain.Rant, []domain.Rant, error) {
	return nil, nil, nil
}

type stubAccount struct{}

func (stubAccount) CurrentAccountID(context.Context) (string, error)                 { return "", nil }
func (stubAccount) CurrentProfile(context.Context) (app.Profile, error)              { return app.Profile{}, nil }
func (stubAccount) UpdateProfile(context.Context, string, string) error              { return nil }
func (stubAccount) BlockUser(context.Context, string) error                          { return nil }
func (stubAccount) ListBlockedUsers(context.Context, int) ([]app.BlockedUser, error) { return nil, nil }
func (stubAccount) UnblockUser(context.Context, string) error                        { return nil }
func (stubAccount) FollowUser(context.Context, string) error                         { return nil }
func (stubAccount) UnfollowUser(context.Context, string) error                       { return nil }
func (stubAccount) LookupFollowing(context.Context, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (stubAccount) ProfileByID(context.Context, string) (app.Profile, error) {
	return app.Profile{}, nil
}
func (stubAccount) PostsByAccount(context.Context, string, int, string) ([]domain.Rant, error) {
	return nil, nil
}

func makeRant(id string, createdAt time.Time, accountID string) domain.Rant {
	return domain.Rant{
		ID:        id,
		AccountID: accountID,
		Author:    "Author " + id,
		Username:  "user" + id,
		Content:   "hello " + domain.AppHashTag,
		CreatedAt: createdAt,
	}
}
