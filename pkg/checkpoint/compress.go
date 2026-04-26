// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package checkpoint

import (
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

// ZstdCompressor implements compression using zstd per spec §5.3.
type ZstdCompressor struct {
	level zstd.EncoderLevel
}

// NewZstdCompressor creates a zstd compressor with default compression level.
func NewZstdCompressor() (*ZstdCompressor, error) {
	return &ZstdCompressor{level: 3}, nil
}

// CompressFile compresses a file to the destination path.
func (c *ZstdCompressor) CompressFile(dstPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dst.Close()

	encoder, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(c.level))
	if err != nil {
		return fmt.Errorf("create encoder: %w", err)
	}

	if _, err := io.Copy(encoder, src); err != nil {
		return fmt.Errorf("compress: %w", err)
	}
	return encoder.Close()
}

// DecompressFile decompresses a file to the destination path.
func (c *ZstdCompressor) DecompressFile(dstPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dst.Close()

	decoder, err := zstd.NewReader(src)
	if err != nil {
		return fmt.Errorf("create decoder: %w", err)
	}
	defer decoder.Close()

	_, err = io.Copy(dst, decoder)
	return err
}