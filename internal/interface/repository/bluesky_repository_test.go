package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kojikubota/quotebot/config"
)

func TestBlueskyRepository_PostMessage(t *testing.T) {
	// Set up test server
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
			name: "success case: initial post successful",
			cfg: &config.Config{
				AccessJWT:    "valid-token",
				RefreshJWT:   "refresh-token",
				DID:         "did:plc:test",
				PDSURL:      server.URL,
				HTTPTimeout: 3 * time.Second,
			},
			message: "test message",
			wantErr: false,
		},
		{
			name: "error case: successful after auth error and refresh",
			cfg: &config.Config{
				AccessJWT:    "invalid-token",
				RefreshJWT:   "refresh-token",
				DID:         "did:plc:test",
				PDSURL:      server.URL,
				HTTPTimeout: 3 * time.Second,
			},
			message: "test message",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewBlueskyRepository(tt.cfg)
			ctx := context.Background()
			err := repo.PostMessage(ctx, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueskyRepository.PostMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlueskyRepository_RefreshToken(t *testing.T) {
	// Set up test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.refreshSession" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

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
			name: "success case: token refresh successful",
			cfg: &config.Config{
				AccessJWT:    "old-token",
				RefreshJWT:   "old-refresh-token",
				DID:         "did:plc:test",
				PDSURL:      server.URL,
				HTTPTimeout: 3 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewBlueskyRepository(tt.cfg)
			ctx := context.Background()
			err := repo.RefreshToken(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueskyRepository.RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
