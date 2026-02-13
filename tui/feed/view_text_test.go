package feed

import (
	"strings"
	"testing"
)

func TestSplitContentAndTags(t *testing.T) {
	content, tags := splitContentAndTags("hi #One there #two\n#ONE")
	if strings.Contains(content, "#") {
		t.Fatalf("content still has hashtag: %q", content)
	}
	if len(tags) != 2 || tags[0] != "#one" || tags[1] != "#two" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestTruncateToTwoLines(t *testing.T) {
	got := truncateToTwoLines("a b c d e f g h i j k l m n o p", 8)
	lines := strings.Split(got, "\n")
	if len(lines) > 2 && !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis when truncated: %q", got)
	}
}

func TestClipLines(t *testing.T) {
	in := "a\nb\nc\nd"
	got := clipLines(in, 2)
	if strings.Count(got, "\n") != 1 || !strings.HasPrefix(got, "a\nb") {
		t.Fatalf("unexpected clipped output: %q", got)
	}
}
