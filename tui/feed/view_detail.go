package feed

import (
	"fmt"
	"strings"

	"github.com/CrestNiraj12/terminalrant/domain"
	"github.com/CrestNiraj12/terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

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
	postWidth := 74
	// Keep detail post width stable regardless of which item is selected.
	// Otherwise selecting replies with/without media causes width jumps.
	if m.showMediaPreview {
		postWidth = m.currentPostPaneWidth()
		if postWidth < 52 {
			postWidth = 52
		}
	}
	contentWidth := max(postWidth-8, 24)

	var b strings.Builder

	// Header Layout (Consistent with feed)
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render(domain.DisplayAppTitle())
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")

	// Active source badge, consistent with feed view.
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel())

	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	postCrumb := crumbStyle.Render(fmt.Sprintf("Post %s", r.ID))

	b.WriteString(title + tagline + "\n")

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
		Width(postWidth)

	var cardContent strings.Builder
	headerAuthor := renderAuthor(r.Username, r.IsOwn, m.isFollowing(r.AccountID))
	if r.IsOwn {
		headerAuthor += common.OwnBadgeStyle.Render("(you)")
	}
	cardContent.WriteString(headerAuthor + " " + common.MetadataStyle.Render("("+r.Author+")") + "\n")
	if m.confirmFollow {
		action := "Follow"
		if !m.followTarget {
			action = "Unfollow"
		}
		cardContent.WriteString(common.ConfirmStyle.Render(fmt.Sprintf("%s @%s? (y/n)", action, m.followUsername)) + "\n")
	}
	cardContent.WriteString(common.TimestampStyle.Render(r.CreatedAt.Format("Monday, Jan 02, 2006 at 15:04")) + "\n")
	if m.confirmDelete {
		cardContent.WriteString(common.ConfirmStyle.Render("Delete this post? (y/n)") + "\n")
	}

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
	content := common.ContentStyle.Width(contentWidth).Render(displayContent)
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
		if !m.showMediaPreview {
			cardContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A0A0A0")).
				Faint(true).
				Render("preview hidden: press i") + "\n")
		}
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

	renderedCard := cardStyle.
		BorderForeground(lipgloss.Color("#45475A")).
		Render(cardContent.String())
	if m.detailCursor == 0 {
		renderedCard = cardStyle.
			BorderForeground(lipgloss.Color("#FF8700")).
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
		parentSummary := truncateToTwoLines(parentContent, contentWidth)

		parentCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(0, 1). // Use 0 padding on top/bottom to keep it compact
			MarginLeft(2).
			Width(postWidth).
			Render(fmt.Sprintf("%s %s\n%s",
				renderAuthor(parent.Username, parent.IsOwn, m.isFollowing(parent.AccountID)),
				common.TimestampStyle.Render(parent.CreatedAt.Format("Jan 02")),
				common.ContentStyle.Render(parentSummary)))

		parentView = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("  Parent Thread:") + "\n" + parentCard + "\n"
	}

	postBlock := parentView + renderedCard
	if panel := m.renderSelectedMediaPreviewPanel(); panel != "" {
		leftHeight := max(lipgloss.Height(postBlock), 1)
		preview := clipLines(panel, leftHeight)
		previewPane := lipgloss.NewStyle().
			MaxHeight(leftHeight).
			Render(preview)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, postBlock, "  ", previewPane))
	} else {
		b.WriteString(postBlock)
	}

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

			author := renderAuthor(r.Username, r.IsOwn, m.isFollowing(r.AccountID))
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
			if mediaLine := renderMediaCompact(r.Media); mediaLine != "" {
				replyContent += "\n  " + indentPrefix + mediaLine
			}

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

func (m Model) renderDetailViewport(content string) string {
	lines := strings.Split(content, "\n")
	if m.height <= 0 {
		return content
	}
	viewHeight := max(m.height-2, 8)
	maxScroll := max(len(lines)-viewHeight, 0)
	scroll := min(max(m.detailScrollLine, 0), maxScroll)
	end := min(scroll+viewHeight, len(lines))
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
