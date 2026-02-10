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
- A Mastodon account + access token

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
- `TERMINALRANT_TOKEN` — Path to a file containing your access token
  - Default: `~/.config/terminalrant/token`
- `TERMINALRANT_HASHTAG` — Hashtag to follow (without `#`)
  - Default: `terminalrant`

### Token file

Create the token file and paste your Mastodon access token into it:

```sh
mkdir -p ~/.config/terminalrant
pbpaste > ~/.config/terminalrant/token
# or: cat > ~/.config/terminalrant/token
```

The token is read as a bearer token and trimmed for whitespace.

## Usage

Start the app:

```sh
TERMINALRANT_INSTANCE="https://your.instance" \
TERMINALRANT_HASHTAG="terminalrant" \
go run .
```

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

