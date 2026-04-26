// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/compress"
	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/image"
	"github.com/coco-sandbox/coco/daemon/coco-checkpoint/restore"
)

type CheckpointConfig struct {
	CheckpointDir    string
	Compression      string
	CompressionLevel int
	Incremental      bool
}

type Checkpoint struct {
	ID        string
	SandboxID string
	Name      string
	Path      string
	SizeBytes int64
	CreatedAt time.Time
}

type CheckpointManager struct {
	checkpointDir string
	imageBuilder  *image.Builder
	compressor    *compress.Compressor
	restorer      *restore.Restorer
}

func NewCheckpointManager(cfg CheckpointConfig) (*CheckpointManager, error) {
	if err := os.MkdirAll(cfg.CheckpointDir, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}
	return &CheckpointManager{
		checkpointDir: cfg.CheckpointDir,
		imageBuilder:  image.NewBuilder(cfg.CheckpointDir),
		compressor:    compress.NewCompressor(compress.CompressionType(cfg.Compression), cfg.CompressionLevel),
		restorer:      restore.NewRestorer(cfg.CheckpointDir),
	}, nil
}

func (cm *CheckpointManager) Create(ctx context.Context, sandboxID, name string) (*Checkpoint, error) {
	id := fmt.Sprintf("cp-%d", time.Now().UnixNano())
	path := filepath.Join(cm.checkpointDir, sandboxID, id)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint path: %w", err)
	}
	return &Checkpoint{
		ID:        id,
		SandboxID: sandboxID,
		Name:      name,
		Path:      path,
		CreatedAt: time.Now(),
	}, nil
}

func (cm *CheckpointManager) List(sandboxID string) ([]Checkpoint, error) { return nil, nil }
func (cm *CheckpointManager) Delete(id string) error                      { return nil }

func main() {
	log.SetFlags(0)
	flag.Parse()
	log.Printf("coco-checkpoint started")
	<-make(chan struct{})
}
