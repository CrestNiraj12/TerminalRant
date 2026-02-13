package feed

import (
	"fmt"
	"strings"
	"terminalrant/domain"

	"terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

// View renders the feed as a string.
func (m Model) View() string {
	if m.showBlocked {
		return m.withKeyDialog(m.renderBlockedView())
	}

	if m.showProfile {
		return m.withKeyDialog(m.renderProfileView())
	}

	// If in detail view, render it exclusively (or as an overlay)
	if m.showDetail {
		return m.withKeyDialog(m.renderDetailView())
	}

	var b strings.Builder
	b.WriteString(m.renderFeedHeader())
	b.WriteString(m.renderFeedBody())
	b.WriteString("\n")
	b.WriteString(m.renderFeedStatusRows())
	if m.hashtagInput {
		b.WriteString(m.renderHashtagInputBar() + "\n")
	}
	b.WriteString(m.helpView())
	return m.withKeyDialog(b.String())
}

func (m Model) withKeyDialog(base string) string {
	if !m.showAllHints {
		return base
	}
	return base + "\n\n" + m.renderKeyDialog()
}

func (m Model) renderFeedHeader() string {
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render(domain.DisplayAppTitle())
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	return title + tagline + "\n" + m.renderTabs() + "\n\n"
}

func (m Model) renderFeedBody() string {
	if m.loading && len(m.rants) == 0 {
		return fmt.Sprintf("  %s Loading rants...\n", m.spinner.View())
	}
	if m.err != nil {
		return common.ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n\n  Press r to retry.\n"
	}
	if len(m.rants) == 0 {
		return "  " + m.emptyFeedMessage(false) + "\n"
	}

	visibleIndices := m.visibleIndices()
	if len(visibleIndices) == 0 {
		return "  " + m.emptyFeedMessage(true) + "\n"
	}

	// Keep feed card width stable across selection changes. Without this,
	// media/non-media selection flips can reflow wrapped lines and cause jumps.
	reservePreviewColumn := m.showMediaPreview
	showPreviewPanel := m.renderSelectedMediaPreviewPanel() != ""
	cardWidth, bodyWidth := m.feedCardWidths(reservePreviewColumn)
	return m.renderFeedList(visibleIndices, cardWidth, bodyWidth, showPreviewPanel)
}

func (m Model) renderFeedStatusRows() string {
	if len(m.rants) == 0 {
		return ""
	}
	var b strings.Builder
	if m.loading {
		fmt.Fprintf(&b, "  %s Refreshing...\n", m.spinner.View())
	} else if m.loadingMore {
		fmt.Fprintf(&b, "  %s Loading older posts...\n", m.spinner.View())
	} else {
		b.WriteString("\n")
	}
	if m.pagingNotice != "" {
		b.WriteString(common.StatusBarStyle.Render("  " + m.pagingNotice))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderHashtagInputBar() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111111")).
		Background(lipgloss.Color("#FFB454")).
		Bold(true).
		Padding(0, 1).
		Render(" Set hashtag: #" + m.hashtagBuffer + " (enter: apply, esc: cancel) ")
}

func (m Model) renderFeedList(visibleIndices []int, cardWidth, bodyWidth int, showPreviewPanel bool) string {
	var listBuilder strings.Builder
	for _, i := range visibleIndices {
		listBuilder.WriteString(m.renderFeedCard(i, cardWidth, bodyWidth))
		listBuilder.WriteString("\n")
	}

	listString := strings.TrimSuffix(listBuilder.String(), "\n")
	listLines := strings.Split(listString, "\n")
	viewHeight := m.feedViewportHeight()
	// Use persisted scroll state; render must not re-anchor viewport.
	maxScroll := max(len(listLines)-viewHeight, 0)
	scroll := min(max(m.scrollLine, 0), maxScroll)
	end := min(scroll+viewHeight, len(listLines))
	visible := listLines[scroll:end]
	for len(visible) < viewHeight {
		visible = append(visible, "")
	}
	gutter := m.feedGutter(scroll, end, len(listLines), len(visible))
	contentWindow := strings.Join(visible, "\n")
	gutterWindow := strings.Join(gutter, "\n")
	listPane := lipgloss.JoinHorizontal(lipgloss.Top, contentWindow, " ", gutterWindow)
	if !showPreviewPanel {
		return listPane
	}
	panel := m.renderSelectedMediaPreviewPanel()
	previewPane := clipLines(panel, viewHeight)
	previewPane = lipgloss.NewStyle().
		Width(56).
		MaxHeight(viewHeight).
		Render(previewPane)
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, "  ", previewPane)
}

func (m Model) feedGutter(scroll, end, total, visibleCount int) []string {
	markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	gutter := make([]string, visibleCount)
	for i := range gutter {
		gutter[i] = " "
	}
	if scroll > 0 && len(gutter) > 0 {
		gutter[0] = markerStyle.Render("▲")
	}
	if end < total && len(gutter) > 0 {
		gutter[len(gutter)-1] = markerStyle.Render("▼")
	}
	return gutter
}

func (m Model) renderFeedCard(i, cardWidth, bodyWidth int) string {
	rantItem := m.rants[i]
	rant := rantItem.Rant
	author := renderAuthor(rant.Username, rant.IsOwn, m.isFollowing(rant.AccountID))
	if rant.IsOwn {
		author += common.OwnBadgeStyle.Render("(you)")
	}
	timestamp := common.TimestampStyle.Render(rant.CreatedAt.Format("Jan 02 15:04"))

	replyIndicator := ""
	if rant.InReplyToID != "" && rant.InReplyToID != "<nil>" && rant.InReplyToID != "0" {
		replyIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(" ↩ reply")
	}

	statusText := ""
	switch rantItem.Status {
	case StatusPendingCreate:
		statusText = common.ConfirmStyle.Render(" (posting...)")
	case StatusPendingUpdate:
		statusText = common.ConfirmStyle.Render(" (updating...)")
	case StatusPendingDelete:
		statusText = common.ConfirmStyle.Render(" (deleting...)")
	case StatusFailed:
		statusText = common.ErrorStyle.Render(" (failed)")
	}
	hiddenText := ""
	isHiddenMarked := m.showHidden && m.isMarkedHidden(rant)
	if isHiddenMarked {
		hiddenText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A9A9A9")).
			Background(lipgloss.Color("#3A3A3A")).
			Faint(true).
			Padding(0, 1).
			Render("HIDDEN")
	}

	content, tags := splitContentAndTags(rant.Content)
	if strings.TrimSpace(content) == "" && len(rant.Media) > 0 {
		content = "(media post)"
	}
	likeIcon := "♡"
	likeStyle := common.MetadataStyle
	if rant.Liked {
		likeIcon = "♥"
		likeStyle = common.LikeActiveStyle
	}
	meta := fmt.Sprintf("%s %d  ↩ %d",
		likeStyle.Render(likeIcon), rant.LikesCount, rant.RepliesCount)

	indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
	preview := truncateToTwoLines(content, bodyWidth)
	previewLines := strings.Split(preview, "\n")
	var bodyBuilder strings.Builder
	for _, line := range previewLines {
		bodyBuilder.WriteString(indicator + common.ContentStyle.Render(line) + "\n")
	}

	body := strings.TrimSuffix(bodyBuilder.String(), "\n")
	tagLine := renderCompactTags(tags, 2)
	mediaLine := renderMediaCompact(rant.Media)
	if mediaLine != "" && !m.showMediaPreview {
		mediaLine += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Faint(true).Render("(preview hidden: press i)")
	}
	itemContent := fmt.Sprintf("%s%s %s  %s%s\n%s\n%s",
		author, statusText, hiddenText, timestamp, replyIndicator, body, common.MetadataStyle.Render(meta))
	if tagLine != "" {
		itemContent = fmt.Sprintf("%s%s %s  %s%s\n%s\n\n%s\n\n%s",
			author, statusText, hiddenText, timestamp, replyIndicator, body, tagLine, common.MetadataStyle.Render(meta))
	}
	if mediaLine != "" {
		itemContent = fmt.Sprintf("%s\n%s", itemContent, mediaLine)
	}

	itemBase := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(cardWidth)
	itemSelected := itemBase.BorderForeground(lipgloss.Color("#FF8700"))
	itemUnselected := itemBase.BorderForeground(lipgloss.Color("#45475A"))

	if i == m.cursor {
		if isHiddenMarked {
			itemContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Faint(true).Render(itemContent)
		}
		itemContent = itemSelected.Render(itemContent)
		if m.confirmDelete {
			itemContent += "\n" + common.ConfirmStyle.Render("  Delete this rant? (y/n)")
		}
		if m.confirmBlock {
			itemContent += "\n" + common.ConfirmStyle.Render(fmt.Sprintf("  Block @%s? (y/n)", m.blockUsername))
		}
		if m.confirmFollow {
			action := "Follow"
			if !m.followTarget {
				action = "Unfollow"
			}
			itemContent += "\n" + common.ConfirmStyle.Render(fmt.Sprintf("  %s @%s? (y/n)", action, m.followUsername))
		}
		return itemContent
	}

	if isHiddenMarked {
		itemContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Faint(true).Render(itemContent)
	}
	return itemUnselected.Render(itemContent)
}

func (m Model) feedCardWidths(reservePreviewColumn bool) (cardWidth int, bodyWidth int) {
	// listPane = gutter + spacer + cards (+ optional preview pane)
	available := m.width - 4 // gutter + spacer + a little safety
	if reservePreviewColumn {
		available -= 58 // preview width + gap
	}
	if available < 44 {
		available = 44
	}
	cardWidth = available
	// Rounded border + horizontal padding consume a few columns.
	bodyWidth = max(cardWidth-10, 20)
	return cardWidth, bodyWidth
}

func (m Model) emptyFeedMessage(hadData bool) string {
	switch m.feedSource {
	case sourceFollowing:
		if hadData {
			return "No posts from people you follow."
		}
		return "No posts from people you follow yet."
	case sourceTrending:
		return "Trending is quiet right now."
	case sourceCustomHashtag:
		tag := strings.TrimSpace(strings.TrimPrefix(m.hashtag, "#"))
		if tag == "" {
			tag = m.defaultHashtag
		}
		return fmt.Sprintf("No posts found for #%s.", tag)
	default:
		if hadData {
			if !m.showHidden {
				return "No posts to show. Press X to reveal hidden posts."
			}
			return "No posts to show."
		}
		return "No #terminalrant posts yet. Start the rant."
	}
}
