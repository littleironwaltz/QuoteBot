package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/littleironwaltz/quotebot/config"
	"github.com/littleironwaltz/quotebot/internal/domain"
)

// BlueskyRepository handles posting to Bluesky
type BlueskyRepository struct {
	cfg          *config.Config
	tokenManager *TokenManager
	httpClient   *HTTPClient
	Done         chan struct{} // Exported for cleanup in main
}

// NewBlueskyRepository creates a new BlueskyRepository instance
func NewBlueskyRepository(cfg *config.Config) *BlueskyRepository {
	// Create the HTTP client
	httpClient := NewHTTPClient(cfg)

	// Create the token encryptor
	encryptor := NewTokenEncryptor()

	// Create the token manager
	tokenManager := NewTokenManager(cfg, encryptor, httpClient)

	return &BlueskyRepository{
		cfg:          cfg,
		tokenManager: tokenManager,
		httpClient:   httpClient,
		Done:         make(chan struct{}),
	}
}

// PostMessage posts the specified message to Bluesky
func (r *BlueskyRepository) PostMessage(ctx context.Context, message string) error {
	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.createRecord", r.cfg.PDSURL)

	// Get access token
	accessToken, err := r.tokenManager.GetToken(AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Create request body
	requestBody := map[string]interface{}{
		"repo":       r.cfg.DID,
		"collection": "app.bsky.feed.post",
		"record": map[string]interface{}{
			"$type":     "app.bsky.feed.post",
			"text":      message,
			"createdAt": time.Now().Format(time.RFC3339),
			"facets":    []interface{}{},
		},
	}

	// Set request headers
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json",
	}

	// Send the request
	resp, err := r.httpClient.DoRequest(ctx, "POST", url, requestBody, headers)
	if err != nil {
		// If unauthorized, try to refresh the token and retry
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == 401 {
			if err := r.tokenManager.RefreshToken(ctx); err != nil {
				return fmt.Errorf("failed to refresh token: %w", err)
			}

			// Get new access token
			accessToken, err = r.tokenManager.GetToken(AccessToken)
			if err != nil {
				return fmt.Errorf("failed to get refreshed access token: %w", err)
			}

			// Update header with new token
			headers["Authorization"] = fmt.Sprintf("Bearer %s", accessToken)

			// Retry the request
			resp, err = r.httpClient.DoRequest(ctx, "POST", url, requestBody, headers)
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

// RefreshToken refreshes the access token
func (r *BlueskyRepository) RefreshToken(ctx context.Context) error {
	return r.tokenManager.RefreshToken(ctx)
}

// PostRandomQuote selects a random quote and posts it
func (r *BlueskyRepository) PostRandomQuote(ctx context.Context, quote *domain.Quote) error {
	if quote == nil {
		return fmt.Errorf("quote cannot be nil")
	}

	formattedMessage := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
	return r.PostMessage(ctx, formattedMessage)
}

// Shutdown cleans up resources
func (r *BlueskyRepository) Shutdown() {
	// Shut down token manager
	r.tokenManager.Shutdown()
	// Signal that we're done
	close(r.Done)
}
