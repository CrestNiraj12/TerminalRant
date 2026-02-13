package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileTokenProvider_AccessToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("  abc123 \n"), 0o600); err != nil {
		t.Fatalf("write token failed: %v", err)
	}

	p := NewFileTokenProvider(path)
	got, err := p.AccessToken()
	if err != nil {
		t.Fatalf("access token failed: %v", err)
	}
	if got != "abc123" {
		t.Fatalf("unexpected token: %q", got)
	}
}

func TestFileTokenProvider_AccessTokenErrors(t *testing.T) {
	p := NewFileTokenProvider(filepath.Join(t.TempDir(), "missing"))
	if _, err := p.AccessToken(); err == nil {
		t.Fatalf("expected missing-file error")
	}

	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, []byte(" \n\t"), 0o600); err != nil {
		t.Fatalf("write empty token failed: %v", err)
	}
	p = NewFileTokenProvider(empty)
	_, err := p.AccessToken()
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty-token error, got: %v", err)
	}
}
