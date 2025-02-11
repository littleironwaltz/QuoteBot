# QuoteBot

QuoteBotは、名言を定期的にBlueskyに投稿するボットアプリケーションです。クリーンアーキテクチャの原則に従い、保守性と拡張性の高い設計を採用しています。

## 機能

- JSONファイルから名言をランダムに選択
- Blueskyへの自動投稿（1時間間隔）
- 環境変数による設定管理
- クリーンアーキテクチャに基づく設計

## 必要条件

- Go 1.21以上
- Bluesky アカウント
- アクセストークン（JWT）

## 環境変数

以下の環境変数を設定する必要があります：

### 必須の環境変数

| 環境変数 | 説明 | 例 |
|----------|------|-----|
| `ACCESS_JWT` | Blueskyのアクセストークン | `ey...` |
| `DID` | BlueskyのDID | `did:plc:...` |

### オプションの環境変数

| 環境変数 | 説明 | デフォルト値 |
|----------|------|------------|
| `PDS_URL` | BlueskyのPDS URL | `https://bsky.social` |
| `QUOTES_FILE` | 名言データのJSONファイル | `quotes.json` |
| `POST_INTERVAL` | 投稿間隔 | `1h` |
| `HTTP_TIMEOUT` | HTTPリクエストのタイムアウト | `10s` |

## 環境変数の設定方法

### Unix/Linux/macOS

```bash
# 必須の環境変数
export ACCESS_JWT="your_access_jwt"
export DID="your_did"

# オプションの環境変数（必要に応じて）
export PDS_URL="https://bsky.social"
export QUOTES_FILE="quotes.json"
export POST_INTERVAL="1h"
export HTTP_TIMEOUT="10s"
```

### Windows (PowerShell)

```powershell
# 必須の環境変数
$env:ACCESS_JWT="your_access_jwt"
$env:DID="your_did"

# オプションの環境変数（必要に応じて）
$env:PDS_URL="https://bsky.social"
$env:QUOTES_FILE="quotes.json"
$env:POST_INTERVAL="1h"
$env:HTTP_TIMEOUT="10s"
```

## プロジェクト構造

```
QuoteBot/
├── config/
│   └── config.go       # アプリケーション設定の管理
├── internal/
│   ├── domain/         # ドメインロジック（エンティティ、値オブジェクト）
│   ├── usecase/        # ユースケース（アプリケーションロジック）
│   └── interface/
│       └── repository/ # データアクセス層（外部APIとの通信）
├── main.go            # アプリケーションのエントリーポイント
└── quotes.json        # 投稿する名言データ
```

## アーキテクチャ設計

本プロジェクトは以下の層で構成されています：

1. **ドメイン層** (`internal/domain/`)
   - ビジネスロジックの中核
   - 外部依存を持たない純粋なビジネスルール

2. **ユースケース層** (`internal/usecase/`)
   - アプリケーションのユースケースを実装
   - ドメイン層のロジックを組み合わせて具体的な機能を実現

3. **インターフェース層** (`internal/interface/`)
   - 外部システム（Bluesky API）とのインタフェース
   - リポジトリパターンを採用し、データアクセスを抽象化

## 設定管理

- 環境変数による設定管理（[envconfig](https://github.com/kelseyhightower/envconfig)使用）
- `.env`ファイルでローカル開発環境の設定を管理
- 必須設定項目の検証機能

## 主要コンポーネント

### Config (`config/config.go`)
- アプリケーション設定の一元管理
- 環境変数からの自動設定読み込み
- デフォルト値の提供

### BlueskyRepository (`internal/interface/repository/bluesky_repository.go`)
- Bluesky APIとの通信を担当
- HTTPリクエストの共通処理
- トークンリフレッシュの自動処理

## エラー処理とロギング

- 構造化されたエラーハンドリング
- エラーの適切なラッピングとコンテキスト付加
- HTTPエラーの専用型による表現

## セットアップ

1. リポジトリをクローン
```bash
git clone https://github.com/yourusername/QuoteBot.git
cd QuoteBot
```

2. 環境変数の設定
```bash
cp .env.example .env
# .envファイルを編集して実際の値を設定
```

必要な環境変数：
- `ACCESS_JWT`: Blueskyのアクセストークン
- `DID`: BlueskyのDID

3. 依存関係のインストール
```bash
go mod tidy
```

## 実行方法

```bash
go run main.go
```

## ライセンス

MIT
