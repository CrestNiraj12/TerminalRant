package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
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
	ID               string                    `json:"id"`
	Content          string                    `json:"content"` // HTML
	CreatedAt        string                    `json:"created_at"`
	URL              string                    `json:"url"`
	Account          mastodonAccount           `json:"account"`
	Favourited       bool                      `json:"favourited"`
	FavouritesCount  int                       `json:"favourites_count"`
	RepliesCount     int                       `json:"replies_count"`
	InReplyToID      interface{}               `json:"in_reply_to_id"` // Can be string or null
	MediaAttachments []mastodonMediaAttachment `json:"media_attachments"`
}

type mastodonAccount struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Acct        string `json:"acct"`
}

type mastodonMediaAttachment struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	PreviewURL  string `json:"preview_url"`
	Description string `json:"description"`
	Meta        struct {
		Original struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"original"`
	} `json:"meta"`
}

func (s *timelineService) FetchByHashtag(_ context.Context, hashtag string, limit int) ([]domain.Rant, error) {
	return s.FetchByHashtagPage(context.Background(), hashtag, limit, "")
}

func (s *timelineService) FetchByHashtagPage(_ context.Context, hashtag string, limit int, maxID string) ([]domain.Rant, error) {
	path := fmt.Sprintf("/api/v1/timelines/tag/%s?limit=%d", hashtag, limit)
	if maxID != "" {
		path += "&max_id=" + url.QueryEscape(maxID)
	}
	return s.fetchTimelinePath(path)
}

func (s *timelineService) FetchHomePage(_ context.Context, limit int, maxID string) ([]domain.Rant, error) {
	path := fmt.Sprintf("/api/v1/timelines/home?limit=%d", limit)
	if maxID != "" {
		path += "&max_id=" + url.QueryEscape(maxID)
	}
	return s.fetchTimelinePath(path)
}

func (s *timelineService) FetchPublicPage(_ context.Context, limit int, maxID string) ([]domain.Rant, error) {
	path := fmt.Sprintf("/api/v1/timelines/public?limit=%d", limit)
	if maxID != "" {
		path += "&max_id=" + url.QueryEscape(maxID)
	}
	return s.fetchTimelinePath(path)
}

func (s *timelineService) FetchTrendingPage(_ context.Context, limit int, maxID string) ([]domain.Rant, error) {
	if limit <= 0 {
		limit = 20
	}
	// Trends endpoint is typically not pageable by max_id. We use trends for
	// the first page, then public timeline pagination for older pages.
	if maxID == "" {
		path := fmt.Sprintf("/api/v1/trends/statuses?limit=%d", limit)
		rants, err := s.fetchTimelinePath(path)
		if err == nil && len(rants) > 0 {
			return rants, nil
		}
		return s.FetchPublicPage(context.Background(), limit, "")
	}

	// When maxID originated from trends IDs, some servers may return nothing.
	// If that happens, retry with the first public page to establish paging.
	rants, err := s.FetchPublicPage(context.Background(), limit, maxID)
	if err != nil {
		return nil, err
	}
	if len(rants) == 0 {
		return s.FetchPublicPage(context.Background(), limit, "")
	}
	return rants, nil
}

func (s *timelineService) fetchTimelinePath(path string) ([]domain.Rant, error) {
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
			AccountID:    st.Account.ID,
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
			Media:        mapMediaAttachments(st.MediaAttachments),
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
			AccountID:    st.Account.ID,
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
			Media:        mapMediaAttachments(st.MediaAttachments),
		})
	}
	return rants
}

func mapMediaAttachments(in []mastodonMediaAttachment) []domain.MediaAttachment {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.MediaAttachment, 0, len(in))
	for _, m := range in {
		out = append(out, domain.MediaAttachment{
			ID:          sanitizeForTerminal(m.ID),
			Type:        sanitizeForTerminal(m.Type),
			URL:         sanitizeForTerminal(m.URL),
			PreviewURL:  sanitizeForTerminal(m.PreviewURL),
			Description: sanitizeForTerminal(strings.TrimSpace(m.Description)),
			Width:       int(m.Meta.Original.Width),
			Height:      int(m.Meta.Original.Height),
		})
	}
	return out
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
	// Decode HTML entities like &lt; &gt; &amp;
	s = html.UnescapeString(s)
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
