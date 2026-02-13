package mastodon

import (
	"strings"
	"testing"
	"time"
)

func TestStripHTML_DecodesEntitiesAndStripsTags(t *testing.T) {
	in := `<p>Hello &lt;world&gt; &amp; crew</p><script>x</script><br/>line2`
	got := stripHTML(in)
	if strings.Contains(got, "<p>") || strings.Contains(got, "<script>") {
		t.Fatalf("expected HTML tags stripped: %q", got)
	}
	if !strings.Contains(got, "<world>") || !strings.Contains(got, "&") {
		t.Fatalf("expected html entities decoded: %q", got)
	}
	if !strings.Contains(got, "line2") {
		t.Fatalf("expected line break retained: %q", got)
	}
}

func TestSanitizeForTerminal_RemovesEscapesAndControls(t *testing.T) {
	in := "ok\x1b[31mred\x1b[0m\x1b]8;;http://x\x07bad\x01\x02"
	got := sanitizeForTerminal(in)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("expected ansi removed: %q", got)
	}
	if strings.ContainsRune(got, '\x01') || strings.ContainsRune(got, '\x02') {
		t.Fatalf("expected controls removed: %q", got)
	}
	if !strings.Contains(got, "ok") || !strings.Contains(got, "red") {
		t.Fatalf("expected plain text preserved: %q", got)
	}
}

func TestMapStatuses_MapsFields(t *testing.T) {
	svc := &timelineService{currentAccountID: "self-id"}
	statuses := []mastodonStatus{{
		ID:              "1",
		Content:         "<p>hi &lt;you&gt;</p>",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		URL:             "https://example/p/1",
		Favourited:      true,
		FavouritesCount: 4,
		RepliesCount:    2,
		InReplyToID:     "42",
		Account: mastodonAccount{
			ID:          "self-id",
			DisplayName: "",
			Acct:        "user",
		},
		MediaAttachments: []mastodonMediaAttachment{{
			ID:         "m1",
			Type:       "image",
			URL:        "https://img",
			PreviewURL: "https://pimg",
			Meta: struct {
				Original struct {
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				} `json:"original"`
			}{},
		}},
	}}

	got := svc.mapStatuses(statuses)
	if len(got) != 1 {
		t.Fatalf("expected one mapped rant")
	}
	r := got[0]
	if !r.IsOwn || r.ID != "1" || r.Username != "user" || r.InReplyToID != "42" {
		t.Fatalf("unexpected mapping: %+v", r)
	}
	if !strings.Contains(r.Content, "<you>") {
		t.Fatalf("expected html entity decode in content: %q", r.Content)
	}
}

func TestSanitizeForTerminal_StripsMalformedSequences(t *testing.T) {
	in := "a\x1b[9999;9999Xb\x1b\\c\x7fd"
	got := sanitizeForTerminal(in)
	if strings.Contains(got, "\x1b") || strings.ContainsRune(got, '\x7f') {
		t.Fatalf("escape/control must be stripped: %q", got)
	}
	if got != "abcd" {
		t.Fatalf("unexpected sanitized content: %q", got)
	}
}

func TestMapStatuses_MissingOptionalFieldsStillMaps(t *testing.T) {
	svc := &timelineService{}
	st := mastodonStatus{
		ID:          "x1",
		Content:     "<p>plain</p>",
		CreatedAt:   "not-a-time",
		URL:         "",
		Account:     mastodonAccount{ID: "", DisplayName: "", Acct: "fallback"},
		InReplyToID: nil,
	}
	got := svc.mapStatuses([]mastodonStatus{st})
	if len(got) != 1 {
		t.Fatalf("expected one mapped item")
	}
	if got[0].Author != "fallback" || got[0].Username != "fallback" {
		t.Fatalf("expected acct fallback for author mapping: %#v", got[0])
	}
	if got[0].InReplyToID != "" {
		t.Fatalf("expected empty in-reply mapping for nil value")
	}
}

func TestMapMediaAttachments_EmptyAndPreviewFallback(t *testing.T) {
	in := []mastodonMediaAttachment{
		{
			ID:          "m1",
			Type:        "image",
			URL:         "",
			PreviewURL:  "https://preview",
			Description: "  desc  ",
			Meta: struct {
				Original struct {
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				} `json:"original"`
			}{},
		},
	}
	out := mapMediaAttachments(in)
	if len(out) != 1 {
		t.Fatalf("expected one mapped attachment")
	}
	if out[0].PreviewURL != "https://preview" || out[0].Description != "desc" {
		t.Fatalf("unexpected media mapping: %#v", out[0])
	}
}
