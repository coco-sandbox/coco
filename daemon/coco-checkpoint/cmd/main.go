// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coco-sandbox/coco/pkg/checkpoint"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := parseFlags()
	log.Printf("coco-checkpoint starting (dir=%s, compress=%s, level=%d)",
		cfg.CheckpointDir, cfg.Compression, cfg.CompressionLevel)

	// Initialize store and manager
	store, err := checkpoint.NewStore(cfg.CheckpointDir)
	if err != nil {
		log.Fatalf("failed to create checkpoint store: %v", err)
	}

	mgr := checkpoint.NewCheckpointManager(cfg.CheckpointDir, store)
	mgr.SetCompressor(&checkpoint.ZstdCompressor{})

	// Health check: verify we can write to the checkpoint dir
	testDir := cfg.CheckpointDir + "/.health"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Fatalf("checkpoint dir not writable: %v", err)
	}
	os.RemoveAll(testDir)

	log.Printf("coco-checkpoint ready")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down")
}

type Config struct {
	CheckpointDir    string
	Compression      string
	CompressionLevel int
}

func parseFlags() *Config {
	dir := flag.String("dir", "/var/lib/coco/checkpoints", "Checkpoint directory")
	comp := flag.String("compress", "zstd", "Compression type (zstd, gzip, lz4)")
	level := flag.Int("level", 3, "Compression level (1=fast, 19=max)")

	flag.Parse()

	return &Config{
		CheckpointDir:    *dir,
		Compression:      *comp,
		CompressionLevel: *level,
	}
}

// serveCLI runs checkpoint operations via CLI.
// This would be replaced by gRPC service in full implementation.
func serveCLI(ctx context.Context, mgr *checkpoint.CheckpointManager, args []string) {
	switch args[0] {
	case "create":
		if len(args) < 3 {
			log.Fatalf("usage: create <sandbox_id> <name>")
		}
		sandboxID, name := args[1], args[2]
		cp, err := mgr.Create(ctx, sandboxID, name)
		if err != nil {
			log.Fatalf("create checkpoint: %v", err)
		}
		log.Printf("Created checkpoint %s for sandbox %s", cp.ID, sandboxID)

	case "list":
		if len(args) < 2 {
			log.Fatalf("usage: list <sandbox_id>")
		}
		sandboxes, err := mgr.List(args[1])
		if err != nil {
			log.Fatalf("list checkpoints: %v", err)
		}
		log.Printf("Found %d checkpoint(s)", len(sandboxes))
		for _, cp := range sandboxes {
			log.Printf("  %s  created=%s", cp.ID, cp.CreatedAt)
		}

	case "delete":
		if len(args) < 2 {
			log.Fatalf("usage: delete <checkpoint_id>")
		}
		if err := mgr.Delete(args[1]); err != nil {
			log.Fatalf("delete checkpoint: %v", err)
		}
		log.Printf("Deleted checkpoint %s", args[1])
	}
}
