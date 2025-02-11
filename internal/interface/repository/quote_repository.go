package repository

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kojikubota/quotebot/config"
	"github.com/kojikubota/quotebot/internal/domain"
)

// QuoteRepository handles persistence of quote data
type QuoteRepository struct {
	quotesFile string
}

// NewQuoteRepository creates a new QuoteRepository instance
func NewQuoteRepository(cfg *config.Config) *QuoteRepository {
	return &QuoteRepository{
		quotesFile: cfg.QuotesFile,
	}
}

// LoadQuotes loads quote data from the file
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
