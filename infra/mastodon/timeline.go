package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"terminalrant/domain"
)

// timelineService implements app.TimelineService using the Mastodon API.
type timelineService struct {
	client           *Client
	currentAccountID string // Set after init to mark own posts.
}

// NewTimelineService creates a TimelineService backed by Mastodon.
// Pass currentAccountID to mark the user's own posts in the feed.
func NewTimelineService(client *Client, currentAccountID string) *timelineService {
	return &timelineService{
		client:           client,
		currentAccountID: currentAccountID,
	}
}

// mastodonStatus is the subset of Mastodon's Status entity we care about.
type mastodonStatus struct {
	ID              string          `json:"id"`
	Content         string          `json:"content"` // HTML
	CreatedAt       string          `json:"created_at"`
	URL             string          `json:"url"`
	Account         mastodonAccount `json:"account"`
	Favourited      bool            `json:"favourited"`
	FavouritesCount int             `json:"favourites_count"`
	RepliesCount    int             `json:"replies_count"`
	InReplyToID     interface{}     `json:"in_reply_to_id"` // Can be string or null
}

type mastodonAccount struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Acct        string `json:"acct"`
}

func (s *timelineService) FetchByHashtag(_ context.Context, hashtag string, limit int) ([]domain.Rant, error) {
	path := fmt.Sprintf("/api/v1/timelines/tag/%s?limit=%d", hashtag, limit)

	data, err := s.client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("fetching timeline: %w", err)
	}

	var statuses []mastodonStatus
	if err := json.Unmarshal(data, &statuses); err != nil {
		return nil, fmt.Errorf("parsing timeline: %w", err)
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
			Author:       author,
			Username:     sanitizeForTerminal(st.Account.Acct),
			Content:      stripHTML(st.Content),
			CreatedAt:    createdAt,
			URL:          sanitizeForTerminal(st.URL),
			IsOwn:        s.currentAccountID != "" && st.Account.ID == s.currentAccountID,
			Liked:        st.Favourited,
			LikesCount:   st.FavouritesCount,
			RepliesCount: st.RepliesCount,
			InReplyToID:  inReplyToID,
		})
	}

	return rants, nil
}

type mastodonContext struct {
	Ancestors   []mastodonStatus `json:"ancestors"`
	Descendants []mastodonStatus `json:"descendants"`
}

func (s *timelineService) FetchThread(_ context.Context, id string) (ancestors, descendants []domain.Rant, err error) {
	path := fmt.Sprintf("/api/v1/statuses/%s/context", id)

	data, err := s.client.Get(path)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching thread: %w", err)
	}

	var ctx mastodonContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, nil, fmt.Errorf("parsing thread: %w", err)
	}

	ancestors = s.mapStatuses(ctx.Ancestors)
	descendants = s.mapStatuses(ctx.Descendants)

	return ancestors, descendants, nil
}

func (s *timelineService) mapStatuses(statuses []mastodonStatus) []domain.Rant {
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
			Author:       author,
			Username:     sanitizeForTerminal(st.Account.Acct),
			Content:      stripHTML(st.Content),
			CreatedAt:    createdAt,
			URL:          sanitizeForTerminal(st.URL),
			IsOwn:        s.currentAccountID != "" && st.Account.ID == s.currentAccountID,
			Liked:        st.Favourited,
			LikesCount:   st.FavouritesCount,
			RepliesCount: st.RepliesCount,
			InReplyToID:  inReplyToID,
		})
	}
	return rants
}

// stripHTML removes HTML tags and decodes common entities.
// Good enough for terminal display; not a security boundary.
var (
	htmlTagRe   = regexp.MustCompile(`<[^>]*>`)
	lineBreakRe = regexp.MustCompile(`(?i)</p>|<br\s*/?>`)
	ansiCSIRe   = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	ansiOSCRe   = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	ansiEscRe   = regexp.MustCompile(`\x1b[@-_]`)
)

func stripHTML(s string) string {
	// Replace paragraph ends and breaks with newlines
	s = lineBreakRe.ReplaceAllString(s, "\n")
	// Strip all remaining tags
	s = htmlTagRe.ReplaceAllString(s, "")
	return sanitizeForTerminal(s)
}

// sanitizeForTerminal removes ANSI escape sequences and control chars that can
// alter terminal behavior. It preserves newlines/tabs for readable formatting.
func sanitizeForTerminal(s string) string {
	s = ansiOSCRe.ReplaceAllString(s, "")
	s = ansiCSIRe.ReplaceAllString(s, "")
	s = ansiEscRe.ReplaceAllString(s, "")

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r >= 0x20 && r != 0x7f && !(r >= 0x80 && r <= 0x9f):
			b.WriteRune(r)
		}
	}
	return b.String()
}
