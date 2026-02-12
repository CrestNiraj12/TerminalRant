package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"terminalrant/app"
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

func (s *accountService) CurrentAccountID(ctx context.Context) (string, error) {
	profile, err := s.CurrentProfile(ctx)
	if err != nil {
		return "", err
	}
	return profile.ID, nil
}

func (s *accountService) CurrentProfile(_ context.Context) (app.Profile, error) {
	if s.cachedID != "" {
		// still fetch full profile to keep display/bio current
	}

	data, err := s.client.Get("/api/v1/accounts/verify_credentials")
	if err != nil {
		return app.Profile{}, fmt.Errorf("fetching account: %w", err)
	}

	var acct struct {
		ID          string `json:"id"`
		Acct        string `json:"acct"`
		DisplayName string `json:"display_name"`
		Note        string `json:"note"`
	}
	if err := json.Unmarshal(data, &acct); err != nil {
		return app.Profile{}, fmt.Errorf("parsing account: %w", err)
	}

	s.cachedID = acct.ID
	return app.Profile{
		ID:          acct.ID,
		Username:    sanitizeForTerminal(acct.Acct),
		DisplayName: sanitizeForTerminal(acct.DisplayName),
		Bio:         stripHTML(acct.Note),
	}, nil
}

func (s *accountService) UpdateProfile(_ context.Context, displayName, bio string) error {
	form := url.Values{}
	form.Set("display_name", strings.TrimSpace(displayName))
	form.Set("note", strings.TrimSpace(bio))

	_, err := s.client.Patch("/api/v1/accounts/update_credentials", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}
	return nil
}

func (s *accountService) BlockUser(_ context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("invalid account id")
	}
	path := fmt.Sprintf("/api/v1/accounts/%s/block", accountID)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("blocking user: %w", err)
	}
	return nil
}
