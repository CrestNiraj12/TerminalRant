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

func main() {
	// 1. Load config from environment.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// 2. Build infrastructure.
	tokenProvider := auth.NewFileTokenProvider(cfg.TokenPath)
	httpClient := mastodon.NewClient(cfg.InstanceURL, tokenProvider)

	// 3. Build services (concrete types satisfy app.* interfaces).
	accountSvc := mastodon.NewAccountService(httpClient)
	// Fetch account ID synchronously for simplicity in wiring.
	accountID, _ := accountSvc.CurrentAccountID(context.Background())

	timelineSvc := mastodon.NewTimelineService(httpClient, accountID)
	postSvc := mastodon.NewPostService(httpClient)
	editorSvc := editor.NewEnvEditor()

	// 4. Wire root TUI model.
	rootModel := tui.NewApp(tui.Deps{
		Timeline: timelineSvc,
		Post:     postSvc,
		Account:  accountSvc,
		Editor:   editorSvc,
		Hashtag:  cfg.Hashtag,
	})

	// 5. Run.
	p := tea.NewProgram(rootModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "terminalrant: %v\n", err)
		os.Exit(1)
	}
}
