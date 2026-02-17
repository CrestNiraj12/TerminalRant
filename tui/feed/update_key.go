package feed

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/app"
)

func (m Model) handleKeyMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
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
				m.returnToProfile = false
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
				m.returnToProfile = false
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
			case key.Matches(msg, m.keys.Like):
				if m.profileCursor <= 0 || m.profileCursor > len(m.profilePosts) {
					return m, nil
				}
				selected := m.profilePosts[m.profileCursor-1]
				return m, func() tea.Msg {
					return LikeRantMsg{
						ID:       selected.ID,
						WasLiked: selected.Liked,
					}
				}
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
					m.returnToProfile = true
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
		case msg.String() == "left":
			if m.hScroll > 0 {
				m.hScroll = max(m.hScroll-4, 0)
			}
			return m, nil
		case msg.String() == "right":
			m.hScroll += 4
			if m.hScroll < 0 {
				m.hScroll = 0
			}
			return m, nil
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
			m.feedSource = m.nextFeedSource(1)
			m.prepareSourceChange()
			m.pagingNotice = "Feed: " + m.sourceLabel()
			m.feedReqSeq++
			return m, tea.Batch(m.fetchRants(m.feedReqSeq), m.emitPrefsChanged())

		case msg.String() == "T":
			if m.showDetail {
				m.pagingNotice = "Exit detail view to switch tabs."
				return m, nil
			}
			m.feedSource = m.nextFeedSource(-1)
			m.prepareSourceChange()
			m.pagingNotice = "Feed: " + m.sourceLabel()
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
			loadMore := m.maybeStartFeedPrefetch()
			if loadMore != nil {
				return m, tea.Batch(loadMore, m.ensureMediaPreviewCmd())
			}
			if m.feedSource == sourceTrending && !m.hasMoreFeed && m.isAtVisibleFeedEnd() {
				m.pagingNotice = "🔥 You reached the end of trending. The rantverse rests... for now."
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
			m.returnToProfile = false
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
			m.hScroll = 0
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
					m.returnToProfile = false
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
				if m.returnToProfile {
					m.showDetail = false
					m.showProfile = true
					m.returnToProfile = false
					m.focusedRant = nil
					m.viewStack = nil
					m.detailStart = 0
					m.detailScrollLine = 0
					return m, nil
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
		return m, nil
	}

	return m, nil
}
