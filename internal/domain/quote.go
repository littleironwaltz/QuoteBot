package domain

// Quote はドメインモデルとして名言とその著者を表します
type Quote struct {
	Text   string
	Author string
}

// Format は名言を表示用にフォーマットします
func (q *Quote) Format() string {
	return q.Text + "\n― " + q.Author
}
