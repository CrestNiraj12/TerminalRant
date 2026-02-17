package feed

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestUpdateKey_ManageBlocks_OpensDialogAndRequestsList(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
	if !updated.showBlocked || !updated.loadingBlocked {
		t.Fatalf("expected blocked users dialog to open and load")
	}
	if cmd == nil {
		t.Fatalf("expected request command for blocked users")
	}
	msg := cmd()
	if _, ok := msg.(RequestBlockedUsersMsg); !ok {
		t.Fatalf("expected RequestBlockedUsersMsg, got %T", msg)
	}
}

func TestUpdateKey_ConfirmFollow_ClearsOnUnrelatedKey(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.confirmFollow = true
	m.followAccountID = "acct-1"
	m.followUsername = "alice"
	m.followTarget = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if updated.confirmFollow || updated.followAccountID != "" || updated.followUsername != "" || updated.followTarget {
		t.Fatalf("unrelated key should cancel pending follow confirmation")
	}
}

func TestUpdateKey_HashtagInputEnter_AppliesAndSchedulesRefresh(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.hashtagInput = true
	m.hashtagBuffer = "golang"
	beforeReqSeq := m.feedReqSeq

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.hashtagInput {
		t.Fatalf("expected hashtag input mode to close on enter")
	}
	if updated.hashtag != "golang" {
		t.Fatalf("expected hashtag to be applied, got %q", updated.hashtag)
	}
	if updated.feedSource != sourceCustomHashtag {
		t.Fatalf("expected custom hashtag source, got %v", updated.feedSource)
	}
	if updated.feedReqSeq != beforeReqSeq+1 {
		t.Fatalf("expected feed req seq increment, got %d want %d", updated.feedReqSeq, beforeReqSeq+1)
	}
	if !updated.loading {
		t.Fatalf("expected source-change refresh to set loading=true")
	}
	if cmd == nil {
		t.Fatalf("expected refresh cmd batch")
	}
}

func TestUpdateKey_ProfileOpenURLAndAvatarAndToggleMedia(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.profile = appProfile("42", "u42")
	m.profile.URL = "https://example.social/@u42"
	m.profile.AvatarURL = "https://cdn.example/avatar.webp"
	m.showMediaPreview = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd == nil {
		t.Fatalf("expected open profile URL command")
	}
	m = updated

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'I'}})
	if cmd == nil {
		t.Fatalf("expected open avatar URL command")
	}
	m = updated

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if updated.showMediaPreview {
		t.Fatalf("expected profile media preview hidden after i")
	}
	if cmd != nil {
		t.Fatalf("expected no fetch command when hiding media")
	}
	m = updated

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if !updated.showMediaPreview {
		t.Fatalf("expected profile media preview shown after second i")
	}
	if cmd == nil {
		t.Fatalf("expected avatar fetch command when showing media")
	}
}

func TestRenderProfileView_HidesPreviewPaneWhenMediaOff(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.width = 140
	m.height = 40
	m.showMediaPreview = false
	m.profile = appProfile("42", "u42")
	m.profile.AvatarURL = "https://cdn.example/avatar.webp"
	m.profilePosts = []domain.Rant{{
		ID:        "p1",
		Username:  "u42",
		AccountID: "42",
		Content:   "hello",
		CreatedAt: time.Now(),
	}}

	out := m.renderProfileView()
	if strings.Contains(out, "Profile Image Preview") {
		t.Fatalf("profile preview pane should be hidden when media is off")
	}
}

func TestUpdateKey_ProfileView_LeftRightAdjustsHorizontalPan(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.hScroll = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if updated.hScroll <= 0 {
		t.Fatalf("expected right to increase horizontal scroll in profile view, got %d", updated.hScroll)
	}
	m = updated

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if updated.hScroll != 0 {
		t.Fatalf("expected left to reduce horizontal scroll in profile view, got %d", updated.hScroll)
	}
}

func TestUpdateKey_DeleteFromDetailTargetsSelectedReply(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{{Rant: domain.Rant{ID: "root", IsOwn: true}}}
	m.cursor = 0
	m.showDetail = true
	m.replies = []domain.Rant{{ID: "reply-own", IsOwn: true}}
	m.detailCursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !updated.confirmDelete || updated.deleteTargetID != "reply-own" {
		t.Fatalf("expected delete confirmation for selected reply, confirm=%v target=%q", updated.confirmDelete, updated.deleteTargetID)
	}
	m = updated

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if updated.confirmDelete || updated.deleteTargetID != "" {
		t.Fatalf("expected delete confirmation cleared after yes")
	}
	if cmd == nil {
		t.Fatalf("expected delete command")
	}
	msg := cmd()
	del, ok := msg.(DeleteRantMsg)
	if !ok || del.ID != "reply-own" {
		t.Fatalf("expected DeleteRantMsg for reply-own, got %T %+v", msg, msg)
	}
}

func TestUpdateKey_DeleteAllowedViaProfileOwnershipFallback(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{{Rant: domain.Rant{ID: "root", AccountID: "acct-me", IsOwn: false}}}
	m.cursor = 0
	m.showDetail = true
	m.profileIsOwn = true
	m.profile = appProfile("acct-me", "me")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !updated.confirmDelete || updated.deleteTargetID != "root" {
		t.Fatalf("expected delete confirmation via ownership fallback, confirm=%v target=%q", updated.confirmDelete, updated.deleteTargetID)
	}
}

func TestUpdateKey_DeleteInDetailNotOwnShowsNotice(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{{Rant: domain.Rant{ID: "root", AccountID: "acct-other", IsOwn: false}}}
	m.cursor = 0
	m.showDetail = true
	m.profileIsOwn = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if updated.confirmDelete {
		t.Fatalf("delete confirmation should not open for non-own post")
	}
	if updated.pagingNotice == "" {
		t.Fatalf("expected notice when delete is not allowed")
	}
}

func TestUpdateKey_DeleteConfirmInDetail_OptimisticAndBackToFeed(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showDetail = true
	m.confirmDelete = true
	m.deleteTargetID = "root"
	m.rants = []RantItem{{Rant: domain.Rant{ID: "root", AccountID: "acct-me", IsOwn: true}}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if updated.showDetail {
		t.Fatalf("expected detail view to close after delete confirm")
	}
	if len(updated.rants) != 0 {
		t.Fatalf("expected optimistic removal from feed list, got %d", len(updated.rants))
	}
	if cmd == nil {
		t.Fatalf("expected delete API command")
	}
	msg := cmd()
	del, ok := msg.(DeleteRantMsg)
	if !ok || del.ID != "root" {
		t.Fatalf("expected DeleteRantMsg for root, got %T %+v", msg, msg)
	}
}

func TestUpdateKey_DeleteConfirmInDetail_ReturnsToProfile(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showDetail = true
	m.returnToProfile = true
	m.showProfile = false
	m.confirmDelete = true
	m.deleteTargetID = "p1"
	m.profilePosts = []domain.Rant{{ID: "p1", AccountID: "acct-me", IsOwn: true}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !updated.showProfile || updated.showDetail {
		t.Fatalf("expected return to profile after delete from profile detail")
	}
}

func TestUpdateKey_ProfileDownUsesHeaderScrollGateBeforeMovingCursor(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.height = 20
	m.profile = appProfile("42", "u42")
	m.profile.Bio = strings.Repeat("line ", 400)
	m.profilePosts = []domain.Rant{{ID: "p1", IsOwn: true}}
	m.profileCursor = 0
	m.detailScrollLine = 0

	gate := m.profileScrollGate()
	if gate <= 0 {
		t.Fatalf("expected positive profile scroll gate, got %d", gate)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.profileCursor != 0 {
		t.Fatalf("cursor should remain on profile card until gate consumed, got %d", updated.profileCursor)
	}
	if updated.detailScrollLine != 1 {
		t.Fatalf("expected detailScrollLine increment, got %d", updated.detailScrollLine)
	}
}
