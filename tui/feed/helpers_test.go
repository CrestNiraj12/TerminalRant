package feed

import (
	"strings"
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestDetailReplyGateAndWrappedLines(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.height = 20
	main := makeRant("x", time.Now(), "acct-a")
	main.Content = strings.Repeat("abcdefghij", 15)
	main.Media = []domain.MediaAttachment{{ID: "m"}}
	m.rants = []RantItem{{Rant: main, Status: StatusNormal}}
	m.ancestors = []domain.Rant{{ID: "p"}}
	gate := m.detailReplyGate()
	if gate < 0 {
		t.Fatalf("detail reply gate must be non-negative")
	}
	if estimateWrappedLines("abc\ndef", 2) < 3 {
		t.Fatalf("wrapped line estimation should account for wrapping + newlines")
	}
}

func TestOrganizeThreadReplies_NestsAndAppendsOrphans(t *testing.T) {
	desc := []domain.Rant{
		{ID: "c1", InReplyToID: "root"},
		{ID: "c2", InReplyToID: "c1"},
		{ID: "orphan", InReplyToID: "x"},
	}
	out := organizeThreadReplies("root", desc)
	if len(out) != 3 {
		t.Fatalf("expected all descendants returned, got %d", len(out))
	}
	if out[0].ID != "c1" || out[1].ID != "c2" {
		t.Fatalf("expected threaded order first, got %#v", out)
	}
}

func TestIsSafeExternalURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "https", in: "https://example.com/post/1", want: true},
		{name: "http", in: "http://example.com/post/1", want: true},
		{name: "javascript", in: "javascript:alert(1)", want: false},
		{name: "file", in: "file:///etc/passwd", want: false},
		{name: "mailto", in: "mailto:a@example.com", want: false},
		{name: "relative", in: "/local/path", want: false},
		{name: "invalid", in: "://bad", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSafeExternalURL(tc.in); got != tc.want {
				t.Fatalf("isSafeExternalURL(%q) got %v want %v", tc.in, got, tc.want)
			}
		})
	}
}
