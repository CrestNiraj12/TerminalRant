package feed

import (
	"fmt"
	"strings"

	"github.com/CrestNiraj12/terminalrant/domain"

	"github.com/charmbracelet/lipgloss"
)

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

func wrapAndTruncate(text string, width int, maxLines int) []string {
	if width < 8 {
		width = 8
	}
	if maxLines < 1 {
		maxLines = 1
	}
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, maxLines)
	line := ""
	consume := func() {
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
	}
	for _, word := range words {
		w := []rune(word)
		if len(w) > width {
			if line != "" {
				consume()
				if len(lines) == maxLines {
					break
				}
			}
			for len(w) > width {
				lines = append(lines, string(w[:width]))
				w = w[width:]
				if len(lines) == maxLines {
					break
				}
			}
			if len(lines) == maxLines {
				break
			}
			line = string(w)
			continue
		}
		if line == "" {
			line = word
			continue
		}
		if len([]rune(line))+1+len([]rune(word)) <= width {
			line += " " + word
			continue
		}
		consume()
		if len(lines) == maxLines {
			break
		}
		line = word
	}
	if len(lines) < maxLines && line != "" {
		lines = append(lines, line)
	}
	full := strings.Join(words, " ")
	visible := strings.Join(lines, " ")
	if len([]rune(visible)) < len([]rune(full)) && len(lines) > 0 {
		last := []rune(lines[len(lines)-1])
		if len(last) > width-3 {
			last = last[:max(width-3, 0)]
		}
		lines[len(lines)-1] = strings.TrimSpace(string(last) + "...")
	}
	return lines
}

func (m Model) renderSelectedMediaPreviewPanel() string {
	if !m.showMediaPreview {
		return ""
	}
	r := m.getSelectedRant()
	if m.showDetail {
		if m.focusedRant != nil {
			r = *m.focusedRant
		} else if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
			r = m.rants[m.cursor].Rant
		}
	}
	targets := mediaPreviewTargets(r.Media)
	if len(targets) == 0 {
		return ""
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FA8DC")).
		Bold(true).
		Render("Media Preview (i: toggle, I: open all)")
	_, _, tileW, tileH := m.previewSizing(len(targets))
	displayTargets := targets
	if !m.showDetail && len(displayTargets) > 1 {
		displayTargets = displayTargets[:1]
	}
	tiles := make([]string, 0, len(displayTargets))
	renderAlt := func(desc string) string {
		desc = strings.TrimSpace(desc)
		if desc == "" {
			desc = "(no alt)"
		}
		lines := wrapAndTruncate(desc, tileW, 3)
		if len(lines) < 2 {
			lines = append(lines, "")
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E8E")).
			Faint(true).
			Width(tileW).
			Render("alt: " + strings.Join(lines, "\n"))
	}
	renderTile := func(i int) string {
		target := displayTargets[i]
		url := target.URL
		baseKey := mediaPreviewBaseKey(url)
		content := "queued"
		if m.mediaLoading[baseKey] {
			content = m.spinner.View() + " loading..."
		} else if preview, ok := m.mediaPreview[baseKey]; ok {
			if preview == "" {
				content = "preview unavailable"
			} else {
				content = preview
			}
		}
		tile := lipgloss.NewStyle().
			Width(tileW).
			Height(tileH).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Padding(0, 0).
			Render(content)
		return renderAlt(target.Description) + "\n" + tile
	}

	body := ""
	columnGap := " "
	rowGap := "\n"
	if m.showDetail {
		columnGap = "   "
		rowGap = "\n\n"
	}
	switch len(displayTargets) {
	case 1:
		body = renderTile(0)
		if !m.showDetail && len(targets) > 1 {
			more := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E8E8E")).
				Bold(true).
				Width(4).
				Height(tileH).
				AlignVertical(lipgloss.Center).
				Render(fmt.Sprintf("+%d", len(targets)-1))
			body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", more)
		}
	case 2:
		// Two images: 1x2 layout.
		tiles = append(tiles, renderTile(0), renderTile(1))
		body = lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], columnGap, tiles[1])
	case 3:
		// 2x2 grid with one empty cell.
		tiles = append(tiles, renderTile(0), renderTile(1), renderTile(2))
		top := lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], columnGap, tiles[1])
		bottom := tiles[2]
		body = top + rowGap + bottom
	default:
		// 4+ images: render all previews as 2-column rows.
		for i := 0; i < len(displayTargets); i += 2 {
			if i+1 < len(displayTargets) {
				row := lipgloss.JoinHorizontal(lipgloss.Top, renderTile(i), columnGap, renderTile(i+1))
				if body == "" {
					body = row
				} else {
					body += rowGap + row
				}
			} else {
				if body == "" {
					body = renderTile(i)
				} else {
					body += rowGap + renderTile(i)
				}
			}
		}
	}
	return header + "\n\n" + body
}
