package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"terminalrant/infra/auth"
	"terminalrant/infra/config"
	"terminalrant/infra/editor"
	"terminalrant/infra/mastodon"
	"terminalrant/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("TerminalRant %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
		return
	}

	// 1. Load config from environment.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// 2. Build infrastructure.
	if err := auth.EnsureOAuthLogin(context.Background(), cfg.InstanceURL, cfg.OAuthTokenPath, cfg.OAuthClientPath, cfg.OAuthCallbackPort); err != nil {
		fmt.Fprintf(os.Stderr, "oauth login: %v\n", err)
		os.Exit(1)
	}

	tokenProvider := auth.NewFileTokenProvider(cfg.OAuthTokenPath)
	httpClient := mastodon.NewClient(cfg.InstanceURL, tokenProvider)

	// 3. Build services (concrete types satisfy app.* interfaces).
	accountSvc := mastodon.NewAccountService(httpClient)
	// Fetch account ID synchronously for simplicity in wiring.
	accountID, _ := accountSvc.CurrentAccountID(context.Background())

	timelineSvc := mastodon.NewTimelineService(httpClient, accountID)
	postSvc := mastodon.NewPostService(httpClient)
	editorSvc := editor.NewEnvEditor()

	uiState, _ := config.LoadUIState(cfg.UIStatePath)
	initialHashtag := cfg.Hashtag
	if uiState.Hashtag != "" {
		initialHashtag = uiState.Hashtag
	}
	initialFeedSource := uiState.FeedSource
	if initialFeedSource == "" {
		initialFeedSource = "terminalrant"
	}

	// 4. Wire root TUI model.
	rootModel := tui.NewApp(tui.Deps{
		Timeline:  timelineSvc,
		Post:      postSvc,
		Account:   accountSvc,
		Editor:    editorSvc,
		Hashtag:   initialHashtag,
		FeedView:  initialFeedSource,
		StatePath: cfg.UIStatePath,
	})

	// 5. Run.
	p := tea.NewProgram(rootModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "terminalrant: %v\n", err)
		os.Exit(1)
	}
}
