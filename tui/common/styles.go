package common

import "github.com/charmbracelet/lipgloss"

var (
	// AppTitleStyle styles the application title. Rendered at call site with content.
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			Padding(1, 2, 0, 1)

	// HashtagStyle styles the hashtag shown next to the title.
	HashtagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95")).
			Bold(true)

	// TaglineStyle styles the app's tagline.
	TaglineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")). // Dimmed grey
			Italic(true).
			MarginLeft(1)

	// AuthorStyle styles the rant author name.
	AuthorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DC4E4"))

	// TimestampStyle styles timestamps.
	TimestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D"))

	// ContentStyle styles rant content text.
	ContentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CAD3F5"))

	// SelectedStyle highlights the currently selected rant.
	SelectedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6600")).
			Padding(0, 1)

	// OwnBadgeStyle highlights posts that belong to the user.
	OwnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95")).
			Bold(true).
			MarginLeft(1)

	// UnselectedStyle gives unselected rants a subtle greyed-out border.
	UnselectedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).
			Padding(0, 1)

	// StatusBarStyle styles the bottom status bar.
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D")).
			Padding(1, 0, 0, 0)

	// ActionActiveStyle styles the currently selected action in the menu.
	ActionActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6600")).
				Bold(true).
				Padding(0, 1)

	// ActionInactiveStyle styles unselected actions in the menu.
	ActionInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6E738D")).
				Padding(0, 1)

	// ConfirmStyle styles the delete confirmation prompt.
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ED8796")).
			Bold(true).
			Padding(0, 1)

	// ErrorStyle styles error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ED8796")).
			Bold(true)

	// SuccessStyle styles success messages.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95")).
			Bold(true)
)
