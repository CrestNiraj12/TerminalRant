package feed

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminalrant/app"
	"terminalrant/domain"
	"terminalrant/tui/common"
)

const (
	defaultLimit    = 20
	replyPageSize   = 20
	prefetchTrigger = 3
	creatorGitHub   = "https://github.com/CrestNiraj12"
)

// RantsLoadedMsg is sent when the timeline fetch completes successfully.
type RantsLoadedMsg struct {
	Rants    []domain.Rant
	QueryKey string
	RawCount int
	ReqSeq   int
}

// RantsErrorMsg is sent when the timeline fetch fails.
type RantsErrorMsg struct {
	Err      error
	QueryKey string
	ReqSeq   int
}

// RantsPageLoadedMsg is sent when an older feed page is loaded.
type RantsPageLoadedMsg struct {
	Rants    []domain.Rant
	QueryKey string
	RawCount int
	ReqSeq   int
}

// RantsPageErrorMsg is sent when loading an older feed page fails.
type RantsPageErrorMsg struct {
	Err      error
	QueryKey string
	ReqSeq   int
}

type BlockUserMsg struct {
	AccountID string
	Username  string
}

type BlockResultMsg struct {
	AccountID string
	Username  string
	Err       error
}

type FollowToggleMsg struct {
	AccountID string
	Username  string
	Follow    bool
}

type FollowToggleResultMsg struct {
	AccountID string
	Username  string
	Follow    bool
	Err       error
}

type HideAuthorPostsMsg struct {
	AccountID string
}

type RequestBlockedUsersMsg struct{}

type BlockedUsersLoadedMsg struct {
	Users []app.BlockedUser
	Err   error
}

type UnblockUserMsg struct {
	AccountID string
	Username  string
}

type UnblockResultMsg struct {
	AccountID string
	Username  string
	Err       error
}

type RelationshipsLoadedMsg struct {
	Following map[string]bool
	Err       error
}

type OpenProfileMsg struct {
	AccountID string
}

type ProfileLoadedMsg struct {
	AccountID string
	Profile   app.Profile
	Posts     []domain.Rant
	Err       error
}

// EditRantMsg is sent when the user selects 'Edit' from the action menu.
type EditRantMsg struct {
	Rant      domain.Rant
	UseInline bool
}

// DeleteResultMsg is sent after a rant deletion attempt.
type DeleteResultMsg struct {
	ID  string
	Err error
}

// LikeRantMsg is sent when the user wants to like a rant.
type LikeRantMsg struct {
	ID       string
	WasLiked bool
}

// LikeResultMsg is sent after a like attempt.
type LikeResultMsg struct {
	ID  string
	Err error
}

// ReplyRantMsg is sent when the user wants to reply to a rant.
type ReplyRantMsg struct {
	Rant      domain.Rant
	UseInline bool
}

// ThreadLoadedMsg is sent when a thread (ancestors and replies) is loaded.
type ThreadLoadedMsg struct {
	ID          string
	Ancestors   []domain.Rant
	Descendants []domain.Rant
}

type MediaPreviewLoadedMsg struct {
	URL     string
	Preview string
	Err     error
}

// ThreadErrorMsg is sent when a thread fetch fails.
type ThreadErrorMsg struct {
	ID  string
	Err error
}

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

type ResetFeedStateMsg struct {
	ForceReset bool
}

type OpenDetailWithoutRepliesMsg struct {
	ID string
}

type threadData struct {
	Ancestors   []domain.Rant
	Descendants []domain.Rant
}

type feedSource int

const (
	sourceTerminalRant feedSource = iota
	sourceTrending
	sourceFollowing
	sourceCustomHashtag
)

type FeedPrefsChangedMsg struct {
	Hashtag string
	Source  string
}

type PrefsSavedMsg struct {
	Err error
}

// ResultMsg is a generic success/fail result for creation or update.
type ResultMsg struct {
	ID         string // Local or Server ID
	Rant       domain.Rant
	IsEdit     bool
	Err        error
	OldContent string
}

// --- Optimistic Update Messages ---

type AddOptimisticRantMsg struct {
	Content string
}

type UpdateOptimisticRantMsg struct {
	ID      string
	Content string
}

type DeleteOptimisticRantMsg struct {
	ID string
}

// --- Status Types ---

type RantStatus int

const (
	StatusNormal RantStatus = iota
	StatusPendingCreate
	StatusPendingUpdate
	StatusPendingDelete
	StatusFailed
)

type RantItem struct {
	Rant       domain.Rant
	Status     RantStatus
	Err        error
	OldContent string // For rollback
}

// --- Model ---

// Model holds the state for the feed (timeline) view.
type Model struct {
	timeline         app.TimelineService
	account          app.AccountService
	defaultHashtag   string
	hashtag          string
	feedSource       feedSource
	rants            []RantItem
	cursor           int
	loading          bool
	loadingMore      bool
	hasMoreFeed      bool
	oldestFeedID     string
	err              error
	keys             common.KeyMap
	spinner          spinner.Model
	confirmDelete    bool // Whether we are in the 'Are you sure?' delete step
	showDetail       bool // Whether we are in full-post view
	width            int  // Terminal width
	height           int  // Terminal height
	startIndex       int  // First visible item in the list (for scrolling)
	scrollLine       int  // Line-based scroll for feed viewport
	ancestors        []domain.Rant
	replies          []domain.Rant
	replyAll         []domain.Rant
	replyVisible     int
	hasMoreReplies   bool
	loadingReplies   bool
	detailCursor     int // 0 for main post, 1...n for replies
	detailScrollLine int
	focusedRant      *domain.Rant
	threadCache      map[string]threadData
	viewStack        []*domain.Rant // To support going back in deep threading
	showAllHints     bool
	pagingNotice     string
	hiddenIDs        map[string]bool
	hiddenAuthors    map[string]bool
	showHidden       bool
	confirmBlock     bool
	blockAccountID   string
	blockUsername    string
	confirmFollow    bool
	followAccountID  string
	followUsername   string
	followTarget     bool
	followingByID    map[string]bool
	recentFollows    []string
	followingDirty   bool
	showBlocked      bool
	loadingBlocked   bool
	blockedErr       error
	blockedUsers     []app.BlockedUser
	blockedCursor    int
	confirmUnblock   bool
	unblockTarget    app.BlockedUser
	hashtagInput     bool
	hashtagBuffer    string
	detailStart      int
	showProfile      bool
	profileIsOwn     bool
	profileLoading   bool
	profileErr       error
	profile          app.Profile
	profilePosts     []domain.Rant
	profileCursor    int // 0 for profile card, 1...n for profile posts
	profileStart     int // first visible profile post
	showMediaPreview bool
	mediaPreview     map[string]string
	mediaLoading     map[string]bool
	feedReqSeq       int
}

// New creates a feed model with injected dependencies.
func New(timeline app.TimelineService, account app.AccountService, hashtag, initialSource string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
	tag := strings.TrimSpace(strings.TrimPrefix(hashtag, "#"))
	if tag == "" {
		tag = "terminalrant"
	}
	source := parseFeedSource(initialSource)
	if source == sourceCustomHashtag && strings.EqualFold(tag, "terminalrant") {
		source = sourceTerminalRant
	}

	return Model{
		timeline:         timeline,
		account:          account,
		defaultHashtag:   "terminalrant",
		hashtag:          tag,
		feedSource:       source,
		keys:             common.DefaultKeyMap(),
		spinner:          s,
		loading:          true,
		hasMoreFeed:      true,
		threadCache:      make(map[string]threadData),
		followingByID:    make(map[string]bool),
		hiddenIDs:        make(map[string]bool),
		hiddenAuthors:    make(map[string]bool),
		showMediaPreview: true,
		mediaPreview:     make(map[string]string),
		mediaLoading:     make(map[string]bool),
	}
}

// Init starts the initial feed fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRants(m.feedReqSeq),
		m.spinner.Tick,
	)
}

// Refresh returns a Cmd that re-fetches the timeline.
func (m Model) Refresh() tea.Cmd {
	return m.fetchRants(m.feedReqSeq)
}

func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		// Use 'open' for Mac. For Linux 'xdg-open', Windows 'rundll32'.
		// Since user is on Mac, 'open' is safe.
		_ = exec.Command("open", url).Start()
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

// Update handles messages for the feed view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureFeedCursorVisible()
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case RantsLoadedMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		// Reconciliation: Merge remote results with inflight optimistic items.
		rants := msg.Rants
		if m.feedSource == sourceFollowing {
			rants = filterOutOwnRants(rants)
		}
		newRants := make([]RantItem, len(rants))
		for i, r := range rants {
			newRants[i] = RantItem{Rant: r, Status: StatusNormal}
		}

		// Keep Pending items that aren't reconciled yet.
		var pendingItems []RantItem
		for _, ri := range m.rants {
			if ri.Status == StatusNormal || ri.Status == StatusFailed {
				continue
			}

			// For PendingUpdate/Delete, try to find the item in the new list.
			found := false
			if ri.Status == StatusPendingUpdate || ri.Status == StatusPendingDelete {
				for i, nr := range newRants {
					if nr.Rant.ID == ri.Rant.ID {
						found = true
						if ri.Status == StatusPendingDelete {
							// Successfully deleted on server but still in list?
						} else {
							// For simplicity, server wins once it arrives.
							ri.Status = StatusNormal
							newRants[i] = ri
						}
						break
					}
				}
			} else if ri.Status == StatusPendingCreate {
				// Match by content for creation. Fuzzy match to ignore hashtags/whitespace.
				for _, nr := range newRants {
					if strings.Contains(nr.Rant.Content, ri.Rant.Content) ||
						strings.Contains(ri.Rant.Content, nr.Rant.Content) {
						found = true
						break
					}
				}
			}

			if !found {
				pendingItems = append(pendingItems, ri)
			}
		}

		m.rants = append(pendingItems, newRants...)
		m.normalizeFeedOrder()
		m.loading = false
		m.loadingMore = false
		m.err = nil
		m.pagingNotice = ""
		m.oldestFeedID = m.lastFeedID()
		if m.feedSource == sourceTrending {
			m.hasMoreFeed = len(msg.Rants) > 0
		} else if m.feedSource == sourceFollowing {
			raw := msg.RawCount
			if raw == 0 {
				raw = len(msg.Rants)
			}
			m.hasMoreFeed = raw == defaultLimit
		} else {
			m.hasMoreFeed = len(rants) == defaultLimit
		}
		if m.cursor >= len(m.rants) {
			m.cursor = 0
		}
		if m.feedSource == sourceFollowing {
			m.followingDirty = false
		}
		m.ensureFeedCursorVisible()
		return m, tea.Batch(m.ensureMediaPreviewCmd(), m.fetchRelationshipsForRants(msg.Rants))

	case RantsErrorMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		m.loading = false
		m.loadingMore = false
		m.err = msg.Err
		return m, nil

	case RantsPageLoadedMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		anchorScroll := m.scrollLine
		anchorID := ""
		if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
			anchorID = m.rants[m.cursor].Rant.ID
		}
		m.loadingMore = false
		m.err = nil
		rants := msg.Rants
		if m.feedSource == sourceFollowing {
			rants = filterOutOwnRants(rants)
		}
		if len(rants) == 0 && msg.RawCount == 0 {
			m.hasMoreFeed = false
			if len(m.rants) > 0 {
				m.pagingNotice = "🚀 End of the rantverse reached."
			}
			return m, nil
		}
		existing := make(map[string]struct{}, len(m.rants))
		for _, ri := range m.rants {
			existing[ri.Rant.ID] = struct{}{}
		}
		added := 0
		for _, r := range rants {
			if _, ok := existing[r.ID]; ok {
				continue
			}
			m.rants = append(m.rants, RantItem{Rant: r, Status: StatusNormal})
			added++
		}
		m.oldestFeedID = m.lastFeedID()
		if m.feedSource == sourceTrending {
			m.hasMoreFeed = added > 0
		} else if m.feedSource == sourceFollowing {
			raw := msg.RawCount
			if raw == 0 {
				raw = len(msg.Rants)
			}
			m.hasMoreFeed = raw == defaultLimit
		} else {
			m.hasMoreFeed = len(rants) == defaultLimit && added > 0
		}
		if added == 0 && len(m.rants) > 0 && m.feedSource != sourceFollowing {
			m.hasMoreFeed = false
			m.pagingNotice = "🚀 End of the rantverse reached."
		} else if m.hasMoreFeed {
			m.pagingNotice = ""
		}
		if anchorID != "" {
			m.setCursorByID(anchorID)
		}
		m.scrollLine = anchorScroll
		if m.scrollLine < 0 {
			m.scrollLine = 0
		}
		return m, m.fetchRelationshipsForRants(msg.Rants)

	case RantsPageErrorMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		m.loadingMore = false
		m.err = msg.Err
		return m, nil

	case AddOptimisticRantMsg:
		newItem := RantItem{
			Rant: domain.Rant{
				ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
				Content:   msg.Content,
				Author:    "You", // Generic placeholder
				Username:  "you",
				IsOwn:     true,
				CreatedAt: time.Now(),
			},
			Status: StatusPendingCreate,
		}

		m.rants = append([]RantItem{newItem}, m.rants...)
		m.cursor = 0     // Focus the new item
		m.startIndex = 0 // Scroll to top
		m.scrollLine = 0
		return m, nil

	case ResetFeedStateMsg:
		if msg.ForceReset {
			m.showDetail = false
			m.confirmDelete = false
			m.confirmBlock = false
			m.confirmFollow = false
			m.blockAccountID = ""
			m.blockUsername = ""
			m.followAccountID = ""
			m.followUsername = ""
			m.followTarget = false
			m.replies = nil
			m.replyAll = nil
			m.replyVisible = 0
			m.hasMoreReplies = false
			m.ancestors = nil
			m.loadingReplies = false
			m.focusedRant = nil
			m.viewStack = nil
			m.detailStart = 0
			m.detailScrollLine = 0
			m.showBlocked = false
			m.loadingBlocked = false
			m.blockedErr = nil
			m.blockedUsers = nil
			m.blockedCursor = 0
			m.confirmUnblock = false
			m.unblockTarget = app.BlockedUser{}
			m.showProfile = false
			m.profileIsOwn = false
			m.profileLoading = false
			m.profileErr = nil
			m.profile = app.Profile{}
			m.profilePosts = nil
			m.profileCursor = 0
			m.profileStart = 0
		}
		return m, nil

	case OpenDetailWithoutRepliesMsg:
		if msg.ID != "" {
			m.setCursorByID(msg.ID)
		}
		if len(m.rants) == 0 {
			return m, nil
		}
		m.showDetail = true
		m.detailCursor = 0
		m.detailStart = 0
		m.detailScrollLine = 0
		m.replies = nil
		m.replyAll = nil
		m.replyVisible = 0
		m.hasMoreReplies = false
		m.ancestors = nil
		m.loadingReplies = false
		m.focusedRant = nil
		m.viewStack = nil
		return m, m.ensureMediaPreviewCmd()

	case ThreadLoadedMsg:
		replies := organizeThreadReplies(msg.ID, msg.Descendants)
		m.threadCache[msg.ID] = threadData{
			Ancestors:   msg.Ancestors,
			Descendants: replies,
		}

		// Ignore stale async responses for previously focused posts.
		if msg.ID != m.currentThreadRootID() {
			return m, nil
		}

		m.replyAll = replies
		m.replyVisible = minInt(replyPageSize, len(m.replyAll))
		m.hasMoreReplies = m.replyVisible < len(m.replyAll)
		m.replies = m.replyAll[:m.replyVisible]
		m.ancestors = msg.Ancestors
		m.loadingReplies = false
		m.ensureDetailCursorVisible()
		all := append([]domain.Rant{}, msg.Ancestors...)
		all = append(all, replies...)
		return m, tea.Batch(m.ensureMediaPreviewCmd(), m.fetchRelationshipsForRants(all))

	case ThreadErrorMsg:
		if msg.ID != m.currentThreadRootID() {
			return m, nil
		}
		m.loadingReplies = false
		return m, nil

	case MediaPreviewLoadedMsg:
		delete(m.mediaLoading, msg.URL)
		if msg.Err != nil {
			m.mediaPreview[msg.URL] = ""
			return m, nil
		}
		m.mediaPreview[msg.URL] = msg.Preview
		return m, nil

	case HideAuthorPostsMsg:
		if msg.AccountID == "" {
			return m, nil
		}
		m.hiddenAuthors[msg.AccountID] = true
		m.ensureVisibleCursor()
		m.ensureFeedCursorVisible()
		return m, nil

	case BlockResultMsg:
		m.confirmBlock = false
		m.blockAccountID = ""
		m.blockUsername = ""
		if msg.Err == nil && msg.AccountID != "" {
			m.hiddenAuthors[msg.AccountID] = true
			m.ensureVisibleCursor()
			m.ensureFeedCursorVisible()
		}
		return m, nil

	case RelationshipsLoadedMsg:
		if msg.Err != nil {
			return m, nil
		}
		for id, following := range msg.Following {
			m.followingByID[id] = following
		}
		return m, nil

	case ProfileLoadedMsg:
		m.profileLoading = false
		m.profileErr = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.profile = msg.Profile
		m.profilePosts = msg.Posts
		m.profileCursor = 0
		m.profileStart = 0
		m.ensureProfileCursorVisible()
		if msg.Profile.ID != "" {
			// Best-effort local relationship hydration for profile header.
			if following, ok := m.followingByID[msg.Profile.ID]; ok {
				_ = following
			}
		}
		return m, m.fetchRelationshipsForRants(msg.Posts)

	case FollowToggleResultMsg:
		m.confirmFollow = false
		m.followAccountID = ""
		m.followUsername = ""
		if msg.Err != nil {
			return m, nil
		}
		if strings.TrimSpace(msg.AccountID) != "" {
			m.followingByID[msg.AccountID] = msg.Follow
			if msg.Follow {
				m.addRecentFollow(msg.AccountID)
			} else {
				m.removeRecentFollow(msg.AccountID)
			}
			if m.showProfile && m.profile.ID == msg.AccountID {
				if msg.Follow {
					m.profile.Followers++
				} else if m.profile.Followers > 0 {
					m.profile.Followers--
				}
			}
		}
		m.followingDirty = true
		if !msg.Follow {
			// Immediately hide unfollowed authors from the Following tab.
			m.ensureVisibleCursor()
			m.ensureFeedCursorVisible()
		}
		if m.feedSource == sourceFollowing {
			m.rants = nil
			m.cursor = 0
			m.startIndex = 0
			m.scrollLine = 0
			m.oldestFeedID = ""
			m.hasMoreFeed = true
			m.loading = true
			m.loadingMore = false
			m.pagingNotice = ""
			m.feedReqSeq++
			return m, m.fetchRants(m.feedReqSeq)
		}
		return m, nil

	case BlockedUsersLoadedMsg:
		m.loadingBlocked = false
		m.blockedErr = msg.Err
		m.blockedUsers = msg.Users
		if m.blockedCursor >= len(m.blockedUsers) {
			m.blockedCursor = 0
		}
		return m, nil

	case UnblockResultMsg:
		m.confirmUnblock = false
		m.unblockTarget = app.BlockedUser{}
		if msg.Err != nil {
			m.blockedErr = msg.Err
			return m, nil
		}
		m.blockedErr = nil
		filtered := make([]app.BlockedUser, 0, len(m.blockedUsers))
		for _, u := range m.blockedUsers {
			if u.AccountID == msg.AccountID {
				continue
			}
			filtered = append(filtered, u)
		}
		m.blockedUsers = filtered
		delete(m.hiddenAuthors, msg.AccountID)
		if m.blockedCursor >= len(m.blockedUsers) && m.blockedCursor > 0 {
			m.blockedCursor--
		}
		m.pagingNotice = "Unblocked @" + msg.Username
		return m, nil

	case AddOptimisticReplyMsg:
		reply := domain.Rant{
			ID:          fmt.Sprintf("local-reply-%d", time.Now().UnixNano()),
			Content:     msg.Content,
			Author:      "You",
			Username:    "you",
			IsOwn:       true,
			CreatedAt:   time.Now(),
			InReplyToID: msg.ParentID,
		}
		if !m.showDetail {
			return m, nil
		}
		threadID := m.currentThreadRootID()
		if threadID == "" || !m.belongsToCurrentThread(msg.ParentID) {
			return m, nil
		}
		m.replyAll = append(m.replyAll, reply)
		m.replyVisible = len(m.replyAll)
		m.replies = m.replyAll
		m.hasMoreReplies = false
		if data, ok := m.threadCache[threadID]; ok {
			data.Descendants = append(data.Descendants, reply)
			m.threadCache[threadID] = data
		}
		return m, nil

	case LikeRantMsg:
		m.applyLikeToggle(msg.ID)
		m.toggleLikeInThreadCache(msg.ID)
		return m, nil

	case LikeResultMsg:
		if msg.Err != nil {
			// Rollback by toggling again.
			m.applyLikeToggle(msg.ID)
			m.toggleLikeInThreadCache(msg.ID)
		}
		return m, nil

	case UpdateOptimisticRantMsg:
		for i, ri := range m.rants {
			if ri.Rant.ID == msg.ID {
				ri.OldContent = ri.Rant.Content
				ri.Rant.Content = msg.Content
				ri.Status = StatusPendingUpdate
				m.rants[i] = ri
				break
			}
		}
		return m, nil

	case ResultMsg:
		if msg.Err != nil {
			// Find the item and set to Failed.
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					ri.Status = StatusFailed
					ri.Err = msg.Err
					if msg.IsEdit {
						ri.Rant.Content = ri.OldContent // Rollback
					}
					m.rants[i] = ri
					break
				}
			}
		} else {
			if msg.Rant.InReplyToID != "" && msg.Rant.InReplyToID != "<nil>" && msg.Rant.InReplyToID != "0" {
				m.reconcileReplyResult(msg.ID, msg.Rant)
				return m, nil
			}
			// Success: replace optimistic item with server version.
			for i, ri := range m.rants {
				// Match by ID OR fuzzy content (for new posts)
				if ri.Rant.ID == msg.ID || (!msg.IsEdit && (strings.Contains(msg.Rant.Content, ri.Rant.Content) || strings.Contains(ri.Rant.Content, msg.Rant.Content))) {
					ri.Rant = msg.Rant
					ri.Status = StatusNormal
					m.rants[i] = ri
					break
				}
			}
			// Also check replies
			for i, r := range m.replies {
				if r.ID == msg.ID {
					m.replies[i] = msg.Rant
					break
				}
			}
		}
		return m, nil

	case DeleteResultMsg:
		if msg.Err != nil {
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					ri.Status = StatusFailed
					ri.Err = msg.Err
					m.rants[i] = ri
					break
				}
			}
		} else {
			// Success: remove from list.
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					m.rants = append(m.rants[:i], m.rants[i+1:]...)
					if m.cursor >= len(m.rants) && m.cursor > 0 {
						m.cursor--
					}
					break
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.showAllHints {
			if key.Matches(msg, m.keys.ToggleHints) || msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.showAllHints = false
			}
			return m, nil
		}
		if m.showProfile {
			if m.confirmFollow && msg.String() != "y" && msg.String() != "n" {
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
			}
			switch {
			case key.Matches(msg, m.keys.ToggleHints):
				m.showAllHints = true
				return m, nil
			case key.Matches(msg, m.keys.Home):
				// h: jump to top within profile view.
				m.profileCursor = 0
				m.profileStart = 0
				return m, nil
			case msg.String() == "H":
				// H: go to feed home.
				m.showProfile = false
				m.profileIsOwn = false
				m.profileLoading = false
				m.profileErr = nil
				m.profile = app.Profile{}
				m.profilePosts = nil
				m.profileCursor = 0
				m.profileStart = 0
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				m.cursor = 0
				m.startIndex = 0
				m.scrollLine = 0
				m.showBlocked = false
				m.confirmUnblock = false
				m.unblockTarget = app.BlockedUser{}
				return m, nil
			case msg.String() == "esc" || msg.String() == "q":
				m.showProfile = false
				m.profileIsOwn = false
				m.profileLoading = false
				m.profileErr = nil
				m.profile = app.Profile{}
				m.profilePosts = nil
				m.profileCursor = 0
				m.profileStart = 0
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				return m, nil
			case key.Matches(msg, m.keys.Up):
				if m.profileCursor > 0 {
					m.profileCursor--
				}
				m.ensureProfileCursorVisible()
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.profileCursor < len(m.profilePosts) {
					m.profileCursor++
				}
				m.ensureProfileCursorVisible()
				return m, nil
			case key.Matches(msg, m.keys.FollowUser):
				if strings.TrimSpace(m.profile.ID) == "" || m.profileIsOwn {
					return m, nil
				}
				already := m.followingByID[m.profile.ID]
				if already {
					// Unfollow requires confirmation.
					m.confirmFollow = true
					m.followAccountID = m.profile.ID
					m.followUsername = m.profile.Username
					m.followTarget = false
					return m, nil
				}
				// Follow immediately.
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				accountID := m.profile.ID
				username := m.profile.Username
				return m, func() tea.Msg {
					return FollowToggleMsg{AccountID: accountID, Username: username, Follow: true}
				}
			case key.Matches(msg, m.keys.ManageBlocks):
				m.showBlocked = true
				m.loadingBlocked = true
				m.blockedErr = nil
				m.blockedUsers = nil
				m.blockedCursor = 0
				m.confirmUnblock = false
				m.unblockTarget = app.BlockedUser{}
				return m, func() tea.Msg { return RequestBlockedUsersMsg{} }
			case msg.String() == "enter":
				if m.profileCursor > 0 && m.profileCursor <= len(m.profilePosts) {
					target := m.profilePosts[m.profileCursor-1]
					m.showProfile = false
					m.profileIsOwn = false
					m.setCursorByID(target.ID)
					m.showDetail = true
					m.detailCursor = 0
					m.detailStart = 0
					m.detailScrollLine = 0
					m.replies = nil
					m.replyAll = nil
					m.replyVisible = 0
					m.hasMoreReplies = false
					m.ancestors = nil
					m.loadingReplies = true
					// Open exactly the selected profile post, even if it isn't in the current feed slice.
					m.focusedRant = &target
					m.viewStack = nil
					return m, m.loadThreadFromCacheOrFetch(target.ID)
				}
				return m, nil
			case msg.String() == "y":
				if m.confirmFollow && m.followAccountID != "" {
					accountID := m.followAccountID
					username := m.followUsername
					follow := m.followTarget
					return m, func() tea.Msg {
						return FollowToggleMsg{
							AccountID: accountID,
							Username:  username,
							Follow:    follow,
						}
					}
				}
				return m, nil
			case msg.String() == "n":
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				return m, nil
			}
			return m, nil
		}
		if m.showBlocked {
			switch {
			case msg.String() == "esc" || msg.String() == "q":
				m.showBlocked = false
				m.confirmUnblock = false
				m.unblockTarget = app.BlockedUser{}
				return m, nil
			case key.Matches(msg, m.keys.Up):
				if m.blockedCursor > 0 {
					m.blockedCursor--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.blockedCursor < len(m.blockedUsers)-1 {
					m.blockedCursor++
				}
				return m, nil
			case msg.String() == "u":
				if len(m.blockedUsers) == 0 || m.blockedCursor < 0 || m.blockedCursor >= len(m.blockedUsers) {
					return m, nil
				}
				m.confirmUnblock = true
				m.unblockTarget = m.blockedUsers[m.blockedCursor]
				return m, nil
			case msg.String() == "y":
				if m.confirmUnblock && m.unblockTarget.AccountID != "" {
					target := m.unblockTarget
					m.confirmUnblock = false
					m.unblockTarget = app.BlockedUser{}
					return m, func() tea.Msg {
						return UnblockUserMsg{AccountID: target.AccountID, Username: target.Username}
					}
				}
				return m, nil
			case msg.String() == "n":
				if m.confirmUnblock {
					m.confirmUnblock = false
					m.unblockTarget = app.BlockedUser{}
				}
				return m, nil
			}
			return m, nil
		}
		if m.confirmFollow && msg.String() != "y" && msg.String() != "n" && msg.String() != "q" && msg.String() != "esc" {
			m.confirmFollow = false
			m.followAccountID = ""
			m.followUsername = ""
			m.followTarget = false
		}
		if m.hashtagInput {
			switch msg.String() {
			case "esc":
				m.hashtagInput = false
				m.hashtagBuffer = ""
				return m, nil
			case "enter":
				tag := strings.TrimSpace(strings.TrimPrefix(m.hashtagBuffer, "#"))
				m.hashtagInput = false
				if tag == "" {
					m.hashtagBuffer = ""
					return m, nil
				}
				m.hashtag = tag
				if strings.EqualFold(tag, m.defaultHashtag) {
					m.feedSource = sourceTerminalRant
				} else {
					m.feedSource = sourceCustomHashtag
				}
				m.hashtagBuffer = ""
				m.prepareSourceChange()
				m.pagingNotice = "Switched to #" + tag
			m.feedReqSeq++
			return m, tea.Batch(
				m.fetchRants(m.feedReqSeq),
				m.emitPrefsChanged(),
			)
			case "backspace":
				if len(m.hashtagBuffer) > 0 {
					r := []rune(m.hashtagBuffer)
					m.hashtagBuffer = string(r[:len(r)-1])
				}
				return m, nil
			}
			if len(msg.Runes) > 0 {
				m.hashtagBuffer += string(msg.Runes)
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.ToggleHints):
			m.showAllHints = true
			return m, nil

		case msg.String() == "i":
			m.showMediaPreview = !m.showMediaPreview
			if m.showMediaPreview {
				return m, m.ensureMediaPreviewCmd()
			}
			return m, nil

		case msg.String() == "I":
			r := m.getSelectedRant()
			if m.showDetail {
				if m.focusedRant != nil {
					r = *m.focusedRant
				} else if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
					r = m.rants[m.cursor].Rant
				}
			}
			if urls := mediaOpenURLs(r.Media); len(urls) > 0 {
				return m, openURLs(urls)
			}
			m.pagingNotice = "No media on selected post."
			return m, nil

		case key.Matches(msg, m.keys.SwitchFeed):
			if m.showDetail {
				m.pagingNotice = "Exit detail view to switch tabs."
				return m, nil
			}
			if m.hasCustomTab() {
				m.feedSource = (m.feedSource + 1) % 4
			} else {
				m.feedSource = (m.feedSource + 1) % 3
			}
			m.prepareSourceChange()
			switch m.feedSource {
			case sourceTerminalRant:
				m.pagingNotice = "Feed: #terminalrant"
			case sourceTrending:
				m.pagingNotice = "Feed: trending"
			case sourceFollowing:
				m.pagingNotice = "Feed: following"
			case sourceCustomHashtag:
				m.pagingNotice = "Feed: #" + m.hashtag
			}
			m.feedReqSeq++
			return m, tea.Batch(m.fetchRants(m.feedReqSeq), m.emitPrefsChanged())

		case msg.String() == "T":
			if m.showDetail {
				m.pagingNotice = "Exit detail view to switch tabs."
				return m, nil
			}
			if m.hasCustomTab() {
				m.feedSource = (m.feedSource + 3) % 4
			} else {
				m.feedSource = (m.feedSource + 2) % 3
			}
			m.prepareSourceChange()
			switch m.feedSource {
			case sourceTerminalRant:
				m.pagingNotice = "Feed: #terminalrant"
			case sourceTrending:
				m.pagingNotice = "Feed: trending"
			case sourceFollowing:
				m.pagingNotice = "Feed: following"
			case sourceCustomHashtag:
				m.pagingNotice = "Feed: #" + m.hashtag
			}
			m.feedReqSeq++
			return m, tea.Batch(m.fetchRants(m.feedReqSeq), m.emitPrefsChanged())

		case key.Matches(msg, m.keys.SetHashtag):
			if m.showDetail {
				m.showDetail = false
				m.focusedRant = nil
				m.viewStack = nil
				m.detailCursor = 0
				m.detailStart = 0
				m.detailScrollLine = 0
				m.cursor = 0
				return m, nil
			}
			m.hashtagInput = true
			m.hashtagBuffer = m.hashtag
			return m, nil

		case key.Matches(msg, m.keys.ManageBlocks):
			m.showBlocked = true
			m.loadingBlocked = true
			m.blockedErr = nil
			m.blockedUsers = nil
			m.blockedCursor = 0
			m.confirmUnblock = false
			m.unblockTarget = app.BlockedUser{}
			return m, func() tea.Msg { return RequestBlockedUsersMsg{} }

		case key.Matches(msg, m.keys.ShowHidden):
			m.showHidden = !m.showHidden
			if m.showHidden {
				m.pagingNotice = "Showing hidden posts"
			} else {
				m.pagingNotice = "Hidden posts concealed"
			}
			m.ensureVisibleCursor()
			m.ensureFeedCursorVisible()
			return m, nil

		case key.Matches(msg, m.keys.HidePost):
			if m.showDetail {
				break
			}
			sel, ok := m.selectedVisibleRant()
			if !ok {
				break
			}
			m.hiddenIDs[sel.ID] = true
			m.pagingNotice = "Post hidden (X to toggle hidden)"
			m.ensureVisibleCursor()
			m.ensureFeedCursorVisible()
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			if m.showDetail {
				id := m.currentThreadRootID()
				delete(m.threadCache, id)
				m.replies = nil
				m.replyAll = nil
				m.replyVisible = 0
				m.hasMoreReplies = false
				m.ancestors = nil
				m.loadingReplies = true
				return m, m.fetchThread(id)
			}
			m.loading = true
			m.loadingMore = false
			m.hasMoreFeed = true
			m.oldestFeedID = ""
			m.pagingNotice = ""
			m.feedReqSeq++
			return m, m.fetchRants(m.feedReqSeq)

		case key.Matches(msg, m.keys.Up):
			if m.showDetail {
				if m.detailCursor > 0 {
					m.detailCursor--
				}
				if m.detailScrollLine > 0 {
					m.detailScrollLine--
				}
				return m, m.ensureMediaPreviewCmd()
			}
			m.confirmDelete = false
			m.moveCursorVisible(-1)
			m.ensureFeedCursorVisible()
			return m, m.ensureMediaPreviewCmd()
		case key.Matches(msg, m.keys.Down):
			if m.showDetail {
				gate := m.detailReplyGate()
				if m.detailCursor == 0 && m.detailScrollLine < gate {
					m.detailScrollLine++
					return m, m.ensureMediaPreviewCmd()
				}
				if m.detailCursor < len(m.replies) {
					m.detailCursor++
				}
				if m.hasMoreReplies && m.detailCursor >= len(m.replies)-prefetchTrigger {
					m.loadMoreReplies()
				}
				m.detailScrollLine++
				return m, m.ensureMediaPreviewCmd()
			}
			m.confirmDelete = false
			m.moveCursorVisible(1)
			m.ensureFeedCursorVisible()
			if !m.loading && len(m.rants) > 0 && m.oldestFeedID != "" && m.hasMoreFeed && !m.loadingMore && m.cursor >= len(m.rants)-prefetchTrigger {
				m.loadingMore = true
				m.feedReqSeq++
				return m, tea.Batch(m.fetchOlderRants(m.feedReqSeq), m.ensureMediaPreviewCmd())
			}
			return m, m.ensureMediaPreviewCmd()

		case key.Matches(msg, m.keys.Home):
			if m.showDetail {
				m.detailScrollLine = 0
				m.detailCursor = 0
				return m, m.ensureMediaPreviewCmd()
			}
			m.showDetail = false
			m.showProfile = false
			m.profileIsOwn = false
			m.confirmDelete = false
			m.confirmBlock = false
			m.confirmFollow = false
			m.blockAccountID = ""
			m.blockUsername = ""
			m.followAccountID = ""
			m.followUsername = ""
			m.followTarget = false
			m.cursor = 0
			m.startIndex = 0
			m.scrollLine = 0
			m.detailCursor = 0
			m.detailStart = 0
			m.detailScrollLine = 0
			m.showBlocked = false
			m.confirmUnblock = false
			m.unblockTarget = app.BlockedUser{}
			return m, nil

		case msg.String() == "enter":
			if len(m.rants) > 0 {
				if !m.showDetail {
					m.showDetail = true
					m.detailCursor = 0
					m.detailStart = 0
					m.detailScrollLine = 0
					m.replies = nil
					m.replyAll = nil
					m.replyVisible = 0
					m.hasMoreReplies = false
					m.ancestors = nil
					m.loadingReplies = true
					m.focusedRant = nil
					m.viewStack = nil
					return m, m.loadThreadFromCacheOrFetch(m.rants[m.cursor].Rant.ID)
				} else if m.detailCursor > 0 && m.detailCursor <= len(m.replies) {
					// Deep threading: focus the selected reply
					selected := m.replies[m.detailCursor-1]
					m.viewStack = append(m.viewStack, m.focusedRant)
					m.focusedRant = &selected
					m.detailCursor = 0
					m.detailStart = 0
					m.detailScrollLine = 0
					m.replies = nil
					m.replyAll = nil
					m.replyVisible = 0
					m.hasMoreReplies = false
					m.ancestors = nil

					m.loadingReplies = true
					return m, m.loadThreadFromCacheOrFetch(selected.ID)
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.LoadMore):
			if m.confirmDelete {
				m.confirmDelete = false
				return m, nil
			}
			if m.confirmBlock {
				m.confirmBlock = false
				m.blockAccountID = ""
				m.blockUsername = ""
				return m, nil
			}
			if m.confirmFollow {
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				return m, nil
			}
			if m.showDetail {
				m.loadMoreReplies()
				return m, nil
			}
			if len(m.rants) == 0 {
				return m, nil
			}
			if m.loading || m.loadingMore {
				m.pagingNotice = "⏳ Loading older posts..."
				return m, nil
			}
			if !m.hasMoreFeed || m.oldestFeedID == "" {
				m.pagingNotice = "🗂️ No older posts left."
				return m, nil
			}
			if m.hasMoreFeed && m.oldestFeedID != "" {
				m.loadingMore = true
				m.feedReqSeq++
				return m, m.fetchOlderRants(m.feedReqSeq)
			}
			return m, nil

		case key.Matches(msg, m.keys.Open):
			if len(m.rants) > 0 {
				r := m.getSelectedRant()
				if r.URL != "" {
					return m, openURL(r.URL)
				}
			}

		case key.Matches(msg, m.keys.GitHub):
			return m, openURL(creatorGitHub)

		case key.Matches(msg, m.keys.Edit):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.Rant.IsOwn {
				return m, func() tea.Msg { return EditRantMsg{Rant: r.Rant, UseInline: false} }
			}

		case key.Matches(msg, m.keys.EditInline):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.Rant.IsOwn {
				return m, func() tea.Msg { return EditRantMsg{Rant: r.Rant, UseInline: true} }
			}

		case key.Matches(msg, m.keys.Like):
			if len(m.rants) == 0 {
				break
			}
			selected := m.getSelectedRant()
			return m, func() tea.Msg {
				return LikeRantMsg{
					ID:       selected.ID,
					WasLiked: selected.Liked,
				}
			}

		case key.Matches(msg, m.keys.Reply):
			if len(m.rants) == 0 {
				break
			}
			return m, func() tea.Msg { return ReplyRantMsg{Rant: m.getSelectedRant(), UseInline: false} }

		case key.Matches(msg, m.keys.ReplyInline):
			if len(m.rants) == 0 {
				break
			}
			return m, func() tea.Msg { return ReplyRantMsg{Rant: m.getSelectedRant(), UseInline: true} }

		case key.Matches(msg, m.keys.BlockUser):
			r := m.getSelectedRant()
			if r.AccountID == "" || r.IsOwn {
				m.pagingNotice = "Cannot block this user."
				break
			}
			m.confirmBlock = true
			m.blockAccountID = r.AccountID
			m.blockUsername = r.Username
			return m, nil

		case key.Matches(msg, m.keys.FollowUser):
			r := m.getSelectedRant()
			if r.AccountID == "" || r.IsOwn {
				m.pagingNotice = "Cannot follow this user."
				break
			}
			m.confirmFollow = true
			m.followAccountID = r.AccountID
			m.followUsername = r.Username
			m.followTarget = !m.isFollowing(r.AccountID)
			m.confirmBlock = false
			m.blockAccountID = ""
			m.blockUsername = ""
			return m, nil

		case key.Matches(msg, m.keys.OpenProfile):
			r := m.getSelectedRant()
			if strings.TrimSpace(r.AccountID) == "" {
				break
			}
			m.showProfile = true
			m.profileIsOwn = r.IsOwn
			m.profileLoading = true
			m.profileErr = nil
			m.profile = app.Profile{}
			m.profilePosts = nil
			m.profileCursor = 0
			m.profileStart = 0
			return m, m.fetchProfile(r.AccountID)

		case key.Matches(msg, m.keys.OpenOwnProfile):
			m.showProfile = true
			m.profileIsOwn = true
			m.profileLoading = true
			m.profileErr = nil
			m.profile = app.Profile{}
			m.profilePosts = nil
			m.profileCursor = 0
			m.profileStart = 0
			return m, m.fetchOwnProfile()

		case key.Matches(msg, m.keys.Delete):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.Rant.IsOwn {
				m.confirmDelete = true
			}

		case msg.String() == "esc", msg.String() == "q":
			if m.confirmBlock {
				m.confirmBlock = false
				m.blockAccountID = ""
				m.blockUsername = ""
				return m, nil
			}
			if m.confirmFollow {
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
				return m, nil
			}
			if m.showDetail {
				if len(m.viewStack) > 0 {
					// Pop from stack
					m.focusedRant = m.viewStack[len(m.viewStack)-1]
					m.viewStack = m.viewStack[:len(m.viewStack)-1]
					m.detailCursor = 0
					m.detailStart = 0
					m.detailScrollLine = 0

					id := m.rants[m.cursor].Rant.ID
					if m.focusedRant != nil {
						id = m.focusedRant.ID
					}

					m.replies = nil
					m.replyAll = nil
					m.replyVisible = 0
					m.hasMoreReplies = false
					m.ancestors = nil
					m.loadingReplies = true
					return m, m.loadThreadFromCacheOrFetch(id)
				}
				m.showDetail = false
				m.focusedRant = nil
				m.viewStack = nil
				m.detailStart = 0
				m.detailScrollLine = 0
				return m, nil
			}
			if msg.String() == "q" {
				return m, tea.Quit
			}
			if m.confirmDelete {
				m.confirmDelete = false
				return m, nil
			}

		case msg.String() == "y":
			if m.confirmDelete {
				ri := m.rants[m.cursor]
				m.confirmDelete = false
				ri.Status = StatusPendingDelete
				m.rants[m.cursor] = ri
				return m, m.deleteRant(ri.Rant.ID)
			}
			if m.confirmBlock && m.blockAccountID != "" {
				accountID := m.blockAccountID
				username := m.blockUsername
				m.confirmBlock = false
				m.blockAccountID = ""
				m.blockUsername = ""
				return m, func() tea.Msg { return BlockUserMsg{AccountID: accountID, Username: username} }
			}
			if m.confirmFollow && m.followAccountID != "" {
				accountID := m.followAccountID
				username := m.followUsername
				follow := m.followTarget
				return m, func() tea.Msg {
					return FollowToggleMsg{AccountID: accountID, Username: username, Follow: follow}
				}
			}
		case msg.String() == "n":
			if m.confirmDelete {
				m.confirmDelete = false
			}
			if m.confirmBlock {
				m.confirmBlock = false
				m.blockAccountID = ""
				m.blockUsername = ""
			}
			if m.confirmFollow {
				m.confirmFollow = false
				m.followAccountID = ""
				m.followUsername = ""
				m.followTarget = false
			}
		case msg.String() == "u":
			if !m.showDetail {
				break
			}

			current := m.getSelectedRant()
			selected := m.getSelectedRant()
			parentID := selected.InReplyToID
			if parentID == "" || parentID == "<nil>" || parentID == "0" {
				if len(m.ancestors) == 0 {
					break
				}
				parentID = m.ancestors[len(m.ancestors)-1].ID
			}

			parent, ok := m.findRantByID(parentID)
			if !ok {
				break
			}

			previous := current
			m.viewStack = append(m.viewStack, &previous)
			m.focusedRant = &parent
			m.detailCursor = 0
			m.detailStart = 0
			m.detailScrollLine = 0
			m.replies = nil
			m.replyAll = nil
			m.replyVisible = 0
			m.hasMoreReplies = false
			m.ancestors = nil
			m.loadingReplies = true
			return m, m.loadThreadFromCacheOrFetch(parentID)
		}
	}

	return m, nil
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
	next := m.replyVisible + replyPageSize
	if next > len(m.replyAll) {
		next = len(m.replyAll)
	}
	m.replyVisible = next
	m.replies = m.replyAll[:m.replyVisible]
	m.hasMoreReplies = m.replyVisible < len(m.replyAll)
	m.ensureDetailCursorVisible()
}

func (m *Model) ensureMediaPreviewCmd() tea.Cmd {
	if !m.showMediaPreview {
		return nil
	}
	r := m.getSelectedRant()
	if m.showDetail {
		if m.focusedRant != nil {
			r = *m.focusedRant
		} else if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
			r = m.rants[m.cursor].Rant
		}
	}
	urls := mediaPreviewURLs(r.Media)
	if len(urls) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(urls))
	for _, url := range urls {
		if _, ok := m.mediaPreview[url]; ok {
			continue
		}
		if m.mediaLoading[url] {
			continue
		}
		m.mediaLoading[url] = true
		cmds = append(cmds, fetchMediaPreview(url))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func firstMediaPreviewURL(media []domain.MediaAttachment) string {
	urls := mediaPreviewURLs(media)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func firstMediaOpenURL(media []domain.MediaAttachment) string {
	urls := mediaOpenURLs(media)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func mediaPreviewURLs(media []domain.MediaAttachment) []string {
	out := make([]string, 0, len(media))
	seen := make(map[string]struct{}, len(media))
	for _, m := range media {
		t := strings.ToLower(strings.TrimSpace(m.Type))
		switch t {
		case "video", "image", "gifv":
			url := strings.TrimSpace(m.PreviewURL)
			if url == "" {
				url = strings.TrimSpace(m.URL)
			}
			if url == "" {
				continue
			}
			if _, ok := seen[url]; ok {
				continue
			}
			seen[url] = struct{}{}
			out = append(out, url)
		}
	}
	return out
}

func mediaOpenURLs(media []domain.MediaAttachment) []string {
	out := make([]string, 0, len(media))
	seen := make(map[string]struct{}, len(media))
	for _, m := range media {
		url := strings.TrimSpace(m.URL)
		if url == "" {
			url = strings.TrimSpace(m.PreviewURL)
		}
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	return out
}

func fetchMediaPreview(url string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 6 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return MediaPreviewLoadedMsg{URL: url, Err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return MediaPreviewLoadedMsg{URL: url, Err: fmt.Errorf("preview status %d", resp.StatusCode)}
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
		if err != nil {
			return MediaPreviewLoadedMsg{URL: url, Err: err}
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return MediaPreviewLoadedMsg{URL: url, Err: err}
		}
		return MediaPreviewLoadedMsg{
			URL:     url,
			Preview: renderANSIThumbnail(img, 8, 4),
		}
	}
}

func renderANSIThumbnail(img image.Image, w, h int) string {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return ""
	}
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}
	var out strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := b.Min.X + x*b.Dx()/w
			sy := b.Min.Y + y*b.Dy()/h
			c := color.NRGBAModel.Convert(img.At(sx, sy)).(color.NRGBA)
			out.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm  \x1b[0m", c.R, c.G, c.B))
		}
		if y < h-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
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
		profile, err := acct.ProfileByID(context.Background(), accountID)
		if err != nil {
			return ProfileLoadedMsg{AccountID: accountID, Err: err}
		}
		posts, err := acct.PostsByAccount(context.Background(), accountID, defaultLimit, "")
		if err != nil {
			return ProfileLoadedMsg{AccountID: accountID, Err: err}
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
func (m Model) Rants() []domain.Rant {
	res := make([]domain.Rant, len(m.rants))
	for i, r := range m.rants {
		res[i] = r.Rant
	}
	return res
}

func (m Model) getSelectedRant() domain.Rant {
	if m.showDetail {
		if m.detailCursor > 0 && m.detailCursor <= len(m.replies) {
			return m.replies[m.detailCursor-1]
		}
		if m.focusedRant != nil {
			return *m.focusedRant
		}
	}
	if len(m.rants) == 0 {
		return domain.Rant{}
	}
	if m.cursor < 0 || m.cursor >= len(m.rants) {
		return domain.Rant{}
	}
	return m.rants[m.cursor].Rant
}

func (m Model) getSelectedRantID() string {
	return m.getSelectedRant().ID
}

func (m Model) lastFeedID() string {
	if len(m.rants) == 0 {
		return ""
	}
	return m.rants[len(m.rants)-1].Rant.ID
}

func (m Model) currentThreadRootID() string {
	if m.focusedRant != nil {
		return m.focusedRant.ID
	}
	if len(m.rants) == 0 {
		return ""
	}
	return m.rants[m.cursor].Rant.ID
}

func (m *Model) normalizeFeedOrder() {
	if len(m.rants) < 2 {
		return
	}
	sort.SliceStable(m.rants, func(i, j int) bool {
		ti := m.rants[i].Rant.CreatedAt
		tj := m.rants[j].Rant.CreatedAt
		if ti.Equal(tj) {
			return m.rants[i].Rant.ID > m.rants[j].Rant.ID
		}
		return ti.After(tj)
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) feedViewportHeight() int {
	h := m.height - m.feedChromeLines()
	// App-level status/confirm bars are rendered outside feed.View().
	h -= 2
	if h < 4 {
		h = 4
	}
	return h
}

func (m Model) feedChromeLines() int {
	lineCount := func(s string) int {
		if s == "" {
			return 0
		}
		return strings.Count(s, "\n") + 1
	}

	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant") + common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	top := lineCount(title) + lineCount(m.renderTabs()) + 1 // trailing blank line under tabs

	bottom := 1 // spacer line before status/help block
	if m.loading && len(m.rants) > 0 {
		bottom++
	} else if m.loadingMore {
		bottom++
	}
	if m.pagingNotice != "" && len(m.rants) > 0 {
		bottom++
	}
	if m.hashtagInput {
		bottom++
	}
	bottom += lineCount(m.helpView())

	return top + bottom
}

func (m *Model) ensureFeedCursorVisible() {
	if m.showDetail {
		return
	}
	visible := m.visibleIndices()
	if len(visible) == 0 {
		m.scrollLine = 0
		return
	}
	m.ensureVisibleCursor()
	cardWidth, bodyWidth := m.feedCardWidthsForModel()
	top := 0
	bottom := 0
	linePos := 0
	for i, idx := range visible {
		if idx == m.cursor {
			lines := m.feedItemRenderedLines(m.rants[idx].Rant, cardWidth, bodyWidth)
			top = linePos
			bottom = linePos + lines - 1
			break
		}
		linePos += m.feedItemRenderedLines(m.rants[idx].Rant, cardWidth, bodyWidth)
		if i < len(visible)-1 {
			linePos += 1
		}
	}
	viewHeight := m.feedViewportHeight()
	if top < m.scrollLine {
		m.scrollLine = top
	} else if bottom >= m.scrollLine+viewHeight {
		m.scrollLine = bottom - viewHeight + 1
	}
	if m.scrollLine < 0 {
		m.scrollLine = 0
	}
}

func (m Model) feedItemRenderedLines(r domain.Rant, cardWidth, bodyWidth int) int {
	content, tags := splitContentAndTags(r.Content)
	if strings.TrimSpace(content) == "" && len(r.Media) > 0 {
		content = "(media post)"
	}
	author := common.AuthorStyle.Render("@" + r.Username)
	timestamp := common.TimestampStyle.Render(r.CreatedAt.Format("Jan 02 15:04"))
	replyIndicator := ""
	if r.InReplyToID != "" && r.InReplyToID != "<nil>" && r.InReplyToID != "0" {
		replyIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(" ↩ reply")
	}
	likeIcon := "♡"
	likeStyle := common.MetadataStyle
	if r.Liked {
		likeIcon = "♥"
		likeStyle = common.LikeActiveStyle
	}
	meta := fmt.Sprintf("%s %d  ↩ %d",
		likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)
	indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
	preview := truncateToTwoLinesForWidth(content, bodyWidth)
	previewLines := strings.Split(preview, "\n")
	var bodyBuilder strings.Builder
	for _, line := range previewLines {
		bodyBuilder.WriteString(indicator + common.ContentStyle.Render(line) + "\n")
	}
	body := strings.TrimSuffix(bodyBuilder.String(), "\n")
	tagLine := renderCompactTags(tags, 2)
	mediaLine := renderMediaCompact(r.Media)
	itemContent := fmt.Sprintf("%s  %s%s\n%s\n%s",
		author, timestamp, replyIndicator, body, common.MetadataStyle.Render(meta))
	if tagLine != "" {
		itemContent = fmt.Sprintf("%s  %s%s\n%s\n\n%s\n\n%s",
			author, timestamp, replyIndicator, body, tagLine, common.MetadataStyle.Render(meta))
	}
	if mediaLine != "" {
		itemContent = itemContent + "\n" + mediaLine
	}
	rendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(cardWidth).
		Render(itemContent)
	// Add spacer line between cards in list.
	return len(strings.Split(rendered, "\n")) + 1
}

func truncateToTwoLinesForWidth(text string, width int) string {
	if width < 12 {
		width = 12
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 2 {
		return wrapped
	}
	return strings.Join(lines[:2], "\n") + "..."
}

func (m Model) feedCardWidthsForModel() (cardWidth int, bodyWidth int) {
	showPreviewPanel := m.feedPreviewPanelVisible()
	available := m.width - 4
	if showPreviewPanel {
		available -= 58
	}
	if available < 44 {
		available = 44
	}
	cardWidth = available
	bodyWidth = cardWidth - 10
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	return cardWidth, bodyWidth
}

func (m Model) feedPreviewPanelVisible() bool {
	if !m.showMediaPreview {
		return false
	}
	r := m.getSelectedRant()
	urls := mediaPreviewURLs(r.Media)
	if len(urls) == 0 {
		return false
	}
	for _, url := range urls {
		if m.mediaLoading[url] {
			return true
		}
		if _, ok := m.mediaPreview[url]; ok {
			return true
		}
	}
	return false
}

func (m Model) currentFeedQueryKey() string {
	switch m.feedSource {
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "tag:" + strings.ToLower(strings.TrimSpace(m.hashtag))
	default:
		return "tag:" + strings.ToLower(strings.TrimSpace(m.defaultHashtag))
	}
}

func (m Model) detailReplyGate() int {
	r := m.getSelectedRant()
	if m.focusedRant != nil {
		r = *m.focusedRant
	}
	content, tags := splitContentAndTags(r.Content)
	if strings.TrimSpace(content) == "" && len(r.Media) > 0 {
		content = "(media post)"
	}
	contentLines := estimateWrappedLines(content, 66)
	mainLines := 18 + contentLines
	if len(tags) > 0 {
		mainLines += 3
	}
	if len(r.Media) > 0 {
		mainLines += 4
	}
	if len(m.ancestors) > 0 {
		mainLines += 6
	}
	viewHeight := m.height - 2
	if viewHeight < 8 {
		viewHeight = 8
	}
	gate := mainLines - (viewHeight - 4)
	if gate < 0 {
		gate = 0
	}
	return gate
}

func estimateWrappedLines(text string, width int) int {
	if width < 1 {
		width = 1
	}
	lines := 0
	for _, ln := range strings.Split(text, "\n") {
		r := []rune(ln)
		if len(r) == 0 {
			lines++
			continue
		}
		lines += (len(r)-1)/width + 1
	}
	if lines < 1 {
		lines = 1
	}
	return lines
}

func (m Model) loadThreadFromCacheOrFetch(id string) tea.Cmd {
	if data, ok := m.threadCache[id]; ok {
		return func() tea.Msg {
			return ThreadLoadedMsg{
				ID:          id,
				Ancestors:   data.Ancestors,
				Descendants: data.Descendants,
			}
		}
	}
	return m.fetchThread(id)
}

func (m Model) findRantByID(id string) (domain.Rant, bool) {
	for _, ri := range m.rants {
		if ri.Rant.ID == id {
			return ri.Rant, true
		}
	}
	for _, r := range m.replies {
		if r.ID == id {
			return r, true
		}
	}
	for _, r := range m.ancestors {
		if r.ID == id {
			return r, true
		}
	}
	if m.focusedRant != nil && m.focusedRant.ID == id {
		return *m.focusedRant, true
	}
	return domain.Rant{}, false
}

func (m *Model) setCursorByID(id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	for i := range m.rants {
		if m.rants[i].Rant.ID == id {
			m.cursor = i
			m.ensureVisibleCursor()
			m.ensureFeedCursorVisible()
			return
		}
	}
}

func (m *Model) toggleLikeInThreadCache(id string) {
	for key, data := range m.threadCache {
		updated := false
		for i := range data.Ancestors {
			if data.Ancestors[i].ID == id {
				if data.Ancestors[i].Liked {
					data.Ancestors[i].Liked = false
					data.Ancestors[i].LikesCount--
				} else {
					data.Ancestors[i].Liked = true
					data.Ancestors[i].LikesCount++
				}
				updated = true
			}
		}
		for i := range data.Descendants {
			if data.Descendants[i].ID == id {
				if data.Descendants[i].Liked {
					data.Descendants[i].Liked = false
					data.Descendants[i].LikesCount--
				} else {
					data.Descendants[i].Liked = true
					data.Descendants[i].LikesCount++
				}
				updated = true
			}
		}
		if updated {
			m.threadCache[key] = data
		}
	}
}

func (m *Model) applyLikeToggle(id string) {
	toggle := func(liked *bool, likesCount *int) {
		if *liked {
			*liked = false
			if *likesCount > 0 {
				*likesCount--
			}
		} else {
			*liked = true
			*likesCount++
		}
	}

	for i, ri := range m.rants {
		if ri.Rant.ID == id {
			toggle(&ri.Rant.Liked, &ri.Rant.LikesCount)
			m.rants[i] = ri
			break
		}
	}
	for i := range m.replies {
		if m.replies[i].ID == id {
			toggle(&m.replies[i].Liked, &m.replies[i].LikesCount)
			break
		}
	}
	for i := range m.replyAll {
		if m.replyAll[i].ID == id {
			toggle(&m.replyAll[i].Liked, &m.replyAll[i].LikesCount)
			break
		}
	}
	for i := range m.ancestors {
		if m.ancestors[i].ID == id {
			toggle(&m.ancestors[i].Liked, &m.ancestors[i].LikesCount)
			break
		}
	}
	if m.focusedRant != nil && m.focusedRant.ID == id {
		toggle(&m.focusedRant.Liked, &m.focusedRant.LikesCount)
	}
}

func (m Model) isHiddenRant(r domain.Rant) bool {
	if m.showHidden {
		return false
	}
	return m.isMarkedHidden(r)
}

func (m Model) isMarkedHidden(r domain.Rant) bool {
	if m.hiddenIDs[r.ID] {
		return true
	}
	if r.AccountID != "" && m.hiddenAuthors[r.AccountID] {
		return true
	}
	return false
}

func (m *Model) ensureDetailCursorVisible() {
	if !m.showDetail {
		m.detailStart = 0
		return
	}
	if m.detailCursor <= 0 {
		m.detailStart = 0
		return
	}
	slots := m.detailReplySlots()
	if slots < 1 {
		slots = 1
	}
	idx := m.detailCursor - 1
	if idx < m.detailStart {
		m.detailStart = idx
	}
	if idx >= m.detailStart+slots {
		m.detailStart = idx - slots + 1
	}
	maxStart := len(m.replies) - slots
	if maxStart < 0 {
		maxStart = 0
	}
	if m.detailStart > maxStart {
		m.detailStart = maxStart
	}
	if m.detailStart < 0 {
		m.detailStart = 0
	}
}

func (m Model) detailReplySlots() int {
	// Header + parent/main card + footer/hints leave room for reply window.
	h := m.height - 30
	if h < 20 {
		h = 20
	}
	slots := h / 5
	if slots < 4 {
		slots = 4
	}
	return slots
}

func (m *Model) ensureProfileCursorVisible() {
	if !m.showProfile {
		m.profileStart = 0
		return
	}
	if m.profileCursor <= 0 {
		m.profileStart = 0
		return
	}
	slots := m.profilePostSlots()
	if slots < 1 {
		slots = 1
	}
	idx := m.profileCursor - 1
	if idx < m.profileStart {
		m.profileStart = idx
	}
	if idx >= m.profileStart+slots {
		m.profileStart = idx - slots + 1
	}
	maxStart := len(m.profilePosts) - slots
	if maxStart < 0 {
		maxStart = 0
	}
	if m.profileStart > maxStart {
		m.profileStart = maxStart
	}
	if m.profileStart < 0 {
		m.profileStart = 0
	}
}

func (m Model) profilePostSlots() int {
	h := m.height - 30
	if h < 20 {
		h = 20
	}
	slots := h / 5
	if slots < 4 {
		slots = 4
	}
	return slots
}

func (m Model) belongsToCurrentThread(parentID string) bool {
	if parentID == "" {
		return false
	}
	threadID := m.currentThreadRootID()
	if parentID == threadID {
		return true
	}
	if m.focusedRant != nil && parentID == m.focusedRant.ID {
		return true
	}
	for _, r := range m.replies {
		if r.ID == parentID {
			return true
		}
	}
	for _, a := range m.ancestors {
		if a.ID == parentID {
			return true
		}
	}
	if data, ok := m.threadCache[threadID]; ok {
		for _, r := range data.Descendants {
			if r.ID == parentID {
				return true
			}
		}
		for _, a := range data.Ancestors {
			if a.ID == parentID {
				return true
			}
		}
	}
	return false
}

func (m *Model) reconcileReplyResult(localID string, server domain.Rant) {
	replace := func(list []domain.Rant) ([]domain.Rant, bool) {
		for i := range list {
			if list[i].ID == server.ID {
				list[i] = server
				return list, true
			}
			if strings.TrimSpace(localID) != "" && list[i].ID == localID {
				list[i] = server
				return list, true
			}
			if strings.HasPrefix(list[i].ID, "local-reply-") &&
				list[i].InReplyToID == server.InReplyToID &&
				strings.TrimSpace(list[i].Content) == strings.TrimSpace(server.Content) {
				list[i] = server
				return list, true
			}
		}
		return append(list, server), false
	}

	m.replyAll, _ = replace(m.replyAll)
	m.replyVisible = len(m.replyAll)
	m.replies = m.replyAll
	m.hasMoreReplies = false

	threadID := m.currentThreadRootID()
	if data, ok := m.threadCache[threadID]; ok {
		data.Descendants, _ = replace(data.Descendants)
		m.threadCache[threadID] = data
	}
}

type AddOptimisticReplyMsg struct {
	LocalID  string
	ParentID string
	Content  string
}

func (m Model) visibleIndices() []int {
	indices := make([]int, 0, len(m.rants))
	for i, ri := range m.rants {
		if !m.isVisibleInFeed(ri.Rant) {
			continue
		}
		indices = append(indices, i)
	}
	return indices
}

func (m Model) isVisibleInFeed(r domain.Rant) bool {
	if m.isHiddenRant(r) {
		return false
	}
	if m.feedSource != sourceFollowing {
		return true
	}
	if r.IsOwn {
		return false
	}
	id := strings.TrimSpace(r.AccountID)
	if id == "" {
		return true
	}
	following, known := m.followingByID[id]
	if !known {
		// Keep it visible until relationship hydration arrives.
		return true
	}
	return following
}

func (m *Model) ensureVisibleCursor() {
	if len(m.rants) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rants) {
		m.cursor = len(m.rants) - 1
	}
	if m.showHidden {
		return
	}
	if m.isVisibleInFeed(m.rants[m.cursor].Rant) {
		return
	}
	for i := m.cursor + 1; i < len(m.rants); i++ {
		if m.isVisibleInFeed(m.rants[i].Rant) {
			m.cursor = i
			return
		}
	}
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isVisibleInFeed(m.rants[i].Rant) {
			m.cursor = i
			return
		}
	}
}

func (m *Model) moveCursorVisible(delta int) {
	if len(m.rants) == 0 || delta == 0 {
		return
	}
	steps := len(m.rants)
	dir := 1
	if delta < 0 {
		dir = -1
	}
	for i := 0; i < steps; i++ {
		next := m.cursor + dir
		if next < 0 || next >= len(m.rants) {
			return
		}
		m.cursor = next
		if m.isVisibleInFeed(m.rants[m.cursor].Rant) {
			return
		}
	}
}

func (m Model) selectedVisibleRant() (domain.Rant, bool) {
	if len(m.rants) == 0 {
		return domain.Rant{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.rants) {
		return domain.Rant{}, false
	}
	r := m.rants[m.cursor].Rant
	if !m.isVisibleInFeed(r) {
		return domain.Rant{}, false
	}
	return r, true
}

func (m Model) isFollowing(accountID string) bool {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return false
	}
	return m.followingByID[accountID]
}

func (m Model) sourceLabel() string {
	switch m.feedSource {
	case sourceTerminalRant:
		return "#terminalrant"
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "#" + m.hashtag
	default:
		return "#terminalrant"
	}
}

func (m Model) sourcePersistValue() string {
	switch m.feedSource {
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "custom"
	default:
		return "terminalrant"
	}
}

func parseFeedSource(v string) feedSource {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "trending":
		return sourceTrending
	case "following":
		return sourceFollowing
	case "personal": // migration from older saved state
		return sourceFollowing
	case "custom":
		return sourceCustomHashtag
	default:
		return sourceTerminalRant
	}
}

func (m Model) emitPrefsChanged() tea.Cmd {
	hashtag := strings.TrimSpace(strings.TrimPrefix(m.hashtag, "#"))
	if hashtag == "" {
		hashtag = "terminalrant"
	}
	source := m.sourcePersistValue()
	return func() tea.Msg {
		return FeedPrefsChangedMsg{
			Hashtag: hashtag,
			Source:  source,
		}
	}
}

func (m Model) hasCustomTab() bool {
	return !strings.EqualFold(strings.TrimSpace(m.hashtag), strings.TrimSpace(m.defaultHashtag))
}

func (m *Model) prepareSourceChange() {
	m.loadingMore = false
	m.cursor = 0
	m.startIndex = 0
	m.scrollLine = 0
	m.rants = nil
	m.oldestFeedID = ""
	m.hasMoreFeed = true
	m.loading = true
}

func (m *Model) addRecentFollow(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return
	}
	out := make([]string, 0, len(m.recentFollows)+1)
	out = append(out, accountID)
	for _, id := range m.recentFollows {
		if id == accountID {
			continue
		}
		out = append(out, id)
		if len(out) >= 20 {
			break
		}
	}
	m.recentFollows = out
}

func (m *Model) removeRecentFollow(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" || len(m.recentFollows) == 0 {
		return
	}
	out := m.recentFollows[:0]
	for _, id := range m.recentFollows {
		if id == accountID {
			continue
		}
		out = append(out, id)
	}
	m.recentFollows = out
}

// ... Loading, Err, Cursor unchanged ...

// IsInDetailView returns true if the detail view is active.
func (m Model) IsInDetailView() bool {
	return m.showDetail
}

// IsDialogOpen reports whether a modal/overlay should capture quit/back keys.
func (m Model) IsDialogOpen() bool {
	return m.showAllHints || m.showBlocked || m.showProfile || m.hashtagInput || m.confirmBlock || m.confirmDelete || m.confirmFollow
}

// SelectedRant returns the currently highlighted rant, if any.
func (m Model) SelectedRant() (domain.Rant, bool) {
	if len(m.rants) == 0 {
		return domain.Rant{}, false
	}
	return m.rants[m.cursor].Rant, true
}

// organizeThreadReplies flattens a thread's descendants into a nested list (depth 2).
func organizeThreadReplies(focusedID string, descendants []domain.Rant) []domain.Rant {
	type replyNode struct {
		rant     domain.Rant
		children []replyNode
	}

	nodeMap := make(map[string]*replyNode)
	for _, r := range descendants {
		nodeMap[r.ID] = &replyNode{rant: r}
	}

	var rootNodes []replyNode
	for _, r := range descendants {
		if r.InReplyToID == focusedID {
			rootNodes = append(rootNodes, *nodeMap[r.ID])
		}
	}

	for i := range rootNodes {
		for _, r := range descendants {
			if r.InReplyToID == rootNodes[i].rant.ID {
				rootNodes[i].children = append(rootNodes[i].children, *nodeMap[r.ID])
			}
		}
	}

	var flatResults []domain.Rant
	var walk func(nodes []replyNode, depth int)
	walk = func(nodes []replyNode, depth int) {
		for _, node := range nodes {
			flatResults = append(flatResults, node.rant)
			if depth < 1 { // Only nest Level 1
				walk(node.children, depth+1)
			}
		}
	}

	walk(rootNodes, 0)

	// Append orphans
	for _, r := range descendants {
		found := false
		for _, fr := range flatResults {
			if fr.ID == r.ID {
				found = true
				break
			}
		}
		if !found {
			flatResults = append(flatResults, r)
		}
	}

	return flatResults
}
