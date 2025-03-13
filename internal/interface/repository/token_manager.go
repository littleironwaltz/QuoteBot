package repository

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/littleironwaltz/quotebot/config"
)

// TokenType defines the type of token
type TokenType string

const (
	// AccessToken is the token used for API access
	AccessToken TokenType = "access"
	// RefreshToken is the token used for refreshing the access token
	RefreshToken TokenType = "refresh"
)

// TokenManager handles token management
type TokenManager struct {
	cfg                  *config.Config
	encryptor            *TokenEncryptor
	httpClient           *HTTPClient
	cachedAccessToken    string
	cachedRefreshToken   string
	encryptedTokensMutex sync.RWMutex // Protects encrypted token storage in config
	cachedTokensMutex    sync.RWMutex // Protects decrypted token cache
	refreshTick          *time.Ticker
	Done                 chan struct{}
}

// NewTokenManager creates a new TokenManager instance
func NewTokenManager(cfg *config.Config, encryptor *TokenEncryptor, httpClient *HTTPClient) *TokenManager {
	tm := &TokenManager{
		cfg:        cfg,
		encryptor:  encryptor,
		httpClient: httpClient,
		Done:       make(chan struct{}),
	}

	// Encrypt initial tokens if they're not already encrypted
	if err := tm.encryptTokensIfNeeded(); err != nil {
		log.Printf("Warning: could not encrypt tokens: %v", err)
	}

	// 初期化時に明示的にトークンリフレッシュを試みる
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout)
	defer cancel()

	log.Println("TokenManager初期化時にトークンリフレッシュを試みます...")
	if err := tm.RefreshToken(ctx); err != nil {
		log.Printf("初期トークンリフレッシュに失敗しましたが、処理を続行します: %v", err)
	} else {
		log.Println("初期トークンリフレッシュに成功しました")
	}

	// Start background token refresh
	tm.refreshTick = time.NewTicker(cfg.TokenRefreshInterval)
	log.Printf("バックグラウンドトークンリフレッシュを開始します（間隔: %v）", cfg.TokenRefreshInterval)
	go tm.backgroundTokenRefresh()

	return tm
}

// encryptTokensIfNeeded encrypts the access and refresh tokens if they are not already encrypted
func (tm *TokenManager) encryptTokensIfNeeded() error {
	// Make a copy of the original tokens
	accessJWT := tm.cfg.AccessJWT
	refreshJWT := tm.cfg.RefreshJWT

	// Check if they are already encrypted
	if tm.encryptor.IsEncrypted(accessJWT) {
		// Already looks like an encrypted token
		return nil
	}

	// Encrypt tokens
	encryptedAccessJWT, err := tm.encryptor.Encrypt(accessJWT)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshJWT, err := tm.encryptor.Encrypt(refreshJWT)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	// Store encrypted tokens
	tm.encryptedTokensMutex.Lock()
	tm.cfg.AccessJWT = encryptedAccessJWT
	tm.cfg.RefreshJWT = encryptedRefreshJWT
	tm.encryptedTokensMutex.Unlock()

	// Cache the decrypted tokens
	tm.cachedTokensMutex.Lock()
	tm.cachedAccessToken = accessJWT
	tm.cachedRefreshToken = refreshJWT
	tm.cachedTokensMutex.Unlock()

	return nil
}

// GetToken returns the requested token (access or refresh)
func (tm *TokenManager) GetToken(tokenType TokenType) (string, error) {
	// First check the cache
	tm.cachedTokensMutex.RLock()
	var cachedToken string
	if tokenType == AccessToken {
		cachedToken = tm.cachedAccessToken
	} else {
		cachedToken = tm.cachedRefreshToken
	}

	if cachedToken != "" {
		tm.cachedTokensMutex.RUnlock()
		return cachedToken, nil
	}
	tm.cachedTokensMutex.RUnlock()

	// Cache miss, need to decrypt
	tm.encryptedTokensMutex.RLock()
	var encryptedToken string
	if tokenType == AccessToken {
		encryptedToken = tm.cfg.AccessJWT
	} else {
		encryptedToken = tm.cfg.RefreshJWT
	}
	tm.encryptedTokensMutex.RUnlock()

	// Lock for writing to cache
	tm.cachedTokensMutex.Lock()
	defer tm.cachedTokensMutex.Unlock()

	// Double-check after acquiring the write lock
	if tokenType == AccessToken && tm.cachedAccessToken != "" {
		return tm.cachedAccessToken, nil
	} else if tokenType == RefreshToken && tm.cachedRefreshToken != "" {
		return tm.cachedRefreshToken, nil
	}

	// Decrypt the token
	decrypted, err := tm.encryptor.Decrypt(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt %s token: %w", tokenType, err)
	}

	// Update the cache
	if tokenType == AccessToken {
		tm.cachedAccessToken = decrypted
	} else {
		tm.cachedRefreshToken = decrypted
	}

	return decrypted, nil
}

// backgroundTokenRefresh runs a background process to periodically refresh tokens
func (tm *TokenManager) backgroundTokenRefresh() {
	for {
		select {
		case <-tm.refreshTick.C:
			log.Printf("バックグラウンドでトークンリフレッシュを開始します（間隔: %v）", tm.cfg.TokenRefreshInterval)
			ctx, cancel := context.WithTimeout(context.Background(), tm.cfg.HTTPTimeout)
			if err := tm.RefreshToken(ctx); err != nil {
				log.Printf("バックグラウンドでのトークンリフレッシュに失敗しました: %v", err)
			} else {
				log.Println("バックグラウンドでのトークンリフレッシュに成功しました")
			}
			cancel()
		case <-tm.Done:
			log.Println("トークンリフレッシュのバックグラウンドタスクを終了します")
			tm.refreshTick.Stop()
			return
		}
	}
}

// RefreshToken uses the refresh token to obtain a new access token
func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	log.Println("トークンのリフレッシュを実行します...")
	// Get the current refresh token
	refreshToken, err := tm.GetToken(RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	url := fmt.Sprintf("%s/xrpc/com.atproto.server.refreshSession", tm.cfg.PDSURL)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", refreshToken),
		"Content-Type":  "application/json",
	}

	// Use the HTTP client to make the request
	resp, err := tm.httpClient.DoRequest(ctx, "POST", url, nil, headers)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var refreshResp struct {
		AccessJWT  string `json:"accessJwt"`
		RefreshJWT string `json:"refreshJwt"`
	}

	if err := tm.httpClient.DecodeJSONResponse(resp, &refreshResp); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Update the cached tokens
	tm.cachedTokensMutex.Lock()
	tm.cachedAccessToken = refreshResp.AccessJWT
	tm.cachedRefreshToken = refreshResp.RefreshJWT
	tm.cachedTokensMutex.Unlock()

	// Encrypt and store the new tokens
	encryptedAccessJWT, err := tm.encryptor.Encrypt(refreshResp.AccessJWT)
	if err != nil {
		return fmt.Errorf("failed to encrypt new access token: %w", err)
	}

	encryptedRefreshJWT, err := tm.encryptor.Encrypt(refreshResp.RefreshJWT)
	if err != nil {
		return fmt.Errorf("failed to encrypt new refresh token: %w", err)
	}

	// Update the encrypted tokens
	tm.encryptedTokensMutex.Lock()
	tm.cfg.AccessJWT = encryptedAccessJWT
	tm.cfg.RefreshJWT = encryptedRefreshJWT
	tm.encryptedTokensMutex.Unlock()

	log.Println("新しいトークンの取得とキャッシュが完了しました")
	return nil
}

// Shutdown stops the background token refresh process
func (tm *TokenManager) Shutdown() {
	close(tm.Done)
}
