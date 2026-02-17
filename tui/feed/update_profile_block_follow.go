package feed

import (
	"maps"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/app"
)

func (m Model) handleProfileBlockFollowMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		maps.Copy(m.followingByID, msg.Following)
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
		m.detailScrollLine = 0
		m.ensureProfileCursorVisible()
		if msg.Profile.ID != "" {
			// Best-effort local relationship hydration for profile header.
			if following, ok := m.followingByID[msg.Profile.ID]; ok {
				_ = following
			}
		}
		return m, tea.Batch(
			m.fetchRelationshipsForRants(msg.Posts),
			m.ensureProfileAvatarPreviewCmd(),
		)

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

	}

	return m, nil
}
