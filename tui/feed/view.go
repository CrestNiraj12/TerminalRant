package feed

import (
	"fmt"
	"strings"

	"terminalrant/tui/common"
)

// View renders the feed as a string.
func (m Model) View() string {
	var b strings.Builder

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
		for i, rant := range m.rants {
			author := common.AuthorStyle.Render(rant.Author)
			if rant.IsOwn {
				author += common.OwnBadgeStyle.Render("(you)")
			}
			timestamp := common.TimestampStyle.Render(rant.CreatedAt.Format("Jan 02 15:04"))

			// Strip the tracked hashtag from display content.
			content := common.StripHashtag(rant.Content, m.hashtag)
			styledContent := common.ContentStyle.Render(content)

			entry := fmt.Sprintf("%s  %s\n%s", author, timestamp, styledContent)

			if i == m.cursor {
				entry = common.SelectedStyle.Render(entry)
				if m.showActions {
					entry += "\n" + m.renderActionBar()
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

func (m Model) renderActionBar() string {
	if m.confirmDelete {
		return common.ConfirmStyle.Render("  Delete this rant? (y/n)")
	}

	actions := []string{"✏️ Edit (buffer)", "📝 Edit (inline)", "🗑 Delete", "✕ Cancel"}
	var rendered []string
	for i, action := range actions {
		if i == m.actionCursor {
			rendered = append(rendered, common.ActionActiveStyle.Render("▸ "+action))
		} else {
			rendered = append(rendered, common.ActionInactiveStyle.Render("  "+action))
		}
	}
	return "  " + strings.Join(rendered, "   ")
}
