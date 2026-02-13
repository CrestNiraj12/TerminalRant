package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ParsesEnvAndDefaults(t *testing.T) {
	t.Setenv("TERMINALRANT_INSTANCE", "https://example.social/")
	t.Setenv("TERMINALRANT_AUTH_DIR", t.TempDir())
	t.Setenv("TERMINALRANT_OAUTH_CALLBACK_PORT", "45146")
	t.Setenv("TERMINALRANT_HASHTAG", "mytag")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.InstanceURL != "https://example.social" {
		t.Fatalf("instance must be normalized: %q", cfg.InstanceURL)
	}
	if cfg.OAuthCallbackPort != 45146 || cfg.Hashtag != "mytag" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoad_RejectsNonHTTPS(t *testing.T) {
	t.Setenv("TERMINALRANT_INSTANCE", "http://insecure.local")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for non-https instance")
	}
}

func TestUIState_LoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ui_state.json")

	st, err := LoadUIState(path)
	if err != nil {
		t.Fatalf("missing state should not error: %v", err)
	}
	if st != (UIState{}) {
		t.Fatalf("expected empty state for missing file")
	}

	want := UIState{Hashtag: "terminalrant", FeedSource: "trending"}
	if err := SaveUIState(path, want); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	got, err := LoadUIState(path)
	if err != nil {
		t.Fatalf("load after save failed: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected loaded state got=%#v want=%#v", got, want)
	}

	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("write corrupt state failed: %v", err)
	}
	if _, err := LoadUIState(path); err == nil {
		t.Fatalf("expected parse error for invalid json")
	}
}
