package feed

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/app"
	"terminalrant/domain"
)

type stubTimeline struct{}

func (stubTimeline) FetchByHashtag(context.Context, string, int) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchByHashtagPage(context.Context, string, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchHomePage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchPublicPage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchTrendingPage(context.Context, int, string) ([]domain.Rant, error) {
	return nil, nil
}
func (stubTimeline) FetchThread(context.Context, string) ([]domain.Rant, []domain.Rant, error) {
	return nil, nil, nil
}

type stubAccount struct{}

func (stubAccount) CurrentAccountID(context.Context) (string, error)                 { return "", nil }
func (stubAccount) CurrentProfile(context.Context) (app.Profile, error)              { return app.Profile{}, nil }
func (stubAccount) UpdateProfile(context.Context, string, string) error              { return nil }
func (stubAccount) BlockUser(context.Context, string) error                          { return nil }
func (stubAccount) ListBlockedUsers(context.Context, int) ([]app.BlockedUser, error) { return nil, nil }
func (stubAccount) UnblockUser(context.Context, string) error                        { return nil }
func (stubAccount) FollowUser(context.Context, string) error                         { return nil }
func (stubAccount) UnfollowUser(context.Context, string) error                       { return nil }
func (stubAccount) LookupFollowing(context.Context, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (stubAccount) ProfileByID(context.Context, string) (app.Profile, error) {
	return app.Profile{}, nil
}
func (stubAccount) PostsByAccount(context.Context, string, int, string) ([]domain.Rant, error) {
	return nil, nil
}

func makeRant(id string, createdAt time.Time, accountID string) domain.Rant {
	return domain.Rant{
		ID:        id,
		AccountID: accountID,
		Author:    "Author " + id,
		Username:  "user" + id,
		Content:   "hello " + domain.AppHashTag,
		CreatedAt: createdAt,
	}
}

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

func TestApplyLikeToggleAndThreadCacheLikeToggle(t *testing.T) {
	now := time.Now()
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.rants = []RantItem{{Rant: makeRant("id-1", now, "acct-a"), Status: StatusNormal}}
	m.replies = []domain.Rant{{ID: "id-2", LikesCount: 1, Liked: true}}
	m.replyAll = []domain.Rant{{ID: "id-2", LikesCount: 1, Liked: true}}
	m.ancestors = []domain.Rant{{ID: "id-3", LikesCount: 0, Liked: false}}
	focused := domain.Rant{ID: "id-4", LikesCount: 0, Liked: false}
	m.focusedRant = &focused
	m.threadCache["root"] = threadData{
		Ancestors:   []domain.Rant{{ID: "id-3", LikesCount: 0, Liked: false}},
		Descendants: []domain.Rant{{ID: "id-2", LikesCount: 1, Liked: true}},
	}

	m.applyLikeToggle("id-1")
	m.applyLikeToggle("id-2")
	m.applyLikeToggle("id-3")
	m.applyLikeToggle("id-4")
	if !m.rants[0].Rant.Liked || m.rants[0].Rant.LikesCount != 1 {
		t.Fatalf("feed like toggle failed: %#v", m.rants[0].Rant)
	}
	if m.replies[0].Liked || m.replies[0].LikesCount != 0 {
		t.Fatalf("reply like toggle failed: %#v", m.replies[0])
	}
	if !m.ancestors[0].Liked || m.ancestors[0].LikesCount != 1 {
		t.Fatalf("ancestor like toggle failed: %#v", m.ancestors[0])
	}
	if !m.focusedRant.Liked || m.focusedRant.LikesCount != 1 {
		t.Fatalf("focused like toggle failed: %#v", *m.focusedRant)
	}

	m.toggleLikeInThreadCache("id-2")
	m.toggleLikeInThreadCache("id-3")
	cached := m.threadCache["root"]
	if cached.Descendants[0].Liked || cached.Descendants[0].LikesCount != 0 {
		t.Fatalf("thread cache descendant toggle failed: %#v", cached.Descendants[0])
	}
	if !cached.Ancestors[0].Liked || cached.Ancestors[0].LikesCount != 1 {
		t.Fatalf("thread cache ancestor toggle failed: %#v", cached.Ancestors[0])
	}
}

func TestDetailAndProfileCursorVisibilityBounds(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.height = 44
	m.showDetail = true
	for i := range 20 {
		m.replies = append(m.replies, makeRant(fmt.Sprintf("r-%d", i), time.Now(), "acct-a"))
	}
	m.detailCursor = 18
	m.ensureDetailCursorVisible()
	if m.detailStart < 0 || m.detailStart >= len(m.replies) {
		t.Fatalf("detailStart out of range: %d", m.detailStart)
	}
	if m.detailReplySlots() < 4 {
		t.Fatalf("detail slots must keep lower bound")
	}

	m.showProfile = true
	m.profileCursor = 16
	for i := range 20 {
		m.profilePosts = append(m.profilePosts, makeRant(fmt.Sprintf("p-%d", i), time.Now(), "acct-a"))
	}
	m.ensureProfileCursorVisible()
	if m.profileStart < 0 || m.profileStart >= len(m.profilePosts) {
		t.Fatalf("profileStart out of range: %d", m.profileStart)
	}
	if m.profilePostSlots() < 4 {
		t.Fatalf("profile slots must keep lower bound")
	}
}

func TestThreadMembershipAndReconcileReplyResult(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	root := makeRant("root", time.Now(), "acct-a")
	m.focusedRant = &root
	m.rants = []RantItem{{Rant: root, Status: StatusNormal}}
	m.ancestors = []domain.Rant{{ID: "a1"}}
	m.replies = []domain.Rant{{ID: "d1"}}
	m.threadCache["root"] = threadData{
		Ancestors:   []domain.Rant{{ID: "a2"}},
		Descendants: []domain.Rant{{ID: "d2"}},
	}
	if !m.belongsToCurrentThread("root") || !m.belongsToCurrentThread("a1") || !m.belongsToCurrentThread("d1") || !m.belongsToCurrentThread("a2") || !m.belongsToCurrentThread("d2") {
		t.Fatalf("expected ids to belong to current thread")
	}
	if m.belongsToCurrentThread("x") {
		t.Fatalf("unexpected thread membership")
	}

	m.replyAll = []domain.Rant{{ID: "local-reply-1", InReplyToID: "root", Content: "hello"}}
	server := domain.Rant{ID: "srv-1", InReplyToID: "root", Content: "hello"}
	m.reconcileReplyResult("local-reply-1", server)
	if len(m.replies) != 1 || m.replies[0].ID != "srv-1" {
		t.Fatalf("expected local reply to be replaced by server reply: %#v", m.replies)
	}
	if len(m.threadCache["root"].Descendants) == 0 {
		t.Fatalf("thread cache descendants must be updated")
	}
}

func TestFeedSourceAndPrefsHelpers(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.hashtag = "GoLang"
	m.defaultHashtag = "terminalrant"

	m.feedSource = sourceTerminalRant
	if m.currentFeedQueryKey() != "tag:terminalrant" || m.sourceLabel() != domain.AppHashTag || m.sourcePersistValue() != "terminalrant" {
		t.Fatalf("unexpected terminalrant source helpers")
	}
	m.feedSource = sourceTrending
	if m.currentFeedQueryKey() != "trending" || m.sourcePersistValue() != "trending" {
		t.Fatalf("unexpected trending helpers")
	}
	m.feedSource = sourceFollowing
	if m.currentFeedQueryKey() != "following" || m.sourcePersistValue() != "following" {
		t.Fatalf("unexpected following helpers")
	}
	m.feedSource = sourceCustomHashtag
	if m.currentFeedQueryKey() != "tag:golang" || m.sourceLabel() != "#GoLang" || m.sourcePersistValue() != "custom" {
		t.Fatalf("unexpected custom helpers")
	}
	if !m.hasCustomTab() {
		t.Fatalf("expected custom tab when hashtag differs from default")
	}
	msg := m.emitPrefsChanged()().(FeedPrefsChangedMsg)
	if msg.Hashtag != "GoLang" || msg.Source != "custom" {
		t.Fatalf("unexpected prefs msg: %#v", msg)
	}
	if parseFeedSource("personal") != sourceFollowing || parseFeedSource("custom") != sourceCustomHashtag || parseFeedSource("bad") != sourceTerminalRant {
		t.Fatalf("unexpected parse feed source mapping")
	}
}

func TestVisibilityCursorAndSelectionHelpers(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "following")
	m.feedSource = sourceFollowing
	m.hiddenIDs["id-hidden"] = true
	m.hiddenAuthors["acct-hidden"] = true
	m.rants = []RantItem{
		{Rant: makeRant("id-own", time.Now(), "acct-me"), Status: StatusNormal},
		{Rant: makeRant("id-hidden", time.Now(), "acct-a"), Status: StatusNormal},
		{Rant: makeRant("id-author-hidden", time.Now(), "acct-hidden"), Status: StatusNormal},
		{Rant: makeRant("id-followed", time.Now(), "acct-b"), Status: StatusNormal},
	}
	m.rants[0].Rant.IsOwn = true
	m.followingByID["acct-b"] = true
	m.followingByID["acct-a"] = true
	m.followingByID["acct-hidden"] = true

	vis := m.visibleIndices()
	if len(vis) != 1 || m.rants[vis[0]].Rant.ID != "id-followed" {
		t.Fatalf("unexpected visible indices: %#v", vis)
	}
	m.cursor = 0
	m.ensureVisibleCursor()
	if m.rants[m.cursor].Rant.ID != "id-followed" {
		t.Fatalf("cursor should jump to visible rant, got %s", m.rants[m.cursor].Rant.ID)
	}
	m.moveCursorVisible(-1)
	if m.cursor != 0 {
		t.Fatalf("cursor should stop at boundary when no visible item exists above, got %d", m.cursor)
	}
	if _, ok := m.selectedVisibleRant(); ok {
		t.Fatalf("selectedVisibleRant should be false when cursor points to hidden/non-followed item")
	}
	m.cursor = 3
	sel, ok := m.selectedVisibleRant()
	if !ok || sel.ID != "id-followed" {
		t.Fatalf("unexpected selected visible rant after reset: ok=%v rant=%#v", ok, sel)
	}
	if !m.isFollowing("acct-b") || m.isFollowing("missing") {
		t.Fatalf("isFollowing helper mismatch")
	}
}

func TestFindSetCursorAndRecentFollowHelpers(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showHidden = true
	m.rants = []RantItem{{Rant: makeRant("a", time.Now(), "acct-a"), Status: StatusNormal}, {Rant: makeRant("b", time.Now(), "acct-b"), Status: StatusNormal}}
	m.replies = []domain.Rant{{ID: "r1"}}
	m.ancestors = []domain.Rant{{ID: "a1"}}
	fr := domain.Rant{ID: "f1"}
	m.focusedRant = &fr

	for _, id := range []string{"a", "r1", "a1", "f1"} {
		if _, ok := m.findRantByID(id); !ok {
			t.Fatalf("findRantByID should locate %s", id)
		}
	}
	m.setCursorByID("b")
	if m.cursor != 1 {
		t.Fatalf("setCursorByID failed: %d", m.cursor)
	}

	m.addRecentFollow("x")
	m.addRecentFollow("y")
	m.addRecentFollow("x")
	if len(m.recentFollows) != 2 || m.recentFollows[0] != "x" {
		t.Fatalf("unexpected recent follows ordering: %#v", m.recentFollows)
	}
	m.removeRecentFollow("x")
	if len(m.recentFollows) != 1 || m.recentFollows[0] != "y" {
		t.Fatalf("unexpected recent follows after remove: %#v", m.recentFollows)
	}
}

func TestPrepareSourceChangeAndDialogFlags(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.loadingMore = true
	m.cursor = 9
	m.startIndex = 7
	m.scrollLine = 12
	m.rants = []RantItem{{Rant: makeRant("x", time.Now(), "acct-a"), Status: StatusNormal}}
	m.oldestFeedID = "x"
	m.hasMoreFeed = false
	m.loading = false

	m.prepareSourceChange()
	if !m.loading || m.loadingMore || m.cursor != 0 || m.startIndex != 0 || m.scrollLine != 0 || len(m.rants) != 0 || m.oldestFeedID != "" || !m.hasMoreFeed {
		t.Fatalf("prepareSourceChange did not reset feed state: %#v", m)
	}

	m.showAllHints = true
	if !m.IsDialogOpen() {
		t.Fatalf("dialog should report open when hints visible")
	}
	m.showAllHints = false
	m.rants = []RantItem{{Rant: makeRant("id-1", time.Now(), "acct-a"), Status: StatusNormal}}
	if _, ok := m.SelectedRant(); !ok {
		t.Fatalf("selected rant should be available")
	}
}

func TestDetailReplyGateAndWrappedLines(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.height = 20
	main := makeRant("x", time.Now(), "acct-a")
	main.Content = strings.Repeat("abcdefghij", 15)
	main.Media = []domain.MediaAttachment{{ID: "m"}}
	m.rants = []RantItem{{Rant: main, Status: StatusNormal}}
	m.ancestors = []domain.Rant{{ID: "p"}}
	gate := m.detailReplyGate()
	if gate < 0 {
		t.Fatalf("detail reply gate must be non-negative")
	}
	if estimateWrappedLines("abc\ndef", 2) < 3 {
		t.Fatalf("wrapped line estimation should account for wrapping + newlines")
	}
}

func TestOrganizeThreadReplies_NestsAndAppendsOrphans(t *testing.T) {
	desc := []domain.Rant{
		{ID: "c1", InReplyToID: "root"},
		{ID: "c2", InReplyToID: "c1"},
		{ID: "orphan", InReplyToID: "x"},
	}
	out := organizeThreadReplies("root", desc)
	if len(out) != 3 {
		t.Fatalf("expected all descendants returned, got %d", len(out))
	}
	if out[0].ID != "c1" || out[1].ID != "c2" {
		t.Fatalf("expected threaded order first, got %#v", out)
	}
}
