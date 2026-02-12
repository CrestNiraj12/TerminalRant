package feed

import (
	"fmt"
	"strings"

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
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel())

	b.WriteString(title + tagline + "\n")
	b.WriteString(m.renderTabs() + "\n")
	b.WriteString(hashtag + "\n")

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
			if m.showHidden && m.isMarkedHidden(rant) {
				hiddenText = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#111111")).
					Background(lipgloss.Color("#ED8796")).
					Bold(true).
					Padding(0, 1).
					Render("HIDDEN")
			}

			content := common.StripHashtag(rant.Content, m.hashtag)

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

			itemContent := fmt.Sprintf("%s%s %s  %s%s\n%s\n%s",
				author, statusText, hiddenText, timestamp, replyIndicator, body, common.MetadataStyle.Render(meta))

			itemBase := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1).
				Height(4)
			itemSelected := itemBase.Copy().BorderForeground(lipgloss.Color("#FF8700"))
			itemUnselected := itemBase.Copy().BorderForeground(lipgloss.Color("#45475A"))

			if i == m.cursor {
				itemContent = itemSelected.Render(itemContent)
				if m.confirmDelete {
					itemContent += "\n" + common.ConfirmStyle.Render("  Delete this rant? (y/n)")
				}
				if m.confirmBlock {
					itemContent += "\n" + common.ConfirmStyle.Render(fmt.Sprintf("  Block @%s? (y/n)", m.blockUsername))
				}
			} else {
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
	displayContent := common.StripHashtag(r.Content, m.hashtag)
	content := common.ContentStyle.Width(66).Render(displayContent)
	cardContent.WriteString(content + "\n\n")

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
				Foreground(lipgloss.Color("#111111")).
				Background(lipgloss.Color("#ED8796")).
				Bold(true).
				Padding(0, 1).
				Render("HIDDEN") + "\n",
		)
	}
	meta := fmt.Sprintf("%s Likes: %d  |  ↩ Replies: %d",
		likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)
	cardContent.WriteString(common.MetadataStyle.Render(meta) + "\n")

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

	// Parent Post Card (if available)
	var parentView string
	if len(m.ancestors) > 0 {
		parent := m.ancestors[len(m.ancestors)-1]
		parentContent := common.StripHashtag(parent.Content, m.hashtag)
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

		start := m.detailStart
		if start < 0 {
			start = 0
		}
		if start > len(m.replies) {
			start = len(m.replies)
		}
		end := start + m.detailReplySlots()
		if end > len(m.replies) {
			end = len(m.replies)
		}

		if start > 0 {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("▲ more above") + "\n")
		}

		for i := start; i < end; i++ {
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
			contentLines := strings.Split(truncateToTwoLines(common.StripHashtag(r.Content, m.hashtag), 56), "\n")

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

			// Depth hint for hidden replies
			if depth == 1 && r.RepliesCount > 0 {
				meta += lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Italic(true).Render(" (press enter to see more replies...)")
			}

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
		if end < len(m.replies) {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("▼ more below") + "\n")
		}
		if m.hasMoreReplies {
			remaining := len(m.replyAll) - len(m.replies)
			if remaining < 0 {
				remaining = 0
			}
			b.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(
				fmt.Sprintf("n: load more replies (%d remaining)", remaining),
			))
		}
	}
	if m.confirmBlock {
		b.WriteString("\n" + common.ConfirmStyle.Render(fmt.Sprintf("  Block @%s? (y/n)", m.blockUsername)))
	}
	b.WriteString("\n\n" + m.helpView())

	return b.String()
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
			"n               load more replies",
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
			"n               load older posts",
			"t               switch feed (#terminalrant/trending/personal/custom)",
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
			"n               load older posts (when available)",
			"t               switch feed (#terminalrant/trending/personal/custom)",
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
		{label: "#" + m.hashtag, source: sourceCustomHashtag},
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
	return lipgloss.NewStyle().MarginLeft(2).Render(strings.Join(rendered, " "))
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
