// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package signer

import (
	"crypto/ed25519"
	"fmt"
)

// Verifier verifies signatures
type Verifier struct {
	publicKeys map[string]PublicKey
}

// NewVerifier creates a new Verifier with trusted public keys
func NewVerifier() *Verifier {
	return &Verifier{
		publicKeys: make(map[string]PublicKey),
	}
}

// AddKey adds a trusted public key by key ID
func (v *Verifier) AddKey(keyID string, pub PublicKey) {
	v.publicKeys[keyID] = pub
}

// RemoveKey removes a trusted public key
func (v *Verifier) RemoveKey(keyID string) {
	delete(v.publicKeys, keyID)
}

// Verify checks if a signature is valid for the message using the named key
func (v *Verifier) Verify(keyID string, message, signature []byte) error {
	pub, ok := v.publicKeys[keyID]
	if !ok {
		return fmt.Errorf("unknown key: %s", keyID)
	}

	if !ed25519.Verify(pub, message, signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// VerifyAny checks if the signature is valid for any of the registered keys
func (v *Verifier) VerifyAny(message, signature []byte) (bool, string) {
	for keyID, pub := range v.publicKeys {
		if ed25519.Verify(pub, message, signature) {
			return true, keyID
		}
	}
	return false, ""
}

// BatchVerify verifies multiple signatures
func (v *Verifier) BatchVerify(entries []VerifyEntry) []error {
	errors := make([]error, len(entries))
	for i, entry := range entries {
		if err := v.Verify(entry.KeyID, entry.Message, entry.Signature); err != nil {
			errors[i] = err
		}
	}
	return errors
}

// VerifyEntry is a message/signature pair to verify
type VerifyEntry struct {
	KeyID     string
	Message   []byte
	Signature []byte
}

// SignatureScheme defines the signature algorithm
type SignatureScheme int

const (
	Ed25519 Scheme = iota
)

// Scheme represents a signature scheme
type Scheme int

func (s Scheme) String() string {
	switch s {
	case Ed25519:
		return "ed25519"
	default:
		return "unknown"
	}
}
