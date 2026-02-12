package feed

import (
	"context"
	"fmt"
	"os/exec"
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
	feedItemLines   = 6
	creatorGitHub   = "https://github.com/CrestNiraj12"
)

// RantsLoadedMsg is sent when the timeline fetch completes successfully.
type RantsLoadedMsg struct {
	Rants []domain.Rant
}

// RantsErrorMsg is sent when the timeline fetch fails.
type RantsErrorMsg struct {
	Err error
}

// RantsPageLoadedMsg is sent when an older feed page is loaded.
type RantsPageLoadedMsg struct {
	Rants []domain.Rant
}

// RantsPageErrorMsg is sent when loading an older feed page fails.
type RantsPageErrorMsg struct {
	Err error
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

type threadData struct {
	Ancestors   []domain.Rant
	Descendants []domain.Rant
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
	timeline       app.TimelineService
	hashtag        string
	rants          []RantItem
	cursor         int
	loading        bool
	loadingMore    bool
	hasMoreFeed    bool
	oldestFeedID   string
	err            error
	keys           common.KeyMap
	spinner        spinner.Model
	confirmDelete  bool // Whether we are in the 'Are you sure?' delete step
	showDetail     bool // Whether we are in full-post view
	height         int  // Terminal height
	startIndex     int  // First visible item in the list (for scrolling)
	scrollLine     int  // Line-based scroll for feed viewport
	ancestors      []domain.Rant
	replies        []domain.Rant
	replyAll       []domain.Rant
	replyVisible   int
	hasMoreReplies bool
	loadingReplies bool
	detailCursor   int // 0 for main post, 1...n for replies
	focusedRant    *domain.Rant
	threadCache    map[string]threadData
	viewStack      []*domain.Rant // To support going back in deep threading
	showAllHints   bool
	pagingNotice   string
}

// New creates a feed model with injected dependencies.
func New(timeline app.TimelineService, hashtag string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))

	return Model{
		timeline:    timeline,
		hashtag:     hashtag,
		keys:        common.DefaultKeyMap(),
		spinner:     s,
		loading:     true,
		hasMoreFeed: true,
		threadCache: make(map[string]threadData),
	}
}

// Init starts the initial feed fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRants(),
		m.spinner.Tick,
	)
}

// Refresh returns a Cmd that re-fetches the timeline.
func (m Model) Refresh() tea.Cmd {
	return m.fetchRants()
}

func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		// Use 'open' for Mac. For Linux 'xdg-open', Windows 'rundll32'.
		// Since user is on Mac, 'open' is safe.
		_ = exec.Command("open", url).Start()
		return nil
	}
}

// Update handles messages for the feed view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.ensureFeedCursorVisible()
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case RantsLoadedMsg:
		// Reconciliation: Merge remote results with inflight optimistic items.
		newRants := make([]RantItem, len(msg.Rants))
		for i, r := range msg.Rants {
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
		m.loading = false
		m.loadingMore = false
		m.err = nil
		m.pagingNotice = ""
		m.oldestFeedID = m.lastFeedID()
		m.hasMoreFeed = len(msg.Rants) == defaultLimit
		if m.cursor >= len(m.rants) {
			m.cursor = 0
		}
		m.ensureFeedCursorVisible()
		return m, nil

	case RantsErrorMsg:
		m.loading = false
		m.loadingMore = false
		m.err = msg.Err
		return m, nil

	case RantsPageLoadedMsg:
		m.loadingMore = false
		m.err = nil
		if len(msg.Rants) == 0 {
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
		for _, r := range msg.Rants {
			if _, ok := existing[r.ID]; ok {
				continue
			}
			m.rants = append(m.rants, RantItem{Rant: r, Status: StatusNormal})
			added++
		}
		m.oldestFeedID = m.lastFeedID()
		m.hasMoreFeed = len(msg.Rants) == defaultLimit && added > 0
		if added == 0 && len(m.rants) > 0 {
			m.hasMoreFeed = false
			m.pagingNotice = "🚀 End of the rantverse reached."
		} else if m.hasMoreFeed {
			m.pagingNotice = ""
		}
		return m, nil

	case RantsPageErrorMsg:
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
			m.replies = nil
			m.replyAll = nil
			m.replyVisible = 0
			m.hasMoreReplies = false
			m.ancestors = nil
			m.loadingReplies = false
			m.focusedRant = nil
			m.viewStack = nil
		}
		return m, nil

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
		return m, nil

	case ThreadErrorMsg:
		if msg.ID != m.currentThreadRootID() {
			return m, nil
		}
		m.loadingReplies = false
		return m, nil

	case LikeRantMsg:
		for i, ri := range m.rants {
			if ri.Rant.ID == msg.ID {
				if ri.Rant.Liked {
					ri.Rant.Liked = false
					ri.Rant.LikesCount--
				} else {
					ri.Rant.Liked = true
					ri.Rant.LikesCount++
				}
				m.rants[i] = ri
				break
			}
		}
		// Also search replies
		for i, r := range m.replies {
			if r.ID == msg.ID {
				if r.Liked {
					r.Liked = false
					r.LikesCount--
				} else {
					r.Liked = true
					r.LikesCount++
				}
				m.replies[i] = r
				break
			}
		}
		m.toggleLikeInThreadCache(msg.ID)
		return m, nil

	case LikeResultMsg:
		if msg.Err != nil {
			// Rollback or show error
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					// Toggle back on error
					if ri.Rant.Liked {
						ri.Rant.Liked = false
						ri.Rant.LikesCount--
					} else {
						ri.Rant.Liked = true
						ri.Rant.LikesCount++
					}
					m.rants[i] = ri
					break
				}
			}
			// Also rollback replies
			for i, r := range m.replies {
				if r.ID == msg.ID {
					if r.Liked {
						r.Liked = false
						r.LikesCount--
					} else {
						r.Liked = true
						r.LikesCount++
					}
					m.replies[i] = r
					break
				}
			}
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

		switch {
		case key.Matches(msg, m.keys.ToggleHints):
			m.showAllHints = true
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
			return m, m.fetchRants()

		case key.Matches(msg, m.keys.Up):
			if m.showDetail {
				if m.detailCursor > 0 {
					m.detailCursor--
				}
				return m, nil
			}
			m.confirmDelete = false
			if m.cursor > 0 {
				m.cursor--
			}
			// Scroll up if necessary
			if m.cursor < m.startIndex {
				m.startIndex = m.cursor
			}
			m.ensureFeedCursorVisible()
		case key.Matches(msg, m.keys.Down):
			if m.showDetail {
				if m.detailCursor < len(m.replies) {
					m.detailCursor++
				}
				if m.hasMoreReplies && m.detailCursor >= len(m.replies)-prefetchTrigger {
					m.loadMoreReplies()
				}
				return m, nil
			}
			m.confirmDelete = false
			if m.cursor < len(m.rants)-1 {
				m.cursor++
			}
			// Scroll down if necessary
			reserved := 9
			availableHeight := m.height - reserved
			if availableHeight < 0 {
				availableHeight = 0
			}
			visibleCount := availableHeight / 5
			if visibleCount < 1 {
				visibleCount = 1
			}

			if m.cursor >= m.startIndex+visibleCount {
				m.startIndex = m.cursor - visibleCount + 1
			}
			m.ensureFeedCursorVisible()
			if m.hasMoreFeed && !m.loadingMore && m.cursor >= len(m.rants)-prefetchTrigger {
				m.loadingMore = true
				return m, m.fetchOlderRants()
			}

		case key.Matches(msg, m.keys.Home):
			m.showDetail = false
			m.confirmDelete = false
			m.cursor = 0
			m.startIndex = 0
			m.scrollLine = 0
			m.detailCursor = 0
			return m, nil

		case msg.String() == "enter":
			if len(m.rants) > 0 {
				if !m.showDetail {
					m.showDetail = true
					m.detailCursor = 0
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
				return m, m.fetchOlderRants()
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

		case key.Matches(msg, m.keys.Delete):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.Rant.IsOwn {
				m.confirmDelete = true
			}

		case msg.String() == "esc", msg.String() == "q":
			if m.showDetail {
				if len(m.viewStack) > 0 {
					// Pop from stack
					m.focusedRant = m.viewStack[len(m.viewStack)-1]
					m.viewStack = m.viewStack[:len(m.viewStack)-1]
					m.detailCursor = 0

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
		case msg.String() == "n":
			if m.confirmDelete {
				m.confirmDelete = false
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

func (m Model) fetchRants() tea.Cmd {
	timeline := m.timeline
	hashtag := m.hashtag
	return func() tea.Msg {
		rants, err := timeline.FetchByHashtag(context.Background(), hashtag, defaultLimit)
		if err != nil {
			return RantsErrorMsg{Err: err}
		}
		return RantsLoadedMsg{Rants: rants}
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
}

func (m Model) fetchOlderRants() tea.Cmd {
	if m.loading || !m.hasMoreFeed || m.oldestFeedID == "" {
		return nil
	}
	timeline := m.timeline
	hashtag := m.hashtag
	maxID := m.oldestFeedID
	return func() tea.Msg {
		rants, err := timeline.FetchByHashtagPage(context.Background(), hashtag, defaultLimit, maxID)
		if err != nil {
			return RantsPageErrorMsg{Err: err}
		}
		return RantsPageLoadedMsg{Rants: rants}
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) feedViewportHeight() int {
	// Reserve stable room for header, footer, and status/help rows.
	const reserved = 13
	h := m.height - reserved
	if h < 8 {
		h = 8
	}
	return h
}

func (m *Model) ensureFeedCursorVisible() {
	if m.showDetail {
		return
	}
	if len(m.rants) == 0 {
		m.scrollLine = 0
		return
	}
	viewHeight := m.feedViewportHeight()
	top := m.cursor * feedItemLines
	bottom := top + feedItemLines - 1
	if top < m.scrollLine {
		m.scrollLine = top
	} else if bottom >= m.scrollLine+viewHeight {
		m.scrollLine = bottom - viewHeight + 1
	}
	if m.scrollLine < 0 {
		m.scrollLine = 0
	}
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

// ... Loading, Err, Cursor unchanged ...

// IsInDetailView returns true if the detail view is active.
func (m Model) IsInDetailView() bool {
	return m.showDetail
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
