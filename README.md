# slack-cli

Slack Web API CLI for AI agents — written in Go, supply-chain conscious.

## Why

- AI agents call this CLI to **gather information from Slack** (read channels, read threads, search messages/channels/users).
- Built mostly **from scratch** with only `slack-go/slack` and `spf13/cobra` as direct dependencies — minimal supply-chain footprint.
- JSON output by default for machine consumption; `--format pretty` for humans.

## Requirements

- Go **1.25** or newer (`slack-go/slack v0.23.1` requires it)
- A Slack App with the required OAuth scopes (see [Tokens](#tokens))

## Build

```bash
make build               # → bin/play-slack
# or
go build -o bin/play-slack ./...
```

## Tokens

The CLI reads tokens from environment variables:

| Variable             | Token type | Used by                                                       |
|----------------------|-----------|----------------------------------------------------------------|
| `SLACK_USER_TOKEN`   | `xoxp-…`  | Default. Required for `search-messages` (`search:read`, legacy) |
| `SLACK_BOT_TOKEN`    | `xoxb-…`  | Optional. Used when `--token-type bot` is passed                |

Use `--token-type user` (default) or `--token-type bot` to select.

Required OAuth scopes per command are documented in each `skills/<name>/SKILL.md`.

> **Note**: `search.messages` is **User Token only** (legacy `search:read`).

## Commands

| Command           | Token        | Purpose                                  |
|-------------------|--------------|------------------------------------------|
| `read-channel`    | user / bot   | Fetch recent messages in a channel       |
| `read-thread`     | user / bot   | Fetch all replies in a thread            |
| `search-messages` | **user**     | Search messages by Slack query           |
| `search-channels` | user / bot   | List / filter channels                   |
| `search-users`    | user / bot   | List / filter users                      |

Each command emits JSON containing both the result and `next_cursor` (or `next_page`) so an AI agent can paginate explicitly.

```bash
./bin/play-slack --help
./bin/play-slack read-channel C0123456 --limit 50
./bin/play-slack search-messages "from:@me incident"
```

## Skills (for AI agents)

Each command ships with a Claude-Code-compatible skill at `skills/<name>/SKILL.md`. The expected layout for the surrounding repo:

```
~/slack-cli/                      ← source of truth (this repo)
~/playground-okayama/.claude/skills/
  ├── slack-read-channel    → ~/slack-cli/skills/slack-read-channel
  ├── slack-read-thread     → ~/slack-cli/skills/slack-read-thread
  ├── slack-search-messages → ~/slack-cli/skills/slack-search-messages
  ├── slack-search-channels → ~/slack-cli/skills/slack-search-channels
  └── slack-search-users    → ~/slack-cli/skills/slack-search-users
~/playground-okayama/tools/slack/cli → ~/slack-cli   (local convenience; gitignored)
```

## Layout

```
.
├── main.go
├── cmd/                 # cobra subcommands
│   ├── root.go
│   ├── read_channel.go
│   ├── read_thread.go
│   ├── search_messages.go
│   ├── search_channels.go
│   └── search_users.go
├── internal/
│   ├── client/          # slack-go/slack thin wrapper (retry + timeout)
│   ├── config/          # env-based token loader
│   └── output/          # JSON / pretty formatter
├── skills/
│   └── slack-*/SKILL.md
├── Makefile
├── go.mod
└── README.md
```

## License

Private / internal. Not for redistribution.
