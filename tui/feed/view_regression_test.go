package feed

import (
	"strings"
	"testing"
	"time"

	"terminalrant/domain"
)

func TestRenderDetailView_ContainsThreadAndMediaSections(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 44
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
	m.replies = []domain.Rant{{ID: "r1", Username: "carol", AccountID: "acct-c", Content: "reply", CreatedAt: time.Now().Add(-time.Minute)}}

	out := m.renderDetailView()
	mustContain := []string{"Post root", "Parent Thread:", "Replies", "Media (1)", "URL:"}
	for _, needle := range mustContain {
		if !strings.Contains(out, needle) {
			t.Fatalf("detail view missing %q", needle)
		}
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
	m.profilePosts = []domain.Rant{{
		ID:        "p1",
		Author:    "User 42",
		Username:  "u42",
		AccountID: "42",
		Content:   "first post #terminalrant",
		CreatedAt: time.Now(),
	}}

	out := m.renderProfileView()
	mustContain := []string{"Profile @u42", "Posts 2  Followers 10  Following 1", "Posts", "first post"}
	for _, needle := range mustContain {
		if !strings.Contains(out, needle) {
			t.Fatalf("profile view missing %q", needle)
		}
	}
}
