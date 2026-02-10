package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EnvEditor prepares an external editor command using $EDITOR (fallback: "vi").
// It does NOT run the editor itself — callers use tea.Exec with the returned
// *exec.Cmd so Bubble Tea properly suspends raw terminal mode.
type EnvEditor struct{}

// NewEnvEditor creates an EnvEditor.
func NewEnvEditor() *EnvEditor {
	return &EnvEditor{}
}

const instructionComment = `<!-- 
TerminalRant: Edit your rant below.

- SAVE and EXIT to post/update (e.g., :wq in vi).
- Emptying the file or making NO CHANGES will cancel.
- The tracked hashtag will be added automatically.
-->

`

// Cmd prepares an *exec.Cmd for the editor and a temp file path.
// It writes the provided content (and an instruction comment) to the temp file.
func (e *EnvEditor) Cmd(content string) (*exec.Cmd, string, error) {
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "vi"
	}

	tmpFile, err := os.CreateTemp("", "terminalrant-*.md")
	if err != nil {
		return nil, "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(instructionComment + content); err != nil {
		os.Remove(tmpPath)
		return nil, "", fmt.Errorf("writing to temp file: %w", err)
	}

	cmd := exec.Command(editorCmd, "+", tmpPath)
	return cmd, tmpPath, nil
}

// ReadContent reads the temp file, trims whitespace, and removes the file.
// It strips the instruction comment before returning.
func (e *EnvEditor) ReadContent(path string) (string, error) {
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading temp file: %w", err)
	}

	content := string(data)
	if idx := strings.Index(content, "-->"); idx != -1 {
		content = content[idx+3:]
	}
	return strings.TrimSpace(content), nil
}
