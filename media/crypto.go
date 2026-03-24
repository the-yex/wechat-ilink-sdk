// Package media provides CDN media handling with AES-128-ECB encryption.
package media

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	aesBlockSize = 16
)

// EncryptAESECB encrypts plaintext using AES-128-ECB with PKCS7 padding.
// The key must be exactly 16 bytes (128 bits).
func EncryptAESECB(plaintext, key []byte) ([]byte, error) {
	if len(key) != aesBlockSize {
		return nil, fmt.Errorf("key must be 16 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Apply PKCS7 padding
	padded := pkcs7Pad(plaintext, aesBlockSize)

	// ECB encryption - encrypt each block independently
	ciphertext := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aesBlockSize {
		block.Encrypt(ciphertext[i:i+aesBlockSize], padded[i:i+aesBlockSize])
	}

	return ciphertext, nil
}

// DecryptAESECB decrypts ciphertext using AES-128-ECB with PKCS7 padding.
// The key must be exactly 16 bytes (128 bits).
// The ciphertext must be a multiple of 16 bytes.
func DecryptAESECB(ciphertext, key []byte) ([]byte, error) {
	if len(key) != aesBlockSize {
		return nil, fmt.Errorf("key must be 16 bytes, got %d", len(key))
	}

	if len(ciphertext)%aesBlockSize != 0 {
		return nil, fmt.Errorf("ciphertext must be multiple of 16 bytes, got %d", len(ciphertext))
	}

	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// ECB decryption - decrypt each block independently
	plaintext := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aesBlockSize {
		block.Decrypt(plaintext[i:i+aesBlockSize], ciphertext[i:i+aesBlockSize])
	}

	// Remove PKCS7 padding
	return pkcs7Unpad(plaintext)
}

// GenerateAESKey generates a random 16-byte AES key.
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, aesBlockSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}

// AESECBPaddedSize returns the ciphertext size after PKCS7 padding.
// PKCS7 padding always adds at least 1 byte.
func AESECBPaddedSize(plaintextSize int) int {
	return ((plaintextSize + aesBlockSize) / aesBlockSize) * aesBlockSize
}

// pkcs7Pad applies PKCS7 padding to the data.
func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	// PKCS7 padding always adds at least 1 byte
	if pad == 0 {
		pad = blockSize
	}
	padding := bytes.Repeat([]byte{byte(pad)}, pad)
	return append(data, padding...)
}

// pkcs7Unpad removes PKCS7 padding from the data.
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	pad := int(data[len(data)-1])

	// Validate padding
	if pad > aesBlockSize {
		return nil, fmt.Errorf("invalid padding: %d > block size", pad)
	}
	if pad > len(data) {
		return nil, fmt.Errorf("invalid padding: %d > data length", pad)
	}

	// Verify all padding bytes are correct
	for i := len(data) - pad; i < len(data); i++ {
		if data[i] != byte(pad) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}

	return data[:len(data)-pad], nil
}

// GenerateFileKey generates a random 16-byte file key (hex encoded).
func GenerateFileKey() (string, error) {
	key := make([]byte, aesBlockSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate file key: %w", err)
	}
	return fmt.Sprintf("%x", key), nil
}