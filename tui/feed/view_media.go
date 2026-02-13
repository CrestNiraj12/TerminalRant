package feed

import (
	"fmt"
	"strings"

	"terminalrant/domain"

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
	urls := mediaPreviewURLs(r.Media)
	if len(urls) == 0 {
		return ""
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FA8DC")).
		Bold(true).
		Render("Media Preview (i: toggle, I: open all)")
	maxTiles := min(len(urls), 4)
	tiles := make([]string, 0, maxTiles)
	renderTile := func(i int, width int, showLabel bool) string {
		url := urls[i]
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
		text := content
		if showLabel {
			text = lipgloss.NewStyle().Foreground(lipgloss.Color("#7A7A7A")).Render(fmt.Sprintf("[%d]", i+1)) + "\n" + content
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475E73")).
			Width(width).
			Padding(0, 1).
			Render(text)
	}

	body := ""
	switch maxTiles {
	case 1:
		// Single image: fill the whole preview area (2x2-equivalent footprint).
		url := urls[0]
		baseKey := mediaPreviewBaseKey(url)
		singleKey := mediaPreviewSingleKey(url)
		content := "queued"
		if preview, ok := m.mediaPreview[singleKey]; ok && preview != "" {
			content = preview
		} else if m.mediaLoading[singleKey] {
			// While high-res is loading, show base preview if available.
			if basePreview, ok := m.mediaPreview[baseKey]; ok && basePreview != "" {
				content = basePreview + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#7A7A7A")).Render("enhancing...")
			} else {
				content = m.spinner.View() + " loading..."
			}
		} else if m.mediaLoading[baseKey] {
			content = m.spinner.View() + " loading..."
		} else if preview, ok := m.mediaPreview[baseKey]; ok {
			if preview == "" {
				content = "preview unavailable"
			} else {
				content = preview
			}
		}
		body = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475E73")).
			Width(49).
			Padding(0, 1).
			Render(content)
	case 2:
		// Two images: 1x2 layout.
		tiles = append(tiles, renderTile(0, 24, true), renderTile(1, 24, true))
		body = lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], " ", tiles[1])
	case 3:
		// 2x2 grid with one empty cell.
		tiles = append(tiles, renderTile(0, 24, true), renderTile(1, 24, true), renderTile(2, 24, true))
		top := lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], " ", tiles[1])
		bottom := tiles[2]
		body = top + "\n" + bottom
	default:
		// 4+ images: 2x2 grid + overflow indicator.
		tiles = append(tiles, renderTile(0, 24, true), renderTile(1, 24, true), renderTile(2, 24, true), renderTile(3, 24, true))
		top := lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], " ", tiles[1])
		bottom := lipgloss.JoinHorizontal(lipgloss.Top, tiles[2], " ", tiles[3])
		body = top + "\n" + bottom
	}
	if len(urls) > 4 {
		body += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E8E")).
			Render(fmt.Sprintf("+%d more", len(urls)-4))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3A4E63")).
		Padding(0, 1).
		Render(header + "\n" + body)
}
