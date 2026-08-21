# Authentication and OAuth scopes

The CLI reads tokens from environment variables. It never prompts for a token.

| Variable | Selected with | Use |
|---|---|---|
| `SLACK_USER_TOKEN` | `--token-type user` (default) | All commands, including workspace search |
| `SLACK_BOT_TOKEN` | `--token-type bot` | Channel, thread, permalink, channel-list, and user-list reads |

`search-messages`, `search-files`, and `user-activity` are User Token only because
Slack's search API requires the legacy `search:read` scope.

## Scope matrix

| Command | Required scopes |
|---|---|
| `read-channel`, `read-thread`, `resolve` | One matching history scope: `channels:history`, `groups:history`, `im:history`, or `mpim:history` |
| `search-channels` | `channels:read`, `groups:read`, `im:read`, or `mpim:read`, according to `--types` |
| `search-users` | `users:read` |
| `search-messages` | `search:read` (User Token) |
| `search-files` | `search:read` and `files:read` (User Token) |
| `user-activity` | `users:read` and `search:read` (User Token) |

For a complete app, the User Token commonly needs:

```text
channels:history groups:history im:history mpim:history
channels:read groups:read im:read mpim:read
files:read search:read users:read
```

The corresponding Bot Token can omit the search scopes and uses:

```text
channels:history groups:history im:history mpim:history
channels:read groups:read im:read mpim:read users:read
```

## Diagnosing errors

The CLI enriches `missing_scope` errors with the scopes expected by the command
and the scopes reported by Slack. Treat these cases differently:

- `missing_scope`: the token is authenticated but lacks a required scope.
- `channel_not_found` or `not_in_channel`: the ID or access is wrong for the selected token.
- `ratelimited`: retry after the server's limit; do not increase pagination immediately.
- Empty search results: a result for the specified query, date range, and page limit—not proof of global absence.

Never print token values. Keep `SLACK_*_TOKEN` in the process environment or a
secret manager, not in a command argument or checked-in file.
