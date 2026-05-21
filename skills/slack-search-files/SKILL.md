---
name: slack-search-files
description: >
  Slack 内のファイル（PDF / 画像 / スニペット / 各種添付）を search.files で
  横断検索する。User Token 必須・legacy search:read scope。
  「あの設計図どこ」「過去にアップした PDF」「○○ さんが貼った画像」に言及された場合に使用。
---

# Slack: search-files

## いつ使うか
- ファイル名・キーワード・投稿者・チャンネル・日付などで Slack 添付を探したい
- 過去にアップロードされた設計図 / 画像 / PDF / スニペットを掘りたい
- `search-messages` でテキストを取り、`search-files` でその時期の添付も合わせて拾うフローを組みたい

## 前提
- `slack` バイナリにパスが通っている
- **`SLACK_USER_TOKEN` (xoxp-) が env に設定済み**（Bot Token 不可）
- App に `search:read` (User Token, legacy) が付与済み

## 必要 OAuth スコープ
| scope | 役割 |
|---|---|
| `search:read` (User Token のみ) | search.files |
| `files:read` | 検索結果のファイルメタデータ閲覧（多くの場合付随的に必要） |

## 使い方
```bash
slack search-files <query> [flags]
```

クエリ構文は `search-messages` と同じ:
- `from:@user`: アップロード者
- `in:#channel`: 投稿チャンネル
- `before:2024-01-01` / `after:2024-01-01`: 日付
- `"phrase"`: ファイル名やタイトルの完全一致
- `has:link`: 任意のフィルタ

主要フラグ:
| flag | 型 | 既定 | 説明 |
|---|---|---|---|
| `--count` | int | 20 | 1 ページの件数 (max 100) |
| `--page` | int | 1 | ページ番号 (1-based) |
| `--sort` | string | score | `score` or `timestamp` |
| `--sort-direction` | string | desc | `asc` or `desc` |
| `--highlight` | bool | false | highlight マーカーを含める |
| `--team-id` | string | "" | Enterprise Grid: workspace 絞り込み |

### 典型例
```bash
# 「設計図」 + 投稿者で絞り込み
slack search-files "design from:@okayama after:2026-05-01"

# 特定チャンネルの画像
slack search-files "schema.png in:#team-dev-ict-develop"

# ファイル種別を変えるなら filetype: クエリも使える（Slack 側の機能）
slack search-files "filetype:pdf 議事録 after:2026-01-01"
```

## 共通フラグ
- `--out <path>` 結果を `<path>` に書き出し、stdout は `{out, format, size_bytes}` のサマリーだけ。ファイル一覧は URL / メタが嵩むので推奨。
- `--include-permalinks` ファイル検索なので **no-op**。各 file には `permalink` / `url_private` フィールドが元から付く。
- ほか `--format json|pretty` / `--token-type` (user 固定) / `--timeout` / `--debug` は全コマンド共通。

## 出力（JSON）
```json
{
  "query": "design from:@okayama",
  "matches": [
    {
      "id": "F0123",
      "name": "schema.png",
      "title": "テナント分離スキーマ",
      "filetype": "png",
      "user": "U08L3MPJB9T",
      "size": 102400,
      "permalink": "https://...",
      "url_private": "https://...",
      "channels": ["C012"],
      "groups": [],
      "ims": [],
      "timestamp": 1700000000
    }
  ],
  "total": 12,
  "page": 1,
  "page_count": 1,
  "next_page": null
}
```

`next_page` が `null` なら終端。最大 100 ページで打ち止め。

## エラー

> **scope エラー診断**: `missing_scope` の場合、CLI はコマンドが期待する scope (`needed:`) と token が実際に持っている scope (`provided:`、レスポンス HTTP ヘッダから取得) を併記する。ギャップをそのまま diff できるので、追加すべき scope が即わかる。

| err | 意味 | 対処 |
|---|---|---|
| `search-files requires --token-type user` | Bot Token を指定した | `--token-type user` に切替 |
| `missing_scope` | `search:read` が App に付いていない | App 設定で追加（User Token Scope） |
| `invalid_auth` | User Token が無効・期限切れ | re-authorize |
| `ratelimited` | レート制限 | 少し待って再実行 |

## 注意
- `search.files` も legacy 扱い。`search.messages` と同じ scope (`search:read`) を要求する。
- 検索結果の `url_private` は token 付き URL。ダウンロードするときは Authorization ヘッダか curl `-H "Authorization: Bearer $SLACK_USER_TOKEN"` が必要。
- 検索インデックスへの反映は遅延するため、直前にアップしたファイルが出ない場合がある（数十秒〜数分）。直近を確実に拾うなら `read-channel` でチャンネルから辿る。
