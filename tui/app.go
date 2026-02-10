package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/app"
	"terminalrant/domain"
	"terminalrant/infra/editor"
	"terminalrant/tui/common"
	"terminalrant/tui/compose"
	"terminalrant/tui/feed"
)

// Deps holds all dependencies the TUI needs. Plain struct, not a DI container.
type Deps struct {
	Timeline app.TimelineService
	Post     app.PostService
	Account  app.AccountService
	Editor   *editor.EnvEditor
	Hashtag  string
}

type activeView int

const (
	feedView activeView = iota
	composeView
)

// App is the root Bubble Tea model. It routes between sub-views.
type App struct {
	deps    Deps
	active  activeView
	feed    feed.Model
	compose compose.Model
	keys    common.KeyMap
	status  string // Transient status message (e.g. "Rant posted!")
}

// NewApp creates the root model with all dependencies wired.
func NewApp(deps Deps) App {
	return App{
		deps:   deps,
		active: feedView,
		feed:   feed.New(deps.Timeline, deps.Hashtag),
		keys:   common.DefaultKeyMap(),
	}
}

// Init delegates to the active sub-model and fetches the current account ID.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.feed.Init(),
		a.initAccount(),
	)
}

func (a App) initAccount() tea.Cmd {
	return func() tea.Msg {
		id, _ := a.deps.Account.CurrentAccountID(context.Background())
		return accountIDMsg{ID: id}
	}
}

type accountIDMsg struct {
	ID string
}

// Update handles messages and routes to the active sub-model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key bindings — handled regardless of active view.
		if a.active == feedView && key.Matches(msg, a.keys.Quit) && !a.feed.IsInDetailView() {
			return a, tea.Quit
		}

		// View-specific key bindings.
		if a.active == feedView {
			if key.Matches(msg, a.keys.NewEditor) {
				a.active = composeView
				a.status = ""
				a.compose = compose.NewEditor(a.deps.Post, a.deps.Editor, a.deps.Hashtag)
				return a, a.compose.Init()
			}

			if key.Matches(msg, a.keys.NewInline) {
				a.active = composeView
				a.status = ""
				a.compose = compose.NewInline(a.deps.Post, a.deps.Hashtag)
				return a, a.compose.Init()
			}
		}

	case accountIDMsg:
		// Once we have the account ID, we need to tell the timeline service (if it's already created)
		// but since we recreated the timeline service logic in NewTimelineService to accept it,
		// we might need a way to refresh or just accept that future fetches will have it.
		// Actually, we pass it to the Feed model which passes it to the TimelineService?
		// Let's assume we can set it on the feed.
		return a, nil

	case feed.EditRantMsg:
		a.active = composeView
		a.status = ""
		content := common.StripHashtag(msg.Rant.Content, a.deps.Hashtag)
		if msg.UseInline {
			a.compose = compose.NewInlineWithContent(a.deps.Post, a.deps.Hashtag, msg.Rant.ID, content)
		} else {
			a.compose = compose.NewEditorWithContent(a.deps.Post, a.deps.Editor, a.deps.Hashtag, msg.Rant.ID, content)
		}
		return a, a.compose.Init()

	case feed.DeleteRantMsg:
		// Optimistic delete
		a.feed, _ = a.feed.Update(feed.DeleteOptimisticRantMsg{ID: msg.ID})
		return a, func() tea.Msg {
			err := a.deps.Post.Delete(context.Background(), msg.ID)
			return feed.DeleteResultMsg{ID: msg.ID, Err: err}
		}

	case feed.DeleteResultMsg:
		a.feed, _ = a.feed.Update(msg)
		if msg.Err != nil {
			a.status = "Error deleting: " + msg.Err.Error()
		} else {
			a.status = "Rant deleted."
		}
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.feed, cmd = a.feed.Update(msg)
		return a, cmd

	case compose.DoneMsg:
		a.active = feedView
		a.feed, _ = a.feed.Update(feed.ResetFeedStateMsg{})
		if msg.Err != nil {
			a.status = "Error: " + msg.Err.Error()
			return a, nil
		}

		if msg.Content == "" {
			a.status = "Cancelled."
			return a, nil
		}

		// Optimistic Update
		if msg.IsEdit {
			a.feed, _ = a.feed.Update(feed.UpdateOptimisticRantMsg{
				ID:      msg.RantID,
				Content: msg.Content,
			})
			a.status = "Updating..."
		} else {
			a.feed, _ = a.feed.Update(feed.AddOptimisticRantMsg{
				Content: msg.Content,
			})
			a.status = "Posting..."
		}

		// Trigger background API call
		return a, func() tea.Msg {
			var rant domain.Rant
			var err error
			if msg.IsEdit {
				rant, err = a.deps.Post.Edit(context.Background(), msg.RantID, msg.Content, a.deps.Hashtag)
			} else {
				rant, err = a.deps.Post.Post(context.Background(), msg.Content, a.deps.Hashtag)
			}
			// Mark as own since we just performed the action
			rant.IsOwn = true
			return feed.ResultMsg{
				ID:     msg.RantID,
				Rant:   rant,
				IsEdit: msg.IsEdit,
				Err:    err,
			}
		}

	case feed.ResultMsg:
		a.feed, _ = a.feed.Update(msg)
		a.feed, _ = a.feed.Update(feed.ResetFeedStateMsg{})
		if msg.Err != nil {
			a.status = "Error: " + msg.Err.Error()
		} else {
			if msg.IsEdit {
				a.status = "🔥 Rant updated!"
			} else {
				a.status = "🔥 Rant posted!"
			}
		}
		return a, nil
	}

	// Delegate to the active sub-model.
	switch a.active {
	case feedView:
		updated, cmd := a.feed.Update(msg)
		a.feed = updated
		return a, cmd
	case composeView:
		updated, cmd := a.compose.Update(msg)
		a.compose = updated
		return a, cmd
	}

	return a, nil
}

// View renders the active sub-model.
func (a App) View() string {
	var s string

	switch a.active {
	case feedView:
		s = a.feed.View()
	case composeView:
		s = a.compose.View()
	}

	// Append transient status if present.
	if a.status != "" {
		s += "\n" + common.StatusBarStyle.Render(a.status)
	}

	return s
}
