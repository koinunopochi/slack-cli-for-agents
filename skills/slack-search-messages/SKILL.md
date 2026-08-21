---
name: slack-search-messages
description: >
  Slack のメッセージを検索する（User Token 必須・legacy search:read scope）。
  「Slack 検索」「メッセージ検索」「search.messages」「Slack を ground search」に言及された場合に使用。
---

# Slack: search-messages

## いつ使うか
- キーワード / 投稿者 / チャンネル指定でメッセージを横断検索したい
- 過去の議論やエラー報告を発掘したい
- AI が「いつ誰が何を言ったか」を ground search したい

## **重要な制約**
- **User Token 専用**（Bot Token では呼べない）。`--token-type user` が必須。
- legacy scope `search:read` を要求。Slack は `assistant.search.context` への移行を推奨しているが、現状 `search.messages` を呼ぶには `search:read` しか手がない。
- 検索 index に依存して **反映遅延** あり（数十秒〜数分）。直近メッセージは `read-channel` を使う。
- ユーザの Slack UI 上の検索フィルタ（例: 参加チャンネルのみ）が API 応答にも適用される。

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` が env に設定済み

## 必要 OAuth スコープ
- User Token: `search:read` （legacy）

## 使い方
```bash
slack search-messages "<query>" [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--count` | int | 20 | 1 ページの件数 (max 100) |
| `--page` | int | 1 | ページ番号 (1-based, max 100 = 最大 10,000 件まで遡及) |
| `--sort` | string | score | "score" or "timestamp" |
| `--sort-direction` | string | desc | "asc" or "desc" |
| `--highlight` | bool | false | highlight マーカーを含める |
| `--team-id` | string | "" | Enterprise Grid で workspace 限定検索 |

### Slack 検索クエリ構文（よく使う）
| 構文 | 意味 |
|---|---|
| `from:@user` | 投稿者で絞り込み |
| `in:#channel` | チャンネルで絞り込み |
| `before:2024-01-01` / `after:2024-01-01` | 日付絞り込み |
| `has:link` / `has:reaction` | 添付・リアクション絞り込み |
| `"完全一致フレーズ"` | 完全一致 |

### 典型例
```bash
slack search-messages "incident from:@me"
slack search-messages "in:#alerts before:2024-12-01" --sort timestamp --sort-direction desc
slack search-messages "deployment failed" --count 100 --page 2
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。ヒット件数が読めないので、検索系では特に推奨。
- `--include-permalinks` `search.messages` の各 match は既に `permalink` を含むので **no-op**。
- ほか `--format json|pretty` / `--token-type user|bot` (user 固定) / `--timeout` / `--debug` は全コマンド共通。

## 出力（JSON）
```json
{
  "query": "incident from:@me",
  "matches": [ /* SearchMessage[] */ ],
  "total": 123,
  "page": 1,
  "page_count": 7,
  "next_page": 2
}
```

`next_page` が `null` なら終端。最大 100 ページ (= 10,000 件) で打ち止め。
それ以上欲しい場合はクエリを時間で分割する（`before:` / `after:`）。

## エラー

> **scope エラー診断**: `missing_scope` の場合、CLI はコマンドが期待する scope (`needed:`) と token が実際に持っている scope (`provided:`、レスポンス HTTP ヘッダから取得) を併記する。ギャップをそのまま diff できるので、追加すべき scope が即わかる。

| err | 意味 | 対処 |
|---|---|---|
| `missing_scope` | `search:read` が App に付いていない | App 設定で User Token Scope に `search:read` 追加 |
| `invalid_auth` | User Token が無効・期限切れ | re-authorize |
| `ratelimited` | レート制限 (Tier 2: 20+/min) | CLI 内蔵リトライ後も失敗。少し待って再実行 |
| `search-messages requires --token-type user` | CLI 側エラー、Bot Token を指定した | `--token-type user` に切替 |

## 注意
- `search.messages` は legacy 扱い。長期的には `assistant.search.context` (Real-time Search API) への移行が推奨される。
- 検索結果は近接マッチが 1 件にまとめられる。
