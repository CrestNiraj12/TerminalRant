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
		return m.renderDetailView()
	}

	// Title + hashtag badge
	// Header Layout
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(fmt.Sprintf("#%s", m.hashtag))

	b.WriteString(title + tagline + "\n")
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
		// Calculate visible items based on height.
		// Reserved height: Header (~5), Status Bar (~2), Bottom Padding (~2) = ~9 lines.
		reserved := 9
		availableHeight := m.height - reserved
		if availableHeight < 0 {
			availableHeight = 0
		}
		// Each box is 5 lines (3 content + 2 border)
		visibleCount := availableHeight / 5
		if visibleCount < 1 {
			visibleCount = 1
		}

		// Ensure startIndex is valid
		if m.startIndex < 0 {
			m.startIndex = 0
		}
		if m.startIndex >= len(m.rants) {
			m.startIndex = len(m.rants) - 1
		}

		endIndex := m.startIndex + visibleCount
		if endIndex > len(m.rants) {
			endIndex = len(m.rants)
		}

		var listBuilder strings.Builder
		for i := m.startIndex; i < endIndex; i++ {
			rantItem := m.rants[i]
			rant := rantItem.Rant
			author := common.AuthorStyle.Render(rant.Author)
			if rant.IsOwn {
				author += common.OwnBadgeStyle.Render("(you)")
			}
			timestamp := common.TimestampStyle.Render(rant.CreatedAt.Format("Jan 02 15:04"))

			replyHint := ""
			if rant.InReplyToID != "" && rant.InReplyToID != "<nil>" && rant.InReplyToID != "0" {
				// We don't have the parent author in the list usually, but it might be in the content?
				// Actually, we can just show a simpler 'Reply' indicator if we don't have the author.
				// But let's try to extract it from the content if it starts with @user?
				// For now, let's just make it a prominent line.
				replyHint = "\n  " + lipgloss.NewStyle().
					Foreground(lipgloss.Color("#555555")).
					Italic(true).
					Render("⬑ Reply")
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
			itemContent := fmt.Sprintf("%s%s  %s%s\n%s\n  %s",
				author, statusText, timestamp, replyHint, body, common.MetadataStyle.Render(meta))

			if i == m.cursor {
				itemContent = common.SelectedStyle.Render(itemContent)
				if m.confirmDelete {
					itemContent += "\n" + common.ConfirmStyle.Render("  Delete this rant? (y/n)")
				}
			} else {
				itemContent = common.UnselectedStyle.Render(itemContent)
			}

			listBuilder.WriteString(itemContent)
			listBuilder.WriteString("\n")
		}

		// Scroll Bar Logic
		totalRants := len(m.rants)
		listString := strings.TrimSuffix(listBuilder.String(), "\n")
		listHeight := lipgloss.Height(listString)

		if totalRants > visibleCount {
			thumbHeight := int(float64(visibleCount) / float64(totalRants) * float64(listHeight))
			if thumbHeight < 1 {
				thumbHeight = 1
			}

			thumbStart := int(float64(m.startIndex) / float64(totalRants) * float64(listHeight))
			if thumbStart+thumbHeight > listHeight {
				thumbStart = listHeight - thumbHeight
			}

			var sb strings.Builder
			for j := 0; j < listHeight; j++ {
				if j >= thumbStart && j < thumbStart+thumbHeight {
					sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700")).Render("┃"))
				} else {
					sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Render("┃"))
				}
				if j < listHeight-1 {
					sb.WriteString("\n")
				}
			}

			// Join list and scroll bar with 12 spaces margin for better separation
			joined := lipgloss.JoinHorizontal(lipgloss.Top,
				listString,
				lipgloss.NewStyle().MarginLeft(12).Render(sb.String()))
			b.WriteString(joined)
		} else {
			b.WriteString(listString)
		}
	}

	b.WriteString("\n")
	if m.loading && len(m.rants) > 0 {
		b.WriteString(fmt.Sprintf("  %s Refreshing...\n", m.spinner.View()))
	}

	b.WriteString(m.helpView())

	return b.String()
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
	ri := m.rants[m.cursor]
	r := ri.Rant

	var b strings.Builder

	// Header Layout (Consistent with feed)
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")

	// Hashtag precisely as used in feed view
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(fmt.Sprintf("#%s", m.hashtag))

	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	postCrumb := crumbStyle.Render(fmt.Sprintf("Post %s", r.ID))

	b.WriteString(title + tagline + "\n")

	breadcrumb := lipgloss.JoinHorizontal(lipgloss.Bottom, hashtag, separator, postCrumb)
	b.WriteString(breadcrumb + "\n")

	// Create a card for the content
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		MarginLeft(2).
		Width(74)

	var cardContent strings.Builder
	cardContent.WriteString(common.AuthorStyle.Render(r.Author) + "\n")
	cardContent.WriteString(common.TimestampStyle.Render(r.CreatedAt.Format("Monday, Jan 02, 2006 at 15:04")) + "\n")

	// Parent Context inside card
	if len(m.ancestors) > 0 {
		parent := m.ancestors[len(m.ancestors)-1]
		parentIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true).
			Render(fmt.Sprintf("⬑ Replying to @%s", parent.Author))
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
	meta := fmt.Sprintf("%s Likes: %d  |  ↩ Replies: %d",
		likeStyle.Render(likeIcon), r.LikesCount, r.RepliesCount)
	cardContent.WriteString(common.MetadataStyle.Render(meta) + "\n")

	if r.URL != "" {
		cardContent.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("🔗 URL: "+r.URL) + "\n")
	}

	if ri.Err != nil {
		cardContent.WriteString("\n" + common.ErrorStyle.Render(fmt.Sprintf("⚠️ Error: %v", ri.Err)))
	}

	b.WriteString(cardStyle.Render(cardContent.String()))

	// Replies Section
	if m.loadingReplies {
		b.WriteString("\n\n  " + m.spinner.View() + " Loading replies...")
	} else if len(m.replies) > 0 {
		b.WriteString("\n\n  " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Replies") + "\n")
		for _, r := range m.replies {
			author := common.AuthorStyle.Render(r.Author)
			timestamp := common.TimestampStyle.Render(r.CreatedAt.Format("Jan 02 15:04"))
			content := common.ContentStyle.Render(common.StripHashtag(r.Content, m.hashtag))

			indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
			replyContent := fmt.Sprintf("  %s %s\n  %s %s", author, timestamp, indicator, content)
			b.WriteString("\n" + replyContent + "\n")
		}
	} else if !m.loadingReplies {
		// b.WriteString("\n\n  No replies yet.")
	}
	b.WriteString("\n\n" + common.StatusBarStyle.Render("  l: like • c/C: reply • o: open • esc/q: back to list"))

	return b.String()
}

func (m Model) helpView() string {
	var items []string

	if m.showDetail {
		items = []string{
			"esc: back",
			"l: like",
			"c/C: reply",
			"o: open",
			"q: quit",
		}
		if len(m.ancestors) > 0 {
			items = append(items[:len(items)-1], "u: parent", "q: quit")
		}
	} else if len(m.rants) > 0 {
		items = []string{
			"j/k: focus",
			"enter: detail",
			"p/P: rant",
			"c/C: reply",
			"l: like",
			"r: refresh",
		}
		r := m.rants[m.cursor].Rant
		if r.IsOwn {
			items = append(items, "e/E: edit", "d: delete")
		}
		items = append(items, "q: quit")
	} else {
		items = []string{
			"p/P: rant",
			"r: refresh",
			"q: quit",
		}
	}

	return common.StatusBarStyle.Render("  " + strings.Join(items, " • "))
}
