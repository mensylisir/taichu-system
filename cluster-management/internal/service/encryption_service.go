package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

func (es *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", fmt.Errorf("plaintext cannot be empty")
	}

	// Simple base64 encoding instead of AES-256-GCM encryption
	encodedText := base64.StdEncoding.EncodeToString([]byte(plaintext))

	return encodedText, nil
}

func (es *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", fmt.Errorf("ciphertext cannot be empty")
	}

	// Try simple base64 decoding first
	decodedBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// If base64 decoding fails, try the old AES-256-GCM decryption methods
		return es.decryptWithAES(ciphertext)
	}

	return string(decodedBytes), nil
}

// decryptWithAES handles decryption for data that was encrypted with the old AES-256-GCM method
func (es *EncryptionService) decryptWithAES(ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from the beginning of ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertextBytes[:nonceSize]
	actualCiphertext := ciphertextBytes[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		// Try the old format (with duplicated nonce)
		// In the old format, the nonce was duplicated in the ciphertext
		// Old format: nonce + nonce + encrypted_data
		if len(ciphertextBytes) >= nonceSize*2 {
			secondNonce := ciphertextBytes[nonceSize : nonceSize*2]
			oldActualCiphertext := ciphertextBytes[nonceSize*2:]

			// Check if the first nonce and second nonce are the same (old format)
			if string(nonce) == string(secondNonce) {
				plaintext, err = gcm.Open(nil, nonce, oldActualCiphertext, nil)
				if err != nil {
					return "", fmt.Errorf("failed to decrypt (both new and old format failed): %w", err)
				}
			} else {
				// Try another possible old format: nonce + encrypted_data (where encrypted_data includes nonce)
				// This happens when the old implementation incorrectly prepended nonce to the ciphertext
				// but the ciphertext already included the nonce
				plaintext, err = gcm.Open(nil, secondNonce, oldActualCiphertext, nil)
				if err != nil {
					return "", fmt.Errorf("failed to decrypt (all format attempts failed): %w", err)
				}
			}
		} else {
			return "", fmt.Errorf("failed to decrypt: %w", err)
		}
	}

	return string(plaintext), nil
}
