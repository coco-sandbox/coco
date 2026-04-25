// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package crypto

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// SnapshotHeader is the header for an encrypted snapshot
type SnapshotHeader struct {
	Magic         uint64 // Magic number: 0x434F434F4658 ("COCOFX")
	Version       uint32 // Format version
	Flags         uint32 // Flags (encrypted, compressed, etc.)
	EncryptedLen  uint64 // Length of encrypted data
	OriginalLen   uint64 // Original length before encryption
	Checksum      [32]byte // SHA-256 checksum of encrypted data
	Nonce         [12]byte // GCM nonce
	Reserved      [16]byte // Reserved for future use
}

const (
	SnapshotMagic   = 0x434F434F4658
	SnapshotVersion = 1

	// Flags
	FlagEncrypted   = 1 << 0
	FlagCompressed  = 1 << 1
	FlagIntegrity   = 1 << 2
)

// EncryptedSnapshot wraps a snapshot with encryption
type EncryptedSnapshot struct {
	Header SnapshotHeader
	Data   []byte
}

// NewEncryptedSnapshot creates a new encrypted snapshot from plaintext
func NewEncryptedSnapshot(plaintext []byte, enc *AES256GCM) (*EncryptedSnapshot, error) {
	// Encrypt the data
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	// Create header
	header := SnapshotHeader{
		Magic:        SnapshotMagic,
		Version:      SnapshotVersion,
		Flags:        FlagEncrypted | FlagIntegrity,
		EncryptedLen: uint64(len(ciphertext)),
		OriginalLen:  uint64(len(plaintext)),
	}

	// Compute checksum
	hash := sha256.Sum256(ciphertext)
	header.Checksum = hash

	// Copy nonce (last 12 bytes of ciphertext for GCM)
	if len(ciphertext) >= 12 {
		copy(header.Nonce[:], ciphertext[len(ciphertext)-12:])
	}

	return &EncryptedSnapshot{
		Header: header,
		Data:   ciphertext,
	}, nil
}

// Decrypt decrypts an encrypted snapshot
func (es *EncryptedSnapshot) Decrypt(enc *AES256GCM) ([]byte, error) {
	if es.Header.Magic != SnapshotMagic {
		return nil, fmt.Errorf("invalid snapshot magic: expected 0x%x, got 0x%x", SnapshotMagic, es.Header.Magic)
	}

	if es.Header.Version != SnapshotVersion {
		return nil, fmt.Errorf("unsupported snapshot version: %d", es.Header.Version)
	}

	if es.Header.Flags&FlagEncrypted == 0 {
		return nil, fmt.Errorf("snapshot is not encrypted")
	}

	// Verify checksum
	hash := sha256.Sum256(es.Data)
	if hash != es.Header.Checksum {
		return nil, fmt.Errorf("checksum mismatch")
	}

	// Decrypt
	plaintext, err := enc.Decrypt(es.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// WriteTo writes the encrypted snapshot to a file
func (es *EncryptedSnapshot) WriteTo(w io.Writer) error {
	// Write header
	headerBytes := make([]byte, 96) // Size of SnapshotHeader
	binary.LittleEndian.PutUint64(headerBytes[0:8], es.Header.Magic)
	binary.LittleEndian.PutUint32(headerBytes[8:12], es.Header.Version)
	binary.LittleEndian.PutUint32(headerBytes[12:16], es.Header.Flags)
	binary.LittleEndian.PutUint64(headerBytes[16:24], es.Header.EncryptedLen)
	binary.LittleEndian.PutUint64(headerBytes[24:32], es.Header.OriginalLen)
	copy(headerBytes[32:64], es.Header.Checksum[:])
	copy(headerBytes[64:76], es.Header.Nonce[:])

	if _, err := w.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	if _, err := w.Write(es.Data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// ReadFrom reads an encrypted snapshot from a file
func ReadFrom(r io.Reader) (*EncryptedSnapshot, error) {
	// Read header
	headerBytes := make([]byte, 96)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var header SnapshotHeader
	header.Magic = binary.LittleEndian.Uint64(headerBytes[0:8])
	header.Version = binary.LittleEndian.Uint32(headerBytes[8:12])
	header.Flags = binary.LittleEndian.Uint32(headerBytes[12:16])
	header.EncryptedLen = binary.LittleEndian.Uint64(headerBytes[16:24])
	header.OriginalLen = binary.LittleEndian.Uint64(headerBytes[24:32])
	copy(header.Checksum[:], headerBytes[32:64])
	copy(header.Nonce[:], headerBytes[64:76])

	// Read data
	data := make([]byte, header.EncryptedLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return &EncryptedSnapshot{
		Header: header,
		Data:   data,
	}, nil
}

// ReadFromFile reads an encrypted snapshot from a file
func ReadFromFile(path string) (*EncryptedSnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return ReadFrom(f)
}

// WriteToFile writes the encrypted snapshot to a file
func (es *EncryptedSnapshot) WriteToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	return es.WriteTo(f)
}
