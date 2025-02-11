package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config はアプリケーション全体の設定を保持します
type Config struct {
	PDSURL       string        `envconfig:"PDS_URL" default:"https://bsky.social"`
	Collection   string        `envconfig:"COLLECTION" default:"app.bsky.feed.post"`
	QuotesFile   string        `envconfig:"QUOTES_FILE" default:"quotes.json"`
	AccessJWT    string        `envconfig:"ACCESS_JWT" required:"true"`
	RefreshJWT   string        `envconfig:"REFRESH_JWT" required:"true"`
	DID          string        `envconfig:"DID" required:"true"`
	PostInterval time.Duration `envconfig:"POST_INTERVAL" default:"1h"`
	HTTPTimeout  time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
}

// New は新しい設定インスタンスを作成します
// 環境変数から設定を自動的に読み込み、デフォルト値がない必須項目が
// 設定されていない場合はエラーを返します
func New() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("環境変数の処理に失敗: %w", err)
	}
	return &cfg, nil
}
