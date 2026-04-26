// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/pkg/checkpoint"
	v1 "github.com/coco-sandbox/coco/pkg/api/v1"
	"github.com/coco-sandbox/coco/pkg/api/v1/v1connect"
	"github.com/coco-sandbox/coco/pkg/types"
	connect "connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	// Create and register the checkpoint service server
	server := &CheckpointServer{
		mgr: mgr,
	}

	// HTTP server with connectrpc
	mux := http.NewServeMux()
	path, handler := v1connect.NewCheckpointServiceHandler(server)
	mux.Handle(path, handler)

	// Readiness probe
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ready","checkpoint_dir":%q}`, cfg.CheckpointDir)
	})

	// Liveness probe
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"alive"}`)
	})

	h2server := &http2.Server{}
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: h2c.NewHandler(mux, h2server),
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("checkpoint service listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Printf("coco-checkpoint ready")
	<-sigChan
	log.Println("Shutting down")
}

type Config struct {
	ListenAddr       string
	CheckpointDir    string
	Compression      string
	CompressionLevel int
}

func parseFlags() *Config {
	listen := flag.String("listen", ":46001", "gRPC listen address")
	dir := flag.String("dir", "/var/lib/coco/checkpoints", "Checkpoint directory")
	comp := flag.String("compress", "zstd", "Compression type (zstd, gzip, lz4)")
	level := flag.Int("level", 3, "Compression level (1=fast, 19=max)")

	flag.Parse()

	return &Config{
		ListenAddr:       *listen,
		CheckpointDir:    *dir,
		Compression:      *comp,
		CompressionLevel: *level,
	}
}

// CheckpointServer implements the CheckpointService connect RPC server.
type CheckpointServer struct {
	mgr *checkpoint.CheckpointManager
}

var _ v1connect.CheckpointServiceHandler = (*CheckpointServer)(nil)

func (s *CheckpointServer) Create(ctx context.Context, req *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	sandboxID := req.Msg.GetSandboxId()
	name := req.Msg.GetName()
	if sandboxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("sandbox_id is required"))
	}
	if name == "" {
		name = fmt.Sprintf("auto-%d", time.Now().Unix())
	}

	cp, err := s.mgr.Create(ctx, sandboxID, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Estimate sizes from checkpoint directory
	var sizeBytes, memSizeBytes int64
	if info, err := os.Stat(cp.Path); err == nil {
		sizeBytes = info.Size()
	}
	memPath := checkpoint.MemoryImagePath(cp)
	if info, err := os.Stat(memPath); err == nil {
		memSizeBytes = info.Size()
	}

	return connect.NewResponse(&v1.CreateResponse{
		CheckpointId:    cp.ID,
		SizeBytes:      sizeBytes,
		MemorySizeBytes: memSizeBytes,
		DiskSizeBytes:  sizeBytes - memSizeBytes,
		DurationMs:     0, // CRIU timing would go here
	}), nil
}

func (s *CheckpointServer) List(ctx context.Context, req *connect.Request[v1.ListRequest]) (*connect.Response[v1.ListResponse], error) {
	sandboxID := req.Msg.GetSandboxId()
	checkpoints, err := s.mgr.List(sandboxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	details := make([]*v1.CheckpointDetail, 0, len(checkpoints))
	for _, cp := range checkpoints {
		details = append(details, checkpointToDetail(cp))
	}

	return connect.NewResponse(&v1.ListResponse{
		Checkpoints: details,
	}), nil
}

func (s *CheckpointServer) Delete(ctx context.Context, req *connect.Request[v1.DeleteRequest]) (*connect.Response[v1.DeleteResponse], error) {
	checkpointID := req.Msg.GetCheckpointId()
	if checkpointID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("checkpoint_id is required"))
	}

	if err := s.mgr.Delete(checkpointID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteResponse{}), nil
}

func (s *CheckpointServer) Restore(ctx context.Context, req *connect.Request[v1.RestoreRequest]) (*connect.Response[v1.RestoreResponse], error) {
	checkpointID := req.Msg.GetCheckpointId()
	targetSandboxID := req.Msg.GetTargetSandboxId()
	if checkpointID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("checkpoint_id is required"))
	}

	cp, err := s.mgr.Get(checkpointID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Delegate to checkpoint manager for actual CRIU restore
	targetID := targetSandboxID
	if targetID == "" {
		targetID = cp.SandboxID
	}

	if err := s.mgr.Restore(ctx, cp.SandboxID, targetID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.RestoreResponse{
		SandboxId:  targetID,
		VsockCid:   0, // Would be assigned by visor
		DurationMs: 0, // CRIU timing would go here
	}), nil
}

func (s *CheckpointServer) GetStatus(ctx context.Context, req *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error) {
	checkpointID := req.Msg.GetCheckpointId()
	if checkpointID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("checkpoint_id is required"))
	}

	cp, err := s.mgr.Get(checkpointID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetStatusResponse{
		Checkpoint:      checkpointToDetail(cp),
		Stage:            v1.CheckpointStage_CHECKPOINT_STAGE_FINALIZING,
		ProgressPercent:  100,
		BytesWritten:     cp.SizeBytes,
		TotalBytes:       cp.SizeBytes,
	}), nil
}

func checkpointToDetail(cp *types.Checkpoint) *v1.CheckpointDetail {
	comp := v1.CheckpointCompression_CHECKPOINT_COMPRESSION_ZSTD
	if cp.Compression != "" {
		switch cp.Compression {
		case "zstd":
			comp = v1.CheckpointCompression_CHECKPOINT_COMPRESSION_ZSTD
		case "none":
			comp = v1.CheckpointCompression_CHECKPOINT_COMPRESSION_NONE
		}
	}

	return &v1.CheckpointDetail{
		Id:                 cp.ID,
		SandboxId:          cp.SandboxID,
		Name:               cp.Name,
		Path:               cp.Path,
		SizeBytes:          cp.SizeBytes,
		IsCompressed:       cp.Compression != "" && cp.Compression != "none",
		Compression:        comp,
		MemorySizeBytes:    int64(cp.MemoryDiffMB) * 1024 * 1024,
		DiskSizeBytes:      int64(cp.StateSizeKB) * 1024,
		Incremental:         false,
		ParentCheckpointId: cp.ParentID,
	}
}
