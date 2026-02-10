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

const defaultLimit = 20

// RantsLoadedMsg is sent when the timeline fetch completes successfully.
type RantsLoadedMsg struct {
	Rants []domain.Rant
}

// RantsErrorMsg is sent when the timeline fetch fails.
type RantsErrorMsg struct {
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
	ID string
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
	Ancestors   []domain.Rant
	Descendants []domain.Rant
}

// ThreadErrorMsg is sent when a thread fetch fails.
type ThreadErrorMsg struct {
	Err error
}

func (m Model) fetchThread(id string) tea.Cmd {
	timeline := m.timeline
	return func() tea.Msg {
		ancestors, descendants, err := timeline.FetchThread(context.Background(), id)
		if err != nil {
			return ThreadErrorMsg{Err: err}
		}
		return ThreadLoadedMsg{Ancestors: ancestors, Descendants: descendants}
	}
}

type ResetFeedStateMsg struct{}

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
	err            error
	keys           common.KeyMap
	spinner        spinner.Model
	confirmDelete  bool // Whether we are in the 'Are you sure?' delete step
	showDetail     bool // Whether we are in full-post view
	height         int  // Terminal height
	startIndex     int  // First visible item in the list (for scrolling)
	ancestors      []domain.Rant
	replies        []domain.Rant
	loadingReplies bool
}

// New creates a feed model with injected dependencies.
func New(timeline app.TimelineService, hashtag string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))

	return Model{
		timeline: timeline,
		hashtag:  hashtag,
		keys:     common.DefaultKeyMap(),
		spinner:  s,
		loading:  true,
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
							// Logic depends on if server still returns it.
							// If server returns it, keep it as PendingDelete until it disappears.
						} else {
							// Update: replace with server version but maybe keep pending if local is "newer"?
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
		m.err = nil
		if m.cursor >= len(m.rants) {
			m.cursor = 0
		}
		return m, nil

	case AddOptimisticRantMsg:
		newItem := RantItem{
			Rant: domain.Rant{
				ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
				Content:   msg.Content,
				Author:    "You", // Generic placeholder
				IsOwn:     true,
				CreatedAt: time.Now(),
			},
			Status: StatusPendingCreate,
		}

		m.rants = append([]RantItem{newItem}, m.rants...)
		m.cursor = 0     // Focus the new item
		m.startIndex = 0 // Scroll to top
		return m, nil

	case ResetFeedStateMsg:
		m.showDetail = false
		m.confirmDelete = false
		m.replies = nil
		m.ancestors = nil
		m.loadingReplies = false
		return m, nil

	case ThreadLoadedMsg:
		m.replies = msg.Descendants
		m.ancestors = msg.Ancestors
		m.loadingReplies = false
		return m, nil

	case ThreadErrorMsg:
		m.loadingReplies = false
		// We could show a specific error for replies, but for now just silence it
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
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, m.fetchRants()

		case key.Matches(msg, m.keys.Up):
			if m.showDetail {
				break
			}
			m.confirmDelete = false
			if m.cursor > 0 {
				m.cursor--
			}
			// Scroll up if necessary
			if m.cursor < m.startIndex {
				m.startIndex = m.cursor
			}
		case key.Matches(msg, m.keys.Down):
			if m.showDetail {
				break
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

		case msg.String() == "enter":
			if len(m.rants) > 0 {
				m.showDetail = !m.showDetail
				if m.showDetail {
					m.loadingReplies = true
					m.replies = nil
					return m, m.fetchThread(m.rants[m.cursor].Rant.ID)
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Open):
			if m.showDetail && len(m.rants) > 0 {
				r := m.rants[m.cursor].Rant
				if r.URL != "" {
					return m, openURL(r.URL)
				}
			}

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
			return m, func() tea.Msg { return LikeRantMsg{ID: m.rants[m.cursor].Rant.ID} }

		case key.Matches(msg, m.keys.Reply):
			if len(m.rants) == 0 {
				break
			}
			return m, func() tea.Msg { return ReplyRantMsg{Rant: m.rants[m.cursor].Rant, UseInline: false} }

		case key.Matches(msg, m.keys.ReplyInline):
			if len(m.rants) == 0 {
				break
			}
			return m, func() tea.Msg { return ReplyRantMsg{Rant: m.rants[m.cursor].Rant, UseInline: true} }

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
				m.showDetail = false
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
			if m.showDetail && len(m.ancestors) > 0 {
				// Navigate to the immediate parent (last ancestor)
				parent := m.ancestors[len(m.ancestors)-1]
				// Find it in our local rants if possible, or just push it?
				// For now, let's keep it simple and just show it as the "detail"
				// but we'd need to update the cursor or something.
				// Better approach: just update the cursor to the parent if it's in the list.
				for i, ri := range m.rants {
					if ri.Rant.ID == parent.ID {
						m.cursor = i
						m.replies = nil
						m.ancestors = nil
						m.loadingReplies = true
						return m, m.fetchThread(parent.ID)
					}
				}
				// If not in list, we could fetch it and prepend, but that's complex.
				// Let's at least allow going back to the parent if it's in the ancestors list.
			}
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

// Rants returns the current rants for external access.
func (m Model) Rants() []domain.Rant {
	res := make([]domain.Rant, len(m.rants))
	for i, r := range m.rants {
		res[i] = r.Rant
	}
	return res
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
