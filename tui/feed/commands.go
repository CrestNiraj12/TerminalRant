package feed

import (
	"context"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/app"
	"github.com/CrestNiraj12/terminalrant/domain"
)

func (m Model) fetchThread(id string) tea.Cmd {
	timeline := m.timeline
	return func() tea.Msg {
		ancestors, descendants, err := timeline.FetchThread(context.Background(), id)
		if err != nil {
			return ThreadErrorMsg{ID: id, Err: err}
		}
		return ThreadLoadedMsg{ID: id, Ancestors: ancestors, Descendants: descendants}
	}
}

func openURL(rawURL string) tea.Cmd {
	return func() tea.Msg {
		if !isSafeExternalURL(rawURL) {
			return nil
		}
		_ = exec.Command("open", rawURL).Start()
		return nil
	}
}

func openURLs(urls []string) tea.Cmd {
	clean := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !isSafeExternalURL(u) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		clean = append(clean, u)
	}
	if len(clean) == 0 {
		return nil
	}
	return func() tea.Msg {
		for _, u := range clean {
			_ = exec.Command("open", u).Start()
		}
		return nil
	}
}

func isSafeExternalURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func (m Model) deleteRant(id string) tea.Cmd {
	timeline := m.timeline
	_ = timeline // Need to use PostService for delete.
	// We'll pass the PostService via Cmd return for root to handle,
	// or provide it here if it's in the model.
	// Actually, Feed model only has TimelineService.
	// Let's emit a msg for the root to handle the actual deletion.
	return func() tea.Msg {
		return DeleteRantMsg{ID: id}
	}
}

type DeleteRantMsg struct {
	ID string
}

func (m Model) fetchRants(reqSeq int) tea.Cmd {
	timeline := m.timeline
	account := m.account
	hashtag := m.hashtag
	defaultHashtag := m.defaultHashtag
	source := m.feedSource
	queryKey := m.currentFeedQueryKey()
	recentFollows := append([]string{}, m.recentFollows...)
	return func() tea.Msg {
		var (
			rants []domain.Rant
			err   error
		)
		switch source {
		case sourceTerminalRant:
			rants, err = timeline.FetchByHashtag(context.Background(), defaultHashtag, defaultLimit)
		case sourceCustomHashtag:
			rants, err = timeline.FetchByHashtag(context.Background(), hashtag, defaultLimit)
		case sourceTrending:
			rants, err = timeline.FetchTrendingPage(context.Background(), defaultLimit, "")
		case sourceFollowing:
			rants, err = timeline.FetchHomePage(context.Background(), defaultLimit, "")
			if err == nil && len(rants) == 0 && len(recentFollows) > 0 && account != nil {
				seeded := make([]domain.Rant, 0, defaultLimit)
				seen := make(map[string]struct{}, defaultLimit)
				for _, accountID := range recentFollows {
					posts, perr := account.PostsByAccount(context.Background(), accountID, 5, "")
					if perr != nil {
						continue
					}
					for _, p := range posts {
						if _, ok := seen[p.ID]; ok {
							continue
						}
						seen[p.ID] = struct{}{}
						seeded = append(seeded, p)
					}
					if len(seeded) >= defaultLimit {
						break
					}
				}
				sort.SliceStable(seeded, func(i, j int) bool {
					return seeded[i].CreatedAt.After(seeded[j].CreatedAt)
				})
				if len(seeded) > defaultLimit {
					seeded = seeded[:defaultLimit]
				}
				rants = seeded
			}
		}
		if err != nil {
			return RantsErrorMsg{Err: err, QueryKey: queryKey, ReqSeq: reqSeq}
		}
		return RantsLoadedMsg{Rants: rants, QueryKey: queryKey, RawCount: len(rants), ReqSeq: reqSeq}
	}
}

func (m *Model) loadMoreReplies() {
	if !m.hasMoreReplies {
		return
	}
	next := min(len(m.replyAll), m.replyVisible+replyPageSize)
	m.replyVisible = next
	m.replies = m.replyAll[:m.replyVisible]
	m.hasMoreReplies = m.replyVisible < len(m.replyAll)
	m.ensureDetailCursorVisible()
}

func (m Model) fetchOlderRants(reqSeq int) tea.Cmd {
	if m.loading || !m.hasMoreFeed || m.oldestFeedID == "" {
		return nil
	}
	timeline := m.timeline
	hashtag := m.hashtag
	defaultHashtag := m.defaultHashtag
	source := m.feedSource
	maxID := m.oldestFeedID
	queryKey := m.currentFeedQueryKey()
	return func() tea.Msg {
		var (
			rants []domain.Rant
			err   error
		)
		switch source {
		case sourceTerminalRant:
			rants, err = timeline.FetchByHashtagPage(context.Background(), defaultHashtag, defaultLimit, maxID)
		case sourceCustomHashtag:
			rants, err = timeline.FetchByHashtagPage(context.Background(), hashtag, defaultLimit, maxID)
		case sourceTrending:
			rants, err = timeline.FetchTrendingPage(context.Background(), defaultLimit, maxID)
		case sourceFollowing:
			rants, err = timeline.FetchHomePage(context.Background(), defaultLimit, maxID)
		}
		if err != nil {
			return RantsPageErrorMsg{Err: err, QueryKey: queryKey, ReqSeq: reqSeq}
		}
		return RantsPageLoadedMsg{Rants: rants, QueryKey: queryKey, RawCount: len(rants), ReqSeq: reqSeq}
	}
}

func filterOutOwnRants(in []domain.Rant) []domain.Rant {
	if len(in) == 0 {
		return in
	}
	out := make([]domain.Rant, 0, len(in))
	for _, r := range in {
		if r.IsOwn {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (m Model) fetchRelationshipsForRants(rants []domain.Rant) tea.Cmd {
	if m.account == nil || len(rants) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rants))
	seen := make(map[string]struct{}, len(rants))
	for _, r := range rants {
		id := strings.TrimSpace(r.AccountID)
		if id == "" || r.IsOwn {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	acct := m.account
	return func() tea.Msg {
		following, err := acct.LookupFollowing(context.Background(), ids)
		return RelationshipsLoadedMsg{Following: following, Err: err}
	}
}

func (m Model) fetchProfile(accountID string) tea.Cmd {
	if m.account == nil || strings.TrimSpace(accountID) == "" {
		return nil
	}
	acct := m.account
	accountID = strings.TrimSpace(accountID)
	return func() tea.Msg {
		var (
			profile app.Profile
			posts   []domain.Rant
			perr    error
			serr    error
		)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			profile, perr = acct.ProfileByID(context.Background(), accountID)
		}()
		go func() {
			defer wg.Done()
			posts, serr = acct.PostsByAccount(context.Background(), accountID, defaultLimit, "")
		}()
		wg.Wait()
		if perr != nil {
			return ProfileLoadedMsg{AccountID: accountID, Err: perr}
		}
		if serr != nil {
			return ProfileLoadedMsg{AccountID: accountID, Err: serr}
		}
		return ProfileLoadedMsg{
			AccountID: accountID,
			Profile:   profile,
			Posts:     posts,
		}
	}
}

func (m Model) fetchOwnProfile() tea.Cmd {
	if m.account == nil {
		return nil
	}
	acct := m.account
	return func() tea.Msg {
		profile, err := acct.CurrentProfile(context.Background())
		if err != nil {
			return ProfileLoadedMsg{Err: err}
		}
		posts, err := acct.PostsByAccount(context.Background(), profile.ID, defaultLimit, "")
		if err != nil {
			return ProfileLoadedMsg{AccountID: profile.ID, Err: err}
		}
		return ProfileLoadedMsg{
			AccountID: profile.ID,
			Profile:   profile,
			Posts:     posts,
		}
	}
}

// Rants returns the current rants for external access.
