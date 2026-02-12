# TerminalRant

Ranting space for developers that live in the terminal.

TerminalRant is a terminal UI (TUI) client for Mastodon. It lets you read,
post, reply, edit, and moderate your feed without leaving the terminal.

## Features

- OAuth login
- Feed tabs:
  - `#terminalrant`
  - `trending`
  - `personal`
  - custom hashtag tab (only shown when custom tag differs from `terminalrant`)
- Switch tabs with `t` (next) and `T` (previous)
- Hashtag controls:
  - Change custom hashtag with `H`
  - Hashtags rendered as small capsules in feed/detail
  - Feed shows compact tags with `+N more`, detail shows all tags
- Media metadata rendering:
  - Feed shows compact media hint (image/video/audio counts + optional alt snippet)
  - Detail shows full media list with type, size (if available), alt text, and URL
- Post creation and replies:
  - Compose with `$EDITOR` (`p` / `c`) or inline composer (`P` / `C`)
  - Optimistic posting/reply updates
- Post interactions:
  - Like/unlike (`l`)
  - Reply (`c`/`C`)
  - Open URL (`o`)
  - Edit/delete own posts (`e`/`E`/`d`)
- Moderation:
  - Hide post locally (`x`)
  - Toggle hidden posts (`X`)
  - Hidden posts shown in muted style with `HIDDEN` label when revealed
  - Block selected author (`b`) with confirmation
  - Manage blocked users dialog (`B`) and unblock with confirmation
- Navigation:
  - Feed and detail views with keyboard navigation
  - Detail view supports full-page scrolling for long threads
  - Parent-thread jump from reply detail (`u`)
- Key help:
  - Minimal hints shown inline
  - Full key dialog via `?`
  - `q` closes dialogs/detail first before quitting from feed root
- Persistent UI state:
  - Remembers custom hashtag and selected feed tab between runs

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

On first run, TerminalRant opens a browser window for OAuth and stores auth
state under `TERMINALRANT_AUTH_DIR`.

### Key bindings

Global:

- `?` — full keymap dialog
- `ctrl+c` — force quit

Feed:

- `j`/`k` or arrow keys — move
- `enter` — open detail
- `t` / `T` — next/previous tab
- `H` — set custom hashtag
- `r` — refresh
- `p` / `P` — new post (`$EDITOR` / inline)
- `c` / `C` — reply (`$EDITOR` / inline)
- `l` — like/unlike
- `e` — edit via `$EDITOR`
- `E` — edit inline
- `d` — delete own post (confirmation)
- `x` / `X` — hide post / toggle hidden posts
- `b` — block selected post author (confirmation)
- `B` — blocked users dialog
- `o` — open post URL
- `g` — open creator GitHub
- `q` — quit (only when no dialog/detail is open)

Detail:

- `j`/`k` or arrow keys — move/scroll detail page
- `enter` — open selected reply thread
- `u` — open parent post
- `r` — refresh thread
- `l` — like/unlike selected
- `c` / `C` — reply
- `o` — open URL
- `esc` / `q` — back

Dialogs:

- `q` / `esc` — close dialog
- Block confirm: `y`/`n`
- Delete confirm: `y`/`n`
- Blocked users dialog:
  - `j`/`k` — select user
  - `u` — unblock selected (confirmation)

## Notes

- The configured hashtag is auto-appended on post/edit/reply if missing.
- For display, HTML returned from Mastodon is stripped for terminal rendering.
- UI state is stored in `ui_state.json` under `TERMINALRANT_AUTH_DIR`.

## Development

- Dependencies are managed with Go modules (`go.mod`, `go.sum`).
- The UI is implemented with:
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/bubbles`
  - `github.com/charmbracelet/lipgloss`

## License

TBD
