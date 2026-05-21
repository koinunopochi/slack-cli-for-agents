---
name: slack-resolve
description: >
  Slack の permalink（メッセージ URL）を 1 個渡すと、(channel, ts) に分解し、
  そのスレッド全体（親 + 全 reply）を JSON で返す。
  「Slack の URL を貼られた」「permalink を読みたい」「あのスレッドの中身を取って」
  に言及された場合に使用。
---

# Slack: resolve

## いつ使うか
- Slack の permalink を 1 個渡されて中身を読む必要がある
- read-thread を呼ぶために channel と thread_ts を手で分解するのが面倒
- スレッド内の 1 発言の URL を渡されて、文脈（スレッド全体）を一緒に欲しい

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` または `SLACK_BOT_TOKEN` が env に設定済み
- conversations.replies を呼ぶので、対応する `*_history` scope が必要

## 必要 OAuth スコープ
| チャンネル種別 | 必要 scope（User / Bot 共通） |
|---|---|
| Public (`C…`) | `channels:history` |
| Private (`G…`) | `groups:history` |
| DM (`D…`) | `im:history` |
| MPIM | `mpim:history` |

## 使い方
```bash
slack resolve <permalink> [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--limit` | int | 200 | replies の最大件数 |
| `--include-metadata` | bool | false | message metadata を含める |

### 受け付ける permalink 形式
- 単発メッセージ: `https://<workspace>.slack.com/archives/<CHANNEL>/p<TS_NO_DOT>`
- スレッド内の返信: `https://<workspace>.slack.com/archives/<CHANNEL>/p<TS>?thread_ts=<TS>&cid=...`

permalink のパス末尾 `p1716000000123456` から `1716000000.123456` を復元し、`thread_ts` が
クエリにあればそれを、なければパスから復元した ts をアンカーにして `conversations.replies` を呼ぶ。

### 典型例
```bash
# スレッド内の 1 発言から、スレッド全体を取得
slack resolve "https://example.slack.com/archives/C012/p1700000001000000?thread_ts=1700000000.123456&cid=C012"

# 単発メッセージ（スレッドが立っていれば返信も付く）
slack resolve "https://example.slack.com/archives/C012/p1700000000123456"

# 大量のリプライを context に乗せたくない場合
slack resolve "<url>" --out .claude/tmp/slack/thread.json
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。長いスレッドでは必ず使うこと。
- `--include-permalinks` 各メッセージに permalink を埋め込む。**有効**。「何件目の発言が引用元か」を後で示すために便利。
- ほか `--format json|pretty` / `--token-type user|bot` / `--timeout` / `--debug` は全コマンド共通。

## 出力（JSON）
```json
{
  "permalink": "<入力 URL>",
  "channel": "C0123456",
  "ts": "1700000000.123456",
  "thread_ts": "1700000000.123456",
  "messages": [ /* 親 + 全 reply */ ],
  "has_more": false,
  "next_cursor": ""
}
```

`thread_ts` は permalink に `?thread_ts=...` が無ければ空文字。
`ts` はパスから復元した値（permalink そのものが指すメッセージ）。

## エラー
| err | 意味 | 対処 |
|---|---|---|
| `not a Slack message permalink` | URL が `/archives/<CH>/p<TS>` の形式に合っていない | permalink を確認 |
| `thread_not_found` | ts に対応するメッセージが無い（削除された等） | URL を確認 |
| `channel_not_found` | チャンネル ID が誤り or アクセス不可 | scope と権限を確認 |
| `not_in_channel` | Bot Token でチャンネルに招待されていない | User Token に切替 or /invite |
| `missing_scope` | scope 不足 | App 設定で `*_history` 追加 |

## 注意
- スレッドが張られていない単発メッセージの場合、`messages` には 1 件だけ入る。
- Enterprise Grid の channel ID は同じ形式 (`C…` / `G…`)。`team-id` がクエリに含まれることもあるが、resolve は使わない。
- 2025-05-29 以降の non-Marketplace 商用 App は `--limit` max=15 制限あり。
