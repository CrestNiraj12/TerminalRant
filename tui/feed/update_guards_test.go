package feed

import (
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestUpdate_StaleRantsLoaded_IgnoredByReqSeq(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.loading = true
	m.rants = []RantItem{{Rant: makeRant("existing", time.Now(), "acct-a"), Status: StatusNormal}}
	m.feedReqSeq = 5

	updated, cmd := m.Update(RantsLoadedMsg{
		Rants:    []domain.Rant{makeRant("new", time.Now(), "acct-b")},
		QueryKey: m.currentFeedQueryKey(),
		RawCount: 1,
		ReqSeq:   4,
	})
	if cmd != nil {
		t.Fatalf("expected nil cmd for stale response")
	}
	if len(updated.rants) != 1 || updated.rants[0].Rant.ID != "existing" {
		t.Fatalf("stale response should not mutate feed")
	}
	if !updated.loading {
		t.Fatalf("stale response should not clear loading state")
	}
}

func TestUpdate_StaleRantsLoaded_IgnoredByQueryKey(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.loading = true
	m.feedReqSeq = 2
	m.rants = []RantItem{{Rant: makeRant("existing", time.Now(), "acct-a"), Status: StatusNormal}}

	updated, _ := m.Update(RantsLoadedMsg{
		Rants:    []domain.Rant{makeRant("new", time.Now(), "acct-b")},
		QueryKey: "tag:some-other-hashtag",
		RawCount: 1,
		ReqSeq:   2,
	})
	if len(updated.rants) != 1 || updated.rants[0].Rant.ID != "existing" {
		t.Fatalf("stale query response should not mutate feed")
	}
	if !updated.loading {
		t.Fatalf("stale query response should not clear loading state")
	}
}

func TestUpdate_StaleRantsPageLoaded_Ignored(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.loadingMore = true
	m.feedReqSeq = 9
	m.rants = []RantItem{{Rant: makeRant("existing", time.Now(), "acct-a"), Status: StatusNormal}}

	updated, _ := m.Update(RantsPageLoadedMsg{
		Rants:    []domain.Rant{makeRant("older", time.Now().Add(-time.Minute), "acct-b")},
		QueryKey: m.currentFeedQueryKey(),
		RawCount: 1,
		ReqSeq:   8,
	})
	if len(updated.rants) != 1 || updated.rants[0].Rant.ID != "existing" {
		t.Fatalf("stale page response should not append")
	}
	if !updated.loadingMore {
		t.Fatalf("stale page response should not change loadingMore")
	}
}
