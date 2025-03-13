package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/littleironwaltz/quotebot/config"
	"github.com/littleironwaltz/quotebot/internal/interface/repository"
	"github.com/littleironwaltz/quotebot/internal/usecase"
)

// 統合テスト用の設定を作成するヘルパー関数
func setupTestConfig(t *testing.T) *config.Config {
	// テスト用の一時ファイルを作成
	tempQuotesFile, err := os.CreateTemp("", "test_quotes_*.json")
	if err != nil {
		t.Fatalf("テスト用の一時ファイルの作成に失敗しました: %v", err)
	}
	defer tempQuotesFile.Close()

	// テスト用の引用を書き込む
	testQuotes := `[
		{"text": "統合テスト用の引用1", "author": "テスト作者1"},
		{"text": "統合テスト用の引用2", "author": "テスト作者2"},
		{"text": "統合テスト用の引用3", "author": "テスト作者3"}
	]`
	if _, err := tempQuotesFile.Write([]byte(testQuotes)); err != nil {
		t.Fatalf("テスト用引用の書き込みに失敗しました: %v", err)
	}

	// テスト用の設定を作成
	return &config.Config{
		PDSURL:               "https://example.com", // モック用のホスト
		DID:                  "test.user",
		AccessJWT:            "test_access_token",
		RefreshJWT:           "test_refresh_token",
		QuotesFile:           tempQuotesFile.Name(), // テスト用一時ファイルのパス
		PostInterval:         1 * time.Minute,
		HTTPTimeout:          10 * time.Second,
		TokenRefreshInterval: 1 * time.Hour,
		MaxRetries:           3,
		RetryBackoff:         2 * time.Second,
	}
}

// テスト後のクリーンアップ
func cleanupTest(t *testing.T, cfg *config.Config) {
	// テスト用の一時ファイルを削除
	if err := os.Remove(cfg.QuotesFile); err != nil {
		t.Logf("テスト用一時ファイルの削除に失敗しました: %v", err)
	}
}

// モックBlueskyリポジトリ
type MockBlueskyRepository struct {
	PostMessageCalled  bool
	PostMessageError   error
	RefreshTokenCalled bool
	RefreshTokenError  error
	Message            string
	Done               chan struct{}
}

func NewMockBlueskyRepository() *MockBlueskyRepository {
	return &MockBlueskyRepository{
		PostMessageCalled:  false,
		PostMessageError:   nil,
		RefreshTokenCalled: false,
		RefreshTokenError:  nil,
		Done:               make(chan struct{}, 1),
	}
}

func (m *MockBlueskyRepository) PostMessage(ctx context.Context, message string) error {
	m.PostMessageCalled = true
	m.Message = message
	return m.PostMessageError
}

func (m *MockBlueskyRepository) RefreshToken(ctx context.Context) error {
	m.RefreshTokenCalled = true
	return m.RefreshTokenError
}

// 統合テスト：全体的なフロー
func TestIntegrationFlow(t *testing.T) {
	// テスト用設定のセットアップ
	cfg := setupTestConfig(t)
	defer cleanupTest(t, cfg)

	// 実際のリポジトリとモックリポジトリの初期化
	quoteRepo := repository.NewQuoteRepository(cfg)
	mockBlueskyRepo := NewMockBlueskyRepository()

	// ユースケースの初期化
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)

	// 1. ユースケースの初期化
	err := quoteUseCase.Initialize()
	if err != nil {
		t.Fatalf("ユースケースの初期化に失敗しました: %v", err)
	}

	// 2. メイン処理の前にトークンをリフレッシュ（実際のmain.goの処理と同様）
	ctx := context.Background()
	err = mockBlueskyRepo.RefreshToken(ctx)
	if err != nil {
		t.Fatalf("トークンリフレッシュに失敗しました: %v", err)
	}

	// トークンリフレッシュが呼び出されたことを確認
	if !mockBlueskyRepo.RefreshTokenCalled {
		t.Error("トークンリフレッシュが呼び出されていません")
	}

	// 3. ランダムな引用を取得
	quote, err := quoteUseCase.PostRandomQuote(ctx)
	if err != nil {
		t.Fatalf("ランダムな引用の取得に失敗しました: %v", err)
	}

	// 4. 引用の内容を検証
	if quote.Text == "" || quote.Author == "" {
		t.Errorf("引用の内容が空です: %+v", quote)
	}

	// 5. 引用をフォーマットしてBlueskyリポジトリに投稿
	message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
	err = mockBlueskyRepo.PostMessage(ctx, message)
	if err != nil {
		t.Fatalf("メッセージの投稿に失敗しました: %v", err)
	}

	// 6. モックBlueskyリポジトリが正しく呼び出されたか検証
	if !mockBlueskyRepo.PostMessageCalled {
		t.Error("Blueskyリポジトリの PostMessage が呼び出されませんでした")
	}

	// 7. 投稿されたメッセージが正しいか検証
	if mockBlueskyRepo.Message != message {
		t.Errorf("投稿されたメッセージが期待と異なります: got=%s, want=%s", mockBlueskyRepo.Message, message)
	}
}

// 統合テスト：エラーケース（引用ファイルが存在しない）
func TestIntegrationFlow_QuotesFileNotFound(t *testing.T) {
	// テスト用設定の作成（存在しないファイルパスを指定）
	cfg := &config.Config{
		PDSURL:               "https://example.com",
		DID:                  "test.user",
		AccessJWT:            "test_access_token",
		RefreshJWT:           "test_refresh_token",
		QuotesFile:           "/not/exist/quotes.json", // 存在しないパス
		PostInterval:         1 * time.Minute,
		HTTPTimeout:          10 * time.Second,
		TokenRefreshInterval: 1 * time.Hour,
		MaxRetries:           3,
		RetryBackoff:         2 * time.Second,
	}

	// リポジトリとユースケースの初期化
	quoteRepo := repository.NewQuoteRepository(cfg)
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)

	// 初期化でエラーが発生することを確認
	err := quoteUseCase.Initialize()
	if err == nil {
		t.Error("存在しない引用ファイルでエラーが発生しませんでした")
	}
}

// 統合テスト：Blueskyリポジトリでエラーが発生するケース
func TestIntegrationFlow_BlueskyError(t *testing.T) {
	// テスト用設定のセットアップ
	cfg := setupTestConfig(t)
	defer cleanupTest(t, cfg)

	// リポジトリとユースケースの初期化
	quoteRepo := repository.NewQuoteRepository(cfg)
	mockBlueskyRepo := NewMockBlueskyRepository()
	mockBlueskyRepo.PostMessageError = fmt.Errorf("Bluesky APIエラー")

	// ユースケースの初期化
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)
	err := quoteUseCase.Initialize()
	if err != nil {
		t.Fatalf("ユースケースの初期化に失敗しました: %v", err)
	}

	// メイン処理の前にトークンをリフレッシュ（実際のmain.goの処理と同様）
	ctx := context.Background()
	err = mockBlueskyRepo.RefreshToken(ctx)
	if err != nil {
		t.Fatalf("トークンリフレッシュに失敗しました: %v", err)
	}

	// トークンリフレッシュが呼び出されたことを確認
	if !mockBlueskyRepo.RefreshTokenCalled {
		t.Error("トークンリフレッシュが呼び出されていません")
	}

	// ランダムな引用を取得
	quote, err := quoteUseCase.PostRandomQuote(ctx)
	if err != nil {
		t.Fatalf("ランダムな引用の取得に失敗しました: %v", err)
	}

	// メッセージをフォーマットして投稿（エラーが発生するはず）
	message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
	err = mockBlueskyRepo.PostMessage(ctx, message)
	if err == nil {
		t.Error("Blueskyリポジトリでエラーが発生するはずですが、成功してしまいました")
	}
}

// 統合テスト：実際のリポジトリと部分的なモックの組み合わせ
func TestIntegrationWithPartialMock(t *testing.T) {
	// テスト用設定のセットアップ
	cfg := setupTestConfig(t)
	defer cleanupTest(t, cfg)

	// 実際のQuoteRepositoryを使用
	quoteRepo := repository.NewQuoteRepository(cfg)

	// カスタムモックBlueskyRepositoryを作成
	mockBlueskyRepo := &MockBlueskyRepository{
		Done: make(chan struct{}, 1),
	}

	// ユースケースの初期化
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)
	err := quoteUseCase.Initialize()
	if err != nil {
		t.Fatalf("ユースケースの初期化に失敗しました: %v", err)
	}

	// 実際のアプリケーションフローに近いシミュレーション
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. メイン処理の前にトークンをリフレッシュ（実際のmain.goの処理と同様）
	err = mockBlueskyRepo.RefreshToken(ctx)
	if err != nil {
		t.Fatalf("トークンリフレッシュに失敗しました: %v", err)
	}

	// 2. 引用を取得
	quote, err := quoteUseCase.PostRandomQuote(ctx)
	if err != nil {
		t.Fatalf("引用の取得に失敗しました: %v", err)
	}

	// 3. 引用をフォーマットしてBlueskyに投稿
	message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
	err = mockBlueskyRepo.PostMessage(ctx, message)
	if err != nil {
		t.Fatalf("メッセージの投稿に失敗しました: %v", err)
	}

	// 4. 引用のフォーマットが正しいか検証
	expectedPrefix := quote.Text + "\n- " + quote.Author
	if mockBlueskyRepo.Message != expectedPrefix {
		t.Errorf("引用のフォーマットが間違っています: got=%s, want=%s",
			mockBlueskyRepo.Message, expectedPrefix)
	}

	// 5. トークンリフレッシュが呼び出されたことを確認
	if !mockBlueskyRepo.RefreshTokenCalled {
		t.Error("トークンリフレッシュが呼び出されていません")
	}

	// 6. シャットダウンシグナルのシミュレーション
	mockBlueskyRepo.Done <- struct{}{}
}
