package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/infra/auth"
	"github.com/CrestNiraj12/terminalrant/infra/config"
	"github.com/CrestNiraj12/terminalrant/infra/editor"
	"github.com/CrestNiraj12/terminalrant/infra/mastodon"
	"github.com/CrestNiraj12/terminalrant/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type cliMode int

const (
	cliRun cliMode = iota
	cliVersion
	cliHelp
	cliInvalid
)

func parseCLIArgs(args []string) (cliMode, string) {
	if len(args) == 0 {
		return cliRun, ""
	}

	switch args[0] {
	case "--version", "-version", "-v":
		return cliVersion, ""
	case "--help", "-h", "help":
		return cliHelp, ""
	default:
		return cliInvalid, fmt.Sprintf("unexpected argument: %s", strings.Join(args, " "))
	}
}

func usage() string {
	return "Usage: terminalrant [--version|-version|-v] [--help|-h]"
}

func main() {
	mode, msg := parseCLIArgs(os.Args[1:])
	switch mode {
	case cliVersion:
		fmt.Printf("TerminalRant %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
		return
	case cliHelp:
		fmt.Println(usage())
		return
	case cliInvalid:
		fmt.Fprintf(os.Stderr, "%s\n%s\n", msg, usage())
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
