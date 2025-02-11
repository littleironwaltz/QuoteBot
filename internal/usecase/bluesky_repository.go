package usecase

import "context"

// BlueskyRepository はBlueskyへの投稿を担当するインターフェースです
type BlueskyRepository interface {
	// PostMessage は指定されたメッセージをBlueskyに投稿します
	PostMessage(ctx context.Context, message string) error
	// RefreshToken はリフレッシュトークンを使用して新しいアクセストークンを取得します
	RefreshToken(ctx context.Context) error
}
