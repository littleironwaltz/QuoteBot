# QuoteBot

Blueskyに定期的に名言を投稿するボット

## 必要条件

- Go 1.21以上
- Blueskyアカウント

## 環境変数

以下の環境変数を設定する必要があります：

### 必須の環境変数

| 環境変数 | 説明 | 例 |
|----------|------|-----|
| `ACCESS_JWT` | Blueskyのアクセストークン | `eyJ0eXAiOi...` |
| `REFRESH_JWT` | Blueskyのリフレッシュトークン | `eyJ0eXAiOi...` |
| `DID` | BlueskyのDID | `did:plc:...` |

### オプションの環境変数

| 環境変数 | 説明 | デフォルト値 |
|----------|------|------------|
| `PDS_URL` | BlueskyのPDS URL | `https://bsky.social` |
| `QUOTES_FILE` | 名言データのJSONファイル | `quotes.json` |
| `POST_INTERVAL` | 投稿間隔（例：30m, 1h, 2h） | `1h` |
| `HTTP_TIMEOUT` | HTTPリクエストのタイムアウト | `10s` |

## 環境変数の設定方法

### Unix/Linux/macOS

```bash
# 必須の環境変数
export ACCESS_JWT="your_access_jwt"
export REFRESH_JWT="your_refresh_jwt"
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
$env:REFRESH_JWT="your_refresh_jwt"
$env:DID="your_did"

# オプションの環境変数（必要に応じて）
$env:PDS_URL="https://bsky.social"
$env:QUOTES_FILE="quotes.json"
$env:POST_INTERVAL="1h"
$env:HTTP_TIMEOUT="10s"
```

## Blueskyトークンの取得方法

1. https://bsky.app にログイン
2. ブラウザの開発者ツール（Chrome/Safariの場合は`F12`または`Command + Option + I`）を開く
3. `Application`（アプリケーション）タブを選択
4. 左側の`Local Storage`から`bsky.social`を選択
5. 以下の値をコピー：
   - `did`: あなたのDID
   - `jwt`: アクセストークン（ACCESS_JWT）
   - `refreshJwt`: リフレッシュトークン（REFRESH_JWT）

## プロジェクト構造

```
.
├── main.go                 # エントリーポイント
├── config/                 # 設定関連
├── internal/               # 内部パッケージ
│   ├── domain/            # ドメインロジック
│   ├── usecase/           # ユースケース
│   └── interface/         # インターフェース
│       └── repository/    # リポジトリ実装
└── quotes.json            # 名言データ
```

## ビルドと実行

```bash
# ビルド
go build -o quotebot

# 実行
./quotebot
```

## 機能

- 設定された間隔（デフォルト1時間）で名言を自動投稿
- アクセストークンの自動更新
- エラー時の自動リトライ
- カスタマイズ可能な投稿間隔
- 日本語名言の投稿

## エラー対応

よくあるエラーと対処方法：

1. `required key XXX missing value`
   - 必須の環境変数が設定されていません
   - 上記の必須環境変数をすべて設定してください

2. `failed to refresh token`
   - トークンの更新に失敗しました
   - ACCESS_JWTとREFRESH_JWTが正しいか確認してください
   - トークンが期限切れの場合は、再度取得してください

3. `failed to post message`
   - 投稿に失敗しました
   - インターネット接続を確認してください
   - トークンが有効か確認してください

## セキュリティ注意事項

- トークン（ACCESS_JWT, REFRESH_JWT）は機密情報です。他人と共有しないでください
- トークンをソースコードやGitリポジトリに保存しないでください
- 定期的にトークンを更新することをお勧めします

## ライセンス

MIT
