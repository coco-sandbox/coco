package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CheckpointManager struct {
	dir        string
	compressor Compressor
}

type Compressor interface {
	Compress(dst, src interface{}) error
	Decompress(dst, src interface{}) error
}

func NewCheckpointManager(dir string) *CheckpointManager {
	return &CheckpointManager{
		dir: dir,
	}
}

func (cm *CheckpointManager) Create(ctx context.Context, sandboxID, name string) (*Checkpoint, error) {
	id := fmt.Sprintf("cp-%d", time.Now().UnixNano())
	path := filepath.Join(cm.dir, sandboxID, id)

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}

	return &Checkpoint{
		ID:        id,
		SandboxID: sandboxID,
		Name:      name,
		Path:      path,
		CreatedAt: time.Now(),
	}, nil
}

func (cm *CheckpointManager) List(sandboxID string) ([]Checkpoint, error) {
	return nil, nil
}

func (cm *CheckpointManager) Get(id string) (*Checkpoint, error) {
	return nil, nil
}

func (cm *CheckpointManager) Delete(id string) error {
	return nil
}

type Checkpoint struct {
	ID        string
	SandboxID string
	Name      string
	Path      string
	SizeBytes int64
	CreatedAt time.Time
}
