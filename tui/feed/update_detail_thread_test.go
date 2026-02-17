package feed

import (
	"errors"
	"testing"
)

func TestMediaPreviewLoaded_ProfileErrorDoesNotCacheUnavailable(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	key := profileAvatarPreviewKey("https://cdn.example/avatar.webp")
	m.mediaLoading[key] = true

	updated, _ := m.handleDetailThreadMsg(MediaPreviewLoadedMsg{
		Key: key,
		Err: errors.New("fetch failed"),
	})
	if _, ok := updated.mediaLoading[key]; ok {
		t.Fatalf("expected loading key to be cleared for profile avatar")
	}
	if _, ok := updated.mediaPreview[key]; ok {
		t.Fatalf("profile avatar failures should not be cached as unavailable")
	}
}

func TestMediaPreviewLoaded_PostErrorCachesUnavailable(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	key := mediaPreviewBaseKey("https://cdn.example/post.webp")
	m.mediaLoading[key] = true

	updated, _ := m.handleDetailThreadMsg(MediaPreviewLoadedMsg{
		Key: key,
		Err: errors.New("fetch failed"),
	})
	if _, ok := updated.mediaLoading[key]; ok {
		t.Fatalf("expected loading key to be cleared for post media")
	}
	if got, ok := updated.mediaPreview[key]; !ok || got != "" {
		t.Fatalf("post media failures should cache empty preview, got ok=%v value=%q", ok, got)
	}
}
