# Command reference

The executable's `--help` output is authoritative for flags and defaults. This
page explains when to choose each command and how to continue from its output.

All examples use placeholder IDs and a placeholder workspace.

## read-channel

Read recent messages from a known channel.

```sh
slack read-channel C0123456789 --limit 100
slack read-channel C0123456789 --oldest 1710000000.000000 --latest 1710086400.000000
```

Use `--oldest` and `--latest` to bound the time window, and `--cursor` with
`next_cursor` to continue. If a message has a `thread_ts` and its replies are
needed, call [`read-thread`](#read-thread). If the input is a permalink, use
[`resolve`](#resolve) instead of parsing the URL by hand.

Required history scope depends on the channel type: `channels:history`,
`groups:history`, `im:history`, or `mpim:history`.

## read-thread

Read a thread when both its channel ID and parent timestamp are known.

```sh
slack read-thread C0123456789 1710000000.000000 --limit 200
slack read-thread C0123456789 1710000000.000000 --exclude-parent
```

The first message is the parent unless `--exclude-parent` is set. Continue with
`--cursor` when `next_cursor` is present. A permalink should go through
[`resolve`](#resolve), which extracts the channel and timestamp safely.

## resolve

Resolve a Slack permalink and fetch the target message plus its enclosing thread.

```sh
slack resolve 'https://example.slack.com/archives/C0123456789/p1710000000000000'
```

Quote the URL for shell safety. A `thread_ts` query parameter selects the
enclosing thread; without it, the URL timestamp is used as the thread anchor.
Use `--limit` for a large thread. Do not guess a channel from a malformed URL or
silently replace an access error with a history search.

## search-channels

List conversations and filter channel names locally to resolve a channel ID.

```sh
slack search-channels --query 'team-example' --max-pages 5
slack search-channels --query 'private' --types private_channel --max-pages 5
```

Slack does not provide a server-side name query here. `--query` filters the
pages already fetched. Increase `--max-pages` or continue with `--cursor` when a
large workspace has no match on the first page. Use `--all` only when a complete
walk is actually required. Check `name`, `is_private`, and `is_archived` before
choosing among multiple candidates.

## search-files

Search Slack attachments using the same query syntax as message search.

```sh
slack search-files 'design from:@alice after:2026-01-01' --sort timestamp
slack search-files 'schema.png in:#team-example'
```

This command requires a User Token and the legacy `search:read` scope. Use
`from:`, `in:`, `after:`, `before:`, `has:link`, and quoted phrases to narrow the
search. Use `--sort timestamp` for a timeline; the default score sort is better
for discovery. Results identify files with metadata and links; do not expose a
`url_private` value.

## search-messages

Search messages across the workspace by phrase, author, channel, or date.

```sh
slack search-messages '"release plan" in:#team-example after:2026-01-01' --sort timestamp --count 100
slack search-messages 'from:@alice has:link before:2026-02-01'
```

This command requires a User Token and the legacy `search:read` scope. Start
with a narrow query. Use the default score sort for discovery and
`--sort timestamp` for chronology. Continue with `--page` only when
`next_page` is non-null; split by date or channel rather than treating the page
limit as a complete export. Use a returned channel and timestamp with
[`read-thread`](#read-thread) when the replies matter.

## search-users

List users and filter names locally to resolve a user ID.

```sh
slack search-users --query 'alice' --max-pages 10
```

`--query` matches `name`, `real_name`, and `display_name` in fetched pages. Use
`--max-pages` and then `--cursor` when the first page is insufficient. Compare
the candidate ID and names before choosing; do not resolve an ambiguous name by
guessing. Avoid exposing email addresses unless the task requires one.

## user-activity

Resolve a user and search their recent messages. Prefer a known user ID so the
name-resolution scan is skipped.

```sh
slack user-activity U0123456789 --days 14 --sort timestamp --sort-direction asc
slack user-activity alice --days 7 --in '#team-example'
```

This command requires a User Token, `users:read`, and legacy `search:read`.
Use `--max-user-pages` only as far as necessary when starting from a name. If
there are multiple matching people, use [`search-users`](#search-users) first.
Use `--days`, `--in`, and `--count` to keep the search bounded. Continue with
`--page` when `next_page` is present.
