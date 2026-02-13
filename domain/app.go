package domain

const (
	appEmoji = "🔥"
	AppTitle = "TerminalRant"
	AppHashTag = "#terminalrant"
)

func DisplayAppTitle() string {
	return appEmoji + " " + AppTitle
}
