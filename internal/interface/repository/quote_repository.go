package repository

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/littleironwaltz/quotebot/config"
	"github.com/littleironwaltz/quotebot/internal/domain"
)

// QuoteRepository は名言データの永続化を処理します
type QuoteRepository struct {
	quotesFile string
}

// NewQuoteRepository は新しいQuoteRepositoryインスタンスを作成します
func NewQuoteRepository(cfg *config.Config) *QuoteRepository {
	return &QuoteRepository{
		quotesFile: cfg.QuotesFile,
	}
}

// LoadQuotes はファイルから名言データを読み込みます
func (r *QuoteRepository) LoadQuotes() ([]domain.Quote, error) {
	file, err := os.Open(r.quotesFile)
	if err != nil {
		return nil, fmt.Errorf("名言ファイルのオープンに失敗しました: %w", err)
	}
	defer file.Close()

	var quotes []domain.Quote
	if err := json.NewDecoder(file).Decode(&quotes); err != nil {
		return nil, fmt.Errorf("名言データのデコードに失敗しました: %w", err)
	}

	return quotes, nil
}
