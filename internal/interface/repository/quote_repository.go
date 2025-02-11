package repository

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kojikubota/quotebot/config"
	"github.com/kojikubota/quotebot/internal/domain"
)

// QuoteRepository は名言データの永続化を担当します
type QuoteRepository struct {
	quotesFile string
}

// NewQuoteRepository は新しいQuoteRepositoryインスタンスを作成します
func NewQuoteRepository(cfg *config.Config) *QuoteRepository {
	return &QuoteRepository{
		quotesFile: cfg.QuotesFile,
	}
}

// LoadQuotes は名言データをファイルから読み込みます
func (r *QuoteRepository) LoadQuotes() ([]domain.Quote, error) {
	file, err := os.Open(r.quotesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open quotes file: %w", err)
	}
	defer file.Close()

	var quotes []domain.Quote
	if err := json.NewDecoder(file).Decode(&quotes); err != nil {
		return nil, fmt.Errorf("failed to decode quotes: %w", err)
	}

	return quotes, nil
}
