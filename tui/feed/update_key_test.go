package feed

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
