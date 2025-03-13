package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/littleironwaltz/quotebot/config"
	"github.com/littleironwaltz/quotebot/internal/interface/repository"
	"github.com/littleironwaltz/quotebot/internal/usecase"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	quoteRepo := repository.NewQuoteRepository(cfg)
	blueskyRepo := repository.NewBlueskyRepository(cfg)
	quoteUseCase := usecase.NewQuoteUseCase(quoteRepo)

	if err := quoteUseCase.Initialize(); err != nil {
		log.Fatalf("ユースケースの初期化に失敗しました: %v", err)
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

	fmt.Printf("QuoteBotが起動しました（投稿間隔: %v）...\n", cfg.PostInterval)

	// 初回投稿
	reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)

	// 投稿前に明示的にトークンをリフレッシュ
	log.Println("初回投稿前にトークンをリフレッシュします...")
	if err := blueskyRepo.RefreshToken(reqCtx); err != nil {
		log.Printf("トークンリフレッシュに失敗しました: %v", err)
	} else {
		log.Println("トークンリフレッシュに成功しました")
	}

	quote, err := quoteUseCase.PostRandomQuote(reqCtx)
	if err != nil {
		log.Printf("初回投稿の実行に失敗しました: %v", err)
	} else {
		message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
		if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
			log.Printf("初回投稿の実行に失敗しました: %v", err)
		} else {
			log.Println("初回投稿に成功しました")
		}
	}
	reqCancel()

	// メインループ
	for {
		select {
		case <-ticker.C:
			reqCtx, reqCancel := context.WithTimeout(ctx, cfg.HTTPTimeout)

			// 定期的な投稿前にもトークンをリフレッシュ
			log.Println("定期投稿前にトークンをリフレッシュします...")
			if err := blueskyRepo.RefreshToken(reqCtx); err != nil {
				log.Printf("トークンリフレッシュに失敗しました: %v", err)
			} else {
				log.Println("トークンリフレッシュに成功しました")
			}

			quote, err := quoteUseCase.PostRandomQuote(reqCtx)
			if err != nil {
				log.Printf("メッセージの投稿に失敗しました: %v", err)
				reqCancel()
				continue
			}
			message := fmt.Sprintf("%s\n- %s", quote.Text, quote.Author)
			if err := blueskyRepo.PostMessage(reqCtx, message); err != nil {
				log.Printf("メッセージの投稿に失敗しました: %v", err)
			} else {
				log.Println("メッセージの投稿に成功しました")
			}
			reqCancel()
		case sig := <-sigChan:
			fmt.Printf("\nシグナル %v を受信しました。シャットダウンします...\n", sig)
			// バックグラウンドのトークン更新プロセスをクリーンアップ
			blueskyRepo.Done <- struct{}{}
			return
		}
	}
}
