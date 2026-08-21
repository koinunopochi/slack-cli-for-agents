# Slack CLI documentation

The CLI is the entry point. Start with the executable's own help, then use the
linked page when the task needs more context:

```sh
slack --help
slack <command> --help
```

Every help page prints a `Detailed documentation` link. The links are kept in
the command help so agents can discover the documentation without a separate
skill or a copied command manual.

## Choose a command by the input you have

| Input or goal | Command |
|---|---|
| A Slack message permalink | [`resolve`](commands.md#resolve) |
| A known channel ID and recent history | [`read-channel`](commands.md#read-channel) |
| A channel ID and thread timestamp | [`read-thread`](commands.md#read-thread) |
| A phrase, author, channel, or date range | [`search-messages`](commands.md#search-messages) |
| A PDF, image, snippet, or other attachment | [`search-files`](commands.md#search-files) |
| A channel name that must become an ID | [`search-channels`](commands.md#search-channels) |
| A display name that must become a user ID | [`search-users`](commands.md#search-users) |
| A user's recent messages | [`user-activity`](commands.md#user-activity) |

## Documentation map

- [Commands](commands.md) — purpose, examples, important flags, and pagination.
- [Authentication](authentication.md) — token variables, OAuth scopes, and scope errors.
- [Output and safety](output.md) — JSON shape, `--out`, pagination, permalinks, and private data.

## Operating boundary

This tool is read-only. It reads Slack Web API resources and does not post,
edit, delete, react, upload, or download files. Use a User Token only where the
command help says it is required. Keep tokens in environment variables and do
not paste token-bearing URLs into logs or replies.

When a result contains a cursor or next page, continue only as far as the task
needs. A bounded search returning no match is not proof that the workspace has
no such message; report the query, time range, and page limit used.
