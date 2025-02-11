package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/kojikubota/quotebot/internal/domain"
)

// QuoteRepository defines the persistence interface for domain models
type QuoteRepository interface {
	LoadQuotes() ([]domain.Quote, error)
}

// QuoteUseCase controls the retrieval and posting of quotes
type QuoteUseCase struct {
	quoteRepo QuoteRepository
	quotes    []domain.Quote
}

// NewQuoteUseCase creates a new QuoteUseCase instance
func NewQuoteUseCase(qr QuoteRepository) *QuoteUseCase {
	return &QuoteUseCase{
		quoteRepo: qr,
	}
}

// Initialize loads the quote list and performs initialization
func (uc *QuoteUseCase) Initialize() error {
	quotes, err := uc.quoteRepo.LoadQuotes()
	if err != nil {
		return fmt.Errorf("failed to load quotes: %w", err)
	}

	uc.quotes = quotes
	rand.Seed(time.Now().UnixNano())
	return nil
}

// PostRandomQuote selects and returns a random quote
func (uc *QuoteUseCase) PostRandomQuote(ctx context.Context) (*domain.Quote, error) {
	if len(uc.quotes) == 0 {
		return nil, fmt.Errorf("no quotes available")
	}

	quote := uc.quotes[rand.Intn(len(uc.quotes))]
	return &quote, nil
}
