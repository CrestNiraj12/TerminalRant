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
	title := common.AppTitleStyle.Render("🔥 TerminalRant")
	hashtag := common.HashtagStyle.Render(fmt.Sprintf(" #%s", m.hashtag))
	b.WriteString(title + hashtag + "\n\n")

	// Content area
	if m.loading && len(m.rants) == 0 {
		b.WriteString(fmt.Sprintf("  %s Loading rants...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(common.ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n  Press r to retry.\n")
	} else if len(m.rants) == 0 {
		b.WriteString("  No rants yet. Be the first!\n")
	} else {
		for i, rantItem := range m.rants {
			rant := rantItem.Rant
			author := common.AuthorStyle.Render(rant.Author)
			if rant.IsOwn {
				author += common.OwnBadgeStyle.Render("(you)")
			}
			timestamp := common.TimestampStyle.Render(rant.CreatedAt.Format("Jan 02 15:04"))

			// Sync status indicator
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

			// Strip the tracked hashtag from display content.
			content := common.StripHashtag(rant.Content, m.hashtag)

			// TRUNCATION: Max 2 lines for preview
			preview := truncateToTwoLines(content, 70) // Width is approximate, ideally use real width

			styledContent := common.ContentStyle.Render(preview)

			entry := fmt.Sprintf("%s%s  %s\n%s", author, statusText, timestamp, styledContent)

			if i == m.cursor {
				entry = common.SelectedStyle.Render(entry)
				if m.confirmDelete {
					entry += "\n" + common.ConfirmStyle.Render("  Delete this rant? (y/n)")
				}
			} else {
				entry = common.UnselectedStyle.Render(entry)
			}

			b.WriteString(entry)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.loading && len(m.rants) > 0 {
		b.WriteString(fmt.Sprintf("  %s Refreshing...\n", m.spinner.View()))
	}

	b.WriteString(common.StatusBarStyle.Render("  p/P: post • e/E: edit • d: delete • r: refresh • j/k: navigate • q: quit"))

	return b.String()
}

// truncateToTwoLines wraps and truncates text to at most 2 lines.
func truncateToTwoLines(text string, width int) string {
	// Simple wordwrap and line counting
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 2 {
		return text
	}
	return strings.Join(lines[:2], "\n") + "..."
}

func (m Model) renderDetailView() string {
	if len(m.rants) == 0 {
		return "No rant selected."
	}
	ri := m.rants[m.cursor]
	r := ri.Rant

	var b strings.Builder

	// Breadcrumb header
	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	separator := crumbStyle.Render(" > ")
	title := common.AppTitleStyle.Render("🔥 TerminalRant")
	hashtag := common.HashtagStyle.Render("#" + m.hashtag)
	detailCrumb := crumbStyle.Render("Detail")

	b.WriteString("  " + title + separator + hashtag + separator + detailCrumb + "\n\n")

	// Create a card for the content
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		MarginLeft(2).
		Width(74)

	var cardContent strings.Builder
	cardContent.WriteString(common.AuthorStyle.Render(r.Author) + "\n")
	cardContent.WriteString(common.TimestampStyle.Render(r.CreatedAt.Format("Monday, Jan 02, 2006 at 15:04")) + "\n\n")

	// Full content (wrapped) - strip hashtag for display
	displayContent := common.StripHashtag(r.Content, m.hashtag)
	content := common.ContentStyle.Width(66).Render(displayContent)
	cardContent.WriteString(content + "\n")

	if r.URL != "" {
		cardContent.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("🔗 URL: "+r.URL) + "\n")
	}

	if ri.Err != nil {
		cardContent.WriteString("\n" + common.ErrorStyle.Render(fmt.Sprintf("⚠️ Error: %v", ri.Err)))
	}

	b.WriteString(cardStyle.Render(cardContent.String()))
	b.WriteString("\n\n" + common.StatusBarStyle.Render("  o: open • esc/q: back to list"))

	return b.String()
}
