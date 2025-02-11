package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kojikubota/quotebot/config"
)

// BlueskyRepository はBlueskyへの投稿を担当します
type BlueskyRepository struct {
	cfg         *config.Config
	client      *http.Client
	bufferPool  *sync.Pool
	refreshTick *time.Ticker
	Done        chan struct{}  // Exported for cleanup in main
}

// NewBlueskyRepository は新しいBlueskyRepositoryインスタンスを作成します
func NewBlueskyRepository(cfg *config.Config) *BlueskyRepository {
	transport := &http.Transport{
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}
	repo := &BlueskyRepository{
		cfg: cfg,
		client: &http.Client{
			Timeout:   cfg.HTTPTimeout,
			Transport: transport,
		},
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		Done: make(chan struct{}),
	}
	
	// Start background token refresh (every 45 minutes)
	repo.refreshTick = time.NewTicker(45 * time.Minute)
	go repo.backgroundTokenRefresh()
	
	return repo
}

// backgroundTokenRefresh は定期的にトークンを更新するバックグラウンドプロセスを実行します
func (r *BlueskyRepository) backgroundTokenRefresh() {
	for {
		select {
		case <-r.refreshTick.C:
			ctx, cancel := context.WithTimeout(context.Background(), r.cfg.HTTPTimeout)
			if err := r.RefreshToken(ctx); err != nil {
				log.Printf("バックグラウンドトークン更新に失敗: %v", err)
			}
			cancel()
		case <-r.Done:
			r.refreshTick.Stop()
			return
		}
	}
}


// RefreshToken はリフレッシュトークンを使用して新しいアクセストークンを取得します
func (r *BlueskyRepository) RefreshToken(ctx context.Context) error {
	url := fmt.Sprintf("%s/xrpc/com.atproto.server.refreshSession", r.cfg.PDSURL)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", r.cfg.RefreshJWT),
		"Content-Type":  "application/json",
	}

	resp, err := doHTTPRequest(ctx, r.client, "POST", url, nil, headers)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	var refreshResp struct {
		AccessJWT  string `json:"accessJwt"`
		RefreshJWT string `json:"refreshJwt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	r.cfg.AccessJWT = refreshResp.AccessJWT
	r.cfg.RefreshJWT = refreshResp.RefreshJWT

	return nil
}

// PostMessage は指定されたメッセージをBlueskyに投稿します
func (r *BlueskyRepository) PostMessage(ctx context.Context, message string) error {
	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.createRecord", r.cfg.PDSURL)

	// リクエストボディの作成
	record := map[string]interface{}{
		"$type":     "app.bsky.feed.post",
		"text":      message,
		"createdAt": time.Now().Format(time.RFC3339),
		"facets":    []interface{}{},
	}

	body := map[string]interface{}{
		"repo":       r.cfg.DID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	// Get buffer from pool
	buf := r.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer r.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return fmt.Errorf("failed to encode request body: %w", err)
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", r.cfg.AccessJWT),
		"Content-Type":  "application/json",
	}

	resp, err := doHTTPRequest(ctx, r.client, "POST", url, buf, headers)
	if err != nil {
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == http.StatusUnauthorized {
			// トークンの更新を試みる
			if err := r.RefreshToken(ctx); err != nil {
				return fmt.Errorf("failed to refresh token: %w", err)
			}

			// 新しいトークンで再試行
			headers["Authorization"] = fmt.Sprintf("Bearer %s", r.cfg.AccessJWT)
			// Reset buffer position for reuse
			buf.Reset()
			if err := json.NewEncoder(buf).Encode(body); err != nil {
				return fmt.Errorf("failed to encode request body for retry: %w", err)
			}
			resp, err = doHTTPRequest(ctx, r.client, "POST", url, buf, headers)
			if err != nil {
				return fmt.Errorf("failed to post message after token refresh: %w", err)
			}
		} else {
			return fmt.Errorf("failed to post message: %w", err)
		}
	}
	defer resp.Body.Close()

	return nil
}

// HTTPError はHTTPリクエスト時のエラー情報を保持します
type HTTPError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error (status %d): %s: %v", e.StatusCode, e.Message, e.Err)
}

func doHTTPRequest(ctx context.Context, client *http.Client, method string, url string, body *bytes.Buffer, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, &HTTPError{StatusCode: resp.StatusCode, Message: resp.Status, Err: err}
	}

	return resp, nil
}
