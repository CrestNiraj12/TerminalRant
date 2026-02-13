package editor

import (
	"os"
	"strings"
	"testing"
)

func TestCmd_UsesEditorAndWritesTemplate(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	e := NewEnvEditor()

	cmd, path, err := e.Cmd("hello", "@alice")
	if err != nil {
		t.Fatalf("cmd failed: %v", err)
	}
	if cmd.Path == "" || cmd.Args[0] == "" {
		t.Fatalf("expected command populated")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file failed: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "Replying to @alice") || !strings.Contains(text, "hello") {
		t.Fatalf("unexpected template content: %q", text)
	}
}

func TestReadContent_StripsInstructionAndDeletesFile(t *testing.T) {
	e := NewEnvEditor()
	f, err := os.CreateTemp("", "terminalrant-test-*.md")
	if err != nil {
		t.Fatalf("create temp failed: %v", err)
	}
	path := f.Name()
	_, _ = f.WriteString(instructionComment + "\nline1\nline2\n")
	_ = f.Close()

	content, err := e.ReadContent(path)
	if err != nil {
		t.Fatalf("read content failed: %v", err)
	}
	if content != "line1\nline2" {
		t.Fatalf("unexpected content: %q", content)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be deleted")
	}
}
