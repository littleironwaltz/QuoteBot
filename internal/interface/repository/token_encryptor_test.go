package repository

import (
	"testing"
)

func TestTokenEncryptor_EncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "正常系: 通常のテキスト",
			plaintext: "test-token-123",
		},
		{
			name:      "正常系: 空文字列",
			plaintext: "",
		},
		{
			name:      "正常系: 長いトークン",
			plaintext: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		},
		{
			name:      "正常系: 特殊文字を含むテキスト",
			plaintext: "special!@#$%^&*()_+{}[]|\\:;\"'<>,.?/~`",
		},
		{
			name:      "正常系: 日本語テキスト",
			plaintext: "こんにちは世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 暗号化器の作成
			encryptor := NewTokenEncryptor()

			// 暗号化
			encrypted, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Errorf("Encrypt() error = %v", err)
				return
			}

			// 暗号化されたテキストが平文と異なることを確認
			if encrypted == tt.plaintext && tt.plaintext != "" {
				t.Errorf("Encrypt() = %v, which is the same as plaintext", encrypted)
			}

			// 復号化
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Errorf("Decrypt() error = %v", err)
				return
			}

			// 復号化されたテキストが元の平文と一致することを確認
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestTokenEncryptor_DecryptInvalid(t *testing.T) {
	tests := []struct {
		name          string
		encryptedText string
		wantErr       bool
	}{
		{
			name:          "異常系: Base64デコードできない文字列",
			encryptedText: "this-is-not-base64!@#",
			wantErr:       true,
		},
		{
			name:          "異常系: Base64だが暗号文として短すぎる",
			encryptedText: "aGVsbG8=", // "hello" in Base64
			wantErr:       true,
		},
		{
			name:          "異常系: 空文字列",
			encryptedText: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 暗号化器の作成
			encryptor := NewTokenEncryptor()

			// 復号化の試行
			_, err := encryptor.Decrypt(tt.encryptedText)

			// エラーのチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenEncryptor_IsEncrypted(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "正常系: Base64エンコードされた文字列",
			text: "aGVsbG8gd29ybGQ=", // "hello world" in Base64
			want: true,
		},
		{
			name: "正常系: 通常のテキスト（Base64ではない）",
			text: "hello world",
			want: false,
		},
		{
			name: "正常系: 空文字列",
			text: "",
			want: true, // 空文字列はBase64としても有効
		},
		{
			name: "正常系: Base64として無効な文字を含む",
			text: "aGVsbG8!=",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 暗号化器の作成
			encryptor := NewTokenEncryptor()

			// IsEncryptedの呼び出し
			got := encryptor.IsEncrypted(tt.text)

			// 期待される結果と比較
			if got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEncryptor_MultipleCalls(t *testing.T) {
	// 暗号化器の作成
	encryptor := NewTokenEncryptor()

	// 複数の異なるトークンを暗号化
	tokens := []string{
		"token1",
		"token2",
		"token3",
	}

	encryptedTokens := make([]string, len(tokens))
	var err error

	// すべてのトークンを暗号化
	for i, token := range tokens {
		encryptedTokens[i], err = encryptor.Encrypt(token)
		if err != nil {
			t.Errorf("Encrypt(%s) error = %v", token, err)
			return
		}
	}

	// すべての暗号化されたトークンが異なることを確認
	for i := 0; i < len(encryptedTokens); i++ {
		for j := i + 1; j < len(encryptedTokens); j++ {
			if encryptedTokens[i] == encryptedTokens[j] {
				t.Errorf("Tokens %d and %d have the same encrypted value", i, j)
			}
		}
	}

	// すべてのトークンを復号化して元の値と一致することを確認
	for i, encryptedToken := range encryptedTokens {
		decrypted, err := encryptor.Decrypt(encryptedToken)
		if err != nil {
			t.Errorf("Decrypt(%s) error = %v", encryptedToken, err)
			return
		}

		if decrypted != tokens[i] {
			t.Errorf("Decrypt() = %v, want %v", decrypted, tokens[i])
		}
	}
}

func TestTokenEncryptor_NewInstanceEachTime(t *testing.T) {
	// 異なるインスタンスで暗号化したトークンは互いに復号化できないことを確認
	encryptor1 := NewTokenEncryptor()
	encryptor2 := NewTokenEncryptor()

	// テスト用のプレーンテキスト
	plaintext := "test-token"

	// 最初の暗号化器で暗号化
	encrypted1, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Errorf("Encryptor1.Encrypt() error = %v", err)
		return
	}

	// 2番目の暗号化器で暗号化
	encrypted2, err := encryptor2.Encrypt(plaintext)
	if err != nil {
		t.Errorf("Encryptor2.Encrypt() error = %v", err)
		return
	}

	// 異なる暗号化器で暗号化した結果は異なるべき
	if encrypted1 == encrypted2 {
		t.Errorf("Encryptor1 and Encryptor2 produced the same encrypted string")
	}

	// 暗号化器1で暗号化したものを暗号化器2で復号化しようとすると失敗するはず
	_, err = encryptor2.Decrypt(encrypted1)
	if err == nil {
		t.Errorf("Encryptor2 should not be able to decrypt text encrypted by Encryptor1")
	}

	// 暗号化器2で暗号化したものを暗号化器1で復号化しようとすると失敗するはず
	_, err = encryptor1.Decrypt(encrypted2)
	if err == nil {
		t.Errorf("Encryptor1 should not be able to decrypt text encrypted by Encryptor2")
	}
}
