package feed

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

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
	viewHeight := max(m.height-2, 8)
	gate := max(mainLines-(viewHeight-4), 0)
	return gate
}

func (m Model) profileScrollGate() int {
	bio := strings.TrimSpace(m.profile.Bio)
	bioLines := 0
	if bio != "" {
		bioLines = estimateWrappedLines(bio, 66)
	}
	// Mirrors detail gate semantics: allow scrolling through the profile
	// header/card before cursor moves into the post list.
	mainLines := 16 + bioLines
	viewHeight := max(m.height-2, 8)
	gate := max(mainLines-(viewHeight-4), 0)
	return gate
}

func estimateWrappedLines(text string, width int) int {
	if width < 1 {
		width = 1
	}
	lines := 0
	for ln := range strings.SplitSeq(text, "\n") {
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
	for i := range m.profilePosts {
		if m.profilePosts[i].ID == id {
			toggle(&m.profilePosts[i].Liked, &m.profilePosts[i].LikesCount)
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
	slots := max(m.detailReplySlots(), 1)
	idx := m.detailCursor - 1
	if idx < m.detailStart {
		m.detailStart = idx
	}
	if idx >= m.detailStart+slots {
		m.detailStart = idx - slots + 1
	}
	maxStart := max(len(m.replies)-slots, 0)
	if m.detailStart > maxStart {
		m.detailStart = maxStart
	}
	if m.detailStart < 0 {
		m.detailStart = 0
	}
}

func (m Model) detailReplySlots() int {
	// Header + parent/main card + footer/hints leave room for reply window.
	h := max(m.height-30, 20)
	slots := max(h/5, 4)
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
	slots := max(m.profilePostSlots(), 1)
	idx := m.profileCursor - 1
	if idx < m.profileStart {
		m.profileStart = idx
	}
	if idx >= m.profileStart+slots {
		m.profileStart = idx - slots + 1
	}
	maxStart := max(len(m.profilePosts)-slots, 0)
	if m.profileStart > maxStart {
		m.profileStart = maxStart
	}
	if m.profileStart < 0 {
		m.profileStart = 0
	}
}

func (m Model) profilePostSlots() int {
	h := max(m.height-30, 20)
	slots := max(h/5, 4)
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
