package feed

import (
	"strings"
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderDetailView_ContainsThreadAndMediaSections(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 68
	m.showDetail = true
	m.rants = []RantItem{{Rant: domain.Rant{
		ID:        "root",
		AccountID: "acct-a",
		Author:    "Alice",
		Username:  "alice",
		Content:   "root post #terminalrant",
		CreatedAt: time.Now(),
		URL:       "https://example.com/post/root",
		Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
	}, Status: StatusNormal}}
	m.cursor = 0
	m.ancestors = []domain.Rant{{ID: "parent", Username: "bob", AccountID: "acct-b", Content: "parent", CreatedAt: time.Now().Add(-time.Hour)}}
	m.replies = []domain.Rant{{
		ID:        "r1",
		Username:  "carol",
		AccountID: "acct-c",
		Content:   "reply",
		CreatedAt: time.Now().Add(-time.Minute),
		Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "reply-u1"}},
	}}

	out := m.renderDetailView()
	mustContain := []string{"Post root", "Parent Thread:", "Replies", "Media (1)", "URL:", "🖼 1"}
	for _, needle := range mustContain {
		if !strings.Contains(out, needle) {
			t.Fatalf("detail view missing %q", needle)
		}
	}
}

func TestRenderDetailView_DeleteConfirmShowsBeforeBody(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 44
	m.showDetail = true
	m.confirmDelete = true
	m.rants = []RantItem{{Rant: domain.Rant{
		ID:        "root",
		AccountID: "acct-a",
		Author:    "Alice",
		Username:  "alice",
		Content:   "body text here",
		CreatedAt: time.Now(),
	}, Status: StatusNormal}}
	m.cursor = 0

	out := m.renderDetailView()
	confirmNeedle := "Delete this post? (y/n)"
	bodyNeedle := "body text here"
	if !strings.Contains(out, confirmNeedle) {
		t.Fatalf("detail delete confirmation missing")
	}
	if !strings.Contains(out, bodyNeedle) {
		t.Fatalf("detail body missing")
	}
	if strings.Index(out, confirmNeedle) > strings.Index(out, bodyNeedle) {
		t.Fatalf("delete confirmation should render before body")
	}
}

func TestRenderProfileView_ContainsProfileCardAndPostsSection(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 44
	m.showProfile = true
	m.profile = appProfile("42", "u42")
	m.profile.DisplayName = "User Forty Two"
	m.profile.PostsCount = 2
	m.profile.Followers = 10
	m.profile.Following = 1
	m.profile.Bio = "bio text"
	m.profile.AvatarURL = "https://cdn/avatar.png"
	m.profilePosts = []domain.Rant{{
		ID:        "p1",
		Author:    "User 42",
		Username:  "u42",
		AccountID: "42",
		Content:   "first post #terminalrant",
		CreatedAt: time.Now(),
	}}
	m.mediaPreview[profileAvatarPreviewKey(m.profile.AvatarURL)] = "AVATAR_ASCII"

	out := m.renderProfileView()
	mustContain := []string{"Profile @u42", "Posts 2  Followers 10  Following 1", "Posts", "AVATAR_ASCII"}
	for _, needle := range mustContain {
		if !strings.Contains(out, needle) {
			t.Fatalf("profile view missing %q", needle)
		}
	}
}

func TestRenderProfileView_LongBioStillShowsProfilePreviewPane(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 160
	m.height = 44
	m.showProfile = true
	m.profile = appProfile("42", "u42")
	m.profile.Bio = strings.Repeat("x", 1000)
	m.profile.AvatarURL = "https://cdn/avatar.png"
	m.mediaPreview[profileAvatarPreviewKey(m.profile.AvatarURL)] = "AVATAR_ASCII"

	out := m.renderProfileView()
	if !strings.Contains(out, "Profile Image Preview") {
		t.Fatalf("profile preview pane header missing")
	}
	if !strings.Contains(out, "AVATAR_ASCII") {
		t.Fatalf("profile preview image missing")
	}
}

func TestRenderProfileView_RendersPostsRegardlessOfProfileStart(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 44
	m.showProfile = true
	m.profile = appProfile("42", "u42")
	m.profileStart = 999
	m.profilePosts = []domain.Rant{{
		ID:        "p1",
		Author:    "User 42",
		Username:  "u42",
		AccountID: "42",
		Content:   "still visible post #terminalrant",
		CreatedAt: time.Now(),
	}}

	out := m.renderProfileView()
	if !strings.Contains(out, "still visible post") {
		t.Fatalf("expected profile post to render independent of profileStart")
	}
}

func TestRenderDetailView_MainCardWidthStableAcrossSelection(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 50
	m.showDetail = true
	m.showMediaPreview = true
	m.rants = []RantItem{{Rant: domain.Rant{
		ID:        "root",
		AccountID: "acct-a",
		Author:    "Alice",
		Username:  "alice",
		Content:   "root content",
		CreatedAt: time.Now(),
		Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
	}, Status: StatusNormal}}
	m.cursor = 0
	m.replies = []domain.Rant{{
		ID:        "r1",
		AccountID: "acct-b",
		Author:    "Bob",
		Username:  "bob",
		Content:   "reply without media",
		CreatedAt: time.Now(),
	}}

	cardBorderWidth := func(out string) int {
		for _, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "╭") && strings.Contains(ln, "╮") {
				return ansi.StringWidth(ln)
			}
		}
		return 0
	}

	m.detailCursor = 0
	wMain := cardBorderWidth(m.renderDetailView())
	m.detailCursor = 1
	wReply := cardBorderWidth(m.renderDetailView())
	if wMain == 0 || wReply == 0 {
		t.Fatalf("failed to detect detail card border widths: main=%d reply=%d", wMain, wReply)
	}
	if wMain != wReply {
		t.Fatalf("detail main card width changed with selection: main=%d reply=%d", wMain, wReply)
	}
}
