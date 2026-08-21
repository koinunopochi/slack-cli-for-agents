---
name: slack-read-channel
description: >
  Slack のチャンネルから最近のメッセージを取得する。
  「Slack チャンネル」「メッセージ履歴」「conversations.history」に言及された場合に使用。
---

# Slack: read-channel

## いつ使うか
- 特定のチャンネルの最近のメッセージを取得したい
- AI が会話の文脈を把握する必要がある
- スレッドが張られているか確認したい（thread_ts が含まれる）

## 前提
- `slack` バイナリにパスが通っている（または本リポの clone 先の `bin/slack`。例: `<project>/tools/cli/slack/bin/slack`）
- `SLACK_USER_TOKEN` または `SLACK_BOT_TOKEN` が env に設定済み

## 必要 OAuth スコープ
| チャンネル種別 | 必要 scope（User / Bot 共通） |
|---|---|
| Public (`C…`) | `channels:history` |
| Private (`G…`) | `groups:history` |
| DM (`D…`) | `im:history` |
| MPIM | `mpim:history` |

## 使い方
```bash
slack read-channel <channel-id> [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--limit` | int | 100 | 1 リクエスト最大件数 (max 999) |
| `--cursor` | string | "" | 前回の `next_cursor` を渡してページ送り |
| `--oldest` | string | "" | Slack ts（"1234567890.123456"）以降のみ |
| `--latest` | string | "" | Slack ts 以前のみ |
| `--inclusive` | bool | false | --oldest/--latest を含む |
| `--include-metadata` | bool | false | message metadata を含める |
| `--token-type` | user/bot | user | どちらのトークンで叩くか |

### 典型例
```bash
slack read-channel C0123456 --limit 50
# 次ページ
slack read-channel C0123456 --cursor dXNlcjpV...
# 期間指定
slack read-channel C0123456 --oldest 1700000000.000000 --latest 1700086400.000000
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。件数が多いときはこれを使ってエージェントの context を圧迫しないこと。
- `--include-permalinks` 各メッセージに permalink を埋め込む（`chat.getPermalink` を 1 件ずつ呼ぶ）。**有効**。レスポンス後に追加 API コールが走る。
- ほか `--format json|pretty` / `--token-type user|bot` / `--timeout` / `--debug` は全コマンド共通。`slack --help` 参照。

## 出力（JSON）
```json
{
  "channel": "C0123456",
  "messages": [ /* slack-go の Message[] */ ],
  "has_more": true,
  "next_cursor": "dXNlcjpV...",
  "pin_count": 3
}
```
`next_cursor` が空文字なら終端。`has_more` も冗長な判定材料。

## エラー

> **scope エラー診断**: `missing_scope` の場合、CLI はコマンドが期待する scope (`needed:`) と token が実際に持っている scope (`provided:`、レスポンス HTTP ヘッダから取得) を併記する。ギャップをそのまま diff できるので、追加すべき scope が即わかる。

| err | 意味 | 対処 |
|---|---|---|
| `channel_not_found` | チャンネル ID が誤り | ID を確認 |
| `not_in_channel` | Bot が招待されていない（Bot Token） | チャンネルに /invite するか User Token に切替 |
| `missing_scope` | scope 不足 | App 設定で対応する `*_history` scope を付与 |
| `ratelimited` | レート制限 | CLI 内蔵リトライ後も失敗。少し待って再実行 |

## 注意
- 2025-05-29 以降に作られた non-Marketplace 商用 App は `--limit` が max=15、レート 1 req/min に制限される。Internal App として作るか Marketplace 公開する。
- private / DM / MPIM のメッセージは対応 scope (`groups:* / im:* / mpim:*`) が必要。
- DM の channel ID は `D…` で始まる。`conversations.list --types im` で取得可能。
