# QuoteBot

Blueskyに定期的に名言を投稿するボット

## 主な機能

- 設定された間隔で自動的に名言を投稿（デフォルト：1時間）
- アクセストークンの自動更新（初期化時、投稿前、バックグラウンドで定期的に）
- エラー時の自動再試行
- カスタマイズ可能な投稿間隔
- HTTPリクエストの再試行とエクスポネンシャルバックオフ
- トークンの安全な暗号化

## 必要要件

- Go 1.21以上
- Blueskyアカウント

## 環境変数

以下の環境変数を設定する必要があります：

### 必須環境変数

| 環境変数 | 説明 | 例 |
|----------|------|-----|
| `ACCESS_JWT` | Blueskyアクセストークン | `eyJ0eXAiOi...` |
| `REFRESH_JWT` | Blueskyリフレッシュトークン | `eyJ0eXAiOi...` |
| `DID` | Bluesky DID | `did:plc:...` |

### オプション環境変数

| 環境変数 | 説明 | デフォルト値 |
|----------|------|------------|
| `PDS_URL` | Bluesky PDS URL | `https://bsky.social` |
| `COLLECTION` | Blueskyのコレクション名 | `app.bsky.feed.post` |
| `QUOTES_FILE` | 名言データのJSONファイル | `quotes.json` |
| `POST_INTERVAL` | 投稿間隔（例：30m, 1h, 2h） | `1h` |
| `HTTP_TIMEOUT` | HTTPリクエストタイムアウト | `10s` |
| `TOKEN_REFRESH_INTERVAL` | バックグラウンドでのトークンリフレッシュ間隔 | `45m` |
| `MAX_RETRIES` | 失敗時の最大再試行回数 | `3` |
| `RETRY_BACKOFF` | 再試行間の基本待機時間 | `5s` |

## 環境変数の設定方法

### Unix/Linux/macOS

```bash
# 必須環境変数
export ACCESS_JWT="your_access_jwt"
export REFRESH_JWT="your_refresh_jwt"
export DID="your_did"

# オプション環境変数（必要に応じて）
export PDS_URL="https://bsky.social"
export QUOTES_FILE="quotes.json"
export POST_INTERVAL="1h"
export HTTP_TIMEOUT="10s"
export TOKEN_REFRESH_INTERVAL="45m"
export MAX_RETRIES="3"
export RETRY_BACKOFF="5s"
```

### Windows (PowerShell)

```powershell
# 必須環境変数
$env:ACCESS_JWT="your_access_jwt"
$env:REFRESH_JWT="your_refresh_jwt"
$env:DID="your_did"

# オプション環境変数（必要に応じて）
$env:PDS_URL="https://bsky.social"
$env:QUOTES_FILE="quotes.json"
$env:POST_INTERVAL="1h"
$env:HTTP_TIMEOUT="10s"
$env:TOKEN_REFRESH_INTERVAL="45m"
$env:MAX_RETRIES="3"
$env:RETRY_BACKOFF="5s"
```

### .env ファイルの使用

環境変数を `.env` ファイルに保存して使用することもできます（サードパーティの環境変数ローダーを使用する場合）：

```
ACCESS_JWT=your_access_jwt
REFRESH_JWT=your_refresh_jwt
DID=your_did
PDS_URL=https://bsky.social
QUOTES_FILE=quotes.json
POST_INTERVAL=1h
HTTP_TIMEOUT=10s
TOKEN_REFRESH_INTERVAL=45m
MAX_RETRIES=3
RETRY_BACKOFF=5s
```

## Blueskyトークンの取得方法

1. https://bsky.app にログインする
2. 開発者ツールを開く（Chromeまたは他のブラウザで `F12` キー、Safariでは `Command + Option + I` を押す）
3. `Application` タブを選択
4. 左側の `Local Storage` から `bsky.social` を選択
5. 以下の値をコピーする：
   - `did`: あなたのDID
   - `jwt`: アクセストークン（ACCESS_JWT）
   - `refreshJwt`: リフレッシュトークン（REFRESH_JWT）

## プロジェクト構造

```
.
├── main.go                  # エントリーポイント
├── config/                  # 設定
│   └── config.go           # 環境変数からの設定読み込み
├── internal/                # 内部パッケージ
│   ├── domain/             # ドメインロジック
│   │   └── quote.go       # 名言のエンティティ
│   ├── usecase/            # ユースケース
│   │   └── quote_usecase.go # 名言投稿のユースケース
│   └── interface/          # インターフェース
│       └── repository/     # リポジトリ実装
│           ├── bluesky_repository.go # Bluesky API操作
│           ├── quote_repository.go   # 名言の管理
│           ├── http_client.go        # HTTPクライアント
│           ├── token_manager.go      # トークン管理
│           └── token_encryptor.go    # トークン暗号化
├── internal/tests/          # テスト
│   └── integration/        # 統合テスト
├── quotes.json              # 名言データ
└── .env.sample              # 環境変数のサンプル
```

## トークンリフレッシュの仕組み

このアプリケーションでは、以下の3つのタイミングでトークンリフレッシュが行われます：

1. **初期化時**: アプリケーションの起動時に自動的にトークンリフレッシュを試みます
2. **投稿前**: 名言投稿の直前に毎回トークンリフレッシュを行います
3. **バックグラウンド**: 設定された間隔（デフォルト45分）で定期的にトークンリフレッシュを行います

これにより、トークン期限切れによるエラーを防止し、安定した運用が可能になります。

## ビルドと実行

```bash
# ビルド
go build -o quotebot

# 実行
./quotebot
```

## テスト

### 単体テスト

```bash
# すべてのテストを実行
go test ./...

# 詳細出力でテスト実行
go test ./... -v

# レース検出を有効にしてテスト実行
go test ./... -race

# テストカバレッジの確認
go test ./... -cover

# テストカバレッジレポートの生成
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
```

### 統合テスト

```bash
# 統合テストのみ実行
go test ./internal/tests/integration -v
```

## トラブルシューティング

よくあるエラーと解決策：

1. `required key XXX missing value`
   - 必須環境変数が設定されていません
   - 上記のすべての必須環境変数を設定してください

2. `failed to refresh token` または `Expired[REDACTED]`
   - トークンの更新に失敗しました
   - ACCESS_JWTとREFRESH_JWTが正しいか確認してください
   - トークンが期限切れの場合は、Blueskyに再ログインして新しいトークンを取得してください
   - プログラムが数日間実行されていなかった場合、リフレッシュトークンも期限切れになっている可能性があります

3. `failed to post message`
   - 投稿に失敗しました
   - インターネット接続を確認してください
   - トークンが有効であることを確認してください

4. プログラムを長期間停止後に再開する場合
   - 再度Blueskyからトークンを取得し、環境変数を更新してから実行してください
   - リフレッシュトークンの有効期限は通常1〜2週間程度です

## 運用のベストプラクティス

1. **定期的な監視**: ログを定期的に確認し、エラーが発生していないか監視してください
2. **長期停止への対策**: 数週間以上停止する予定がある場合は、再開時に新しいトークンを取得してください
3. **cron設定**: リフレッシュトークンの期限切れを防ぐため、少なくとも週に1回は実行されるようにスケジュールすることを検討してください

## セキュリティに関する注意

- トークン（ACCESS_JWT、REFRESH_JWT）は機密情報です。他人と共有しないでください
- トークンをソースコードやGitリポジトリに保存しないでください
- `.gitignore` ファイルを使用して、機密情報を含むファイルをリポジトリから除外してください
- デプロイ時は環境変数やシークレット管理サービスを使用してトークンを安全に管理してください

## ライセンス

MIT
