package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/compress"
	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/image"
	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/restore"
)

type CheckpointManager struct {
	checkpointDir string
	imageBuilder  *image.Builder
	compressor   *compress.Compressor
	restorer    *restore.Restorer
}

type CheckpointConfig struct {
	CheckpointDir   string
	Compression    string
	CompressionLevel int
	Incremental    bool
}

type Checkpoint struct {
	ID        string
	SandboxID string
	Name      string
	Path      string
	SizeBytes int64
	CreatedAt time.Time
}

func NewCheckpointManager(cfg CheckpointConfig) (*CheckpointManager, error) {
	if err := os.MkdirAll(cfg.CheckpointDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	comp := compress.NewCompressor(compress.CompressionType(cfg.Compression), cfg.CompressionLevel)
	imgBuilder := image.NewBuilder(cfg.CheckpointDir)
	restorer := restore.NewRestorer(cfg.CheckpointDir)

	return &CheckpointManager{
		checkpointDir: cfg.CheckpointDir,
		imageBuilder:  imgBuilder,
		compressor:   comp,
		restorer:    restorer,
	}, nil
}

func (cm *CheckpointManager) CreateCheckpoint(ctx context.Context, sandboxID, name string, incremental bool) (*Checkpoint, error) {
	checkpointID := fmt.Sprintf("cp-%d", time.Now().UnixNano())
	checkpointPath := filepath.Join(cm.checkpointDir, sandboxID, checkpointID)

	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	checkpoint := &Checkpoint{
		ID:        checkpointID,
		SandboxID: sandboxID,
		Name:      name,
		Path:      checkpointPath,
		CreatedAt: time.Now(),
	}

	if err := cm.imageBuilder.CreateImage(ctx, sandboxID, checkpointPath); err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	if err := cm.compressor.CompressFile(
		filepath.Join(checkpointPath, "memory.img"),
		filepath.Join(checkpointPath, "memory.img.zst"),
	); err != nil {
		return nil, fmt.Errorf("failed to compress memory: %w", err)
	}

	info, err := os.Stat(checkpointPath)
	if err != nil {
		checkpoint.SizeBytes = 0
	} else {
		checkpoint.SizeBytes = info.Size()
	}

	log.Printf("Created checkpoint %s for sandbox %s", checkpoint.ID, sandboxID)
	return checkpoint, nil
}

func (cm *CheckpointManager) RestoreCheckpoint(ctx context.Context, checkpointID, sandboxID, targetNode string) error {
	checkpointPath := filepath.Join(cm.checkpointDir, checkpointID)

	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		return fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	memoryPath := filepath.Join(checkpointPath, "memory.img.zst")
	decompressedPath := filepath.Join(checkpointPath, "memory.img")

	if err := cm.compressor.DecompressFile(memoryPath, decompressedPath); err != nil {
		return fmt.Errorf("failed to decompress memory: %w", err)
	}

	if err := cm.restorer.Restore(ctx, decompressedPath, sandboxID, targetNode); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	log.Printf("Restored checkpoint %s to sandbox %s on node %s", checkpointID, sandboxID, targetNode)
	return nil
}

func (cm *CheckpointManager) ListCheckpoints(sandboxID string) ([]Checkpoint, error) {
	sandboxDir := filepath.Join(cm.checkpointDir, sandboxID)

	entries, err := os.ReadDir(sandboxDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	checkpoints := make([]Checkpoint, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		checkpointPath := filepath.Join(sandboxDir, entry.Name())
		sizeBytes := int64(0)

		filepath.Walk(checkpointPath, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				sizeBytes += info.Size()
			}
			return nil
		})

		checkpoints = append(checkpoints, Checkpoint{
			ID:        entry.Name(),
			SandboxID: sandboxID,
			Path:      checkpointPath,
			SizeBytes: sizeBytes,
			CreatedAt: info.ModTime(),
		})
	}

	return checkpoints, nil
}

func (cm *CheckpointManager) DeleteCheckpoint(checkpointID string) error {
	checkpointPath := filepath.Join(cm.checkpointDir, checkpointID)

	if err := os.RemoveAll(checkpointPath); err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	log.Printf("Deleted checkpoint %s", checkpointID)
	return nil
}