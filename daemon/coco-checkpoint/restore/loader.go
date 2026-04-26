package restore

import (
	"fmt"
	"os"
	"path/filepath"
)

type Loader struct {
	baseDir string
}

func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

func (l *Loader) LoadCheckpoint(checkpointID string) (*CheckpointInfo, error) {
	checkpointPath := filepath.Join(l.baseDir, checkpointID)

	info, err := os.Stat(checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("checkpoint not found: %w", err)
	}

	memoryPath := filepath.Join(checkpointPath, "memory.img")
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("memory image not found in checkpoint")
	}

	memoryInfo, _ := os.Stat(memoryPath)

	return &CheckpointInfo{
		ID:          checkpointID,
		Path:        checkpointPath,
		MemorySize:  memoryInfo.Size(),
		CreatedAt:   info.ModTime(),
	}, nil
}

type CheckpointInfo struct {
	ID        string
	Path      string
	MemorySize int64
	CreatedAt  os.FileInfo
}
