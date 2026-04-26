// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionZstd CompressionType = "zstd"
)

type Compressor struct {
	compressionType  CompressionType
	compressionLevel int
}

func NewCompressor(compressionType CompressionType, level int) *Compressor {
	if level == 0 {
		level = 3
	}
	return &Compressor{
		compressionType:  compressionType,
		compressionLevel: level,
	}
}

func (c *Compressor) CompressFile(src, dst string) error {
	if c.compressionType == CompressionNone {
		return copyFile(src, dst)
	}

	switch c.compressionType {
	case CompressionGzip:
		return compressGzip(src, dst, c.compressionLevel)
	case CompressionZstd:
		return compressZstd(src, dst, c.compressionLevel)
	default:
		return fmt.Errorf("unsupported compression type: %s", c.compressionType)
	}
}

func (c *Compressor) DecompressFile(src, dst string) error {
	if c.compressionType == CompressionNone {
		return copyFile(src, dst)
	}

	switch c.compressionType {
	case CompressionGzip:
		return decompressGzip(src, dst)
	case CompressionZstd:
		return decompressZstd(src, dst)
	default:
		return fmt.Errorf("unsupported compression type: %s", c.compressionType)
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func compressGzip(src, dst string, level int) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter, err := gzip.NewWriterLevel(dstFile, level)
	if err != nil {
		return err
	}
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, srcFile)
	return err
}

func decompressGzip(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, gzReader)
	return err
}
