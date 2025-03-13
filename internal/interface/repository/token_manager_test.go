package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/littleironwaltz/quotebot/config"
)

func TestTokenManager_GetToken(t *testing.T) {
	// 初期トークン
	initialAccessToken := "initial-access-token"
	initialRefreshToken := "initial-refresh-token"

	tests := []struct {
		name      string
		tokenType TokenType
		wantToken string
		wantErr   bool
	}{
		{
			name:      "正常系: アクセストークンを取得",
			tokenType: AccessToken,
			wantToken: initialAccessToken,
			wantErr:   false,
		},
		{
			name:      "正常系: リフレッシュトークンを取得",
			tokenType: RefreshToken,
			wantToken: initialRefreshToken,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 設定の作成
			cfg := &config.Config{
				AccessJWT:            initialAccessToken,
				RefreshJWT:           initialRefreshToken,
				TokenRefreshInterval: 1 * time.Hour,
				HTTPTimeout:          3 * time.Second,
			}

			// 実際のコンポーネントの作成
			encryptor := NewTokenEncryptor()
			httpClient := NewHTTPClient(cfg)
			tm := NewTokenManager(cfg, encryptor, httpClient)

			// トークンの取得
			got, err := tm.GetToken(tt.tokenType)

			// エラーのチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("GetToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// トークンの値のチェック
			if err == nil && got != tt.wantToken {
				t.Errorf("GetToken() = %v, want %v", got, tt.wantToken)
			}

			// リソースのクリーンアップ
			tm.Shutdown()
		})
	}
}

func TestTokenManager_RefreshToken(t *testing.T) {
	// テストサーバーの設定
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

		// リフレッシュトークンの検証（Authorizationヘッダーから）
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 成功レスポンス
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"accessJwt": "new-access-token",
			"refreshJwt": "new-refresh-token"
		}`))
	}))
	defer server.Close()

	tests := []struct {
		name        string
		setupConfig func() *config.Config
		wantErr     bool
	}{
		{
			name: "正常系: トークンの更新成功",
			setupConfig: func() *config.Config {
				return &config.Config{
					AccessJWT:            "old-access-token",
					RefreshJWT:           "refresh-token",
					PDSURL:               server.URL,
					TokenRefreshInterval: 1 * time.Hour,
					HTTPTimeout:          3 * time.Second,
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: 無効なPDSURL",
			setupConfig: func() *config.Config {
				return &config.Config{
					AccessJWT:            "old-access-token",
					RefreshJWT:           "refresh-token",
					PDSURL:               "http://invalid-url",
					TokenRefreshInterval: 1 * time.Hour,
					HTTPTimeout:          3 * time.Second,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()

			// 実際のコンポーネントの作成
			encryptor := NewTokenEncryptor()
			httpClient := NewHTTPClient(cfg)
			tm := NewTokenManager(cfg, encryptor, httpClient)

			// トークンの更新
			ctx := context.Background()
			err := tm.RefreshToken(ctx)

			// エラーのチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 成功した場合、トークンが更新されているか確認
			if err == nil {
				// キャッシュからアクセストークンを取得
				newAccessToken, err := tm.GetToken(AccessToken)
				if err != nil {
					t.Errorf("GetToken(AccessToken) after refresh error = %v", err)
					return
				}

				// 新しいトークンになっているか確認
				if newAccessToken != "new-access-token" {
					t.Errorf("After RefreshToken(), access token = %v, want %v", newAccessToken, "new-access-token")
				}

				// キャッシュからリフレッシュトークンを取得
				newRefreshToken, err := tm.GetToken(RefreshToken)
				if err != nil {
					t.Errorf("GetToken(RefreshToken) after refresh error = %v", err)
					return
				}

				// 新しいトークンになっているか確認
				if newRefreshToken != "new-refresh-token" {
					t.Errorf("After RefreshToken(), refresh token = %v, want %v", newRefreshToken, "new-refresh-token")
				}
			}

			// リソースのクリーンアップ
			tm.Shutdown()
		})
	}
}

func TestTokenManager_BackgroundRefresh(t *testing.T) {
	// カウンター用の変数とミューテックス
	var refreshCallCount int
	var counterMutex sync.Mutex

	// テストサーバーの設定
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xrpc/com.atproto.server.refreshSession" {
			// カウンターの増加をミューテックスで保護
			counterMutex.Lock()
			refreshCallCount++
			count := refreshCallCount // ローカル変数にコピーして安全にアクセス
			counterMutex.Unlock()

			fmt.Printf("Token refresh called %d times\n", count)

			// リフレッシュトークンの検証（Authorizationヘッダーから）
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// 成功レスポンス
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"accessJwt": "new-access-token",
				"refreshJwt": "new-refresh-token"
			}`))
		}
	}))
	defer server.Close()

	// 設定の作成
	cfg := &config.Config{
		AccessJWT:            "access-token",
		RefreshJWT:           "refresh-token",
		PDSURL:               server.URL,
		TokenRefreshInterval: 100 * time.Millisecond, // 短い間隔でテスト
		HTTPTimeout:          3 * time.Second,
	}

	// TokenManagerの作成
	encryptor := NewTokenEncryptor()
	httpClient := NewHTTPClient(cfg)
	tm := NewTokenManager(cfg, encryptor, httpClient)

	// しばらく待機してバックグラウンド更新が何回か実行されるのを確認
	time.Sleep(350 * time.Millisecond)

	// TokenManagerのシャットダウン
	tm.Shutdown()

	// カウンターの取得をミューテックスで保護
	counterMutex.Lock()
	count := refreshCallCount
	counterMutex.Unlock()

	// 初期化時に1回 + バックグラウンドで3回程度（タイミングによって2〜4回）のリフレッシュが想定される
	if count < 3 {
		t.Errorf("Expected at least 3 refresh calls (including the initial one), but got %d", count)
	}
}
