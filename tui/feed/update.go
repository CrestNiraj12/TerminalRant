package feed

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureFeedCursorVisible()
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		m.advanceMediaFrames()
		return m, tea.Batch(cmd, m.ensureMediaPreviewCmd())
	}

	switch msg.(type) {
	case SwitchToTerminalRantMsg:
		if m.feedSource == sourceTerminalRant {
			return m, nil
		}
		m.feedSource = sourceTerminalRant
		m.prepareSourceChange()
		m.pagingNotice = "Feed: " + m.sourceLabel()
		m.feedReqSeq++
		return m, tea.Batch(m.fetchRants(m.feedReqSeq), m.emitPrefsChanged())
	case RantsLoadedMsg, RantsErrorMsg, RantsPageLoadedMsg, RantsPageErrorMsg:
		return m.handleFeedLoadingMsg(msg)
	case ResetFeedStateMsg, OpenDetailWithoutRepliesMsg, ThreadLoadedMsg, ThreadErrorMsg, MediaPreviewLoadedMsg:
		return m.handleDetailThreadMsg(msg)
	case HideAuthorPostsMsg, BlockResultMsg, RelationshipsLoadedMsg, ProfileLoadedMsg, FollowToggleResultMsg, BlockedUsersLoadedMsg, UnblockResultMsg:
		return m.handleProfileBlockFollowMsg(msg)
	case AddOptimisticRantMsg, AddOptimisticReplyMsg, LikeRantMsg, LikeResultMsg, UpdateOptimisticRantMsg, DeleteOptimisticRantMsg, ResultMsg, DeleteResultMsg:
		return m.handleOptimisticMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}
