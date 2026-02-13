package feed

import (
	"fmt"
	"strings"
	"time"

	"terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) helpView() string {
	var items []string

	if m.showProfile {
		items = []string{
			"j/k: focus",
			"enter: open",
			"f: follow",
			"B: blocked",
			"esc/q: back",
			"?: all keys",
		}
	} else if m.showDetail {
		items = []string{
			"j/k: focus",
			"enter: open",
			"l: like",
			"f: follow",
			"z/Z: profile",
			"h/H: top/home",
			"esc/q: back",
			"?: all keys",
		}
	} else if len(m.rants) > 0 {
		items = []string{
			"j/k: focus",
			"enter: detail",
			"p/P: rant",
			"l: like",
			"f: follow",
			"z/Z: profile",
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

	wrapWidth := max(m.width-2, 16)
	hints := common.StatusBarStyle.
		Width(wrapWidth).
		Render("  " + strings.Join(items, " • "))
	creator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Italic(true).
		PaddingTop(1).
		Width(wrapWidth).
		Render(fmt.Sprintf("  Created by @CrestNiraj12 • https://github.com/CrestNiraj12 • g: visit\n  © %d CrestNiraj12", time.Now().Year()))
	return hints + "\n" + creator
}

func (m Model) renderKeyDialog() string {
	var lines []string
	if m.showProfile {
		lines = []string{
			"j/k or up/down  move focus",
			"enter           open selected post detail",
			"f               follow/unfollow profile owner",
			"B               show blocked users",
			"esc / q         back",
			"ctrl+c          force quit",
			"?               toggle this dialog",
		}
	} else if m.showDetail {
		lines = []string{
			"j/k or up/down  move focus",
			"enter           open selected reply thread",
			"l               like/dislike selected post",
			"f               follow/unfollow selected user",
			"z               open selected user profile",
			"Z               open own profile",
			"i               toggle image previews",
			"I               open selected media",
			"c / C           reply via editor / inline",
			"x / X           hide post / toggle hidden posts",
			"b               block selected user",
			"B               show blocked users",
			"u               open parent post",
			"r               refresh replies",
			"o               open post URL",
			"v               edit profile",
			"g               open creator GitHub",
			"h               scroll to top of post",
			"H               go to feed home",
			"esc / q         back",
			"ctrl+c          force quit",
			"?               toggle this dialog",
		}
	} else if len(m.rants) > 0 {
		lines = []string{
			"j/k or up/down  move focus",
			"enter           open detail",
			"t / T           next/prev tab",
			"i               toggle image previews",
			"I               open selected media",
			"H               set hashtag feed tag",
			"p / P           new rant via editor / inline",
			"v               edit profile",
			"c / C           reply via editor / inline",
			"l               like/dislike selected post",
			"f               follow/unfollow selected user",
			"z               open selected user profile",
			"Z               open own profile",
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
			"i               toggle image previews",
			"I               open selected media",
			"H               set hashtag feed tag",
			"v               edit profile",
			"Z               open own profile",
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
		{label: "following", source: sourceFollowing},
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

func (m Model) renderBlockedView() string {
	var b strings.Builder
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel())
	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	blockedCrumb := crumbStyle.Render("Blocked Users")

	b.WriteString(title + tagline + "\n")
	b.WriteString(m.renderTabs() + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, hashtag, separator, blockedCrumb) + "\n\n")
	b.WriteString(m.renderBlockedUsersDialog())
	return b.String()
}

func (m Model) renderProfileView() string {
	var b strings.Builder
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render("🔥 TerminalRant")
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	b.WriteString(title + tagline + "\n")
	b.WriteString(m.renderTabs() + "\n")
	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	profileLabel := "Profile"
	if strings.TrimSpace(m.profile.Username) != "" {
		profileLabel = "Profile @" + m.profile.Username
	}
	breadcrumb := lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel()),
		separator,
		crumbStyle.Render(profileLabel),
	)
	b.WriteString(breadcrumb + "\n")

	if m.profileLoading {
		b.WriteString("  " + m.spinner.View() + " Loading profile...\n")
		b.WriteString("\n" + m.helpView())
		return b.String()
	}
	if m.profileErr != nil {
		b.WriteString(common.ErrorStyle.Render("  Error: " + m.profileErr.Error()))
		b.WriteString("\n\n" + m.helpView())
		return b.String()
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475A")).
		Padding(1, 2).
		MarginLeft(2).
		Width(74)
	if m.profileCursor == 0 {
		cardStyle = cardStyle.BorderForeground(lipgloss.Color("#FF8700"))
	}

	var card strings.Builder
	headerAuthor := renderAuthor(m.profile.Username, false, m.isFollowing(m.profile.ID))
	if strings.TrimSpace(m.profile.DisplayName) != "" {
		headerAuthor += " " + common.MetadataStyle.Render("("+m.profile.DisplayName+")")
	}
	card.WriteString(headerAuthor + "\n")
	card.WriteString(common.MetadataStyle.Render(
		fmt.Sprintf("Posts %d  Followers %d  Following %d", m.profile.PostsCount, m.profile.Followers, m.profile.Following),
	) + "\n\n")
	if !m.profileIsOwn && strings.TrimSpace(m.profile.ID) != "" {
		followLabel := "not following"
		if m.isFollowing(m.profile.ID) {
			followLabel = "following"
		}
		card.WriteString(common.MetadataStyle.Render("Follow: "+followLabel) + "\n")
		card.WriteString(common.MetadataStyle.Render("Keymap: f follow/unfollow") + "\n")
		if m.confirmFollow {
			card.WriteString(common.ConfirmStyle.Render("Unfollow? (y/n)") + "\n")
		}
		card.WriteString("\n")
	}
	if strings.TrimSpace(m.profile.Bio) != "" {
		card.WriteString(common.ContentStyle.Width(66).Render(m.profile.Bio) + "\n")
	}
	b.WriteString(cardStyle.Render(card.String()))

	b.WriteString("\n\n  " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Posts") + "\n")
	if len(m.profilePosts) == 0 {
		b.WriteString("\n  No posts.\n")
	} else {
		start := max(m.profileStart, 0)
		slots := m.profilePostSlots()
		end := min(start+slots, len(m.profilePosts))
		if start > 0 {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB454")).Bold(true).Render("▲ more posts above") + "\n")
		}
		for i := start; i < end; i++ {
			p := m.profilePosts[i]
			author := renderAuthor(p.Username, p.IsOwn, m.isFollowing(p.AccountID))
			ts := common.TimestampStyle.Render(p.CreatedAt.Format("Jan 02 15:04"))
			content, _ := splitContentAndTags(p.Content)
			content = strings.TrimSpace(content)
			if content == "" && len(p.Media) > 0 {
				content = "(media post)"
			}
			if content == "" {
				content = "(empty)"
			}
			lines := strings.Split(truncateToTwoLines(content, 56), "\n")
			indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
			var body strings.Builder
			for _, ln := range lines {
				body.WriteString("  " + indicator + common.ContentStyle.Render(ln) + "\n")
			}
			likeIcon := "♡"
			likeStyle := common.MetadataStyle
			if p.Liked {
				likeIcon = "♥"
				likeStyle = common.LikeActiveStyle
			}
			meta := fmt.Sprintf("%s %d  ↩ %d", likeStyle.Render(likeIcon), p.LikesCount, p.RepliesCount)
			item := fmt.Sprintf("  %s %s\n%s  %s", author, ts, strings.TrimSuffix(body.String(), "\n"), common.MetadataStyle.Render(meta))
			if m.profileCursor == i+1 {
				item = lipgloss.NewStyle().Background(lipgloss.Color("#333333")).Foreground(lipgloss.Color("#FFFFFF")).Render(item)
			}
			b.WriteString("\n" + item + "\n")
		}
		if end < len(m.profilePosts) {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#8BD5CA")).Bold(true).Render("▼ more posts below") + "\n")
		}
	}
	b.WriteString("\n\n" + m.helpView())
	return m.renderDetailViewport(b.String())
}
