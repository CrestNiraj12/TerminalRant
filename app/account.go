package app

import "context"

// AccountService provides information about the authenticated user.
type AccountService interface {
	// CurrentAccountID returns the account ID of the authenticated user.
	CurrentAccountID(ctx context.Context) (string, error)
}
