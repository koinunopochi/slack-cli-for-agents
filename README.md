# slack-cli-for-agents

Slack Web API CLI for AI agents, written in Go. It is a small, read-only
wrapper around Slack Web API: it reads channels, threads, messages, files,
channels, and users, but does not manage Slack apps or change Slack data.

## Quick start

```bash
make build
export SLACK_USER_TOKEN=xoxp-...
./bin/slack --help
./bin/slack read-channel C0123456789 --limit 50
```

Use `--token-type bot` with `SLACK_BOT_TOKEN` when a read operation is available
to the bot. Workspace search commands (`search-messages`, `search-files`, and
`user-activity`) require a User Token.

## Documentation

The executable help is the entry point and prints the detailed documentation
URL:

```bash
slack --help
slack <command> --help
```

See [docs/README.md](docs/README.md) for command selection, authentication,
pagination, output, and safety details. Agent-oriented instructions live in
[AGENTS.md](AGENTS.md); [CLAUDE.md](CLAUDE.md) points to the same file.

## Commands

| Command | Purpose |
|---|---|
| `read-channel` | Read recent messages from a known channel ID |
| `read-thread` | Read a thread from a channel ID and thread timestamp |
| `resolve` | Resolve a Slack permalink and read its enclosing thread |
| `search-messages` | Search messages across the workspace |
| `search-files` | Search Slack attachments |
| `search-channels` | List and locally filter conversations |
| `search-users` | List and locally filter users |
| `user-activity` | Find a user's recent messages |

## Requirements

- Go 1.25 or newer
- A Slack App with the OAuth scopes described in [authentication.md](docs/authentication.md)

## Build and test

```bash
make build
make test
make lint
```

## Layout

```text
.
├── cmd/          # Cobra commands and help text
├── internal/     # API client, config, errors, output, permalink helpers
├── docs/         # Detailed operator and agent documentation
├── AGENTS.md     # Agent entry point
├── CLAUDE.md     # Symlink to AGENTS.md
└── bin/slack     # Local build output
```

## License

Private / internal. Not for redistribution.
