---
name: slack-user-activity
description: >
  Slack のユーザ名（または User ID）を 1 個渡すと、ID 解決して
  そのユーザの直近 N 日の発言を search.messages で取得する。
  「あの人の最近の発言を見たい」「PR 作成者の最近の文脈が欲しい」
  「依頼者の直近の動きを追いたい」に言及された場合に使用。
  search.messages 経由なので User Token 必須。
---

# Slack: user-activity

## いつ使うか
- 名前（display_name / real_name / Slack name）からその人の直近発言を一気に出したい
- GitHub PR の author を Slack で同定して、直近の動きから背景を拾いたい
- User ID が分かっているなら直接渡してもよい

> **強く推奨**: 可能なら **Slack user ID (`U0123456789` のような `U`/`W` 始まりの ID) を直接渡す**。
> 名前解決は `users.list` を最大 `--max-user-pages` 回スキャンする実装なので、大規模 workspace では
> ヒットするまで時間がかかる or デフォルト 30 ページ（= 6,000 ユーザー）で届かないことがある。
> ID が分からないときはまず `slack search-users --query <name> --max-pages 50` で当たりを付けると速い。

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` (xoxp-) が env に設定済み（`SLACK_BOT_TOKEN` 不可）
- App に `users:read` と `search:read` の両方が付与済み

## 必要 OAuth スコープ
| scope | 役割 |
|---|---|
| `users:read` | users.list / users.info で名前 → ID 解決 |
| `search:read` (User Token のみ) | search.messages で発言取得 |
| `users:read.email` | 任意。profile.email が必要なときだけ |

## 使い方
```bash
slack user-activity <name-or-id> [flags]
```

引数:
- ID 形式（`U…` / `W…`、9 文字以上、英大文字+数字のみ）→ users.info で即解決
- それ以外 → users.list を paginated に回し、name / real_name / display_name の
  完全一致 → 部分一致の順でマッチ。複数ヒットすると候補一覧を載せてエラー。

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--days` | int | 14 | 過去 N 日（Slack 検索構文の `after:<YYYY-MM-DD>` を組み立てる） |
| `--count` | int | 50 | 1 ページの件数 (max 100) |
| `--page` | int | 1 | ページ番号 (1-based, max 100) |
| `--in` | string | "" | チャンネル絞り込み。例 `#team-example`（`#` 自動補完） |
| `--sort` | string | timestamp | `score` or `timestamp` |
| `--sort-direction` | string | desc | `asc` or `desc` |
| `--max-user-pages` | int | 10 | users.list のスキャンページ上限。大規模 ws ではこれを増やす |

### 典型例
```bash
# Slack 名で直近 7 日
slack user-activity alice --days 7

# User ID で（解決ステップをスキップ、最速）
slack user-activity U0123456789 --days 30

# チャンネル絞り込み（# は自動補完される）
slack user-activity alice --days 14 --in team-example

# 大量取れることが見えてるなら必ず --out で context を守る
slack user-activity alice --days 30 --count 100 --out .claude/tmp/slack/alice-activity.json
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。30 日も取ると重くなるので推奨。
- `--include-permalinks` `search.messages` の各 match は元から `permalink` を含むので **no-op**。
- ほか `--format json|pretty` / `--token-type` (user 固定) / `--timeout` / `--debug` は全コマンド共通。

## 出力（JSON）
```json
{
  "user": {
    "id": "U0123456789",
    "name": "alice",
    "real_name": "Alice Example",
    "display_name": "ali"
  },
  "query": "from:@alice after:2026-05-07",
  "matches": [ /* SearchMessage[] (permalink つき) */ ],
  "total": 42,
  "page": 1,
  "page_count": 1,
  "next_page": null
}
```

`query` には実際に Slack に投げた検索クエリが入る（再現性のため）。

## エラー

> **scope エラー診断**: `missing_scope` の場合、CLI はコマンドが期待する scope (`needed:`) と token が実際に持っている scope (`provided:`、レスポンス HTTP ヘッダから取得) を併記する。ギャップをそのまま diff できるので、追加すべき scope が即わかる。

| err | 意味 | 対処 |
|---|---|---|
| `user-activity requires --token-type user` | Bot Token を指定した | `--token-type user` に切替 |
| `no user matched "..."` | 候補ゼロ | スペル確認、`--max-user-pages` を増やす |
| `ambiguous user "..." — N candidates: ...` | 部分一致が複数 | 候補リストから絞ったキーワード or User ID で再指定 |
| `users.info(U...): ...` | Slack 側のエラー（user_not_found 等） | ID の typo を確認 |
| `missing_scope` | `users:read` か `search:read` 不足 | App 設定で追加 |
| `ratelimited` | レート制限 | 少し待って再実行 |

## 注意
- `--in` の値は `channel-name` でも `#channel-name` でも `<#C0123|name>` でも OK。先頭に `#` を自動で付ける。
- `from:@<username>` は **Slack の username** を見るので、表示名 (display_name) で当たらない場合がある。その場合は User ID で渡すか、display_name を name に揃える。
- search.messages は legacy 扱い。長期的には `assistant.search.context` 移行候補。
- 「自分の発言」を取るなら `slack user-activity <self-id> --days 7` の方が `search-messages "from:@me"` より早い（解決ステップをスキップ）。
