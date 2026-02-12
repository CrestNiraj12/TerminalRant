package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"terminalrant/app"
	"terminalrant/domain"
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
		ID             string `json:"id"`
		Acct           string `json:"acct"`
		DisplayName    string `json:"display_name"`
		Note           string `json:"note"`
		StatusesCount  int    `json:"statuses_count"`
		FollowersCount int    `json:"followers_count"`
		FollowingCount int    `json:"following_count"`
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
		PostsCount:  acct.StatusesCount,
		Followers:   acct.FollowersCount,
		Following:   acct.FollowingCount,
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

func (s *accountService) FollowUser(_ context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("invalid account id")
	}
	path := fmt.Sprintf("/api/v1/accounts/%s/follow", accountID)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("following user: %w", err)
	}
	return nil
}

func (s *accountService) UnfollowUser(_ context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("invalid account id")
	}
	path := fmt.Sprintf("/api/v1/accounts/%s/unfollow", accountID)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("unfollowing user: %w", err)
	}
	return nil
}

func (s *accountService) LookupFollowing(_ context.Context, accountIDs []string) (map[string]bool, error) {
	res := make(map[string]bool)
	if len(accountIDs) == 0 {
		return res, nil
	}
	form := url.Values{}
	seen := make(map[string]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		form.Add("id[]", id)
	}
	if len(seen) == 0 {
		return res, nil
	}
	data, err := s.client.Get("/api/v1/accounts/relationships?" + form.Encode())
	if err != nil {
		return nil, fmt.Errorf("fetching relationships: %w", err)
	}
	var rels []struct {
		ID        string `json:"id"`
		Following bool   `json:"following"`
	}
	if err := json.Unmarshal(data, &rels); err != nil {
		return nil, fmt.Errorf("parsing relationships: %w", err)
	}
	for _, r := range rels {
		res[sanitizeForTerminal(r.ID)] = r.Following
	}
	return res, nil
}

func (s *accountService) ProfileByID(_ context.Context, accountID string) (app.Profile, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return app.Profile{}, fmt.Errorf("invalid account id")
	}
	path := fmt.Sprintf("/api/v1/accounts/%s", accountID)
	data, err := s.client.Get(path)
	if err != nil {
		return app.Profile{}, fmt.Errorf("fetching profile: %w", err)
	}
	var acct struct {
		ID             string `json:"id"`
		Acct           string `json:"acct"`
		DisplayName    string `json:"display_name"`
		Note           string `json:"note"`
		StatusesCount  int    `json:"statuses_count"`
		FollowersCount int    `json:"followers_count"`
		FollowingCount int    `json:"following_count"`
	}
	if err := json.Unmarshal(data, &acct); err != nil {
		return app.Profile{}, fmt.Errorf("parsing profile: %w", err)
	}
	return app.Profile{
		ID:          sanitizeForTerminal(acct.ID),
		Username:    sanitizeForTerminal(acct.Acct),
		DisplayName: sanitizeForTerminal(acct.DisplayName),
		Bio:         stripHTML(acct.Note),
		PostsCount:  acct.StatusesCount,
		Followers:   acct.FollowersCount,
		Following:   acct.FollowingCount,
	}, nil
}

func (s *accountService) PostsByAccount(_ context.Context, accountID string, limit int, maxID string) ([]domain.Rant, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("invalid account id")
	}
	if limit <= 0 {
		limit = 20
	}
	path := fmt.Sprintf("/api/v1/accounts/%s/statuses?limit=%d", accountID, limit)
	if strings.TrimSpace(maxID) != "" {
		path += "&max_id=" + url.QueryEscape(maxID)
	}
	data, err := s.client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("fetching profile posts: %w", err)
	}

	var statuses []mastodonStatus
	if err := json.Unmarshal(data, &statuses); err != nil {
		return nil, fmt.Errorf("parsing profile posts: %w", err)
	}

	rants := make([]domain.Rant, 0, len(statuses))
	for _, st := range statuses {
		createdAt, _ := time.Parse(time.RFC3339, st.CreatedAt)
		author := sanitizeForTerminal(st.Account.DisplayName)
		if author == "" {
			author = sanitizeForTerminal(st.Account.Acct)
		}
		inReplyToID := ""
		if st.InReplyToID != nil {
			inReplyToID = fmt.Sprintf("%v", st.InReplyToID)
		}
		rants = append(rants, domain.Rant{
			ID:           st.ID,
			AccountID:    st.Account.ID,
			Author:       author,
			Username:     sanitizeForTerminal(st.Account.Acct),
			Content:      stripHTML(st.Content),
			CreatedAt:    createdAt,
			URL:          sanitizeForTerminal(st.URL),
			IsOwn:        s.cachedID != "" && st.Account.ID == s.cachedID,
			Liked:        st.Favourited,
			LikesCount:   st.FavouritesCount,
			RepliesCount: st.RepliesCount,
			InReplyToID:  inReplyToID,
			Media:        mapMediaAttachments(st.MediaAttachments),
		})
	}
	return rants, nil
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

func (s *accountService) ListBlockedUsers(_ context.Context, limit int) ([]app.BlockedUser, error) {
	if limit <= 0 {
		limit = 40
	}
	path := fmt.Sprintf("/api/v1/blocks?limit=%d", limit)
	data, err := s.client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("fetching blocked users: %w", err)
	}

	var blocked []struct {
		ID          string `json:"id"`
		Acct        string `json:"acct"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(data, &blocked); err != nil {
		return nil, fmt.Errorf("parsing blocked users: %w", err)
	}

	out := make([]app.BlockedUser, 0, len(blocked))
	for _, u := range blocked {
		out = append(out, app.BlockedUser{
			AccountID:   sanitizeForTerminal(u.ID),
			Username:    sanitizeForTerminal(u.Acct),
			DisplayName: sanitizeForTerminal(u.DisplayName),
		})
	}
	return out, nil
}

func (s *accountService) UnblockUser(_ context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("invalid account id")
	}
	path := fmt.Sprintf("/api/v1/accounts/%s/unblock", accountID)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("unblocking user: %w", err)
	}
	return nil
}
