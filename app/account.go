package app

import "context"

type Profile struct {
	ID          string
	Username    string
	DisplayName string
	Bio         string
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
}
