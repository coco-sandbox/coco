package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Builder struct {
	baseDir string
}

func NewBuilder(baseDir string) *Builder {
	return &Builder{baseDir: baseDir}
}

func (b *Builder) CreateImage(ctx context.Context, sandboxID, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	memoryImg := filepath.Join(outputDir, "memory.img")
	if _, err := os.Create(memoryImg); err != nil {
		return fmt.Errorf("failed to create memory image: %w", err)
	}

	metadata := map[string]interface{}{
		"sandbox_id": sandboxID,
		"version":    "1.0",
	}

	metadataFile := filepath.Join(outputDir, "metadata.json")
	metadataData := fmt.Sprintf(`{"sandbox_id": "%s", "version": "1.0"}`, sandboxID)
	if err := os.WriteFile(metadataFile, []byte(metadataData), 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	_ = metadata

	return nil
}

func (b *Builder) ListImages() ([]string, error) {
	entries, err := os.ReadDir(b.baseDir)
	if err != nil {
		return nil, err
	}

	images := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			images = append(images, entry.Name())
		}
	}

	return images, nil
}
