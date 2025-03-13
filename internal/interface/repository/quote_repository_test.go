package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/littleironwaltz/quotebot/config"
	"github.com/littleironwaltz/quotebot/internal/domain"
)

func TestQuoteRepository_LoadQuotes(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "quotebot_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗しました: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 有効なJSONファイルを作成
	validJSON := `[
		{
			"text": "テスト名言1",
			"author": "テスト著者1"
		},
		{
			"text": "テスト名言2",
			"author": "テスト著者2"
		}
	]`
	validJSONPath := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(validJSONPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗しました: %v", err)
	}

	// 無効なJSONファイルを作成
	invalidJSON := `{ invalid json }`
	invalidJSONPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJSONPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗しました: %v", err)
	}

	// 存在しないファイルパス
	nonExistentPath := filepath.Join(tempDir, "nonexistent.json")

	tests := []struct {
		name        string
		quotesFile  string
		wantQuotes  []domain.Quote
		wantErr     bool
		wantErrText string
	}{
		{
			name:       "正常系: 有効なJSONファイルを読み込む",
			quotesFile: validJSONPath,
			wantQuotes: []domain.Quote{
				{Text: "テスト名言1", Author: "テスト著者1"},
				{Text: "テスト名言2", Author: "テスト著者2"},
			},
			wantErr: false,
		},
		{
			name:        "異常系: 無効なJSONファイルを読み込む",
			quotesFile:  invalidJSONPath,
			wantQuotes:  nil,
			wantErr:     true,
			wantErrText: "名言データのデコードに失敗しました",
		},
		{
			name:        "異常系: 存在しないファイルを読み込む",
			quotesFile:  nonExistentPath,
			wantQuotes:  nil,
			wantErr:     true,
			wantErrText: "名言ファイルのオープンに失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用の設定を作成
			cfg := &config.Config{
				QuotesFile: tt.quotesFile,
			}

			// リポジトリを作成
			r := NewQuoteRepository(cfg)

			// 名言を読み込む
			quotes, err := r.LoadQuotes()

			// エラー確認
			if (err != nil) != tt.wantErr {
				t.Errorf("QuoteRepository.LoadQuotes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーメッセージの確認
			if err != nil && tt.wantErrText != "" {
				if errmsg := err.Error(); errmsg == "" || errmsg[:len(tt.wantErrText)] != tt.wantErrText {
					t.Errorf("QuoteRepository.LoadQuotes() エラーテキスト '%v' が '%v' を含んでいません", errmsg, tt.wantErrText)
				}
				return
			}

			// 正常系の場合はクオート数とその内容を確認
			if !tt.wantErr {
				if len(quotes) != len(tt.wantQuotes) {
					t.Errorf("QuoteRepository.LoadQuotes() が返した名言の数 = %d, 期待値 %d", len(quotes), len(tt.wantQuotes))
					return
				}

				// 各名言の内容を確認
				for i, wantQuote := range tt.wantQuotes {
					if quotes[i].Text != wantQuote.Text || quotes[i].Author != wantQuote.Author {
						t.Errorf("QuoteRepository.LoadQuotes()[%d] = %+v, 期待値 %+v", i, quotes[i], wantQuote)
					}
				}
			}
		})
	}
}
