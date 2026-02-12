package common

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines shared key bindings across all views.
type KeyMap struct {
	Quit         key.Binding
	ForceQuit    key.Binding // ctrl+c — force quit from any view
	ToggleHints  key.Binding // ? — toggle hidden key hints
	Refresh      key.Binding
	LoadMore     key.Binding // disabled (legacy key)
	BlockUser    key.Binding // b — block selected user
	ManageBlocks key.Binding // B — manage blocked users
	HidePost     key.Binding // x — hide selected post locally
	ShowHidden   key.Binding // X — toggle hidden posts visibility
	EditProfile  key.Binding // v — edit current profile
	SwitchFeed   key.Binding // t — switch feed source
	SetHashtag   key.Binding // H — change hashtag
	NewEditor    key.Binding // p — compose via $EDITOR
	NewInline    key.Binding // P — compose via inline textarea
	Edit         key.Binding // e — fast edit own post (buffer)
	EditInline   key.Binding // E — fast edit own post (inline)
	Delete       key.Binding // d — fast delete own post
	Like         key.Binding // l — like/favorite
	Reply        key.Binding // r — reply via $EDITOR
	ReplyInline  key.Binding // ctrl+r — reply inline
	Up           key.Binding
	Down         key.Binding
	Open         key.Binding // o — open in browser
	GitHub       key.Binding // g — open creator GitHub profile
	Home         key.Binding // h — back to top of home feed
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		ToggleHints: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "show/hide all keys"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		LoadMore: key.NewBinding(
			key.WithHelp("", ""),
		),
		BlockUser: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "block user"),
		),
		ManageBlocks: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "blocked users"),
		),
		HidePost: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "hide post"),
		),
		ShowHidden: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "toggle hidden"),
		),
		EditProfile: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "edit profile"),
		),
		SwitchFeed: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "switch feed"),
		),
		SetHashtag: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "set hashtag"),
		),
		NewEditor: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "rant ($EDITOR)"),
		),
		NewInline: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "rant (inline)"),
		),
		Reply: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "reply ($EDITOR)"),
		),
		ReplyInline: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "reply (inline)"),
		),
		Like: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "like"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit (buffer)"),
		),
		EditInline: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "edit (inline)"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open"),
		),
		GitHub: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "creator github"),
		),
		Home: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "home"),
		),
	}
}
