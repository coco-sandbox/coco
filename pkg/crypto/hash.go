// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
)

// Hash256 represents a SHA-256 hash
type Hash256 [32]byte

// NewHash creates a new Hash256 hasher
func NewHash() hash.Hash {
	return sha256.New()
}

// Sum computes SHA-256 hash and returns raw bytes
func Sum(data []byte) Hash256 {
	h := sha256.Sum256(data)
	return h
}

// SumHex computes SHA-256 hash and returns hex string
func SumHex(data []byte) string {
	h := Sum(data)
	return hex.EncodeToString(h[:])
}

// Verify checks if data matches the expected hash
func Verify(data []byte, expected Hash256) bool {
	actual := Sum(data)
	return actual == expected
}

// HashWriter wraps a hash for streaming writes
type HashWriter struct {
	h hash.Hash
}

// NewHashWriter creates a new HashWriter
func NewHashWriter() *HashWriter {
	return &HashWriter{h: sha256.New()}
}

// Write adds data to the hash
func (w *HashWriter) Write(data []byte) (int, error) {
	return w.h.Write(data)
}

// Sum returns the final hash
func (w *HashWriter) Sum() Hash256 {
	var h [32]byte
	copy(h[:], w.h.Sum(nil))
	return Hash256(h)
}

// SumHex returns the final hash as hex string
func (w *HashWriter) SumHex() string {
	return hex.EncodeToString(w.h.Sum(nil))
}

// Checksum computes a simple checksum (XOR of all bytes)
// Used for quick integrity checks, not cryptographic
func Checksum(data []byte) uint8 {
	var sum uint8
	for _, b := range data {
		sum ^= b
	}
	return sum
}

// VerifyChecksum checks if data matches expected checksum
func VerifyChecksum(data []byte, expected uint8) bool {
	return Checksum(data) == expected
}

// HashPair holds two hashes for merkle tree operations
type HashPair struct {
	Left  Hash256
	Right Hash256
}

// Combine combines two hashes into one (for merkle trees)
func Combine(left, right Hash256) Hash256 {
	h := sha256.New()
	h.Write(left[:])
	h.Write(right[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// Marshal serializes a Hash256 to bytes
func (h Hash256) Marshal() []byte {
	return h[:]
}

// Unmarshal deserializes bytes to a Hash256
func (h *Hash256) Unmarshal(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid hash length: %d", len(data))
	}
	copy(h[:], data)
	return nil
}