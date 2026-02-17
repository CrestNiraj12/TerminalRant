package feed

import (
	"fmt"
	"strings"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
	"github.com/CrestNiraj12/terminalrant/tui/common"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) helpView() string {
	var items []string

	if m.showProfile {
		items = []string{
			"j/k: focus",
			"enter: open",
			"i: media",
			"I: open image",
			"o: open profile",
			"v/V: edit profile",
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
	var (
		core        []string
		includeMove bool
	)
	if m.showProfile {
		includeMove = true
		core = []string{
			"enter           open selected post detail",
			"i               toggle profile image preview",
			"I               open profile image in browser",
			"o               open profile URL in browser",
			"v / V           edit profile via editor / inline",
			"f               follow/unfollow profile owner",
			"B               show blocked users",
			"esc / q         back",
		}
	} else if m.showDetail {
		includeMove = true
		core = []string{
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
		}
		if m.canDeleteRant(m.getSelectedRant()) {
			core = append(core, "d               delete selected post")
		}
	} else if len(m.rants) > 0 {
		includeMove = true
		core = []string{
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
		}
		r := m.rants[m.cursor].Rant
		if r.IsOwn {
			core = append(core, "e / E           edit via editor / inline", "d               delete selected post")
		}
	} else {
		core = []string{
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
		}
	}
	lines := buildKeyDialogLines(core, includeMove)

	body := "Keyboard Shortcuts\n\n" + strings.Join(lines, "\n") + "\n\nPress ?, esc, q, or enter to close."
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF8700")).
		Padding(1, 2).
		Margin(1, 2).
		Render(body)
}

func buildKeyDialogLines(core []string, includeMove bool) []string {
	out := make([]string, 0, len(core)+5)
	if includeMove {
		out = append(out, "j/k or up/down  move focus")
	}
	out = append(out, "left/right      pan horizontally")
	out = append(out, core...)
	out = append(out, "ctrl+c          force quit", "?               toggle this dialog")
	return out
}

func (m Model) renderTabs() string {
	tabs := []struct {
		label  string
		source feedSource
	}{
		{label: domain.AppHashTag, source: sourceTerminalRant},
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
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render(domain.DisplayAppTitle())
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	hashtag := common.HashtagStyle.Margin(0, 0, 1, 2).Render(m.sourceLabel())
	crumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginBottom(1)
	separator := crumbStyle.Render(" > ")
	blockedCrumb := crumbStyle.Render("Blocked Users")

	b.WriteString(title + tagline + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, hashtag, separator, blockedCrumb) + "\n\n")
	b.WriteString(m.renderBlockedUsersDialog())
	return b.String()
}

func (m Model) renderProfileView() string {
	var b strings.Builder
	title := common.AppTitleStyle.Padding(1, 0, 0, 1).Render(domain.DisplayAppTitle())
	tagline := common.TaglineStyle.Render("<Why leave terminal to rant!!>")
	b.WriteString(title + tagline + "\n")
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
	profilePreviewPanel := m.renderProfileAvatarPreviewPanel()
	hasProfilePreview := m.showMediaPreview
	postWidth := 74
	if hasProfilePreview {
		postWidth = m.currentPostPaneWidth()
		if postWidth < 40 {
			postWidth = 40
		}
	}
	contentWidth := max(postWidth-8, 24)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475A")).
		Padding(1, 2).
		MarginLeft(2).
		Width(postWidth)
	if m.profileCursor == 0 {
		cardStyle = cardStyle.BorderForeground(lipgloss.Color("#FF8700"))
	}

	var card strings.Builder
	headerAuthor := renderAuthor(m.profile.Username, m.profileIsOwn, m.isFollowing(m.profile.ID))
	displayName := strings.TrimSpace(m.profile.DisplayName)
	if displayName != "" {
		headerAuthor += " " + common.MetadataStyle.Render("("+displayName+")")
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
		card.WriteString(common.ContentStyle.Width(contentWidth).Render(m.profile.Bio) + "\n")
	}
	var body strings.Builder
	body.WriteString(cardStyle.Render(card.String()))

	body.WriteString("\n\n  " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Posts") + "\n")
	if len(m.profilePosts) == 0 {
		body.WriteString("\n  No posts.\n")
	} else {
		for i := 0; i < len(m.profilePosts); i++ {
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
			lines := strings.Split(truncateToTwoLines(content, max(contentWidth-10, 20)), "\n")
			indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("┃ ")
			var postBody strings.Builder
			for _, ln := range lines {
				postBody.WriteString("  " + indicator + common.ContentStyle.Render(ln) + "\n")
			}
			likeIcon := "♡"
			likeStyle := common.MetadataStyle
			if p.Liked {
				likeIcon = "♥"
				likeStyle = common.LikeActiveStyle
			}
			meta := fmt.Sprintf("%s %d  ↩ %d", likeStyle.Render(likeIcon), p.LikesCount, p.RepliesCount)
			item := fmt.Sprintf("  %s %s\n%s  %s", author, ts, strings.TrimSuffix(postBody.String(), "\n"), common.MetadataStyle.Render(meta))
			if mediaLine := renderMediaCompact(p.Media); mediaLine != "" {
				item += "\n  " + mediaLine
			}
			if m.profileCursor == i+1 {
				item = lipgloss.NewStyle().Background(lipgloss.Color("#333333")).Foreground(lipgloss.Color("#FFFFFF")).Render(item)
			}
			body.WriteString("\n" + item + "\n")
		}
	}
	body.WriteString("\n\n" + m.helpView())
	if hasProfilePreview {
		left := clampLinesToWidth(body.String(), postWidth+8)
		leftHeight := max(lipgloss.Height(left), 1)
		preview := clipLines(profilePreviewPanel, leftHeight)
		previewPane := lipgloss.NewStyle().
			MaxHeight(leftHeight).
			Render(preview)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", previewPane))
	} else {
		b.WriteString(body.String())
	}
	return m.renderDetailViewport(b.String())
}

func (m Model) renderProfileAvatarPreviewPanel() string {
	_, _, tw, th := m.previewSizing(1)
	avatar := "no avatar URL from API"
	if strings.TrimSpace(m.profile.AvatarURL) != "" {
		avatarKey := profileAvatarPreviewKey(m.profile.AvatarURL)
		avatar = "queued"
		if m.mediaLoading[avatarKey] {
			avatar = m.spinner.View() + " loading..."
		} else if p, ok := m.mediaPreview[avatarKey]; ok {
			if p == "" {
				avatar = "preview unavailable"
			} else {
				avatar = p
			}
		} else {
			if p, _, err := loadStaticMediaPreview(m.profile.AvatarURL, max(tw/2, 4), max(th, 2), false); err == nil && p != "" {
				m.mediaPreview[avatarKey] = p
				avatar = p
			} else {
				avatar = "preview unavailable"
			}
		}
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FA8DC")).
		Bold(true).
		Render("Profile Image Preview")
	body := lipgloss.NewStyle().
		Width(tw).
		Height(th).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(avatar)
	return header + "\n\n" + body
}
