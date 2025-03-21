package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/littleironwaltz/quotebot/internal/domain"
)

// QuoteRepository はドメインモデルの永続化インターフェースを定義します
type QuoteRepository interface {
	LoadQuotes() ([]domain.Quote, error)
}

// QuoteUseCase は名言の取得と投稿を制御します
type QuoteUseCase struct {
	quoteRepo QuoteRepository
	quotes    []domain.Quote
}

// NewQuoteUseCase は新しいQuoteUseCaseインスタンスを作成します
func NewQuoteUseCase(qr QuoteRepository) *QuoteUseCase {
	return &QuoteUseCase{
		quoteRepo: qr,
	}
}

// Initialize は名言リストを読み込み、初期化を実行します
func (uc *QuoteUseCase) Initialize() error {
	quotes, err := uc.quoteRepo.LoadQuotes()
	if err != nil {
		return fmt.Errorf("名言の読み込みに失敗しました: %w", err)
	}

	uc.quotes = quotes
	rand.Seed(time.Now().UnixNano())
	return nil
}

// PostRandomQuote はランダムな名言を選択して返します
func (uc *QuoteUseCase) PostRandomQuote(ctx context.Context) (*domain.Quote, error) {
	if len(uc.quotes) == 0 {
		return nil, fmt.Errorf("利用可能な名言がありません")
	}

	quote := uc.quotes[rand.Intn(len(uc.quotes))]
	return &quote, nil
}
