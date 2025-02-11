package config

import (
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "正常系: 必須環境変数あり",
			envVars: map[string]string{
				"ACCESS_JWT":  "test-access-token",
				"REFRESH_JWT": "test-refresh-token",
				"DID":        "test-did",
			},
			want: &Config{
				PDSURL:       "https://bsky.social",
				Collection:   "app.bsky.feed.post",
				QuotesFile:   "quotes.json",
				AccessJWT:    "test-access-token",
				RefreshJWT:   "test-refresh-token",
				DID:         "test-did",
				PostInterval: time.Hour,
				HTTPTimeout:  10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "正常系: カスタム値指定",
			envVars: map[string]string{
				"ACCESS_JWT":    "test-access-token",
				"REFRESH_JWT":   "test-refresh-token",
				"DID":          "test-did",
				"PDS_URL":      "https://custom.social",
				"POST_INTERVAL": "30m",
				"HTTP_TIMEOUT":  "5s",
			},
			want: &Config{
				PDSURL:       "https://custom.social",
				Collection:   "app.bsky.feed.post",
				QuotesFile:   "quotes.json",
				AccessJWT:    "test-access-token",
				RefreshJWT:   "test-refresh-token",
				DID:         "test-did",
				PostInterval: 30 * time.Minute,
				HTTPTimeout:  5 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "異常系: 必須環境変数なし",
			envVars: map[string]string{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: 無効な時間形式",
			envVars: map[string]string{
				"ACCESS_JWT":    "test-access-token",
				"REFRESH_JWT":   "test-refresh-token",
				"DID":          "test-did",
				"POST_INTERVAL": "invalid",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			os.Clearenv()

			// テスト用の環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 設定値の検証
			if got.PDSURL != tt.want.PDSURL {
				t.Errorf("PDSURL = %v, want %v", got.PDSURL, tt.want.PDSURL)
			}
			if got.Collection != tt.want.Collection {
				t.Errorf("Collection = %v, want %v", got.Collection, tt.want.Collection)
			}
			if got.QuotesFile != tt.want.QuotesFile {
				t.Errorf("QuotesFile = %v, want %v", got.QuotesFile, tt.want.QuotesFile)
			}
			if got.AccessJWT != tt.want.AccessJWT {
				t.Errorf("AccessJWT = %v, want %v", got.AccessJWT, tt.want.AccessJWT)
			}
			if got.RefreshJWT != tt.want.RefreshJWT {
				t.Errorf("RefreshJWT = %v, want %v", got.RefreshJWT, tt.want.RefreshJWT)
			}
			if got.DID != tt.want.DID {
				t.Errorf("DID = %v, want %v", got.DID, tt.want.DID)
			}
			if got.PostInterval != tt.want.PostInterval {
				t.Errorf("PostInterval = %v, want %v", got.PostInterval, tt.want.PostInterval)
			}
			if got.HTTPTimeout != tt.want.HTTPTimeout {
				t.Errorf("HTTPTimeout = %v, want %v", got.HTTPTimeout, tt.want.HTTPTimeout)
			}
		})
	}
}
