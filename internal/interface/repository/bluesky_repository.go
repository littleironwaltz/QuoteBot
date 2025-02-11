package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kojikubota/quotebot/config"
)

// BlueskyRepository はBlueskyへの投稿を担当します
type BlueskyRepository struct {
	cfg    *config.Config
	client *http.Client
}

// NewBlueskyRepository は新しいBlueskyRepositoryインスタンスを作成します
func NewBlueskyRepository(cfg *config.Config) *BlueskyRepository {
	return &BlueskyRepository{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
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

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", r.cfg.AccessJWT),
		"Content-Type":  "application/json",
	}

	resp, err := doHTTPRequest(ctx, r.client, "POST", url, bytes.NewBuffer(bodyBytes), headers)
	if err != nil {
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == http.StatusUnauthorized {
			// トークンの更新を試みる
			if err := r.RefreshToken(ctx); err != nil {
				return fmt.Errorf("failed to refresh token: %w", err)
			}

			// 新しいトークンで再試行
			headers["Authorization"] = fmt.Sprintf("Bearer %s", r.cfg.AccessJWT)
			resp, err = doHTTPRequest(ctx, r.client, "POST", url, bytes.NewBuffer(bodyBytes), headers)
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
