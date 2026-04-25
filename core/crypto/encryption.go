// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// AES256GCM provides AES-256-GCM encryption
type AES256GCM struct {
	key []byte
}

// NewAES256GCM creates a new AES-256-GCM encrypter with the given key
func NewAES256GCM(key []byte) (*AES256GCM, error) {
	// Derive a 256-bit key from the input using SHA-256
	hash := sha256.Sum256(key)
	return &AES256GCM{key: hash[:]}, nil
}

// NewAES256GCMFromPassword creates a new AES-256-GCM encrypter from a password
func NewAES256GCMFromPassword(password string) *AES256GCM {
	hash := sha256.Sum256([]byte(password))
	return &AES256GCM{key: hash[:]}
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns base64-encoded ciphertext: nonce + ciphertext + tag
func (e *AES256GCM) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with AES-256-GCM
// Expects base64-encoded ciphertext: nonce + ciphertext + tag
func (e *AES256GCM) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptToBase64 encrypts and returns base64-encoded string
func (e *AES256GCM) EncryptToBase64(plaintext []byte) (string, error) {
	ciphertext, err := e.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decodes base64 and decrypts
func (e *AES256GCM) DecryptFromBase64(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return e.Decrypt(ciphertext)
}

// GenerateKey generates a new random 256-bit key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateNonce generates a random nonce for AES-GCM
func GenerateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// KeyDerivation derives a key from a master key and context using HKDF-like derivation
func KeyDerivation(masterKey []byte, context string) []byte {
	// Simple derivation: hash(masterKey || context)
	h := sha256.Sum256(append(masterKey, []byte(context)...))
	return h[:]
}
