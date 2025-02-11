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
		log.Fatalf("設定の読み込みに失敗: %v", err)
	}

	quoteRepo := repository.NewQuoteRepository(cfg)
	blueskyRepo := repository.NewBlueskyRepository(cfg)
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)

	if err := quoteUseCase.Initialize(); err != nil {
		log.Fatalf("ユースケースの初期化に失敗: %v", err)
	}

	// シグナル処理の設定
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// タイマーの設定
	ticker := time.NewTicker(cfg.PostInterval)
	defer ticker.Stop()

	fmt.Printf("QuoteBot を起動しました（投稿間隔: %v）...\n", cfg.PostInterval)

	// 初回投稿
	ctx := context.Background()
	quote, err := quoteUseCase.PostRandomQuote(ctx)
	if err != nil {
		log.Printf("初回投稿エラー: %v", err)
	} else {
		message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
		if err := blueskyRepo.PostMessage(ctx, message); err != nil {
			log.Printf("初回投稿エラー: %v", err)
		}
	}

	// メインループ
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout)
			quote, err := quoteUseCase.PostRandomQuote(ctx)
			if err != nil {
				log.Printf("投稿エラー: %v", err)
				cancel()
				continue
			}
			message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
			if err := blueskyRepo.PostMessage(ctx, message); err != nil {
				log.Printf("投稿エラー: %v", err)
			}
			cancel()
		case sig := <-sigChan:
			fmt.Printf("\nシグナル %v を受信しました。シャットダウンします...\n", sig)
			return
		}
	}
}