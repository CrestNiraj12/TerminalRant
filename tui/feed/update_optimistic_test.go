package feed

import (
	"testing"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestAddOptimisticRant_OnlyOnTerminalRantSource(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.feedSource = sourceTrending

	updated, _ := m.Update(AddOptimisticRantMsg{Content: "hello"})
	if len(updated.rants) != 0 {
		t.Fatalf("expected no optimistic rant outside terminalrant source")
	}

	updated, _ = m.Update(SwitchToTerminalRantMsg{})
	updated, _ = updated.Update(AddOptimisticRantMsg{Content: "hello"})
	if len(updated.rants) != 1 {
		t.Fatalf("expected optimistic rant on terminalrant source, got %d", len(updated.rants))
	}
}

func TestDeleteOptimisticRant_RemovesItemImmediately(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{
		{Rant: domain.Rant{ID: "a"}, Status: StatusNormal},
		{Rant: domain.Rant{ID: "b"}, Status: StatusNormal},
	}

	updated, _ := m.Update(DeleteOptimisticRantMsg{ID: "a"})
	if len(updated.rants) != 1 || updated.rants[0].Rant.ID != "b" {
		t.Fatalf("expected optimistic delete to remove item immediately, got %#v", updated.rants)
	}
}
