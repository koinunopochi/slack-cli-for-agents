---
name: slack-search-channels
description: >
  Slack のチャンネル一覧を取得し、ローカル substring 検索で絞り込む（conversations.list）。
  「Slack チャンネル一覧」「チャンネル検索」「チャンネル ID 取得」「conversations.list」に言及された場合に使用。
---

# Slack: search-channels

## いつ使うか
- チャンネル名から channel ID を引きたい
- 全チャンネルを舐めて命名規約違反を見つけたい
- private / DM / MPIM も含めて一覧したい

## 前提
- `slack` バイナリにパスが通っている
- `SLACK_USER_TOKEN` または `SLACK_BOT_TOKEN` が env に設定済み

## 必要 OAuth スコープ
| チャンネル種別 | scope（User / Bot 共通） |
|---|---|
| Public | `channels:read` |
| Private | `groups:read` |
| DM | `im:read` |
| MPIM | `mpim:read` |

`--types` で含めた種別すべての scope が必要。

## 使い方
```bash
slack search-channels [flags]
```

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--types` | string | public_channel | comma-separated: public_channel,private_channel,mpim,im |
| `--limit` | int | 200 | 1 リクエスト件数 (max 999) |
| `--cursor` | string | "" | 次ページの cursor |
| `--exclude-archived` | bool | true | archived を除外 |
| `--include-archived` | bool | false | archived を含める（exclude を上書き） |
| `--query` | string | "" | チャンネル名で substring 絞り込み (case-insensitive, ローカル適用) |
| `--team-id` | string | "" | Enterprise Grid: workspace 限定 |

### 典型例
```bash
# public チャンネル一覧
slack search-channels --limit 500

# private 含めて name に "incident" を含むもの
slack search-channels --types public_channel,private_channel --query incident

# 次ページ
slack search-channels --cursor dXNlcjpV...
```

## 出力（JSON）
```json
{
  "channels": [
    {
      "id": "C0123456",
      "name": "general",
      "is_private": false,
      "is_archived": false,
      "num_members": 42
    }
  ],
  "next_cursor": "dXNlcjpV...",
  "query": "incident"
}
```

`next_cursor` が空文字なら終端。

## エラー
| err | 意味 | 対処 |
|---|---|---|
| `missing_scope` | `--types` に対応する read scope が無い | App 設定で `*:read` を追加 |
| `invalid_arguments` | `--types` の値が不正 | public_channel / private_channel / mpim / im のみ |
| `ratelimited` | レート制限 (Tier 2: 20+/min) | リトライ後失敗時は少し待つ |

## 注意
- `--exclude-archived` の filter は **取得後** に適用されるため、1 ページの返却数が `--limit` より少なくなる可能性あり。
- `--query` は取得後の **ローカル filter**。サーバー検索ではないので、1 ページ内に絞り込み結果が出るとは限らない。広い `--limit` で取って絞ると確実。
- `--types` を public_channel 以外指定する場合、対応 scope (`groups:read` 等) が必須。
