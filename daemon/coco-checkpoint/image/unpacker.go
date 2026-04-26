package image

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Unpacker struct {
	baseDir string
}

func NewUnpacker(baseDir string) *Unpacker {
	return &Unpacker{baseDir: baseDir}
}

func (u *Unpacker) UnpackImage(imagePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read image directory: %w", err)
	}

	for _, entry := range entries {
		src := filepath.Join(imagePath, entry.Name())
		dst := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
		}
	}

	return nil
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
