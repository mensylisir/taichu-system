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

	// Prepend nonce to ciphertext
	ciphertext := make([]byte, 0, len(nonce)+len(plaintext)+gcm.Overhead())
	ciphertext = append(ciphertext, nonce...)
	ciphertext = gcm.Seal(ciphertext, nonce, []byte(plaintext), nil)

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

	// The ciphertext already contains the nonce at the beginning
	// We need to extract it for verification
	nonceFromCiphertext := ciphertextBytes[:len(nonceBytes)]
	actualCiphertext := ciphertextBytes[len(nonceBytes):]

	// Verify nonce matches
	if string(nonceFromCiphertext) != string(nonceBytes) {
		return "", fmt.Errorf("nonce mismatch")
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonceBytes, actualCiphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
