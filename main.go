package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kojikubota/quotebot/config"
	"github.com/kojikubota/quotebot/internal/interface/repository"
	"github.com/kojikubota/quotebot/internal/usecase"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	quoteRepo := repository.NewQuoteRepository(cfg)
	blueskyRepo := repository.NewBlueskyRepository(cfg)
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)

	if err := quoteUseCase.Initialize(); err != nil {
		log.Fatalf("Failed to initialize use case: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up timer
	ticker := time.NewTicker(cfg.PostInterval)
	defer ticker.Stop()

	// Create application-wide context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("QuoteBot started (posting interval: %v)...\n", cfg.PostInterval)

	// Initial post
	reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	quote, err := quoteUseCase.PostRandomQuote(reqCtx)
	if err != nil {
		log.Printf("Failed to make initial post: %v", err)
	} else {
		message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
		if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
			log.Printf("Failed to make initial post: %v", err)
		}
	}
	reqCancel()

	// Main loop
	for {
		select {
		case <-ticker.C:
			reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
			quote, err := quoteUseCase.PostRandomQuote(reqCtx)
			if err != nil {
				log.Printf("Failed to post message: %v", err)
				reqCancel()
				continue
			}
			message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
			if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
				log.Printf("Failed to post message: %v", err)
			}
			reqCancel()
		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v. Shutting down...\n", sig)
			// Clean up background token refresh process
			blueskyRepo.Done <- struct{}{}
			return
		}
	}
}
