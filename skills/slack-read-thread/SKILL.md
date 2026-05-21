---
name: slack-read-thread
description: >
  Slack のスレッド（特定の thread_ts に紐づく返信全て）を取得する。
  「Slack スレッド」「スレッド読み取り」「conversations.replies」「thread_ts」に言及された場合に使用。
---

# Slack: read-thread

## いつ使うか
- 特定のスレッド全件を取り、議論の流れを把握したい
- 返信件数の多いスレッドを順次取得（ページネーション）したい
- 親メッセージを除いて返信だけほしい（`--exclude-parent`）

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` または `SLACK_BOT_TOKEN` が env に設定済み

## 必要 OAuth スコープ
| チャンネル種別 | scope（User / Bot 共通） |
|---|---|
| Public | `channels:history` |
| Private | `groups:history` |
| DM | `im:history` |
| MPIM | `mpim:history` |

## 使い方
```bash
slack read-thread <channel-id> <thread-ts> [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--limit` | int | 200 | 1 リクエスト件数（推奨 max 200） |
| `--cursor` | string | "" | 次ページの cursor |
| `--oldest` | string | "" | Slack ts 以降のみ |
| `--latest` | string | "" | Slack ts 以前のみ |
| `--inclusive` | bool | false | --oldest/--latest を含む |
| `--exclude-parent` | bool | false | 親メッセージを除外 |
| `--include-metadata` | bool | false | metadata を含める |

### 典型例
```bash
# まず read-channel で thread_ts を見つける
slack read-channel C0123456 --limit 100 | jq '.messages[] | select(.thread_ts) | .ts'

# その ts を渡してスレッド取得
slack read-thread C0123456 1700000000.123456

# 親なしで返信だけ
slack read-thread C0123456 1700000000.123456 --exclude-parent
```

## 出力（JSON）
```json
{
  "channel": "C0123456",
  "thread_ts": "1700000000.123456",
  "messages": [ /* 親 + 全 reply（exclude-parent 指定時は親なし） */ ],
  "has_more": false,
  "next_cursor": ""
}
```

`next_cursor` が空文字なら終端。

## エラー
| err | 意味 | 対処 |
|---|---|---|
| `thread_not_found` | thread_ts に対応する親が無い、または subtype がスレッド不可（channel_join 等） | thread_ts を確認 |
| `channel_not_found` | チャンネル ID が誤り | ID 確認 |
| `not_in_channel` | Bot が招待されていない | /invite or User Token に切替 |
| `missing_scope` | scope 不足 | App 設定で `*_history` を付与 |

## 注意
- 戻り値の 1 件目は **親メッセージ自身**（ts == thread_ts）。`--exclude-parent` で除外可。
- 2025-05-29 以降の non-Marketplace 商用 App は `--limit` max=15 制限あり。
