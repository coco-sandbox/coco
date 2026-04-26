package restore

import (
	"context"
	"fmt"
	"os"
)

type Restorer struct {
	baseDir string
}

func NewRestorer(baseDir string) *Restorer {
	return &Restorer{baseDir: baseDir}
}

func (r *Restorer) Restore(ctx context.Context, memoryImage, sandboxID, targetNode string) error {
	fmt.Printf("Restoring sandbox %s to node %s\n", sandboxID, targetNode)

	if _, err := os.Stat(memoryImage); os.IsNotExist(err) {
		return fmt.Errorf("memory image not found: %s", memoryImage)
	}

	return nil
}

func (r *Restorer) GetRestoreStatus(sandboxID string) (string, error) {
	return "unknown", nil
}
