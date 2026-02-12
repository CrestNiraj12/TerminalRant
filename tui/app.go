package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/app"
	"terminalrant/domain"
	"terminalrant/infra/config"
	"terminalrant/infra/editor"
	"terminalrant/tui/common"
	"terminalrant/tui/compose"
	"terminalrant/tui/feed"
)

// Deps holds all dependencies the TUI needs. Plain struct, not a DI container.
type Deps struct {
	Timeline  app.TimelineService
	Post      app.PostService
	Account   app.AccountService
	Editor    *editor.EnvEditor
	Hashtag   string
	FeedView  string
	StatePath string
}

type activeView int

const (
	feedView activeView = iota
	composeView
)

// App is the root Bubble Tea model. It routes between sub-views.
type App struct {
	deps        Deps
	active      activeView
	feed        feed.Model
	compose     compose.Model
	keys        common.KeyMap
	status      string // Transient status message (e.g. "Rant posted!")
	confirmQuit bool
}

// NewApp creates the root model with all dependencies wired.
func NewApp(deps Deps) App {
	return App{
		deps:   deps,
		active: feedView,
		feed:   feed.New(deps.Timeline, deps.Hashtag, deps.FeedView),
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

type profileLoadedMsg struct {
	Profile app.Profile
	Err     error
}

type profileEditorDoneMsg struct {
	TmpPath string
	Err     error
}

type profileSaveResultMsg struct {
	Err error
}

// Update handles messages and routes to the active sub-model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key bindings — handled regardless of active view.
		if key.Matches(msg, a.keys.ForceQuit) {
			return a, tea.Quit
		}

		if a.confirmQuit {
			switch msg.String() {
			case "y", "Y":
				return a, tea.Quit
			case "n", "N", "esc", "q":
				a.confirmQuit = false
				return a, nil
			default:
				return a, nil
			}
		}

		if a.active == feedView && key.Matches(msg, a.keys.Quit) && !a.feed.IsInDetailView() && !a.feed.IsDialogOpen() {
			a.confirmQuit = true
			return a, nil
		}

		// View-specific key bindings.
		if a.active == feedView {
			if key.Matches(msg, a.keys.EditProfile) {
				return a, a.loadProfileForEdit()
			}

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
			a.compose = compose.NewInlineWithContent(a.deps.Post, a.deps.Hashtag, msg.Rant.ID, content, true, false, "", "")
		} else {
			a.compose = compose.NewEditorWithContent(a.deps.Post, a.deps.Editor, a.deps.Hashtag, msg.Rant.ID, content, true, false, "", "")
		}
		return a, a.compose.Init()

	case feed.ReplyRantMsg:
		a.active = composeView
		a.status = ""
		parentSummary := msg.Rant.Content
		if len(parentSummary) > 50 {
			parentSummary = parentSummary[:47] + "..."
		}
		parentSummary = fmt.Sprintf("@%s: %s", msg.Rant.Username, parentSummary)

		if msg.UseInline {
			a.compose = compose.NewInlineWithContent(a.deps.Post, a.deps.Hashtag, msg.Rant.ID, "", false, true, msg.Rant.Username, parentSummary)
		} else {
			a.compose = compose.NewEditorWithContent(a.deps.Post, a.deps.Editor, a.deps.Hashtag, msg.Rant.ID, "", false, true, msg.Rant.Username, parentSummary)
		}
		return a, a.compose.Init()

	case feed.LikeRantMsg:
		// Optimistic like
		a.feed, _ = a.feed.Update(msg)
		return a, func() tea.Msg {
			var err error
			if msg.WasLiked {
				err = a.deps.Post.Unlike(context.Background(), msg.ID)
			} else {
				err = a.deps.Post.Like(context.Background(), msg.ID)
			}
			return feed.LikeResultMsg{ID: msg.ID, Err: err}
		}

	case feed.LikeResultMsg:
		a.feed, _ = a.feed.Update(msg)
		if msg.Err != nil {
			a.status = "Error liking: " + msg.Err.Error()
		}
		return a, nil

	case feed.BlockUserMsg:
		a.status = "Blocking @" + msg.Username + "..."
		return a, func() tea.Msg {
			err := a.deps.Account.BlockUser(context.Background(), msg.AccountID)
			return feed.BlockResultMsg{AccountID: msg.AccountID, Username: msg.Username, Err: err}
		}

	case feed.BlockResultMsg:
		a.feed, _ = a.feed.Update(msg)
		if msg.Err != nil {
			a.status = "Error blocking @" + msg.Username + ": " + msg.Err.Error()
		} else {
			a.status = "Blocked @" + msg.Username + ". Their posts are hidden."
		}
		return a, nil

	case feed.RequestBlockedUsersMsg:
		return a, func() tea.Msg {
			users, err := a.deps.Account.ListBlockedUsers(context.Background(), 80)
			return feed.BlockedUsersLoadedMsg{Users: users, Err: err}
		}

	case feed.UnblockUserMsg:
		return a, func() tea.Msg {
			err := a.deps.Account.UnblockUser(context.Background(), msg.AccountID)
			return feed.UnblockResultMsg{
				AccountID: msg.AccountID,
				Username:  msg.Username,
				Err:       err,
			}
		}

	case feed.FeedPrefsChangedMsg:
		if strings.TrimSpace(a.deps.StatePath) == "" {
			return a, nil
		}
		return a, func() tea.Msg {
			err := config.SaveUIState(a.deps.StatePath, config.UIState{
				Hashtag:    msg.Hashtag,
				FeedSource: msg.Source,
			})
			if err != nil {
				return feed.PrefsSavedMsg{Err: err}
			}
			return feed.PrefsSavedMsg{}
		}

	case feed.PrefsSavedMsg:
		if msg.Err != nil {
			a.status = "Could not save view settings: " + msg.Err.Error()
		}
		return a, nil

	case profileLoadedMsg:
		if msg.Err != nil {
			a.status = "Profile error: " + msg.Err.Error()
			return a, nil
		}
		cmd, tmpPath, err := a.deps.Editor.Cmd(formatProfileDraft(msg.Profile), "")
		if err != nil {
			a.status = "Editor error: " + err.Error()
			return a, nil
		}
		a.status = "Editing profile..."
		return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return profileEditorDoneMsg{TmpPath: tmpPath, Err: err}
		})

	case profileEditorDoneMsg:
		if msg.Err != nil {
			a.status = "Profile edit cancelled."
			return a, nil
		}
		content, err := a.deps.Editor.ReadContent(msg.TmpPath)
		if err != nil {
			a.status = "Profile read error: " + err.Error()
			return a, nil
		}
		displayName, bio, ok := parseProfileDraft(content)
		if !ok {
			a.status = "Invalid profile format. Keep 'Display Name:' and 'Bio:' sections."
			return a, nil
		}
		return a, func() tea.Msg {
			err := a.deps.Account.UpdateProfile(context.Background(), displayName, bio)
			return profileSaveResultMsg{Err: err}
		}

	case profileSaveResultMsg:
		if msg.Err != nil {
			a.status = "Profile update failed: " + msg.Err.Error()
		} else {
			a.status = "Profile updated."
		}
		return a, nil

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
		a.feed, _ = a.feed.Update(feed.ResetFeedStateMsg{ForceReset: false})
		if msg.Err != nil {
			a.status = "Error: " + msg.Err.Error()
			return a, nil
		}

		if msg.Content == "" {
			a.status = "Cancelled."
			return a, nil
		}

		localReplyID := ""
		// Optimistic Update
		if msg.IsEdit {
			a.feed, _ = a.feed.Update(feed.UpdateOptimisticRantMsg{
				ID:      msg.RantID,
				Content: msg.Content,
			})
			a.status = "Updating..."
		} else if msg.IsReply {
			localReplyID = fmt.Sprintf("local-reply-%d", time.Now().UnixNano())
			a.feed, _ = a.feed.Update(feed.AddOptimisticReplyMsg{
				LocalID:  localReplyID,
				ParentID: msg.ParentID,
				Content:  msg.Content,
			})
			a.status = "Replying..."
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
			} else if msg.IsReply {
				rant, err = a.deps.Post.Reply(context.Background(), msg.ParentID, msg.Content, a.deps.Hashtag)
			} else {
				rant, err = a.deps.Post.Post(context.Background(), msg.Content, a.deps.Hashtag)
			}
			// Mark as own since we just performed the action
			rant.IsOwn = true
			resultID := msg.RantID
			if msg.IsReply {
				resultID = localReplyID
			}
			return feed.ResultMsg{
				ID:     resultID,
				Rant:   rant,
				IsEdit: msg.IsEdit,
				Err:    err,
			}
		}

	case feed.ResultMsg:
		a.feed, _ = a.feed.Update(msg)
		a.feed, _ = a.feed.Update(feed.ResetFeedStateMsg{ForceReset: false})
		if msg.Err != nil {
			a.status = "Error: " + msg.Err.Error()
		} else {
			if msg.IsEdit {
				a.status = "🔥 Rant updated!"
			} else {
				a.status = "🔥 Rant posted!"
				// Only auto-open detail for new top-level posts.
				if msg.Rant.InReplyToID == "" || msg.Rant.InReplyToID == "<nil>" || msg.Rant.InReplyToID == "0" {
					a.feed, _ = a.feed.Update(feed.OpenDetailWithoutRepliesMsg{ID: msg.Rant.ID})
				}
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

func (a App) loadProfileForEdit() tea.Cmd {
	return func() tea.Msg {
		profile, err := a.deps.Account.CurrentProfile(context.Background())
		return profileLoadedMsg{Profile: profile, Err: err}
	}
}

func formatProfileDraft(p app.Profile) string {
	var b strings.Builder
	b.WriteString("Display Name:\n")
	b.WriteString(strings.TrimSpace(p.DisplayName))
	b.WriteString("\n\nBio:\n")
	b.WriteString(strings.TrimSpace(p.Bio))
	b.WriteString("\n")
	return b.String()
}

func parseProfileDraft(content string) (displayName string, bio string, ok bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	parts := strings.Split(content, "\n\nBio:\n")
	if len(parts) != 2 {
		return "", "", false
	}
	head := parts[0]
	if !strings.HasPrefix(head, "Display Name:\n") {
		return "", "", false
	}
	displayName = strings.TrimSpace(strings.TrimPrefix(head, "Display Name:\n"))
	bio = strings.TrimSpace(parts[1])
	return displayName, bio, true
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
	if a.confirmQuit {
		s = common.ConfirmStyle.Render(" Exit TerminalRant? (y/n) ") + "\n" + s
	}

	return s
}
