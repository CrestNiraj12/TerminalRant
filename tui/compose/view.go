package compose

import (
	"fmt"
	"strings"

	"terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

// View renders the compose view based on the active mode.
func (m Model) View() string {
	if m.err != nil {
		return common.ErrorStyle.Render("Error: "+m.err.Error()) + "\n"
	}

	switch m.mode {
	case editorMode:
		return m.status + "\n"

	case inlineMode:
		var b strings.Builder
		b.WriteString(common.AppTitleStyle.Render("🔥 TerminalRant"))
		if m.isReply && m.parentSummary != "" {
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("#555555")).
				Italic(true).
				Render(fmt.Sprintf("  Replying to: %s", m.parentSummary)) + "\n\n")
		} else if m.isEdit {
			b.WriteString("  Edit Rant\n\n")
		} else {
			b.WriteString("  New Rant\n\n")
		}
		b.WriteString(m.textarea.View())
		b.WriteString("\n\n")

		if m.status != "" {
			b.WriteString(common.StatusBarStyle.Render(m.status))
		} else {
			b.WriteString(common.StatusBarStyle.Render(
				fmt.Sprintf("  ctrl+d: post • esc: cancel • %d/500 chars",
					len(m.textarea.Value())),
			))
		}

		return b.String()
	}

	return ""
}
