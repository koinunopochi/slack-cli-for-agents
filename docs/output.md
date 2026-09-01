# Output, pagination, and safety

## Formats

JSON is the default and is intended for agents and shell pipelines:

```sh
slack search-messages 'deploy' --format json
```

Use pretty output only for a human inspection:

```sh
slack read-channel C0123456789 --format pretty
```

All commands accept `--timeout`, `--debug`, and `--token-type`. Message commands
also accept `--include-permalinks`; it makes an extra `chat.getPermalink` call
per message. Search results already include permalinks, so the flag is a no-op
there.

## Context cache

With a User Token, normal commands ensure a local rolling 30-day context cache
once every 24 hours. The first check performs the bounded `search.messages`
walk; later checks reuse the cache until it is due for refresh. The cache is
keyed by workspace and user below the operating system's user cache directory,
with restrictive local permissions. A full rolling rebuild replaces the old
channel aggregates, which removes activity outside the window.

`slack context` returns an envelope like this:

```json
{
  "status": "fresh",
  "refreshed": true,
  "refresh_error": "",
  "context": {
    "workspace": {"id": "T0123456789"},
    "user": {"id": "U0123456789"},
    "window": {"days": 30},
    "partial": false,
    "channels": []
  }
}
```

The cache contains identifiers and activity aggregates only. It never contains
message text or token values. `partial` indicates that the bounded search page
walk did not cover all reported pages. If a refresh fails after a usable cache
exists, the CLI continues with `status: "stale"` and reports a coarse failure
category; a first-run failure is returned as an error. Normal commands also
write stale/partial warnings to stderr so their JSON result remains usable.

## Keep large payloads out of the context

`--out <path>` writes the full payload to a file and emits only a summary on
stdout:

```json
{"out":"/tmp/slack-result.json","format":"json","size_bytes":12345}
```

The summary is not the Slack result. Read the saved file and extract the fields
needed for the task. Treat the output path as local scratch data and do not
commit private messages.

## Pagination contracts

- `read-channel`, `read-thread`, and `resolve` return `has_more` and
  `next_cursor` when Slack has another cursor-based page.
- `search-channels` and `search-users` return `next_cursor`; their name query is
  local, so a first-page miss is inconclusive.
- `search-messages`, `search-files`, and `user-activity` return `page`,
  `page_count`, and `next_page` when another page is available.

Start with a bounded limit, count, or date range. Continue only when the output
pointer is present and the task needs more data. Report the query, channel/user
filter, time range, and page limit used when the result may be incomplete.

## Private data

The CLI is read-only, but Slack messages and files can still be sensitive. Do
not paste private channel or DM content beyond the request. Never expose
`url_private` values or other token-bearing URLs; identify attachments with
filename, file type, permalink, author, and timestamp instead.
