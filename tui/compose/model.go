package compose

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/app"
	"terminalrant/infra/editor"
)

// --- Mode ---

type mode int

const (
	editorMode mode = iota
	inlineMode
)

// --- Messages ---

// DoneMsg is sent when composing is complete (success or cancel).
type DoneMsg struct {
	Content string // Empty if cancelled
	RantID  string // ID of the rant being edited
	IsEdit  bool
	Err     error
}

// editorFinishedMsg is sent after the external editor exits.
type editorFinishedMsg struct {
	tmpPath string
	err     error
}

// --- Model ---

// Model holds the state for the compose view.
type Model struct {
	mode     mode
	post     app.PostService
	editor   *editor.EnvEditor
	hashtag  string
	status   string
	err      error
	textarea textarea.Model // Only used in inline mode
	tmpPath  string         // Temp file path for editor mode
	isEdit   bool
	rantID   string
	content  string // Initial content for editing
}

// NewEditor creates a compose model that opens $EDITOR via tea.Exec.
func NewEditor(post app.PostService, ed *editor.EnvEditor, hashtag string) Model {
	return Model{
		mode:    editorMode,
		post:    post,
		editor:  ed,
		hashtag: hashtag,
		status:  "Opening editor...",
	}
}

// NewEditorWithContent creates a compose model for editing an existing rant.
func NewEditorWithContent(post app.PostService, ed *editor.EnvEditor, hashtag string, rantID string, content string) Model {
	return Model{
		mode:    editorMode,
		post:    post,
		editor:  ed,
		hashtag: hashtag,
		status:  "Opening editor...",
		isEdit:  true,
		rantID:  rantID,
		content: content,
	}
}

// NewInline creates a compose model with an inline Bubble Tea textarea.
func NewInline(post app.PostService, hashtag string) Model {
	ta := textarea.New()
	ta.Placeholder = "What's grinding your gears?"
	ta.CharLimit = 500
	ta.SetWidth(72)
	ta.SetHeight(6)
	ta.Focus()

	return Model{
		mode:     inlineMode,
		post:     post,
		hashtag:  hashtag,
		textarea: ta,
	}
}

// NewInlineWithContent creates a compose model for editing an existing rant inline.
func NewInlineWithContent(post app.PostService, hashtag string, rantID string, content string) Model {
	ta := textarea.New()
	ta.SetValue(content)
	ta.SetWidth(72)
	ta.SetHeight(6)
	ta.Focus()

	return Model{
		mode:     inlineMode,
		post:     post,
		hashtag:  hashtag,
		textarea: ta,
		isEdit:   true,
		rantID:   rantID,
		content:  content,
	}
}

// Init returns the initial command for the active mode.
func (m Model) Init() tea.Cmd {
	switch m.mode {
	case editorMode:
		return m.launchEditor()
	case inlineMode:
		return textarea.Blink
	}
	return nil
}

// launchEditor prepares the editor command and uses tea.Exec to properly
// suspend Bubble Tea's raw terminal mode while the editor runs.
func (m *Model) launchEditor() tea.Cmd {
	cmd, tmpPath, err := m.editor.Cmd(m.content)
	if err != nil {
		return func() tea.Msg {
			return DoneMsg{Err: fmt.Errorf("preparing editor: %w", err)}
		}
	}
	m.tmpPath = tmpPath

	// tea.ExecProcess suspends Bubble Tea, runs the command with full terminal
	// control, then resumes Bubble Tea and delivers the callback message.
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{tmpPath: tmpPath, err: err}
	})
}

// Update handles messages for the compose view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	// --- Editor mode messages ---

	case editorFinishedMsg:
		if msg.err != nil {
			return m, done(DoneMsg{Err: fmt.Errorf("editor: %w", msg.err), IsEdit: m.isEdit})
		}

		// Read content from temp file.
		content, err := m.editor.ReadContent(msg.tmpPath)
		if err != nil {
			return m, done(DoneMsg{Err: err, IsEdit: m.isEdit, RantID: m.rantID})
		}

		if content == "" || content == m.content {
			return m, done(DoneMsg{IsEdit: m.isEdit, RantID: m.rantID}) // Cancel
		}

		return m, done(DoneMsg{Content: content, IsEdit: m.isEdit, RantID: m.rantID})

	// --- Inline mode messages ---

	case tea.KeyMsg:
		if m.mode != inlineMode {
			break
		}

		switch msg.String() {
		case "esc":
			return m, done(DoneMsg{IsEdit: m.isEdit}) // Cancel.

		case "ctrl+d":
			content := m.textarea.Value()
			if content == "" || content == m.content {
				return m, done(DoneMsg{IsEdit: m.isEdit, RantID: m.rantID})
			}
			return m, done(DoneMsg{Content: content, IsEdit: m.isEdit, RantID: m.rantID})
		}

		// Delegate to textarea for normal typing.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

		// --- Shared messages ---

	}

	// Pass through any remaining messages to textarea in inline mode.
	if m.mode == inlineMode {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	return m, nil
}

// done wraps a DoneMsg into a tea.Cmd for immediate delivery.
func done(msg DoneMsg) tea.Cmd {
	return func() tea.Msg { return msg }
}
