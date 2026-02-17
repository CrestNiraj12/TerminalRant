package feed

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func truncateToTwoLines(text string, width int) string {
	if width < 12 {
		width = 12
	}
	// Render with width to handle both explicit newlines and wrapping.
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 2 {
		return wrapped
	}
	// Take first 2 lines and append ellipsis
	return strings.Join(lines[:2], "\n") + "..."
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

func authorStyleFor(username string, isOwn bool) lipgloss.Style {
	if isOwn {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A6DA95"))
	}
	palette := []string{
		"#7DC4E4", "#8BD5CA", "#F5A97F", "#C6A0F6", "#EBA0AC",
		"#A6DA95", "#F9E2AF", "#89B4FA", "#F38BA8", "#94E2D5",
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(username))))
	idx := int(h.Sum32()) % len(palette)
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(palette[idx]))
}

func renderAuthor(username string, isOwn bool, followed bool) string {
	local, domain := splitUsernameDomain(username)
	out := authorStyleFor(username, isOwn).Render("@" + local)
	if domain != "" {
		out += " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E8E")).
			Faint(true).
			Render("@" + domain)
	}
	if followed && !isOwn {
		out += lipgloss.NewStyle().Foreground(lipgloss.Color("#8BD5CA")).Faint(true).Render(" ✓")
	}
	return out
}

func splitUsernameDomain(username string) (local, domain string) {
	u := strings.TrimSpace(username)
	if u == "" {
		return "", ""
	}
	parts := strings.SplitN(u, "@", 2)
	local = strings.TrimSpace(parts[0])
	if local == "" {
		local = u
	}
	if len(parts) > 1 {
		domain = strings.TrimSpace(parts[1])
	}
	return local, domain
}

func clipLines(text string, maxLines int) string {
	if maxLines < 1 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n")
}

func clampLinesToWidth(text string, width int) string {
	if width <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		if ansi.StringWidth(ln) <= width {
			continue
		}
		lines[i] = ansi.Cut(ln, 0, width)
	}
	return strings.Join(lines, "\n")
}
