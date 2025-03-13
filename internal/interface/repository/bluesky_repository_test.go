package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/littleironwaltz/quotebot/config"
)

func TestBlueskyRepository_PostMessage(t *testing.T) {
	// テストサーバーの設定
	var refreshCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.repo.createRecord":
			if r.Header.Get("Authorization") == "Bearer invalid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid token",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"uri": "at://did:plc:test/app.bsky.feed.post/test",
			})
		case "/xrpc/com.atproto.server.refreshSession":
			refreshCount++
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"accessJwt":  "new-valid-token",
				"refreshJwt": "new-refresh-token",
			})
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		cfg     *config.Config
		message string
		wantErr bool
	}{
		{
			name: "正常系: 初回投稿成功",
			cfg: &config.Config{
				AccessJWT:            "valid-token",
				RefreshJWT:           "refresh-token",
				DID:                  "did:plc:test",
				PDSURL:               server.URL,
				HTTPTimeout:          3 * time.Second,
				TokenRefreshInterval: 1 * time.Hour,
				MaxRetries:           3,
				RetryBackoff:         5 * time.Second,
			},
			message: "テストメッセージ",
			wantErr: false,
		},
		{
			name: "エラー後の回復: 認証エラー後にトークンを更新して成功",
			cfg: &config.Config{
				AccessJWT:            "invalid-token",
				RefreshJWT:           "refresh-token",
				DID:                  "did:plc:test",
				PDSURL:               server.URL,
				HTTPTimeout:          3 * time.Second,
				TokenRefreshInterval: 1 * time.Hour,
				MaxRetries:           3,
				RetryBackoff:         5 * time.Second,
			},
			message: "テストメッセージ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refreshCount = 0
			repo := NewBlueskyRepository(tt.cfg)
			ctx := context.Background()

			// 初期化時に最低1回トークンリフレッシュが呼ばれる
			if refreshCount < 1 {
				t.Errorf("初期化時のトークンリフレッシュが実行されませんでした。実行回数: %d", refreshCount)
			}

			// 投稿前に明示的なリフレッシュを行う（main.goの動作に合わせる）
			beforeRefreshCount := refreshCount
			err := repo.RefreshToken(ctx)
			if err != nil {
				t.Errorf("明示的なトークンリフレッシュに失敗しました: %v", err)
			}

			// リフレッシュ回数が増えていることを確認
			if refreshCount <= beforeRefreshCount {
				t.Errorf("トークンリフレッシュが実行されていません。実行前: %d, 実行後: %d", beforeRefreshCount, refreshCount)
			}

			err = repo.PostMessage(ctx, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueskyRepository.PostMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			repo.Shutdown()
		})
	}
}

func TestBlueskyRepository_RefreshToken(t *testing.T) {
	// テストサーバーの設定
	var refreshCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.refreshSession" {
			t.Errorf("予期しないパス: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("予期しないメソッド: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		refreshCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"accessJwt":  "new-valid-token",
			"refreshJwt": "new-refresh-token",
		})
	}))
	defer server.Close()

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "正常系: トークン更新成功",
			cfg: &config.Config{
				AccessJWT:            "old-token",
				RefreshJWT:           "old-refresh-token",
				DID:                  "did:plc:test",
				PDSURL:               server.URL,
				HTTPTimeout:          3 * time.Second,
				TokenRefreshInterval: 1 * time.Hour,
				MaxRetries:           3,
				RetryBackoff:         5 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refreshCount = 0
			repo := NewBlueskyRepository(tt.cfg)
			ctx := context.Background()

			// 初期化時に最低1回トークンリフレッシュが呼ばれる
			initialRefreshCount := refreshCount
			if initialRefreshCount < 1 {
				t.Errorf("初期化時のトークンリフレッシュが実行されませんでした。実行回数: %d", initialRefreshCount)
			}

			// 明示的なトークンリフレッシュ
			beforeRefreshCount := refreshCount
			err := repo.RefreshToken(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueskyRepository.RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			// リフレッシュ回数が増えていることを確認
			if refreshCount <= beforeRefreshCount {
				t.Errorf("トークンリフレッシュが実行されていません。実行前: %d, 実行後: %d", beforeRefreshCount, refreshCount)
			}

			repo.Shutdown()
		})
	}
}
