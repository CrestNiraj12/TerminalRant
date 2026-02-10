package feed

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminalrant/app"
	"terminalrant/domain"
	"terminalrant/tui/common"
)

const defaultLimit = 20

// --- Messages ---

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
	Err error
}

// --- Model ---

// Model holds the state for the feed (timeline) view.
type Model struct {
	timeline      app.TimelineService
	hashtag       string
	rants         []domain.Rant
	cursor        int
	loading       bool
	err           error
	keys          common.KeyMap
	spinner       spinner.Model
	showActions   bool // Whether the action menu is open
	actionCursor  int  // 0: Edit (buffer), 1: Edit (inline), 2: Delete, 3: Cancel
	confirmDelete bool // Whether we are in the 'Are you sure?' delete step
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

// Update handles messages for the feed view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case RantsLoadedMsg:
		m.rants = msg.Rants
		m.loading = false
		m.err = nil
		m.cursor = 0
		return m, nil

	case RantsErrorMsg:
		m.err = msg.Err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Refresh):
			if m.showActions {
				break
			}
			m.loading = true
			return m, m.fetchRants()

		case key.Matches(msg, m.keys.Up):
			m.showActions = false
			m.confirmDelete = false
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			m.showActions = false
			m.confirmDelete = false
			if m.cursor < len(m.rants)-1 {
				m.cursor++
			}

		case key.Matches(msg, m.keys.Edit):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.IsOwn {
				m.showActions = false
				return m, func() tea.Msg { return EditRantMsg{Rant: r, UseInline: false} }
			}

		case key.Matches(msg, m.keys.EditInline):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.IsOwn {
				m.showActions = false
				return m, func() tea.Msg { return EditRantMsg{Rant: r, UseInline: true} }
			}

		case key.Matches(msg, m.keys.Delete):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if r.IsOwn {
				m.showActions = true
				m.actionCursor = 1 // Position at Delete
				m.confirmDelete = true
			}

		case key.Matches(msg, m.keys.Enter):
			if len(m.rants) == 0 {
				break
			}
			r := m.rants[m.cursor]
			if !r.IsOwn {
				break
			}

			if !m.showActions {
				m.showActions = true
				m.actionCursor = 0
				m.confirmDelete = false
			} else {
				// Confirm action
				switch m.actionCursor {
				case 0: // Edit (buffer)
					m.showActions = false
					return m, func() tea.Msg { return EditRantMsg{Rant: r, UseInline: false} }
				case 1: // Edit (inline)
					m.showActions = false
					return m, func() tea.Msg { return EditRantMsg{Rant: r, UseInline: true} }
				case 2: // Delete
					if !m.confirmDelete {
						m.confirmDelete = true
					} else {
						return m, m.deleteRant(r.ID)
					}
				case 3: // Cancel
					m.showActions = false
				}
			}

		case msg.String() == "esc":
			if m.showActions {
				m.showActions = false
				m.confirmDelete = false
				return m, nil
			}

		case msg.String() == "left" || msg.String() == "h":
			if m.showActions && !m.confirmDelete {
				if m.actionCursor > 0 {
					m.actionCursor--
				}
			}

		case msg.String() == "right" || msg.String() == "l":
			if m.showActions && !m.confirmDelete {
				if m.actionCursor < 3 {
					m.actionCursor++
				}
			}

		case msg.String() == "y":
			if m.confirmDelete {
				r := m.rants[m.cursor]
				return m, m.deleteRant(r.ID)
			}
		case msg.String() == "n":
			if m.confirmDelete {
				m.confirmDelete = false
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
	return m.rants
}

// Loading returns whether the feed is currently loading.
func (m Model) Loading() bool {
	return m.loading
}

// Err returns the current error, if any.
func (m Model) Err() error {
	return m.err
}

// Cursor returns the current cursor position.
func (m Model) Cursor() int {
	return m.cursor
}

// SelectedRant returns the currently highlighted rant, if any.
func (m Model) SelectedRant() (domain.Rant, bool) {
	if len(m.rants) == 0 {
		return domain.Rant{}, false
	}
	return m.rants[m.cursor], true
}
