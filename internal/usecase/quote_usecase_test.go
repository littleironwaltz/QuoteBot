package usecase

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/littleironwaltz/quotebot/internal/domain"
)

// モックリポジトリの実装
type mockQuoteRepository struct {
	quotes []domain.Quote
	err    error
}

func (m *mockQuoteRepository) LoadQuotes() ([]domain.Quote, error) {
	return m.quotes, m.err
}

func TestQuoteUseCase_Initialize(t *testing.T) {
	tests := []struct {
		name       string
		mockRepo   *mockQuoteRepository
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "正常系: 名言の読み込み成功",
			mockRepo: &mockQuoteRepository{
				quotes: []domain.Quote{
					{Text: "テスト名言1", Author: "著者1"},
					{Text: "テスト名言2", Author: "著者2"},
				},
				err: nil,
			},
			wantErr: false,
		},
		{
			name: "異常系: 名言の読み込み失敗",
			mockRepo: &mockQuoteRepository{
				quotes: nil,
				err:    errors.New("ファイル読み込みエラー"),
			},
			wantErr:    true,
			wantErrMsg: "名言の読み込みに失敗しました: ファイル読み込みエラー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewQuoteUseCase(tt.mockRepo)
			err := uc.Initialize()

			// エラー確認
			if (err != nil) != tt.wantErr {
				t.Errorf("QuoteUseCase.Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーメッセージ確認（エラーが期待される場合）
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("QuoteUseCase.Initialize() error message = %v, want %v", err.Error(), tt.wantErrMsg)
				return
			}

			// 正常系の場合は名言が読み込まれているか確認
			if !tt.wantErr {
				if len(uc.quotes) != len(tt.mockRepo.quotes) {
					t.Errorf("QuoteUseCase.Initialize() loaded %d quotes, want %d", len(uc.quotes), len(tt.mockRepo.quotes))
				}
			}
		})
	}
}

func TestQuoteUseCase_PostRandomQuote(t *testing.T) {
	// 乱数の再現性のためにシード値固定
	seed := time.Now().UnixNano()
	rand.Seed(seed)

	// テスト終了後に乱数生成器をリセット
	defer func() {
		rand.Seed(time.Now().UnixNano())
	}()

	tests := []struct {
		name        string
		quotes      []domain.Quote
		emptyQuotes bool
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "正常系: ランダムな名言を取得",
			quotes: []domain.Quote{
				{Text: "テスト名言1", Author: "著者1"},
				{Text: "テスト名言2", Author: "著者2"},
				{Text: "テスト名言3", Author: "著者3"},
			},
			emptyQuotes: false,
			wantErr:     false,
		},
		{
			name:        "異常系: 利用可能な名言がない",
			quotes:      []domain.Quote{},
			emptyQuotes: true,
			wantErr:     true,
			wantErrMsg:  "利用可能な名言がありません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックリポジトリの設定
			mockRepo := &mockQuoteRepository{
				quotes: tt.quotes,
				err:    nil,
			}

			// ユースケースの初期化
			uc := NewQuoteUseCase(mockRepo)

			// テスト用に初期化
			if !tt.emptyQuotes {
				if err := uc.Initialize(); err != nil {
					t.Fatalf("QuoteUseCase.Initialize() failed: %v", err)
				}
			}

			// ランダムな名言を取得
			ctx := context.Background()
			quote, err := uc.PostRandomQuote(ctx)

			// エラー確認
			if (err != nil) != tt.wantErr {
				t.Errorf("QuoteUseCase.PostRandomQuote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーメッセージ確認（エラーが期待される場合）
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("QuoteUseCase.PostRandomQuote() error message = %v, want %v", err.Error(), tt.wantErrMsg)
				return
			}

			// 正常系の場合は返却された名言が元の名言リストに含まれているか確認
			if !tt.wantErr {
				found := false
				for _, q := range tt.quotes {
					if q.Text == quote.Text && q.Author == quote.Author {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("QuoteUseCase.PostRandomQuote() returned quote not in original list: %+v", quote)
				}
			}
		})
	}
}
