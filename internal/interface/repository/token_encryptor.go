package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
)

// TokenEncryptor handles encryption and decryption of tokens
type TokenEncryptor struct {
	encryptKey []byte
	aesBlock   cipher.Block
}

// NewTokenEncryptor creates a new TokenEncryptor instance
func NewTokenEncryptor() *TokenEncryptor {
	// Generate encryption key
	encryptKey := make([]byte, DefaultKeySize)
	if _, err := rand.Read(encryptKey); err != nil {
		log.Printf("Warning: failed to generate secure encryption key: %v", err)
		// Use a fallback mechanism to ensure we have a key, but log a warning
		for i := range encryptKey {
			encryptKey[i] = byte(i)
		}
	}

	block, err := aes.NewCipher(encryptKey)
	if err != nil {
		log.Printf("Warning: failed to initialize AES cipher: %v", err)
	}

	return &TokenEncryptor{
		encryptKey: encryptKey,
		aesBlock:   block,
	}
}

// Encrypt encrypts a string using AES-GCM
func (te *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	var block cipher.Block
	var err error

	// Use pre-initialized AES block if available
	if te.aesBlock != nil {
		block = te.aesBlock
	} else {
		// Fallback: create a new block if needed
		block, err = aes.NewCipher(te.encryptKey)
		if err != nil {
			return "", fmt.Errorf("failed to create cipher: %w", err)
		}
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a string using AES-GCM
func (te *TokenEncryptor) Decrypt(encryptedString string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	var block cipher.Block

	// Use pre-initialized AES block if available
	if te.aesBlock != nil {
		block = te.aesBlock
	} else {
		// Fallback: create a new block if needed
		block, err = aes.NewCipher(te.encryptKey)
		if err != nil {
			return "", fmt.Errorf("failed to create cipher: %w", err)
		}
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted attempts to determine if a string is already encrypted
// by trying to decode it as base64
func (te *TokenEncryptor) IsEncrypted(text string) bool {
	_, err := base64.StdEncoding.DecodeString(text)
	return err == nil
}
