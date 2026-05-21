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
make build               # → bin/slack
# or
go build -o bin/slack ./...
```

## Tokens

The CLI reads tokens from environment variables:

| Variable             | Token type | Used by                                                       |
|----------------------|-----------|----------------------------------------------------------------|
| `SLACK_USER_TOKEN`   | `xoxp-…`  | Default. Required for `search-messages` (`search:read`, legacy) |
| `SLACK_BOT_TOKEN`    | `xoxb-…`  | Optional. Used when `--token-type bot` is passed                |

Use `--token-type user` (default) or `--token-type bot` to select.

> **Note**: `search.messages` / `search.files` / `user-activity` are **User Token only** (legacy `search:read`).

### Recommended OAuth scopes (paste these into your Slack App)

To use **every** command in this CLI, add the following scopes to your Slack
App's OAuth & Permissions page and reinstall the app to your workspace.

**User Token Scopes** (`SLACK_USER_TOKEN`, recommended primary token):

```
channels:history   groups:history   im:history   mpim:history
channels:read      groups:read      im:read      mpim:read
files:read         search:read      users:read
```

**Bot Token Scopes** (`SLACK_BOT_TOKEN`, optional — Bot Tokens cannot call
`search.messages` / `search.files` / `user-activity`):

```
channels:history   groups:history   im:history   mpim:history
channels:read      groups:read      im:read      mpim:read
users:read
```

### Per-command scope matrix

| Command           | Token        | Scopes Slack will check                                                |
|-------------------|--------------|------------------------------------------------------------------------|
| `read-channel`    | user / bot   | `channels:history` / `groups:history` / `im:history` / `mpim:history`  |
| `read-thread`     | user / bot   | same as `read-channel`                                                 |
| `resolve`         | user / bot   | same as `read-channel`                                                 |
| `search-channels` | user / bot   | `channels:read` / `groups:read` / `im:read` / `mpim:read` (per `--types`) |
| `search-users`    | user / bot   | `users:read`                                                            |
| `search-messages` | **user**     | `search:read`                                                          |
| `search-files`    | **user**     | `search:read`, `files:read`                                            |
| `user-activity`   | **user**     | `users:read`, `search:read`                                            |

When a call fails with `missing_scope`, the CLI prints which scopes the command
expects and which scopes the token actually has (from the `x-oauth-scopes`
header captured on the most recent response), so the gap is easy to spot.

## Commands

| Command           | Token        | Purpose                                                      |
|-------------------|--------------|--------------------------------------------------------------|
| `read-channel`    | user / bot   | Fetch recent messages in a channel                           |
| `read-thread`     | user / bot   | Fetch all replies in a thread                                |
| `resolve`         | user / bot   | Resolve a Slack permalink and fetch the enclosing thread     |
| `search-messages` | **user**     | Search messages by Slack query                               |
| `search-files`    | **user**     | Search files (PDF / image / snippet) by Slack query          |
| `search-channels` | user / bot   | List / filter channels                                       |
| `search-users`    | user / bot   | List / filter users                                          |
| `user-activity`   | **user**     | Resolve a user (name → ID) and list their recent messages    |

Each command emits JSON containing both the result and `next_cursor` (or `next_page`) so an AI agent can paginate explicitly.

```bash
./bin/slack --help
./bin/slack read-channel C0123456 --limit 50
./bin/slack search-messages "from:@me incident"
./bin/slack resolve "https://example.slack.com/archives/C012/p1700000000123456"
./bin/slack user-activity okayama --days 7
```

### Common flags (any command)

| Flag                    | Effect                                                                                                        |
|-------------------------|---------------------------------------------------------------------------------------------------------------|
| `--out <path>`          | Write the payload to `<path>` instead of stdout; stdout receives a small JSON summary (`{out, format, size_bytes}`). Parent dirs are created automatically. Use this to keep large payloads out of the agent's context window. |
| `--include-permalinks`  | Populate `permalink` on each message via `chat.getPermalink`. Costs one extra API call per message. `search-messages` already returns permalinks so this is a no-op there. |
| `--format json\|pretty` | Output format. `json` is single-line for piping; `pretty` indents.                                            |
| `--token-type user\|bot`| Which token to use. `search-messages`, `search-files`, and `user-activity` require `user`.                    |
| `--timeout <duration>`  | HTTP timeout per call. Default 30s.                                                                           |
| `--debug`               | Enable slack-go debug logging to stderr.                                                                      |

## Skills (for AI agents)

Each command ships with a Claude-Code-compatible skill at `skills/<name>/SKILL.md`. The expected layout for the surrounding repo:

```
~/slack-cli/                      ← source of truth (this repo)
~/playground-okayama/.claude/skills/
  ├── slack-read-channel    → ~/slack-cli/skills/slack-read-channel
  ├── slack-read-thread     → ~/slack-cli/skills/slack-read-thread
  ├── slack-resolve         → ~/slack-cli/skills/slack-resolve
  ├── slack-search-messages → ~/slack-cli/skills/slack-search-messages
  ├── slack-search-files    → ~/slack-cli/skills/slack-search-files
  ├── slack-search-channels → ~/slack-cli/skills/slack-search-channels
  ├── slack-search-users    → ~/slack-cli/skills/slack-search-users
  └── slack-user-activity   → ~/slack-cli/skills/slack-user-activity
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
│   ├── resolve.go
│   ├── search_channels.go
│   ├── search_files.go
│   ├── search_messages.go
│   ├── search_users.go
│   └── user_activity.go
├── internal/
│   ├── client/          # slack-go/slack thin wrapper (retry + timeout)
│   ├── config/          # env-based token loader
│   ├── output/          # JSON / pretty formatter, --out file writer
│   └── permalink/       # chat.getPermalink enrichment for messages
├── skills/
│   └── slack-*/SKILL.md
├── Makefile
├── go.mod
└── README.md
```

## License

Private / internal. Not for redistribution.
