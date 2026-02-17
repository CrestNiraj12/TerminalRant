package feed

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminalrant/domain"
	"terminalrant/tui/common"
)

func (m Model) Rants() []domain.Rant {
	res := make([]domain.Rant, len(m.rants))
	for i, r := range m.rants {
		res[i] = r.Rant
	}
	return res
}

func (m Model) getSelectedRant() domain.Rant {
	if m.showDetail {
		if m.detailCursor > 0 && m.detailCursor <= len(m.replies) {
			return m.replies[m.detailCursor-1]
		}
		if m.focusedRant != nil {
			return *m.focusedRant
		}
	}
	if len(m.rants) == 0 {
		return domain.Rant{}
	}
	if m.cursor < 0 || m.cursor >= len(m.rants) {
		return domain.Rant{}
	}
	return m.rants[m.cursor].Rant
}

func (m Model) getSelectedRantID() string {
	return m.getSelectedRant().ID
}

func (m Model) lastFeedID() string {
	if len(m.rants) == 0 {
		return ""
	}
	return m.rants[len(m.rants)-1].Rant.ID
}

func (m Model) currentThreadRootID() string {
	if m.focusedRant != nil {
		return m.focusedRant.ID
	}
	if len(m.rants) == 0 {
		return ""
	}
	return m.rants[m.cursor].Rant.ID
}

func (m *Model) normalizeFeedOrder() {
	if len(m.rants) < 2 {
		return
	}
	sort.SliceStable(m.rants, func(i, j int) bool {
		ti := m.rants[i].Rant.CreatedAt
		tj := m.rants[j].Rant.CreatedAt
		if ti.Equal(tj) {
			return m.rants[i].Rant.ID > m.rants[j].Rant.ID
		}
		return ti.After(tj)
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) feedViewportHeight() int {
	h := m.height - m.feedChromeLines()
	// App-level status/confirm bars are rendered outside feed.View().
	h -= 2
	if h < 4 {
		h = 4
	}
	return h
}

func (m Model) feedChromeLines() int {
	lineCount := func(s string) int {
		if s == "" {
			return 0
		}
		return strings.Count(s, "\n") + 1
	}

	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render(domain.DisplayAppTitle()) + common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	top := lineCount(title) + lineCount(m.renderTabs()) + 1 // trailing blank line under tabs

	bottom := 1 // spacer line before status/help block
	// Keep feed viewport height stable while loading/pagination state changes.
	// Reserve fixed rows for loader and notice whenever feed has data.
	if len(m.rants) > 0 {
		bottom += 2
	}
	if m.hashtagInput {
		bottom++
	}
	bottom += lineCount(m.helpView())

	return top + bottom
}

func (m *Model) ensureFeedCursorVisible() {
	if m.showDetail {
		return
	}
	visible := m.visibleIndices()
	if len(visible) == 0 {
		m.scrollLine = 0
		return
	}
	m.ensureVisibleCursor()
	spans := m.feedVisibleSpans(visible)
	if len(spans) == 0 {
		m.scrollLine = 0
		return
	}
	viewHeight := m.feedViewportHeight()
	totalLines := spans[len(spans)-1].bottom + 1
	maxScroll := max(totalLines-viewHeight, 0)
	selectedPos := -1
	for i := range spans {
		if spans[i].idx == m.cursor {
			selectedPos = i
			break
		}
	}
	if selectedPos < 0 {
		return
	}

	if m.startIndex < 0 || m.startIndex >= len(spans) {
		m.startIndex = m.feedStartPosFromScrollLine(spans, m.scrollLine)
	}
	// If another code path changed scrollLine (anchor restore, resize), realign
	// the item window anchor from the current line offset.
	if spans[m.startIndex].top != m.scrollLine {
		m.startIndex = m.feedStartPosFromScrollLine(spans, m.scrollLine)
	}

	if selectedPos < m.startIndex {
		m.startIndex = selectedPos
	} else {
		slots := max(m.feedVisibleSlotsFrom(spans, m.startIndex, viewHeight), 1)
		last := m.startIndex + slots - 1
		if selectedPos > last {
			m.startIndex += selectedPos - last
		}
	}
	if m.startIndex < 0 {
		m.startIndex = 0
	}
	if m.startIndex >= len(spans) {
		m.startIndex = len(spans) - 1
	}

	m.scrollLine = min(maxScroll, spans[m.startIndex].top)
}

func (m Model) feedStartPosFromScrollLine(spans []feedItemSpan, scrollLine int) int {
	if len(spans) == 0 {
		return 0
	}
	if scrollLine <= 0 {
		return 0
	}
	for i := range spans {
		if spans[i].bottom >= scrollLine {
			return i
		}
	}
	return len(spans) - 1
}

func (m Model) feedVisibleSlotsFrom(spans []feedItemSpan, startPos, viewHeight int) int {
	if len(spans) == 0 || startPos < 0 || startPos >= len(spans) {
		return 0
	}
	if viewHeight < 1 {
		viewHeight = 1
	}
	windowTop := spans[startPos].top
	windowBottom := windowTop + viewHeight - 1
	slots := 0
	for i := startPos; i < len(spans); i++ {
		// Only count cards that fully fit inside the viewport.
		if spans[i].bottom > windowBottom {
			break
		}
		slots++
	}
	return slots
}

type feedItemSpan struct {
	idx    int
	top    int
	bottom int
}

func (m Model) feedVisibleSpans(visible []int) []feedItemSpan {
	if len(visible) == 0 {
		return nil
	}
	cardWidth, bodyWidth := m.feedCardWidthsForModel()
	spans := make([]feedItemSpan, 0, len(visible))
	linePos := 0
	for i, idx := range visible {
		lines := m.feedItemRenderedLines(m.rants[idx].Rant, cardWidth, bodyWidth)
		top := linePos
		bottom := top + lines - 1
		spans = append(spans, feedItemSpan{
			idx:    idx,
			top:    top,
			bottom: bottom,
		})
		linePos += lines
		if i < len(visible)-1 {
			linePos += 1
		}
	}
	return spans
}

func (m *Model) maybeStartFeedPrefetch() tea.Cmd {
	if m.loading || m.loadingMore || len(m.rants) == 0 {
		return nil
	}
	if m.feedSource == sourceTrending {
		return nil
	}
	if !m.hasMoreFeed || m.oldestFeedID == "" {
		return nil
	}
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return nil
	}
	selectedPos := -1
	for i, idx := range visible {
		if idx == m.cursor {
			selectedPos = i
			break
		}
	}
	if selectedPos < 0 || selectedPos < len(visible)-prefetchTrigger {
		return nil
	}
	m.loadingMore = true
	m.feedReqSeq++
	return m.fetchOlderRants(m.feedReqSeq)
}

func (m Model) feedItemRenderedLines(r domain.Rant, cardWidth, bodyWidth int) int {
	content, tags := splitContentAndTags(r.Content)
	if strings.TrimSpace(content) == "" && len(r.Media) > 0 {
		content = "(media post)"
	}
	author := common.AuthorStyle.Render("@" + r.Username)
	timestamp := common.TimestampStyle.Render(r.CreatedAt.Format("Jan 02 15:04"))
	replyIndicator := ""
	if r.InReplyToID != "" && r.InReplyToID != "<nil>" && r.InReplyToID != "0" {
		replyIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(" ↩ reply")
	}
	likeIcon := "♡"
	likeStyle := common.MetadataStyle
	if r.Liked {
		likeIcon = "♥"
		likeStyle = common.LikeActiveStyle
	}
	meta := fmt.Sprintf("%s %d  ↩ %d",
		likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)
	indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
	preview := truncateToTwoLinesForWidth(content, bodyWidth)
	previewLines := strings.Split(preview, "\n")
	var bodyBuilder strings.Builder
	for _, line := range previewLines {
		bodyBuilder.WriteString(indicator + common.ContentStyle.Render(line) + "\n")
	}
	body := strings.TrimSuffix(bodyBuilder.String(), "\n")
	tagLine := renderCompactTags(tags, 2)
	mediaLine := renderMediaCompact(r.Media)
	itemContent := fmt.Sprintf("%s  %s%s\n%s\n%s",
		author, timestamp, replyIndicator, body, common.MetadataStyle.Render(meta))
	if tagLine != "" {
		itemContent = fmt.Sprintf("%s  %s%s\n%s\n\n%s\n\n%s",
			author, timestamp, replyIndicator, body, tagLine, common.MetadataStyle.Render(meta))
	}
	if mediaLine != "" {
		itemContent = itemContent + "\n" + mediaLine
	}
	rendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(cardWidth).
		Render(itemContent)
	// Add spacer line between cards in list.
	return len(strings.Split(rendered, "\n")) + 1
}

func truncateToTwoLinesForWidth(text string, width int) string {
	if width < 12 {
		width = 12
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 2 {
		return wrapped
	}
	return strings.Join(lines[:2], "\n") + "..."
}

func (m Model) feedCardWidthsForModel() (cardWidth int, bodyWidth int) {
	showPreviewPanel := m.feedPreviewPanelVisible()
	available := m.width - 4
	if showPreviewPanel {
		available -= 58
	}
	if available < 44 {
		available = 44
	}
	cardWidth = available
	bodyWidth = max(cardWidth-10, 20)
	return cardWidth, bodyWidth
}

func (m Model) feedPreviewPanelVisible() bool {
	// Match view behavior: reserve preview column whenever preview mode is on,
	// regardless of whether current selected post has media.
	return m.showMediaPreview
}

func (m Model) captureFeedTopAnchor() (id string, offset int, ok bool) {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return "", 0, false
	}
	spans := m.feedVisibleSpans(visible)
	if len(spans) == 0 {
		return "", 0, false
	}
	startPos := m.startIndex
	if startPos < 0 || startPos >= len(spans) {
		startPos = m.feedStartPosFromScrollLine(spans, m.scrollLine)
	}
	idx := spans[startPos].idx
	return m.rants[idx].Rant.ID, 0, true
}

func (m *Model) restoreFeedTopAnchor(id string, offset int) {
	if strings.TrimSpace(id) == "" {
		return
	}
	if offset < 0 {
		offset = 0
	}
	visible := m.visibleIndices()
	if len(visible) == 0 {
		m.startIndex = 0
		m.scrollLine = 0
		return
	}
	spans := m.feedVisibleSpans(visible)
	foundPos := -1
	for pos := range spans {
		idx := spans[pos].idx
		if m.rants[idx].Rant.ID == id {
			foundPos = pos
			break
		}
	}
	if foundPos < 0 {
		return
	}
	m.startIndex = foundPos
	m.scrollLine = spans[foundPos].top
}

func (m Model) feedTotalLines(cardWidth, bodyWidth int) int {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return 0
	}
	total := 0
	for i, idx := range visible {
		total += m.feedItemRenderedLines(m.rants[idx].Rant, cardWidth, bodyWidth)
		if i < len(visible)-1 {
			total += 1
		}
	}
	return total
}

func (m Model) currentFeedQueryKey() string {
	switch m.feedSource {
	case sourceTrending:
		return "trending"
	case sourceFollowing:
		return "following"
	case sourceCustomHashtag:
		return "tag:" + strings.ToLower(strings.TrimSpace(m.hashtag))
	default:
		return "tag:" + strings.ToLower(strings.TrimSpace(m.defaultHashtag))
	}
}
