package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
)

// accountService implements app.AccountService using the Mastodon API.
type accountService struct {
	client   *Client
	cachedID string // Cache the account ID after first fetch.
}

// NewAccountService creates an AccountService backed by Mastodon.
func NewAccountService(client *Client) *accountService {
	return &accountService{client: client}
}

func (s *accountService) CurrentAccountID(_ context.Context) (string, error) {
	if s.cachedID != "" {
		return s.cachedID, nil
	}

	data, err := s.client.Get("/api/v1/accounts/verify_credentials")
	if err != nil {
		return "", fmt.Errorf("fetching account: %w", err)
	}

	var acct struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &acct); err != nil {
		return "", fmt.Errorf("parsing account: %w", err)
	}

	s.cachedID = acct.ID
	return s.cachedID, nil
}
