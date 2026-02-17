package feed

import (
	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/app"
	"terminalrant/domain"
)

func (m Model) handleDetailThreadMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			m.returnToProfile = false
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
		m.returnToProfile = false
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
		delete(m.mediaLoading, msg.Key)
		if msg.Err != nil {
			m.mediaPreview[msg.Key] = ""
			delete(m.mediaFrames, msg.Key)
			delete(m.mediaFrameIndex, msg.Key)
			return m, nil
		}
		m.mediaPreview[msg.Key] = msg.Preview
		if len(msg.Frames) > 1 {
			m.mediaFrames[msg.Key] = msg.Frames
			m.mediaFrameIndex[msg.Key] = 0
		} else {
			delete(m.mediaFrames, msg.Key)
			delete(m.mediaFrameIndex, msg.Key)
		}
		return m, nil
	}

	return m, nil
}
