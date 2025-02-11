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

	// アプリケーション全体のコンテキストを作成
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("QuoteBot を起動しました（投稿間隔: %v）...\n", cfg.PostInterval)

	// 初回投稿
	reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	quote, err := quoteUseCase.PostRandomQuote(reqCtx)
	if err != nil {
		log.Printf("初回投稿エラー: %v", err)
	} else {
		message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
		if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
			log.Printf("初回投稿エラー: %v", err)
		}
	}
	reqCancel()

	// メインループ
	for {
		select {
		case <-ticker.C:
			reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
			quote, err := quoteUseCase.PostRandomQuote(reqCtx)
			if err != nil {
				log.Printf("投稿エラー: %v", err)
				reqCancel()
				continue
			}
			message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
			if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
				log.Printf("投稿エラー: %v", err)
			}
			reqCancel()
		case sig := <-sigChan:
			fmt.Printf("\nシグナル %v を受信しました。シャットダウンします...\n", sig)
			// バックグラウンドトークン更新プロセスのクリーンアップ
			blueskyRepo.done <- struct{}{}
			return
		}
	}
}
