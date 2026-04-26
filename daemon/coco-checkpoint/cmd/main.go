package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	checkpointDir := flag.String("checkpoint-dir", "/var/lib/coco/checkpoints", "Checkpoint directory")
	compression := flag.String("compression", "zstd", "Compression type (none, gzip, zstd)")
	level := flag.Int("level", 3, "Compression level")
	flag.Parse()

	log.Printf("coco-checkpoint starting (checkpoint_dir=%s, compression=%s)", *checkpointDir, *compression)

	cfg := CheckpointConfig{
		CheckpointDir:    *checkpointDir,
		Compression:     *compression,
		CompressionLevel: *level,
	}

	manager, err := NewCheckpointManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down checkpoint manager")
		cancel()
	}()

	<-ctx.Done()
	fmt.Println("coco-checkpoint stopped")
}
