package app

import (
	"context"

	"github.com/CrestNiraj12/terminalrant/domain"
)

type Profile struct {
	ID          string
	Username    string
	DisplayName string
	Bio         string
	AvatarURL   string
	PostsCount  int
	Followers   int
	Following   int
}

type BlockedUser struct {
	AccountID   string
	Username    string
	DisplayName string
}

// AccountService provides information about the authenticated user.
type AccountService interface {
	// CurrentAccountID returns the account ID of the authenticated user.
	CurrentAccountID(ctx context.Context) (string, error)

	// CurrentProfile returns the authenticated user's profile.
	CurrentProfile(ctx context.Context) (Profile, error)

	// UpdateProfile updates display name and bio.
	UpdateProfile(ctx context.Context, displayName, bio string) error

	// BlockUser blocks a user by account ID.
	BlockUser(ctx context.Context, accountID string) error

	// ListBlockedUsers returns blocked accounts for the authenticated user.
	ListBlockedUsers(ctx context.Context, limit int) ([]BlockedUser, error)

	// UnblockUser unblocks a user by account ID.
	UnblockUser(ctx context.Context, accountID string) error

	// FollowUser follows a user by account ID.
	FollowUser(ctx context.Context, accountID string) error

	// UnfollowUser unfollows a user by account ID.
	UnfollowUser(ctx context.Context, accountID string) error

	// LookupFollowing returns follow-state for account IDs.
	LookupFollowing(ctx context.Context, accountIDs []string) (map[string]bool, error)

	// ProfileByID returns profile details for a specific account.
	ProfileByID(ctx context.Context, accountID string) (Profile, error)

	// PostsByAccount returns posts for an account, newest first.
	PostsByAccount(ctx context.Context, accountID string, limit int, maxID string) ([]domain.Rant, error)
}
