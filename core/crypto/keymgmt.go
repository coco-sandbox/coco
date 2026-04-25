// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// KeyManager handles encryption key management
type KeyManager struct {
	mu         sync.RWMutex
	masterKey  []byte
	keyPath    string
	derivedKey map[string][]byte // context -> derived key
}

// KeyManagerConfig holds configuration for the key manager
type KeyManagerConfig struct {
	MasterKeyPath string // Path to master key file
	KeysDir      string // Directory for key files
	UseHSM      bool   // Use HSM/TPM (future)
	UseTPM      bool   // Use TPM (future)
}

// NewKeyManager creates a new key manager
func NewKeyManager(config KeyManagerConfig) (*KeyManager, error) {
	km := &KeyManager{
		keyPath:    config.KeysDir,
		derivedKey: make(map[string][]byte),
	}

	// Ensure keys directory exists
	if config.KeysDir != "" {
		if err := os.MkdirAll(config.KeysDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create keys directory: %w", err)
		}
	}

	// Load or generate master key
	if config.MasterKeyPath != "" {
		key, err := loadMasterKey(config.MasterKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load master key: %w", err)
		}
		km.masterKey = key
	} else {
		// Generate a new master key
		key, err := GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate master key: %w", err)
		}
		km.masterKey = key

		// Save the master key if KeysDir is configured
		if config.KeysDir != "" {
			if err := saveMasterKey(config.KeysDir+"/master.key", key); err != nil {
				return nil, fmt.Errorf("failed to save master key: %w", err)
			}
		}
	}

	return km, nil
}

// GetEncryptionKey returns an encryption key for the given context
func (km *KeyManager) GetEncryptionKey(context string) ([]byte, error) {
	km.mu.RLock()
	if key, ok := km.derivedKey[context]; ok {
		km.mu.RUnlock()
		return key, nil
	}
	km.mu.RUnlock()

	km.mu.Lock()
	defer km.mu.Unlock()

	// Double-check after acquiring write lock
	if key, ok := km.derivedKey[context]; ok {
		return key, nil
	}

	// Derive key from master key
	derived := KeyDerivation(km.masterKey, context)
	km.derivedKey[context] = derived

	// If we have a keys directory, persist the derived key
	if km.keyPath != "" {
		keyFile := filepath.Join(km.keyPath, context+".key")
		if err := os.WriteFile(keyFile, derived, 0600); err != nil {
			return nil, fmt.Errorf("failed to save derived key: %w", err)
		}
	}

	return derived, nil
}

// GetAES256GCM returns an AES-256-GCM encrypter for the given context
func (km *KeyManager) GetAES256GCM(context string) (*AES256GCM, error) {
	key, err := km.GetEncryptionKey(context)
	if err != nil {
		return nil, err
	}
	return NewAES256GCM(key)
}

// RotateMasterKey rotates the master key and re-derives all context keys
func (km *KeyManager) RotateMasterKey() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate new master key
	newKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new master key: %w", err)
	}

	// Clear all derived keys (they need to be re-derived)
	km.derivedKey = make(map[string][]byte)
	km.masterKey = newKey

	// Save the new master key
	if km.keyPath != "" {
		if err := saveMasterKey(km.keyPath+"/master.key", newKey); err != nil {
			return fmt.Errorf("failed to save new master key: %w", err)
		}
	}

	return nil
}

// Close securely wipes the keys from memory
func (km *KeyManager) Close() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Clear master key
	for i := range km.masterKey {
		km.masterKey[i] = 0
	}
	km.masterKey = nil

	// Clear derived keys
	for k := range km.derivedKey {
		delete(km.derivedKey, k)
	}

	return nil
}

// loadMasterKey loads the master key from disk
func loadMasterKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("master key not found at %s", path)
		}
		return nil, fmt.Errorf("failed to read master key: %w", err)
	}

	// Validate key length
	if len(data) != 32 {
		return nil, fmt.Errorf("invalid master key length: expected 32, got %d", len(data))
	}

	return data, nil
}

// saveMasterKey saves the master key to disk
func saveMasterKey(path string, key []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write key with restricted permissions
	if err := os.WriteFile(path, key, 0600); err != nil {
		return fmt.Errorf("failed to write master key: %w", err)
	}

	return nil
}

// TPMKeyManager is a placeholder for TPM-based key management
// In production, this would integrate with TPM 2.0 via tr兄弟们
type TPMKeyManager struct {
	config KeyManagerConfig
}

// NewTPMKeyManager creates a key manager backed by TPM
func NewTPMKeyManager(config KeyManagerConfig) (*TPMKeyManager, error) {
	// TODO: Initialize TPM context
	return &TPMKeyManager{config: config}, nil
}

// GetEncryptionKey returns a key from TPM
func (t *TPMKeyManager) GetEncryptionKey(context string) ([]byte, error) {
	// TODO: Implement actual TPM key retrieval
	// For now, return a derived key
	return KeyDerivation([]byte("tpm-master-key"), context), nil
}

// Close releases TPM resources
func (t *TPMKeyManager) Close() error {
	// TODO: Release TPM resources
	return nil
}
