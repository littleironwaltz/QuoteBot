package domain

import (
	"testing"
)

func TestQuote_Format(t *testing.T) {
	tests := []struct {
		name  string
		quote Quote
		want  string
	}{
		{
			name: "通常の名言と著者",
			quote: Quote{
				Text:   "我思う、ゆえに我あり。",
				Author: "ルネ・デカルト",
			},
			want: "我思う、ゆえに我あり。\n― ルネ・デカルト",
		},
		{
			name: "空の名言",
			quote: Quote{
				Text:   "",
				Author: "著者名",
			},
			want: "\n― 著者名",
		},
		{
			name: "空の著者",
			quote: Quote{
				Text:   "テキスト内容",
				Author: "",
			},
			want: "テキスト内容\n― ",
		},
		{
			name: "特殊文字を含む名言",
			quote: Quote{
				Text:   "これは「特殊」な\n文字列です。",
				Author: "テスト 作者！",
			},
			want: "これは「特殊」な\n文字列です。\n― テスト 作者！",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.quote.Format()
			if got != tt.want {
				t.Errorf("Quote.Format() = %v, want %v", got, tt.want)
			}
		})
	}
}
