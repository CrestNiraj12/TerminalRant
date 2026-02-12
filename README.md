# TerminalRant

Ranting space for developers that live in the terminal.

TerminalRant is a terminal UI (TUI) client for Mastodon timelines that follow a
single hashtag. It lets you read, post, edit, and delete short “rants” without
leaving your terminal.

## Features

- View latest posts from a hashtag timeline (newest first)
- Post new rants
- Edit and delete your own rants
- Compose using either:
  - Your `$EDITOR` (full-screen editor)
  - An inline TUI textarea

## Requirements

- Go 1.25+
- A Mastodon account
- Browser access for OAuth login

## Install

```sh
go install ./...
```

Or run directly:

```sh
go run .
```

## Configuration

TerminalRant is configured via environment variables:

- `TERMINALRANT_INSTANCE` — Mastodon instance base URL
  - Default: `https://mastodon.social`
- `TERMINALRANT_AUTH_DIR` — Directory used to store OAuth token/client state
  - Default: `~/.config/terminalrant`
- `TERMINALRANT_OAUTH_CALLBACK_PORT` — Local callback port for OAuth login
  - Default: `45145`
- `TERMINALRANT_HASHTAG` — Hashtag to follow (without `#`)
  - Default: `terminalrant`

## Usage

Start the app:

```sh
TERMINALRANT_INSTANCE="https://your.instance" \
TERMINALRANT_HASHTAG="terminalrant" \
go run .
```

On first run, TerminalRant opens a browser window for OAuth login and saves a
session token under `TERMINALRANT_AUTH_DIR`.

### Key bindings

From the timeline:

- `q` / `ctrl+c` — quit
- `r` — refresh
- `j`/`k` or arrow keys — navigate
- `p` — compose via `$EDITOR`
- `P` — compose inline

When a post is yours (marked with `(you)`):

- `enter` — open actions menu
- `e` — edit via `$EDITOR`
- `E` — edit inline
- `d` — delete (with confirmation)

While composing inline:

- `ctrl+d` — post/update
- `esc` — cancel

## Notes

- The tracked hashtag is automatically appended on post/edit if it’s not already
  present.
- For display, HTML returned from Mastodon is stripped for terminal rendering.

## Development

- Dependencies are managed with Go modules (`go.mod`, `go.sum`).
- The UI is implemented with:
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/bubbles`
  - `github.com/charmbracelet/lipgloss`

## License

TBD
