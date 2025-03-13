package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/littleironwaltz/quotebot/config"
)

func TestHTTPClient_NewHTTPClient(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "正常系: デフォルト値で作成",
			cfg: &config.Config{
				HTTPTimeout:  10 * time.Second,
				MaxRetries:   3,
				RetryBackoff: 5 * time.Second,
			},
		},
		{
			name: "正常系: カスタム値で作成",
			cfg: &config.Config{
				HTTPTimeout:  5 * time.Second,
				MaxRetries:   5,
				RetryBackoff: 2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHTTPClient(tt.cfg)
			if client == nil {
				t.Errorf("NewHTTPClient() = nil, want non-nil")
				return
			}
			if client.client.Timeout != tt.cfg.HTTPTimeout {
				t.Errorf("client.Timeout = %v, want %v", client.client.Timeout, tt.cfg.HTTPTimeout)
			}
			if client.retryPolicy.MaxRetries != tt.cfg.MaxRetries {
				t.Errorf("retryPolicy.MaxRetries = %v, want %v", client.retryPolicy.MaxRetries, tt.cfg.MaxRetries)
			}
			if client.retryPolicy.RetryBackoff != tt.cfg.RetryBackoff {
				t.Errorf("retryPolicy.RetryBackoff = %v, want %v", client.retryPolicy.RetryBackoff, tt.cfg.RetryBackoff)
			}
		})
	}
}

func TestHTTPClient_DoRequest(t *testing.T) {
	// テストサーバーのセットアップ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
		case "/client-error":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "client error"})
		case "/rate-limit":
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
		case "/slow":
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "slow response"})
		case "/check-body":
			var reqBody map[string]string
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if reqBody["test"] != "value" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/check-headers":
			if r.Header.Get("X-Test-Header") != "test-value" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	tests := []struct {
		name         string
		method       string
		url          string
		body         interface{}
		headers      map[string]string
		retryPolicy  RetryPolicy
		timeout      time.Duration
		wantErr      bool
		wantErrExact error
		wantStatus   int
	}{
		{
			name:       "正常系: 成功レスポンス",
			method:     "GET",
			url:        server.URL + "/success",
			body:       nil,
			headers:    nil,
			wantErr:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:    "異常系: サーバーエラー（再試行）",
			method:  "GET",
			url:     server.URL + "/server-error",
			body:    nil,
			headers: nil,
			retryPolicy: RetryPolicy{
				MaxRetries:   2,
				RetryBackoff: 10 * time.Millisecond,
			},
			wantErr:    true,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "異常系: クライアントエラー（再試行なし）",
			method:     "GET",
			url:        server.URL + "/client-error",
			body:       nil,
			headers:    nil,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "正常系: リクエストボディを送信",
			method: "POST",
			url:    server.URL + "/check-body",
			body:   map[string]string{"test": "value"},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantErr:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:   "正常系: ヘッダーを送信",
			method: "GET",
			url:    server.URL + "/check-headers",
			body:   nil,
			headers: map[string]string{
				"X-Test-Header": "test-value",
			},
			wantErr:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:    "異常系: タイムアウト",
			method:  "GET",
			url:     server.URL + "/slow",
			body:    nil,
			headers: nil,
			timeout: 50 * time.Millisecond,
			wantErr: true,
		},
		{
			name:    "異常系: 無効なURL",
			method:  "GET",
			url:     "http://invalid-url-that-does-not-exist",
			body:    nil,
			headers: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// タイムアウトのデフォルト値
			timeout := 1 * time.Second
			if tt.timeout > 0 {
				timeout = tt.timeout
			}

			// 設定の作成
			cfg := &config.Config{
				HTTPTimeout:  timeout,
				MaxRetries:   DefaultMaxRetries,
				RetryBackoff: 50 * time.Millisecond,
			}

			// HTTPクライアントの作成
			client := NewHTTPClient(cfg)

			// リトライポリシーのカスタマイズ
			if tt.retryPolicy.MaxRetries > 0 {
				client.retryPolicy = tt.retryPolicy
			}

			// コンテキストの作成
			ctx := context.Background()

			// リクエストの実行
			resp, err := client.DoRequest(ctx, tt.method, tt.url, tt.body, tt.headers)

			// エラーのチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("DoRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 期待されるステータスコードのチェック（エラーがない場合）
			if err == nil && resp.StatusCode != tt.wantStatus {
				t.Errorf("DoRequest() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestHTTPClient_DecodeJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		target   interface{}
		wantErr  bool
	}{
		{
			name:     "正常系: 有効なJSONを正常にデコード",
			jsonBody: `{"key": "value", "number": 123}`,
			target:   &map[string]interface{}{},
			wantErr:  false,
		},
		{
			name:     "異常系: 無効なJSONをデコード",
			jsonBody: `{"key": "value", invalid json}`,
			target:   &map[string]interface{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTPレスポンスのモックを作成
			resp := &http.Response{
				Body: io.NopCloser(strings.NewReader(tt.jsonBody)),
			}

			// クライアントの作成
			cfg := &config.Config{
				HTTPTimeout:  1 * time.Second,
				MaxRetries:   3,
				RetryBackoff: 10 * time.Millisecond,
			}
			client := NewHTTPClient(cfg)

			// JSONのデコード
			err := client.DecodeJSONResponse(resp, tt.target)

			// エラーのチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJSONResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 正常系の場合はデコードされた結果を確認
			if !tt.wantErr {
				result := tt.target.(*map[string]interface{})
				if (*result)["key"] != "value" || (*result)["number"] != float64(123) {
					t.Errorf("DecodeJSONResponse() result = %v, want map with key=value and number=123", *result)
				}
			}
		})
	}
}

func TestHTTPClient_CalculateBackoff(t *testing.T) {
	tests := []struct {
		name        string
		retryPolicy RetryPolicy
		attempt     int
		want        time.Duration
		wantMax     bool
	}{
		{
			name: "正常系: 初回リトライ",
			retryPolicy: RetryPolicy{
				RetryBackoff: 100 * time.Millisecond,
			},
			attempt: 1,
			want:    100 * time.Millisecond,
			wantMax: false,
		},
		{
			name: "正常系: 2回目リトライ（指数バックオフ）",
			retryPolicy: RetryPolicy{
				RetryBackoff: 100 * time.Millisecond,
			},
			attempt: 2,
			want:    200 * time.Millisecond,
			wantMax: false,
		},
		{
			name: "正常系: 3回目リトライ（指数バックオフ）",
			retryPolicy: RetryPolicy{
				RetryBackoff: 100 * time.Millisecond,
			},
			attempt: 3,
			want:    400 * time.Millisecond,
			wantMax: false,
		},
		{
			name: "正常系: 最大バックオフを超える",
			retryPolicy: RetryPolicy{
				RetryBackoff: 20 * time.Second,
			},
			attempt: 2,
			want:    MaxBackoffDuration,
			wantMax: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クライアントの作成
			cfg := &config.Config{HTTPTimeout: 1 * time.Second}
			client := NewHTTPClient(cfg)
			client.retryPolicy = tt.retryPolicy

			// バックオフの計算
			got := client.calculateBackoff(tt.attempt)

			// 最大値を超える場合は最大値を確認
			if tt.wantMax {
				if got != MaxBackoffDuration {
					t.Errorf("calculateBackoff() = %v, want maximum of %v", got, MaxBackoffDuration)
				}
				return
			}

			// 期待される値との比較
			if got != tt.want {
				t.Errorf("calculateBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPClient_ShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		attempt    int
		maxRetries int
		want       bool
	}{
		{
			name:       "正常系: 最大試行回数未満のサーバーエラー",
			err:        &HTTPError{StatusCode: 500, Message: "Internal Server Error"},
			attempt:    0,
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "正常系: レート制限エラー",
			err:        &HTTPError{StatusCode: 429, Message: "Too Many Requests"},
			attempt:    1,
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "異常系: クライアントエラー（再試行なし）",
			err:        &HTTPError{StatusCode: 400, Message: "Bad Request"},
			attempt:    0,
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "異常系: 認証エラー（再試行なし）",
			err:        &HTTPError{StatusCode: 401, Message: "Unauthorized"},
			attempt:    0,
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "異常系: 最大試行回数に到達",
			err:        &HTTPError{StatusCode: 500, Message: "Internal Server Error"},
			attempt:    3,
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "正常系: ネットワークエラー（再試行する）",
			err:        errors.New("network error"),
			attempt:    1,
			maxRetries: 3,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クライアントの作成
			cfg := &config.Config{HTTPTimeout: 1 * time.Second}
			client := NewHTTPClient(cfg)
			client.retryPolicy.MaxRetries = tt.maxRetries

			// 再試行判定
			got := client.shouldRetry(tt.err, tt.attempt)

			// 期待される結果と比較
			if got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPError_Error(t *testing.T) {
	tests := []struct {
		name      string
		httpError HTTPError
		want      string
	}{
		{
			name: "正常系: 基本的なHTTPエラー",
			httpError: HTTPError{
				StatusCode: 500,
				Message:    "Internal Server Error",
				Err:        errors.New("original error"),
			},
			want: "HTTP error (status 500): Internal Server Error: original error",
		},
		{
			name: "正常系: 認証エラー",
			httpError: HTTPError{
				StatusCode: 401,
				Message:    "Unauthorized",
				Err:        nil,
			},
			want: "HTTP error (status 401): Unauthorized: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.httpError.Error()
			if got != tt.want {
				t.Errorf("HTTPError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
