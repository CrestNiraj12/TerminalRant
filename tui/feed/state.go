package feed

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CrestNiraj12/terminalrant/app"
	"github.com/CrestNiraj12/terminalrant/domain"
	"github.com/CrestNiraj12/terminalrant/tui/common"
)

const (
	defaultLimit    = 20
	replyPageSize   = 20
	prefetchTrigger = 3
	creatorGitHub   = "https://github.com/CrestNiraj12"
)

// RantsLoadedMsg is sent when the timeline fetch completes successfully.
type RantsLoadedMsg struct {
	Rants    []domain.Rant
	QueryKey string
	RawCount int
	ReqSeq   int
}

// RantsErrorMsg is sent when the timeline fetch fails.
type RantsErrorMsg struct {
	Err      error
	QueryKey string
	ReqSeq   int
}

// RantsPageLoadedMsg is sent when an older feed page is loaded.
type RantsPageLoadedMsg struct {
	Rants    []domain.Rant
	QueryKey string
	RawCount int
	ReqSeq   int
}

// RantsPageErrorMsg is sent when loading an older feed page fails.
type RantsPageErrorMsg struct {
	Err      error
	QueryKey string
	ReqSeq   int
}

type BlockUserMsg struct {
	AccountID string
	Username  string
}

type BlockResultMsg struct {
	AccountID string
	Username  string
	Err       error
}

type FollowToggleMsg struct {
	AccountID string
	Username  string
	Follow    bool
}

type FollowToggleResultMsg struct {
	AccountID string
	Username  string
	Follow    bool
	Err       error
}

type HideAuthorPostsMsg struct {
	AccountID string
}

type RequestBlockedUsersMsg struct{}

type BlockedUsersLoadedMsg struct {
	Users []app.BlockedUser
	Err   error
}

type UnblockUserMsg struct {
	AccountID string
	Username  string
}

type UnblockResultMsg struct {
	AccountID string
	Username  string
	Err       error
}

type RelationshipsLoadedMsg struct {
	Following map[string]bool
	Err       error
}

type OpenProfileMsg struct {
	AccountID string
}

type ProfileLoadedMsg struct {
	AccountID string
	Profile   app.Profile
	Posts     []domain.Rant
	Err       error
}

type EditProfileMsg struct {
	UseInline bool
}

// EditRantMsg is sent when the user selects 'Edit' from the action menu.
type EditRantMsg struct {
	Rant      domain.Rant
	UseInline bool
}

// DeleteResultMsg is sent after a rant deletion attempt.
type DeleteResultMsg struct {
	ID  string
	Err error
}

// LikeRantMsg is sent when the user wants to like a rant.
type LikeRantMsg struct {
	ID       string
	WasLiked bool
}

// LikeResultMsg is sent after a like attempt.
type LikeResultMsg struct {
	ID  string
	Err error
}

// ReplyRantMsg is sent when the user wants to reply to a rant.
type ReplyRantMsg struct {
	Rant      domain.Rant
	UseInline bool
}

// ThreadLoadedMsg is sent when a thread (ancestors and replies) is loaded.
type ThreadLoadedMsg struct {
	ID          string
	Ancestors   []domain.Rant
	Descendants []domain.Rant
}

type MediaPreviewLoadedMsg struct {
	Key     string
	Preview string
	Frames  []string
	Err     error
}

// ThreadErrorMsg is sent when a thread fetch fails.
type ThreadErrorMsg struct {
	ID  string
	Err error
}

type ResetFeedStateMsg struct {
	ForceReset bool
}

type OpenDetailWithoutRepliesMsg struct {
	ID string
}

type threadData struct {
	Ancestors   []domain.Rant
	Descendants []domain.Rant
}

type feedSource int

const (
	sourceTerminalRant feedSource = iota
	sourceTrending
	sourceFollowing
	sourceCustomHashtag
)

type FeedPrefsChangedMsg struct {
	Hashtag string
	Source  string
}

type PrefsSavedMsg struct {
	Err error
}

// ResultMsg is a generic success/fail result for creation or update.
type ResultMsg struct {
	ID         string // Local or Server ID
	Rant       domain.Rant
	IsEdit     bool
	Err        error
	OldContent string
}

// --- Optimistic Update Messages ---

type AddOptimisticRantMsg struct {
	Content string
}

type UpdateOptimisticRantMsg struct {
	ID      string
	Content string
}

type DeleteOptimisticRantMsg struct {
	ID string
}

// --- Status Types ---

type RantStatus int

const (
	StatusNormal RantStatus = iota
	StatusPendingCreate
	StatusPendingUpdate
	StatusPendingDelete
	StatusFailed
)

type RantItem struct {
	Rant       domain.Rant
	Status     RantStatus
	Err        error
	OldContent string // For rollback
}

// --- Model ---

type modelServices struct {
	timeline app.TimelineService
	account  app.AccountService
}

type feedState struct {
	defaultHashtag string
	hashtag        string
	feedSource     feedSource
	rants          []RantItem
	cursor         int
	loading        bool
	loadingMore    bool
	hasMoreFeed    bool
	oldestFeedID   string
	err            error
	pagingNotice   string
	feedReqSeq     int
}

type uiState struct {
	keys         common.KeyMap
	spinner      spinner.Model
	width        int // Terminal width
	height       int // Terminal height
	startIndex   int // First visible item in the list (for scrolling)
	scrollLine   int // Line-based scroll for feed viewport
	hScroll      int // Horizontal pan offset (columns)
	showAllHints bool
}

type detailState struct {
	confirmDelete    bool // Whether we are in the 'Are you sure?' delete step
	showDetail       bool // Whether we are in full-post view
	ancestors        []domain.Rant
	replies          []domain.Rant
	replyAll         []domain.Rant
	replyVisible     int
	hasMoreReplies   bool
	loadingReplies   bool
	detailCursor     int // 0 for main post, 1...n for replies
	detailStart      int
	detailScrollLine int
	focusedRant      *domain.Rant
	threadCache      map[string]threadData
	viewStack        []*domain.Rant // To support going back in deep threading
}

type moderationState struct {
	hiddenIDs      map[string]bool
	hiddenAuthors  map[string]bool
	showHidden     bool
	confirmBlock   bool
	blockAccountID string
	blockUsername  string
	showBlocked    bool
	loadingBlocked bool
	blockedErr     error
	blockedUsers   []app.BlockedUser
	blockedCursor  int
	confirmUnblock bool
	unblockTarget  app.BlockedUser
}

type relationshipState struct {
	confirmFollow   bool
	followAccountID string
	followUsername  string
	followTarget    bool
	followingByID   map[string]bool
	recentFollows   []string
	followingDirty  bool
}

type hashtagState struct {
	hashtagInput  bool
	hashtagBuffer string
}

type profileState struct {
	showProfile     bool
	returnToProfile bool
	profileIsOwn    bool
	profileLoading  bool
	profileErr      error
	profile         app.Profile
	profilePosts    []domain.Rant
	profileCursor   int // 0 for profile card, 1...n for profile posts
	profileStart    int // first visible profile post
}

type mediaState struct {
	showMediaPreview bool
	mediaPreview     map[string]string
	mediaFrames      map[string][]string
	mediaFrameIndex  map[string]int
	mediaLoading     map[string]bool
}

// Model holds the state for the feed (timeline) view.
type Model struct {
	modelServices
	feedState
	uiState
	detailState
	moderationState
	relationshipState
	hashtagState
	profileState
	mediaState
}

// New creates a feed model with injected dependencies.
func New(timeline app.TimelineService, account app.AccountService, hashtag, initialSource string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
	tag := strings.TrimSpace(strings.TrimPrefix(hashtag, "#"))
	if tag == "" {
		tag = "terminalrant"
	}
	source := parseFeedSource(initialSource)
	if source == sourceCustomHashtag && strings.EqualFold(tag, "terminalrant") {
		source = sourceTerminalRant
	}

	return Model{
		modelServices: modelServices{
			timeline: timeline,
			account:  account,
		},
		feedState: feedState{
			defaultHashtag: "terminalrant",
			hashtag:        tag,
			feedSource:     source,
			loading:        true,
			hasMoreFeed:    true,
		},
		uiState: uiState{
			keys:    common.DefaultKeyMap(),
			spinner: s,
		},
		detailState: detailState{
			threadCache: make(map[string]threadData),
		},
		moderationState: moderationState{
			hiddenIDs:     make(map[string]bool),
			hiddenAuthors: make(map[string]bool),
		},
		relationshipState: relationshipState{
			followingByID: make(map[string]bool),
		},
		mediaState: mediaState{
			showMediaPreview: true,
			mediaPreview:     make(map[string]string),
			mediaFrames:      make(map[string][]string),
			mediaFrameIndex:  make(map[string]int),
			mediaLoading:     make(map[string]bool),
		},
	}
}

// Init starts the initial feed fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRants(m.feedReqSeq),
		m.spinner.Tick,
	)
}

// Refresh returns a Cmd that re-fetches the timeline.
func (m Model) Refresh() tea.Cmd {
	return m.fetchRants(m.feedReqSeq)
}

// Update handles messages for the feed view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m.update(msg)
}
