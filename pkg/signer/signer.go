// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package signer

import (
	"crypto/ed25519"
	"fmt"
)

// PrivateKey is an Ed25519 private key
type PrivateKey ed25519.PrivateKey

// PublicKey is an Ed25519 public key
type PublicKey = ed25519.PublicKey

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (PrivateKey, PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	return PrivateKey(priv), pub, nil
}

// Sign signs a message with the private key
func Sign(priv PrivateKey, message []byte) ([]byte, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(priv), message)
	return sig, nil
}

// Verify verifies a signature with the public key
func Verify(pub PublicKey, message, signature []byte) bool {
	return ed25519.Verify(pub, message, signature)
}

// SignHash signs a hash (pre-hashed message)
func SignHash(priv PrivateKey, hash []byte) ([]byte, error) {
	return Sign(priv, hash)
}

// VerifyHash verifies a signature on a hash
func VerifyHash(pub PublicKey, hash, signature []byte) bool {
	return Verify(pub, hash, signature)
}
