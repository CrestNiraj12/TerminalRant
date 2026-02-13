//go:build smoke

package mastodon

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type envToken struct{}

func (envToken) AccessToken() (string, error) {
	tok := strings.TrimSpace(os.Getenv("TERMINALRANT_TOKEN"))
	if tok == "" {
		return "", fmt.Errorf("TERMINALRANT_TOKEN is empty")
	}
	return tok, nil
}

func smokeClient(t *testing.T) *Client {
	t.Helper()
	base := strings.TrimSpace(os.Getenv("TERMINALRANT_BASE_URL"))
	if base == "" {
		t.Skip("TERMINALRANT_BASE_URL not set")
	}
	if strings.TrimSpace(os.Getenv("TERMINALRANT_TOKEN")) == "" {
		t.Skip("TERMINALRANT_TOKEN not set")
	}
	return NewClient(base, envToken{})
}

func TestSmoke_FetchTimelinesAndThread(t *testing.T) {
	client := smokeClient(t)
	timeline := NewTimelineService(client, "")

	home, err := timeline.FetchHomePage(context.Background(), 5, "")
	if err != nil {
		t.Fatalf("home timeline failed: %v", err)
	}
	if len(home) > 0 {
		_, _, err := timeline.FetchThread(context.Background(), home[0].ID)
		if err != nil {
			t.Fatalf("thread fetch failed: %v", err)
		}
	}
	_, _ = timeline.FetchByHashtag(context.Background(), "terminalrant", 5)
}

func TestSmoke_MutationRoundtrip_OptIn(t *testing.T) {
	if os.Getenv("SMOKE_ALLOW_MUTATION") != "true" {
		t.Skip("SMOKE_ALLOW_MUTATION=true required")
	}
	client := smokeClient(t)
	post := NewPostService(client)

	marker := fmt.Sprintf("smoke-%d", time.Now().Unix())
	r, err := post.Post(context.Background(), "smoke post "+marker, "terminalrant")
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if err := post.Delete(context.Background(), r.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
