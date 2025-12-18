package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

type EncryptionService struct {
	key []byte
}

func NewEncryptionService(encryptionKey string) (*EncryptionService, error) {
	if len(encryptionKey) == 0 {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}

	hash := sha256.Sum256([]byte(encryptionKey))
	return &EncryptionService{
		key: hash[:],
	}, nil
}

func (es *EncryptionService) Encrypt(plaintext string) (string, string, error) {
	if plaintext == "" {
		return "", "", fmt.Errorf("plaintext cannot be empty")
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)
	encodedNonce := base64.StdEncoding.EncodeToString(nonce)

	return encodedCiphertext, encodedNonce, nil
}

func (es *EncryptionService) Decrypt(ciphertext, nonce string) (string, error) {
	if ciphertext == "" {
		return "", fmt.Errorf("ciphertext cannot be empty")
	}
	if nonce == "" {
		return "", fmt.Errorf("nonce cannot be empty")
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
