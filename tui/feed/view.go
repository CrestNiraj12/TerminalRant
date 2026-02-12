package feed

import (
	"fmt"
	"regexp"
	"strings"

	"terminalrant/domain"
	"terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

// View renders the feed as a string.
func (m Model) View() string {
	var b strings.Builder

	// If in detail view, render it exclusively (or as an overlay)
	if m.showDetail {
		base := m.renderDetailView()
		if m.showBlocked {
			base += "\n\n" + m.renderBlockedUsersDialog()
		}
		if m.showAllHints {
			base += "\n\n" + m.renderKeyDialog()
		}
		return base
	}

	// Title + hashtag badge
	// Header Layout
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")

	b.WriteString(title + tagline + "\n")
	b.WriteString(m.renderTabs() + "\n")
	b.WriteString("\n")

	// Content area
	if m.loading && len(m.rants) == 0 {
		b.WriteString(fmt.Sprintf("  %s Loading rants...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(common.ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n  Press r to retry.\n")
	} else if len(m.rants) == 0 {
		b.WriteString("  No rants yet. Be the first!\n")
	} else {
		var listBuilder strings.Builder
		visibleIndices := m.visibleIndices()
		for _, i := range visibleIndices {
			rantItem := m.rants[i]
			rant := rantItem.Rant
			author := common.AuthorStyle.Render("@" + rant.Username)
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

			// Metadata: Likes and Replies
			likeIcon := "♡"
			likeStyle := common.MetadataStyle
			if rant.Liked {
				likeIcon = "♥"
				likeStyle = common.LikeActiveStyle
			}
			meta := fmt.Sprintf("%s %d  ↩ %d",
				likeStyle.Render(likeIcon), rant.LikesCount, rant.RepliesCount)

			indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
			preview := truncateToTwoLines(content, 70)
			previewLines := strings.Split(preview, "\n")
			var bodyBuilder strings.Builder
			for _, line := range previewLines {
				bodyBuilder.WriteString(indicator + common.ContentStyle.Render(line) + "\n")
			}

			body := strings.TrimSuffix(bodyBuilder.String(), "\n")
			tagLine := renderCompactTags(tags, 2)
			mediaLine := renderMediaCompact(rant.Media)
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
				Padding(0, 1)
			itemSelected := itemBase.Copy().BorderForeground(lipgloss.Color("#FF8700"))
			itemUnselected := itemBase.Copy().BorderForeground(lipgloss.Color("#45475A"))

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
			} else {
				if isHiddenMarked {
					itemContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Faint(true).Render(itemContent)
				}
				itemContent = itemUnselected.Render(itemContent)
			}

			listBuilder.WriteString(itemContent)
			listBuilder.WriteString("\n")
		}

		listString := strings.TrimSuffix(listBuilder.String(), "\n")
		if strings.TrimSpace(listString) == "" {
			listString = "  No visible posts. Press X to show hidden posts."
		}
		listLines := strings.Split(listString, "\n")
		viewHeight := m.feedViewportHeight()
		maxScroll := len(listLines) - viewHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scrollLine > maxScroll {
			m.scrollLine = maxScroll
		}
		if m.scrollLine < 0 {
			m.scrollLine = 0
		}
		end := m.scrollLine + viewHeight
		if end > len(listLines) {
			end = len(listLines)
		}
		visible := listLines[m.scrollLine:end]
		for len(visible) < viewHeight {
			visible = append(visible, "")
		}
		markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		gutter := make([]string, len(visible))
		for i := range gutter {
			gutter[i] = " "
		}
		if m.scrollLine > 0 && len(gutter) > 0 {
			gutter[0] = markerStyle.Render("▲")
		}
		if end < len(listLines) && len(gutter) > 0 {
			gutter[len(gutter)-1] = markerStyle.Render("▼")
		}
		contentWindow := strings.Join(visible, "\n")
		gutterWindow := strings.Join(gutter, "\n")
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, gutterWindow, " ", contentWindow))
	}

	b.WriteString("\n")
	if m.loading && len(m.rants) > 0 {
		b.WriteString(fmt.Sprintf("  %s Refreshing...\n", m.spinner.View()))
	} else if m.loadingMore {
		b.WriteString(fmt.Sprintf("  %s Loading older posts...\n", m.spinner.View()))
	}
	if m.pagingNotice != "" && len(m.rants) > 0 {
		b.WriteString(common.StatusBarStyle.Render("  " + m.pagingNotice))
		b.WriteString("\n")
	}
	if m.hashtagInput {
		b.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#111111")).
				Background(lipgloss.Color("#FFB454")).
				Bold(true).
				Padding(0, 1).
				Render(" Set hashtag: #" + m.hashtagBuffer + " (enter: apply, esc: cancel) "),
		)
		b.WriteString("\n")
	}

	b.WriteString(m.helpView())

	base := b.String()
	if m.showBlocked {
		base += "\n\n" + m.renderBlockedUsersDialog()
	}
	if m.showAllHints {
		base += "\n\n" + m.renderKeyDialog()
	}
	return base
}

// truncateToTwoLines wraps and truncates text to at most 2 lines.
func truncateToTwoLines(text string, width int) string {
	// Render with width to handle both explicit newlines and wrapping.
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 2 {
		return wrapped
	}
	// Take first 2 lines and append ellipsis
	return strings.Join(lines[:2], "\n") + "..."
}

func (m Model) renderDetailView() string {
	if len(m.rants) == 0 {
		return "No rant selected."
	}
	r := m.rants[m.cursor].Rant
	status := m.rants[m.cursor].Status
	err := m.rants[m.cursor].Err
	if m.focusedRant != nil {
		r = *m.focusedRant
		status = StatusNormal // Focused rants from thread are usually normal
		err = nil
	}

	var b strings.Builder

	// Header Layout (Consistent with feed)
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")

	// Active source badge, consistent with feed view.
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel())

	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	postCrumb := crumbStyle.Render(fmt.Sprintf("Post %s", r.ID))

	b.WriteString(title + tagline + "\n")
	b.WriteString(m.renderTabs() + "\n")

	breadcrumb := lipgloss.JoinHorizontal(lipgloss.Bottom, hashtag, separator, postCrumb)
	if len(m.viewStack) > 0 {
		depthStr := crumbStyle.Render(fmt.Sprintf(" (depth %d)", len(m.viewStack)))
		breadcrumb = lipgloss.JoinHorizontal(lipgloss.Bottom, breadcrumb, depthStr)
	}
	b.WriteString(breadcrumb + "\n")

	// Create a card for the content
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		MarginLeft(2).
		Width(74)

	var cardContent strings.Builder
	cardContent.WriteString(common.AuthorStyle.Render("@"+r.Username) + " " + common.MetadataStyle.Render("("+r.Author+")") + "\n")
	cardContent.WriteString(common.TimestampStyle.Render(r.CreatedAt.Format("Monday, Jan 02, 2006 at 15:04")) + "\n")

	// Parent Context inside card (minimalist)
	if len(m.ancestors) > 0 {
		parent := m.ancestors[len(m.ancestors)-1]
		parentIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true).
			Render(fmt.Sprintf("⬑ Reply to @%s", parent.Username))
		cardContent.WriteString(parentIndicator + "\n")
	}
	cardContent.WriteString("\n")

	// Full content (wrapped) - strip hashtag for display
	displayContent, tags := splitContentAndTags(r.Content)
	if strings.TrimSpace(displayContent) == "" && len(r.Media) > 0 {
		displayContent = "(media post)"
	}
	content := common.ContentStyle.Width(66).Render(displayContent)
	cardContent.WriteString(content + "\n\n")
	if len(tags) > 0 {
		cardContent.WriteString(renderAllTags(tags) + "\n\n")
	}

	// Metadata: Likes and Replies
	likeIcon := "♡"
	likeStyle := common.MetadataStyle
	if r.Liked {
		likeIcon = "♥"
		likeStyle = common.LikeActiveStyle
	}
	if m.showHidden && m.isMarkedHidden(r) {
		cardContent.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A9A9A9")).
				Background(lipgloss.Color("#3A3A3A")).
				Faint(true).
				Padding(0, 1).
				Render("HIDDEN") + "\n",
		)
	}
	meta := fmt.Sprintf("%s Likes: %d  |  ↩ Replies: %d",
		likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)
	cardContent.WriteString(common.MetadataStyle.Render(meta) + "\n")
	if len(r.Media) > 0 {
		cardContent.WriteString("\n" + renderMediaDetail(r.Media) + "\n")
	}

	if r.URL != "" {
		cardContent.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("🔗 URL: "+r.URL) + "\n")
	}

	if err != nil {
		cardContent.WriteString("\n" + common.ErrorStyle.Render(fmt.Sprintf("⚠️ Error: %v", err)))
	}
	if status != StatusNormal {
		// Show status hint if not normal
		statusText := ""
		switch status {
		case StatusPendingCreate:
			statusText = " (posting...)"
		case StatusPendingUpdate:
			statusText = " (updating...)"
		case StatusPendingDelete:
			statusText = " (deleting...)"
		}
		if statusText != "" {
			cardContent.WriteString("\n" + common.ConfirmStyle.Render(statusText))
		}
	}

	renderedCard := cardStyle.Render(cardContent.String())
	if m.detailCursor == 0 {
		renderedCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF")).
			Padding(1, 2).
			MarginLeft(2).
			Width(74).
			Render(cardContent.String())
	}
	if m.showHidden && m.isMarkedHidden(r) {
		renderedCard = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Faint(true).Render(renderedCard)
	}

	// Parent Post Card (if available)
	var parentView string
	if len(m.ancestors) > 0 {
		parent := m.ancestors[len(m.ancestors)-1]
		parentContent, _ := splitContentAndTags(parent.Content)
		if strings.TrimSpace(parentContent) == "" && len(parent.Media) > 0 {
			parentContent = "(media post)"
		}
		parentSummary := truncateToTwoLines(parentContent, 66)

		parentCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(0, 1). // Use 0 padding on top/bottom to keep it compact
			MarginLeft(2).
			Width(74).
			Render(fmt.Sprintf("%s %s\n%s",
				common.AuthorStyle.Render("@"+parent.Username),
				common.TimestampStyle.Render(parent.CreatedAt.Format("Jan 02")),
				common.ContentStyle.Render(parentSummary)))

		parentView = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("  Parent Thread:") + "\n" + parentCard + "\n"
	}

	b.WriteString(parentView + renderedCard)

	// Replies Section
	if m.loadingReplies {
		b.WriteString("\n\n  " + m.spinner.View() + " Loading replies...")
	} else if len(m.replies) > 0 {
		b.WriteString("\n\n  " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Replies") + "\n")

		for i := 0; i < len(m.replies); i++ {
			r := m.replies[i]
			// Calculate depth based on relationship to focused rant
			depth := 0
			if r.InReplyToID != "" && r.InReplyToID != "<nil>" && r.InReplyToID != "0" && r.InReplyToID != r.ID {
				threadRootID := m.rants[m.cursor].Rant.ID
				if m.focusedRant != nil {
					threadRootID = m.focusedRant.ID
				}
				if r.InReplyToID != threadRootID {
					depth = 1 // Level 2
				}
			}

			author := common.AuthorStyle.Render("@" + r.Username)
			timestamp := common.TimestampStyle.Render(r.CreatedAt.Format("Jan 02 15:04"))
			replyContentClean, _ := splitContentAndTags(r.Content)
			if strings.TrimSpace(replyContentClean) == "" && len(r.Media) > 0 {
				replyContentClean = "(media post)"
			}
			contentLines := strings.Split(truncateToTwoLines(replyContentClean, 56), "\n")

			indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
			indicator := indicatorStyle.Render("┃ ")

			indentPrefix := ""
			for j := 0; j < depth; j++ {
				indentPrefix += "  "
			}

			var replyBody strings.Builder
			for _, line := range contentLines {
				replyBody.WriteString("  " + indentPrefix + indicator + common.ContentStyle.Render(line) + "\n")
			}

			// Metadata for reply
			likeIcon := "♡"
			likeStyle := common.MetadataStyle
			if r.Liked {
				likeIcon = "♥"
				likeStyle = common.LikeActiveStyle
			}
			meta := fmt.Sprintf("%s %d  ↩ %d",
				likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)

			replyContent := fmt.Sprintf("  %s%s %s\n%s\n  %s%s",
				indentPrefix, author, timestamp, strings.TrimSuffix(replyBody.String(), "\n"), indentPrefix, common.MetadataStyle.Render(meta))

			if m.detailCursor == i+1 {
				replyContent = lipgloss.NewStyle().
					Background(lipgloss.Color("#333333")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Render(replyContent)
			}
			b.WriteString("\n" + replyContent + "\n")
		}
	}
	if m.confirmBlock {
		b.WriteString("\n" + common.ConfirmStyle.Render(fmt.Sprintf("  Block @%s? (y/n)", m.blockUsername)))
	}
	b.WriteString("\n\n" + m.helpView())

	return m.renderDetailViewport(b.String())
}

func (m Model) helpView() string {
	var items []string

	if m.showDetail {
		items = []string{
			"j/k: focus",
			"enter: open",
			"l: like",
			"esc/q: back",
			"?: all keys",
		}
	} else if len(m.rants) > 0 {
		items = []string{
			"j/k: focus",
			"enter: detail",
			"p/P: rant",
			"l: like",
			"q: quit",
			"?: all keys",
		}
	} else {
		items = []string{
			"p/P: rant",
			"q: quit",
			"?: all keys",
		}
	}

	hints := common.StatusBarStyle.Render("  " + strings.Join(items, " • "))
	creator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Italic(true).
		PaddingTop(1).
		Render("  Made by @CrestNiraj12 • https://github.com/CrestNiraj12 • g: visit")
	return hints + "\n" + creator
}

func (m Model) renderKeyDialog() string {
	var lines []string
	if m.showDetail {
		lines = []string{
			"j/k or up/down  move focus",
			"enter           open selected reply thread",
			"l               like/dislike selected post",
			"c / C           reply via editor / inline",
			"x / X           hide post / toggle hidden posts",
			"b               block selected user",
			"B               show blocked users",
			"u               open parent post",
			"r               refresh replies",
			"o               open post URL",
			"v               edit profile",
			"g               open creator GitHub",
			"h               jump to feed home",
			"esc / q         back",
			"ctrl+c          force quit",
			"?               toggle this dialog",
		}
	} else if len(m.rants) > 0 {
		lines = []string{
			"j/k or up/down  move focus",
			"enter           open detail",
			"t / T           next/prev tab",
			"H               set hashtag feed tag",
			"p / P           new rant via editor / inline",
			"v               edit profile",
			"c / C           reply via editor / inline",
			"l               like/dislike selected post",
			"x / X           hide post / toggle hidden posts",
			"b               block selected user",
			"B               show blocked users",
			"r               refresh timeline",
			"o               open post URL",
			"g               open creator GitHub",
			"h               jump to top",
			"q               quit",
			"ctrl+c          force quit",
			"?               toggle this dialog",
		}
		r := m.rants[m.cursor].Rant
		if r.IsOwn {
			lines = append(lines, "e / E           edit via editor / inline", "d               delete selected post")
		}
	} else {
		lines = []string{
			"p / P           new rant via editor / inline",
			"t / T           next/prev tab",
			"H               set hashtag feed tag",
			"v               edit profile",
			"B               show blocked users",
			"r               refresh timeline",
			"g               open creator GitHub",
			"q               quit",
			"ctrl+c          force quit",
			"?               toggle this dialog",
		}
	}

	body := "Keyboard Shortcuts\n\n" + strings.Join(lines, "\n") + "\n\nPress ?, esc, q, or enter to close."
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		Margin(1, 2).
		Render(body)
}

func (m Model) renderTabs() string {
	tabs := []struct {
		label  string
		source feedSource
	}{
		{label: "#terminalrant", source: sourceTerminalRant},
		{label: "trending", source: sourceTrending},
		{label: "personal", source: sourcePersonal},
	}
	if m.hasCustomTab() {
		tabs = append(tabs, struct {
			label  string
			source feedSource
		}{label: "#" + m.hashtag, source: sourceCustomHashtag})
	}
	active := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111111")).
		Background(lipgloss.Color("#FFB454")).
		Bold(true).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B3B3B3")).
		Background(lipgloss.Color("#2B2B2B")).
		Padding(0, 1)

	rendered := make([]string, 0, len(tabs))
	for _, t := range tabs {
		if m.feedSource == t.source {
			rendered = append(rendered, active.Render(t.label))
		} else {
			rendered = append(rendered, inactive.Render(t.label))
		}
	}
	return lipgloss.NewStyle().MarginLeft(2).PaddingTop(1).Render(strings.Join(rendered, " "))
}

func (m Model) renderBlockedUsersDialog() string {
	var body strings.Builder
	body.WriteString("Blocked Users\n\n")
	if m.loadingBlocked {
		body.WriteString(m.spinner.View() + " Loading blocked users...\n")
	} else if m.blockedErr != nil {
		body.WriteString(common.ErrorStyle.Render("Error: " + m.blockedErr.Error()))
		body.WriteString("\n")
	} else if len(m.blockedUsers) == 0 {
		body.WriteString("No blocked users.\n")
	} else {
		for i, u := range m.blockedUsers {
			prefix := "  "
			if i == m.blockedCursor {
				prefix = "▶ "
			}
			name := "@" + u.Username
			if strings.TrimSpace(u.DisplayName) != "" {
				name += " (" + u.DisplayName + ")"
			}
			body.WriteString(prefix + name + "\n")
		}
	}
	if m.confirmUnblock {
		body.WriteString("\n" + common.ConfirmStyle.Render(fmt.Sprintf("Unblock @%s? (y/n)", m.unblockTarget.Username)))
	}
	body.WriteString("\n\nj/k: move • u: unblock • esc/q: close")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		Margin(1, 2).
		Width(74).
		Render(body.String())
}

func (m Model) renderDetailViewport(content string) string {
	lines := strings.Split(content, "\n")
	if m.height <= 0 {
		return content
	}
	viewHeight := m.height - 2
	if viewHeight < 8 {
		viewHeight = 8
	}
	maxScroll := len(lines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.detailScrollLine
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[scroll:end]
	for len(visible) < viewHeight {
		visible = append(visible, "")
	}
	markerTop := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB454")).Bold(true).Render("▲ more above")
	markerBottom := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BD5CA")).Bold(true).Render("▼ more below")
	if scroll > 0 && len(visible) > 0 {
		visible[0] = markerTop
	}
	if end < len(lines) && len(visible) > 0 {
		visible[len(visible)-1] = markerBottom
	}
	return strings.Join(visible, "\n")
}

var hashtagRe = regexp.MustCompile(`(?i)#[a-z0-9_]+`)

func splitContentAndTags(content string) (string, []string) {
	found := hashtagRe.FindAllString(content, -1)
	tags := uniqueLower(found)
	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, ln := range lines {
		line := hashtagRe.ReplaceAllString(ln, "")
		line = strings.Join(strings.Fields(line), " ")
		cleaned = append(cleaned, strings.TrimSpace(line))
	}
	out := strings.TrimSpace(strings.Join(cleaned, "\n"))
	return out, tags
}

func uniqueLower(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		low := strings.ToLower(strings.TrimSpace(t))
		if low == "" {
			continue
		}
		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		out = append(out, low)
	}
	return out
}

func renderCompactTags(tags []string, max int) string {
	if len(tags) == 0 {
		return ""
	}
	if max < 1 {
		max = 1
	}
	show := tags
	if len(show) > max {
		show = show[:max]
	}
	capStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A9A9A9")).
		Background(lipgloss.Color("#2F2F2F")).
		Padding(0, 1).
		Faint(true)
	parts := make([]string, 0, len(show)+1)
	for _, t := range show {
		parts = append(parts, capStyle.Render(t))
	}
	if len(tags) > max {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#777777")).Faint(true).Render(fmt.Sprintf("+%d more", len(tags)-max)))
	}
	return strings.Join(parts, " ")
}

func renderAllTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	capStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A9A9A9")).
		Background(lipgloss.Color("#2F2F2F")).
		Padding(0, 1).
		Faint(true)
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		parts = append(parts, capStyle.Render(t))
	}
	return strings.Join(parts, " ")
}

func renderMediaCompact(media []domain.MediaAttachment) string {
	if len(media) == 0 {
		return ""
	}
	imageCount := 0
	videoCount := 0
	audioCount := 0
	otherCount := 0
	for _, m := range media {
		switch strings.ToLower(strings.TrimSpace(m.Type)) {
		case "image":
			imageCount++
		case "video", "gifv":
			videoCount++
		case "audio":
			audioCount++
		default:
			otherCount++
		}
	}
	parts := make([]string, 0, 4)
	if imageCount > 0 {
		parts = append(parts, fmt.Sprintf("🖼 %d", imageCount))
	}
	if videoCount > 0 {
		parts = append(parts, fmt.Sprintf("🎬 %d", videoCount))
	}
	if audioCount > 0 {
		parts = append(parts, fmt.Sprintf("🔊 %d", audioCount))
	}
	if otherCount > 0 {
		parts = append(parts, fmt.Sprintf("📎 %d", otherCount))
	}
	line := strings.Join(parts, "  ")
	firstAlt := ""
	for _, m := range media {
		if strings.TrimSpace(m.Description) != "" {
			firstAlt = m.Description
			break
		}
	}
	if firstAlt != "" {
		r := []rune(firstAlt)
		if len(r) > 40 {
			firstAlt = string(r[:40]) + "..."
		}
		line += "  alt: " + firstAlt
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FA8DC")).
		Faint(true).
		Render(line)
}

func renderMediaDetail(media []domain.MediaAttachment) string {
	if len(media) == 0 {
		return ""
	}
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FA8DC")).
		Bold(true).
		Render(fmt.Sprintf("Media (%d)", len(media)))
	var b strings.Builder
	b.WriteString(title + "\n")
	row := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A7A7A"))
	for i, m := range media {
		entry := fmt.Sprintf("  %d. %s", i+1, strings.ToLower(strings.TrimSpace(m.Type)))
		if m.Width > 0 && m.Height > 0 {
			entry += fmt.Sprintf(" %dx%d", m.Width, m.Height)
		}
		if strings.TrimSpace(m.Description) != "" {
			entry += " — " + m.Description
		}
		if m.URL != "" {
			entry += " [" + m.URL + "]"
		}
		b.WriteString(row.Render(entry))
		if i < len(media)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
