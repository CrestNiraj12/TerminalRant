package common

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines shared key bindings across all views.
type KeyMap struct {
	Quit        key.Binding
	Refresh     key.Binding
	NewEditor   key.Binding // p — compose via $EDITOR
	NewInline   key.Binding // P — compose via inline textarea
	Edit        key.Binding // e — fast edit own post (buffer)
	EditInline  key.Binding // E — fast edit own post (inline)
	Delete      key.Binding // d — fast delete own post
	Like        key.Binding // l — like/favorite
	Reply       key.Binding // r — reply via $EDITOR
	ReplyInline key.Binding // ctrl+r — reply inline
	Up          key.Binding
	Down        key.Binding
	Open        key.Binding // o — open in browser
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
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
	}
}
