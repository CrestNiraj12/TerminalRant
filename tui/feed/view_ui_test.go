package feed

import (
	"strings"
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/app"
	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestView_RendersExpectedModeSections(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 40

	feed := m.View()
	if !strings.Contains(feed, "TerminalRant") {
		t.Fatalf("feed view missing title")
	}

	m.showBlocked = true
	blocked := m.View()
	if !strings.Contains(blocked, "Blocked Users") {
		t.Fatalf("blocked view missing section")
	}

	m.showBlocked = false
	m.showProfile = true
	m.profile = appProfile("42", "u42")
	profile := m.View()
	if !strings.Contains(profile, "Profile @u42") {
		t.Fatalf("profile view missing breadcrumb")
	}

	m.showProfile = false
	m.showDetail = true
	m.rants = []RantItem{{Rant: makeRant("r1", time.Now(), "acct-a"), Status: StatusNormal}}
	detail := m.View()
	if !strings.Contains(detail, "Post r1") {
		t.Fatalf("detail view missing breadcrumb")
	}
}

func TestRenderTabs_DoesNotDuplicateDefaultTag(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 120
	out := m.renderTabs()
	if strings.Count(strings.ToLower(out), domain.AppHashTag) != 1 {
		t.Fatalf("expected single "+domain.AppHashTag+" tab: %q", out)
	}

	m.hashtag = "golang"
	out = m.renderTabs()
	if !strings.Contains(strings.ToLower(out), "#golang") {
		t.Fatalf("expected custom hashtag tab to appear")
	}
}

func TestRenderFeedCard_HiddenPostLabelShownWhenShowHidden(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 120
	m.height = 40
	now := time.Now()
	r := domain.Rant{ID: "x", AccountID: "a1", Author: "A", Username: "u1", Content: "hello", CreatedAt: now}
	m.rants = []RantItem{{Rant: r, Status: StatusNormal}}
	m.hiddenIDs[r.ID] = true
	m.showHidden = true

	card := m.renderFeedCard(0, 80, 60)
	if !strings.Contains(card, "HIDDEN") {
		t.Fatalf("expected hidden marker when showing hidden posts: %q", card)
	}
}

func TestEmptyFeedMessage_ByTab(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")

	m.feedSource = sourceFollowing
	if got := m.emptyFeedMessage(false); !strings.Contains(got, "follow") {
		t.Fatalf("unexpected following empty message: %q", got)
	}

	m.feedSource = sourceTrending
	if got := m.emptyFeedMessage(false); !strings.Contains(strings.ToLower(got), "trending") {
		t.Fatalf("unexpected trending empty message: %q", got)
	}

	m.feedSource = sourceCustomHashtag
	m.hashtag = "go"
	if got := m.emptyFeedMessage(false); !strings.Contains(got, "#go") {
		t.Fatalf("unexpected custom hashtag empty message: %q", got)
	}
}

func TestRenderKeyDialog_AllModes(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{{Rant: makeRant("x", time.Now(), "acct-a"), Status: StatusNormal}}
	if out := m.renderKeyDialog(); !strings.Contains(out, "Keyboard Shortcuts") || !strings.Contains(out, "toggle this dialog") {
		t.Fatalf("unexpected feed key dialog: %q", out)
	}
	m.showDetail = true
	if out := m.renderKeyDialog(); !strings.Contains(out, "open parent post") && !strings.Contains(out, "open selected reply thread") {
		t.Fatalf("unexpected detail key dialog: %q", out)
	}
	m.profileIsOwn = false
	m.profile = appProfile("acct-other", "u42")
	if out := m.renderKeyDialog(); strings.Contains(out, "delete selected post") {
		t.Fatalf("detail key dialog should not include delete for non-own selected post: %q", out)
	}
	m.profileIsOwn = true
	m.profile = appProfile("acct-a", "u42")
	if out := m.renderKeyDialog(); !strings.Contains(out, "delete selected post") {
		t.Fatalf("expected detail key dialog to include delete for own selected post: %q", out)
	}
	m.showDetail = false
	m.showProfile = true
	if out := m.renderKeyDialog(); !strings.Contains(out, "open selected post detail") {
		t.Fatalf("unexpected profile key dialog: %q", out)
	}
}

func TestRenderBlockedUsersDialog_States(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.loadingBlocked = true
	if out := m.renderBlockedUsersDialog(); !strings.Contains(out, "Loading blocked users") {
		t.Fatalf("expected loading state in blocked dialog")
	}
	m.loadingBlocked = false
	m.blockedErr = errString("boom")
	if out := m.renderBlockedUsersDialog(); !strings.Contains(out, "Error: boom") {
		t.Fatalf("expected error state in blocked dialog")
	}
	m.blockedErr = nil
	m.blockedUsers = []app.BlockedUser{{AccountID: "1", Username: "u1", DisplayName: "User One"}}
	m.confirmUnblock = true
	m.unblockTarget = m.blockedUsers[0]
	if out := m.renderBlockedUsersDialog(); !strings.Contains(out, "Unblock @u1? (y/n)") {
		t.Fatalf("expected unblock confirmation in blocked dialog: %q", out)
	}
}

func TestApplyHorizontalPan_SlicesOverflowedLines(t *testing.T) {
	in := "0123456789\nshort"
	out := applyHorizontalPan(in, 3, 5)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "34567" {
		t.Fatalf("unexpected panned first line: %q", lines[0])
	}
	if lines[1] != "rt" {
		t.Fatalf("unexpected second line: %q", lines[1])
	}
}

func TestApplyHorizontalPan_NoOverflowNoChange(t *testing.T) {
	in := "short\ntiny"
	out := applyHorizontalPan(in, 5, 20)
	if out != in {
		t.Fatalf("expected no change when content does not overflow; got %q", out)
	}
}

func appProfile(id, username string) app.Profile {
	return app.Profile{ID: id, Username: username, DisplayName: username}
}

type errString string

func (e errString) Error() string { return string(e) }
