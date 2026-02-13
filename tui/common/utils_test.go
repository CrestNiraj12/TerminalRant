package common

import "testing"

func TestStripHashtag(t *testing.T) {
	if got := StripHashtag("Hello #TerminalRant", "terminalrant"); got != "Hello" {
		t.Fatalf("unexpected strip result: %q", got)
	}
	if got := StripHashtag("Hello #other", "terminalrant"); got != "Hello #other" {
		t.Fatalf("should not strip other hashtag: %q", got)
	}
	if got := StripHashtag("Hello", ""); got != "Hello" {
		t.Fatalf("empty hashtag should be no-op")
	}
}
