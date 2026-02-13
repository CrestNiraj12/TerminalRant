package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"terminalrant/domain"
)

// postService implements app.PostService using the Mastodon API.
type postService struct {
	client *Client
}

const requiredHashtag = domain.AppHashTag

// NewPostService creates a PostService backed by Mastodon.
func NewPostService(client *Client) *postService {
	return &postService{client: client}
}

func (s *postService) Post(_ context.Context, content string, _ string) (domain.Rant, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Rant{}, domain.ErrEmptyRant
	}

	content = ensureRequiredHashtag(content)

	form := url.Values{}
	form.Set("status", content)
	form.Set("visibility", "public")

	data, err := s.client.Post("/api/v1/statuses", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.Rant{}, fmt.Errorf("posting rant: %w", err)
	}

	return s.parseStatus(data)
}

func (s *postService) Edit(_ context.Context, id string, content string, _ string) (domain.Rant, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Rant{}, domain.ErrEmptyRant
	}

	content = ensureRequiredHashtag(content)

	form := url.Values{}
	form.Set("status", content)

	path := fmt.Sprintf("/api/v1/statuses/%s", id)
	data, err := s.client.Put(path, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.Rant{}, fmt.Errorf("editing rant: %w", err)
	}

	return s.parseStatus(data)
}

func (s *postService) Delete(_ context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/statuses/%s", id)
	_, err := s.client.Delete(path)
	if err != nil {
		return fmt.Errorf("deleting rant: %w", err)
	}
	return nil
}

func (s *postService) Like(_ context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/statuses/%s/favourite", id)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("liking rant: %w", err)
	}
	return nil
}

func (s *postService) Unlike(_ context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/statuses/%s/unfavourite", id)
	_, err := s.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("unliking rant: %w", err)
	}
	return nil
}

func (s *postService) Reply(_ context.Context, parentID string, content string, _ string) (domain.Rant, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Rant{}, domain.ErrEmptyRant
	}

	content = ensureRequiredHashtag(content)

	form := url.Values{}
	form.Set("status", content)
	form.Set("in_reply_to_id", parentID)
	form.Set("visibility", "public")

	data, err := s.client.Post("/api/v1/statuses", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.Rant{}, fmt.Errorf("replying to rant: %w", err)
	}

	return s.parseStatus(data)
}

func ensureRequiredHashtag(content string) string {
	if strings.Contains(strings.ToLower(content), requiredHashtag) {
		return content
	}
	return content + "\n\n" + requiredHashtag
}

func (s *postService) parseStatus(data []byte) (domain.Rant, error) {
	var st mastodonStatus
	if err := json.Unmarshal(data, &st); err != nil {
		return domain.Rant{}, fmt.Errorf("parsing status response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, st.CreatedAt)

	author := sanitizeForTerminal(st.Account.DisplayName)
	if author == "" {
		author = sanitizeForTerminal(st.Account.Acct)
	}

	return domain.Rant{
		ID:           st.ID,
		AccountID:    st.Account.ID,
		Author:       author,
		Username:     sanitizeForTerminal(st.Account.Acct),
		Content:      stripHTML(st.Content),
		CreatedAt:    createdAt,
		URL:          sanitizeForTerminal(st.URL),
		Liked:        st.Favourited,
		LikesCount:   st.FavouritesCount,
		RepliesCount: st.RepliesCount,
		InReplyToID:  fmt.Sprintf("%v", st.InReplyToID),
		Media:        mapMediaAttachments(st.MediaAttachments),
	}, nil
}
