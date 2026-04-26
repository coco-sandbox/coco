package compress

import (
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

func compressZstd(src, dst string, level int) error {
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

	encoder, err := zstd.NewWriter(dstFile, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
	if err != nil {
		return err
	}
	defer encoder.Close()

	_, err = io.Copy(encoder, srcFile)
	if err != nil {
		return err
	}

	return encoder.Close()
}

func decompressZstd(src, dst string) error {
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

	decoder, err := zstd.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer decoder.Close()

	_, err = io.Copy(dstFile, decoder)
	return err
}
