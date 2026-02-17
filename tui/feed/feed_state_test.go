package feed

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestPaginationAppendStability_PreservesSelectionAndTopAnchor(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 140
	m.height = 40
	m.loading = false
	m.showMediaPreview = false

	now := time.Now().Add(-1 * time.Hour)
	for i := range 30 {
		id := fmt.Sprintf("id-%02d", i)
		r := makeRant(id, now.Add(time.Duration(30-i)*time.Minute), "acct-a")
		m.rants = append(m.rants, RantItem{Rant: r, Status: StatusNormal})
	}
	m.cursor = 25
	m.scrollLine = 45
	m.hasMoreFeed = true
	m.oldestFeedID = "id-29"

	beforeSelected := m.rants[m.cursor].Rant.ID
	beforeTopID, beforeOffset, ok := m.captureFeedTopAnchor()
	if !ok {
		t.Fatalf("expected top anchor before pagination")
	}

	older := make([]domain.Rant, 0, 5)
	for i := 30; i < 35; i++ {
		id := fmt.Sprintf("id-%02d", i)
		older = append(older, makeRant(id, now.Add(time.Duration(30-i)*time.Minute), "acct-a"))
	}

	updated, _ := m.Update(RantsPageLoadedMsg{Rants: older, QueryKey: m.currentFeedQueryKey(), RawCount: len(older), ReqSeq: m.feedReqSeq})
	afterSelected := updated.rants[updated.cursor].Rant.ID
	if afterSelected != beforeSelected {
		t.Fatalf("selected rant changed after append: got %q want %q", afterSelected, beforeSelected)
	}

	afterTopID, afterOffset, ok := updated.captureFeedTopAnchor()
	if !ok {
		t.Fatalf("expected top anchor after pagination")
	}
	if afterTopID != beforeTopID || afterOffset != beforeOffset {
		t.Fatalf("top anchor changed after append: got (%s,%d) want (%s,%d)", afterTopID, afterOffset, beforeTopID, beforeOffset)
	}
}

func TestFeedViewportHeight_NoDriftAcrossLoadingState(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 120
	m.height = 40
	m.rants = []RantItem{{Rant: makeRant("id-1", time.Now(), "acct-a"), Status: StatusNormal}}

	base := m.feedViewportHeight()
	m.loadingMore = true
	a := m.feedViewportHeight()
	m.loadingMore = false
	m.loading = true
	b := m.feedViewportHeight()
	m.loading = false
	m.pagingNotice = "hello"
	c := m.feedViewportHeight()

	if a != base || b != base || c != base {
		t.Fatalf("viewport height drifted: base=%d loadingMore=%d loading=%d notice=%d", base, a, b, c)
	}
}

func TestFollowingVisibilityFilter_HidesUnfollowedImmediately(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "following")
	m.feedSource = sourceFollowing
	m.rants = []RantItem{
		{Rant: makeRant("a", time.Now(), "acct-a"), Status: StatusNormal},
		{Rant: makeRant("b", time.Now(), "acct-b"), Status: StatusNormal},
	}
	m.followingByID["acct-a"] = true
	m.followingByID["acct-b"] = false

	vis := m.visibleIndices()
	if len(vis) != 1 || m.rants[vis[0]].Rant.AccountID != "acct-a" {
		t.Fatalf("unexpected visible set: %#v", vis)
	}
}

func TestProfileEnter_OpensExactSelectedProfilePost(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.profileIsOwn = true
	m.profilePosts = []domain.Rant{makeRant("profile-id", time.Now(), "acct-x")}
	m.profileCursor = 1
	m.rants = []RantItem{{Rant: makeRant("feed-id", time.Now(), "acct-y"), Status: StatusNormal}}
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.showDetail {
		t.Fatalf("expected detail view to open")
	}
	if updated.focusedRant == nil || updated.focusedRant.ID != "profile-id" {
		t.Fatalf("expected focused rant profile-id, got %+v", updated.focusedRant)
	}
	if !updated.returnToProfile {
		t.Fatalf("expected returnToProfile=true when opening detail from profile")
	}
}

func TestDetailBackFromProfile_ReturnsToProfile(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showProfile = true
	m.profileIsOwn = true
	m.profile = appProfile("42", "u42")
	m.profilePosts = []domain.Rant{makeRant("profile-id", time.Now(), "acct-x")}
	m.profileCursor = 1
	m.rants = []RantItem{{Rant: makeRant("feed-id", time.Now(), "acct-y"), Status: StatusNormal}}
	m.cursor = 0

	opened, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !opened.showDetail || !opened.returnToProfile {
		t.Fatalf("expected detail opened from profile")
	}

	closed, _ := opened.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !closed.showProfile || closed.showDetail {
		t.Fatalf("expected q from detail to return to profile; showProfile=%v showDetail=%v", closed.showProfile, closed.showDetail)
	}
}

func TestDownNearPrefetchTrigger_StartsLoadingOlderPosts(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 120
	m.height = 40
	m.loading = false
	m.loadingMore = false
	m.hasMoreFeed = true
	m.oldestFeedID = "oldest"
	for i := range 8 {
		m.rants = append(m.rants, RantItem{Rant: makeRant(fmt.Sprintf("id-%d", i), time.Now(), "acct-a"), Status: StatusNormal})
	}
	m.cursor = len(m.rants) - prefetchTrigger

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd == nil {
		t.Fatalf("expected prefetch command to be returned")
	}
	if !updated.loadingMore {
		t.Fatalf("expected loadingMore=true after prefetch trigger")
	}
}

func TestEnsureFeedCursorVisible_OldSchoolItemStep(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 120
	m.height = 56
	m.showMediaPreview = false
	m.loading = false
	now := time.Now()
	for i := range 10 {
		m.rants = append(m.rants, RantItem{
			Rant:   makeRant(fmt.Sprintf("id-%d", i), now.Add(-time.Duration(i)*time.Minute), "acct-a"),
			Status: StatusNormal,
		})
	}

	m.cursor = 0
	m.scrollLine = 0
	m.ensureFeedCursorVisible()
	if m.scrollLine != 0 {
		t.Fatalf("expected initial scroll to remain at top, got %d", m.scrollLine)
	}

	visible := m.visibleIndices()
	spans := m.feedVisibleSpans(visible)
	if len(spans) < 3 {
		t.Fatalf("expected enough items for stepping test, got %d", len(spans))
	}

	lastVisiblePos := m.feedVisibleSlotsFrom(spans, 0, m.feedViewportHeight()) - 1
	if lastVisiblePos <= 0 || lastVisiblePos >= len(spans)-1 {
		t.Fatalf("need a partially visible window to test stepping; lastVisiblePos=%d len=%d", lastVisiblePos, len(spans))
	}

	// Move to last item currently visible: should not scroll.
	m.cursor = spans[lastVisiblePos].idx
	m.ensureFeedCursorVisible()
	scrollBefore := m.scrollLine
	if scrollBefore != 0 {
		t.Fatalf("expected no scroll while selection stays inside window, got %d", scrollBefore)
	}

	// Move one item past the current window: should scroll by exactly one item.
	m.cursor = spans[lastVisiblePos+1].idx
	m.ensureFeedCursorVisible()
	want := spans[1].top
	if m.scrollLine != want {
		t.Fatalf("expected one-item step scroll=%d, got %d", want, m.scrollLine)
	}
}

func TestFollowToggleResultInFollowing_RefreshesFeedState(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "following")
	m.feedSource = sourceFollowing
	m.rants = []RantItem{{Rant: makeRant("id-1", time.Now(), "acct-a"), Status: StatusNormal}}
	m.cursor = 0

	updated, cmd := m.Update(FollowToggleResultMsg{AccountID: "acct-a", Username: "user", Follow: true})
	if cmd == nil {
		t.Fatalf("expected refresh command on following feed")
	}
	if !updated.loading {
		t.Fatalf("expected loading=true after follow toggle in following feed")
	}
	if len(updated.rants) != 0 {
		t.Fatalf("expected feed reset before refetch")
	}
}

func TestNoLoadMoreWhileInitialFeedLoading(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.width = 100
	m.height = 40
	m.loading = true
	m.loadingMore = false
	m.hasMoreFeed = true
	m.oldestFeedID = "old"
	m.rants = []RantItem{
		{Rant: makeRant("id-1", time.Now(), "acct-a"), Status: StatusNormal},
		{Rant: makeRant("id-2", time.Now(), "acct-a"), Status: StatusNormal},
	}
	m.cursor = len(m.rants) - 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatalf("must not schedule load-more while initial load is running")
	}
	if updated.loadingMore {
		t.Fatalf("loadingMore must remain false during initial load")
	}
}

func TestTrendingDoesNotAutoPaginateAtEnd(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "trending")
	m.width = 120
	m.height = 40
	m.feedSource = sourceTrending
	m.loading = false
	m.loadingMore = false
	m.hasMoreFeed = false
	m.oldestFeedID = "id-2"
	m.rants = []RantItem{
		{Rant: makeRant("id-1", time.Now(), "acct-a"), Status: StatusNormal},
		{Rant: makeRant("id-2", time.Now(), "acct-b"), Status: StatusNormal},
	}
	m.cursor = len(m.rants) - 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatalf("expected no pagination command in trending mode")
	}
	if updated.loadingMore {
		t.Fatalf("expected loadingMore=false in trending mode")
	}
	if !strings.Contains(strings.ToLower(updated.pagingNotice), "end of trending") {
		t.Fatalf("expected end-of-trending fun message, got %q", updated.pagingNotice)
	}
}

func TestHorizontalScrollKeys_AdjustOffset(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.hScroll = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if updated.hScroll != 0 {
		t.Fatalf("left should clamp at zero, got %d", updated.hScroll)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	if updated.hScroll <= 0 {
		t.Fatalf("right should increase horizontal scroll, got %d", updated.hScroll)
	}
}

func TestTrendingLoaded_DisablesOlderPagination(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "trending")
	m.feedSource = sourceTrending
	m.loading = true
	m.hasMoreFeed = true
	m.oldestFeedID = "seed"

	rants := []domain.Rant{
		makeRant("id-1", time.Now(), "acct-a"),
		makeRant("id-2", time.Now().Add(-time.Minute), "acct-b"),
	}
	updated, _ := m.Update(RantsLoadedMsg{
		Rants:    rants,
		QueryKey: m.currentFeedQueryKey(),
		RawCount: len(rants),
		ReqSeq:   m.feedReqSeq,
	})

	if updated.hasMoreFeed {
		t.Fatalf("trending should be snapshot-only; hasMoreFeed must be false")
	}
	if updated.oldestFeedID != "" {
		t.Fatalf("trending should not keep oldestFeedID, got %q", updated.oldestFeedID)
	}
}

func TestNextFeedSource_OrderIncludesCustomLast(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.defaultHashtag = "terminalrant"
	m.hashtag = "golang"

	m.feedSource = sourceTerminalRant
	if got := m.nextFeedSource(1); got != sourceTrending {
		t.Fatalf("next from terminalrant: got %v", got)
	}
	m.feedSource = sourceTrending
	if got := m.nextFeedSource(1); got != sourceFollowing {
		t.Fatalf("next from trending: got %v", got)
	}
	m.feedSource = sourceFollowing
	if got := m.nextFeedSource(1); got != sourceCustomHashtag {
		t.Fatalf("next from following: got %v", got)
	}
	m.feedSource = sourceCustomHashtag
	if got := m.nextFeedSource(1); got != sourceTerminalRant {
		t.Fatalf("next from custom: got %v", got)
	}
}
