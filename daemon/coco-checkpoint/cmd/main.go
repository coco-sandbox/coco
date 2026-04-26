package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := parseFlags()

	log.Printf("coco-checkpoint starting (dir=%s)", cfg.CheckpointDir)

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
	comp := flag.String("compress", "zstd", "Compression type")
	level := flag.Int("level", 3, "Compression level")

	flag.Parse()

	return &Config{
		CheckpointDir:    *dir,
		Compression:      *comp,
		CompressionLevel: *level,
	}
}
