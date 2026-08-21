# Agent instructions

Use the CLI help as the source of truth for invocation:

1. Run `slack --help` to see the read-only commands and the documentation entry point.
2. Run `slack <command> --help` before using a command. Its `Detailed documentation` link points to the relevant section in `docs/commands.md`.
3. Read `docs/README.md` and the linked page when the task needs routing, scopes, pagination, or output details.

Do not recreate a second command manual in an agent skill. Keep tokens in
`SLACK_USER_TOKEN` / `SLACK_BOT_TOKEN` and never print them. This CLI only reads
Slack; it does not post, edit, delete, react, upload, or download files.
