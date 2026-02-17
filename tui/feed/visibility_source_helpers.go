package feed

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func (m Model) visibleIndices() []int {
	indices := make([]int, 0, len(m.rants))
	for i, ri := range m.rants {
		if !m.isVisibleInFeed(ri.Rant) {
			continue
		}
		indices = append(indices, i)
	}
	return indices
}

func (m Model) isVisibleInFeed(r domain.Rant) bool {
	if m.isHiddenRant(r) {
		return false
	}
	if m.feedSource != sourceFollowing {
		return true
	}
	if r.IsOwn {
		return false
	}
	id := strings.TrimSpace(r.AccountID)
	if id == "" {
		return true
	}
	following, known := m.followingByID[id]
	if !known {
		// Keep it visible until relationship hydration arrives.
		return true
	}
	return following
}

func (m *Model) ensureVisibleCursor() {
	if len(m.rants) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rants) {
		m.cursor = len(m.rants) - 1
	}
	if m.showHidden {
		return
	}
	if m.isVisibleInFeed(m.rants[m.cursor].Rant) {
		return
	}
	for i := m.cursor + 1; i < len(m.rants); i++ {
		if m.isVisibleInFeed(m.rants[i].Rant) {
			m.cursor = i
			return
		}
	}
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isVisibleInFeed(m.rants[i].Rant) {
			m.cursor = i
			return
		}
	}
}

func (m *Model) moveCursorVisible(delta int) {
	if len(m.rants) == 0 || delta == 0 {
		return
	}
	steps := len(m.rants)
	dir := 1
	if delta < 0 {
		dir = -1
	}
	for range steps {
		next := m.cursor + dir
		if next < 0 || next >= len(m.rants) {
			return
		}
		m.cursor = next
		if m.isVisibleInFeed(m.rants[m.cursor].Rant) {
			return
		}
	}
}

func (m Model) selectedVisibleRant() (domain.Rant, bool) {
	if len(m.rants) == 0 {
		return domain.Rant{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.rants) {
		return domain.Rant{}, false
	}
	r := m.rants[m.cursor].Rant
	if !m.isVisibleInFeed(r) {
		return domain.Rant{}, false
	}
	return r, true
}

func (m Model) isFollowing(accountID string) bool {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return false
	}
	return m.followingByID[accountID]
}

func (m Model) sourceLabel() string {
	switch m.feedSource {
	case sourceTerminalRant:
		return domain.AppHashTag
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "#" + m.hashtag
	default:
		return domain.AppHashTag
	}
}

func (m Model) sourcePersistValue() string {
	switch m.feedSource {
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "custom"
	default:
		return "terminalrant"
	}
}

func parseFeedSource(v string) feedSource {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "trending":
		return sourceTrending
	case "following":
		return sourceFollowing
	case "personal": // migration from older saved state
		return sourceFollowing
	case "custom":
		return sourceCustomHashtag
	default:
		return sourceTerminalRant
	}
}

func (m Model) emitPrefsChanged() tea.Cmd {
	hashtag := strings.TrimSpace(strings.TrimPrefix(m.hashtag, "#"))
	if hashtag == "" {
		hashtag = "terminalrant"
	}
	source := m.sourcePersistValue()
	return func() tea.Msg {
		return FeedPrefsChangedMsg{
			Hashtag: hashtag,
			Source:  source,
		}
	}
}

func (m Model) hasCustomTab() bool {
	return !strings.EqualFold(strings.TrimSpace(m.hashtag), strings.TrimSpace(m.defaultHashtag))
}

func (m Model) tabOrder() []feedSource {
	order := []feedSource{sourceTerminalRant, sourceTrending, sourceFollowing}
	if m.hasCustomTab() {
		order = append(order, sourceCustomHashtag)
	}
	return order
}

func (m Model) nextFeedSource(step int) feedSource {
	order := m.tabOrder()
	if len(order) == 0 {
		return sourceTerminalRant
	}
	cur := 0
	for i, src := range order {
		if src == m.feedSource {
			cur = i
			break
		}
	}
	n := (cur + step) % len(order)
	if n < 0 {
		n += len(order)
	}
	return order[n]
}

func (m Model) isAtVisibleFeedEnd() bool {
	visible := m.visibleIndices()
	if len(visible) == 0 || m.cursor < 0 {
		return false
	}
	return visible[len(visible)-1] == m.cursor
}

func (m *Model) prepareSourceChange() {
	m.loadingMore = false
	m.cursor = 0
	m.startIndex = 0
	m.scrollLine = 0
	m.hScroll = 0
	m.returnToProfile = false
	m.rants = nil
	m.oldestFeedID = ""
	m.hasMoreFeed = true
	m.loading = true
}

func (m *Model) addRecentFollow(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return
	}
	out := make([]string, 0, len(m.recentFollows)+1)
	out = append(out, accountID)
	for _, id := range m.recentFollows {
		if id == accountID {
			continue
		}
		out = append(out, id)
		if len(out) >= 20 {
			break
		}
	}
	m.recentFollows = out
}

func (m *Model) removeRecentFollow(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" || len(m.recentFollows) == 0 {
		return
	}
	out := m.recentFollows[:0]
	for _, id := range m.recentFollows {
		if id == accountID {
			continue
		}
		out = append(out, id)
	}
	m.recentFollows = out
}

// ... Loading, Err, Cursor unchanged ...

// IsInDetailView returns true if the detail view is active.
func (m Model) IsInDetailView() bool {
	return m.showDetail
}

// IsDialogOpen reports whether a modal/overlay should capture quit/back keys.
func (m Model) IsDialogOpen() bool {
	return m.showAllHints || m.showBlocked || m.showProfile || m.hashtagInput || m.confirmBlock || m.confirmDelete || m.confirmFollow
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
