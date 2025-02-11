package usecase

import "context"

// BlueskyRepository is an interface for posting to Bluesky
type BlueskyRepository interface {
	// PostMessage posts the specified message to Bluesky
	PostMessage(ctx context.Context, message string) error
	// RefreshToken uses the refresh token to obtain a new access token
	RefreshToken(ctx context.Context) error
}
