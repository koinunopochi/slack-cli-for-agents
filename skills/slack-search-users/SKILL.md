---
name: slack-search-users
description: >
  Slack のユーザ一覧を取得し、ローカル substring 検索で絞り込む（users.list / GetUsersPaginated）。
  「Slack ユーザ一覧」「ユーザ検索」「user ID 取得」「users.list」に言及された場合に使用。
---

# Slack: search-users

## いつ使うか
- ユーザ名から user ID (`U…`) を引きたい
- 全ユーザを舐めて mention に使う名前を確認したい
- 削除済ユーザも含めたい

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` または `SLACK_BOT_TOKEN` が env に設定済み

## 必要 OAuth スコープ
- `users:read` （User / Bot 共通）
- email を取りたいときは `users:read.email` を追加

## 使い方
```bash
slack search-users [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--limit` | int | 200 | 1 リクエスト件数（推奨 max 200） |
| `--cursor` | string | "" | 開始 cursor（前回 next_cursor を渡す） |
| `--query` | string | "" | name / real_name / display_name で絞り込み (case-insensitive, ローカル) |
| `--max-pages` | int | 1 | この回数だけページ取得して停止（大規模ws 防御） |
| `--include-deleted` | bool | false | 削除済ユーザも含む |
| `--team-id` | string | "" | Enterprise Grid: org-level token 用 |

### 典型例
```bash
# 200 人だけ取得
slack search-users

# 名前で絞り込み
slack search-users --query alice

# 全ユーザを 10 ページまで取得
slack search-users --max-pages 10 --query .

# 削除済も含む
slack search-users --include-deleted

# 続きを取得（前回出力の next_cursor を渡す）
slack search-users --cursor 'dXNlcjpV...' --max-pages 5
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。大規模 workspace で `--max-pages` を増やす時は必ず使うこと。
- `--include-permalinks` ユーザーリストを返すコマンドなので **no-op**。
- ほか `--format json|pretty` / `--token-type user|bot` / `--timeout` / `--debug` は全コマンド共通。

## 出力（JSON）
```json
{
  "users": [
    {
      "id": "U0123456",
      "name": "alice",
      "real_name": "Alice Example",
      "profile": { "display_name": "alice", "email": "..." },
      "is_bot": false,
      "deleted": false
    }
  ],
  "next_cursor": "dXNlcjpV...",
  "page_count": 1,
  "query": "alice"
}
```

`next_cursor` が空文字なら全件取得済み（または `--max-pages` で打ち切ったが次ページが無い）。
空でなければ次は `--cursor <next_cursor>` で続きを取れる。

## エラー

> **scope エラー診断**: `missing_scope` の場合、CLI はコマンドが期待する scope (`needed:`) と token が実際に持っている scope (`provided:`、レスポンス HTTP ヘッダから取得) を併記する。ギャップをそのまま diff できるので、追加すべき scope が即わかる。

| err | 意味 | 対処 |
|---|---|---|
| `missing_scope` | `users:read` が App に付いていない | App 設定で `users:read` 追加。email なら `users:read.email` も |
| `limit_required` | 大規模 workspace で `--limit` 未指定 | `--limit 200` 指定（CLI のデフォ値なので通常起きない） |
| `ratelimited` | レート制限 (Tier 2: 20+/min) | リトライ後失敗時は少し待つ |

## 注意
- `GetUsers()` (全件取得) は大規模 workspace で爆発するので、本 CLI は内部的に `GetUsersPaginated` を使い `--max-pages` で件数を制御している。**広く取りたいときは `--max-pages` を明示**。
- `--query` は **ローカル filter**。サーバー側検索ではない。ヒット 0 件のとき、もっとページを取れば見つかる可能性があるので `--max-pages` を増やす。
- email は `users:read.email` scope がないと profile.email が空になる。
