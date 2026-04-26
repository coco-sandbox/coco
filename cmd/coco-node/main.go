// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/pkg/api/v1/v1connect"
	"github.com/coco-sandbox/coco/pkg/config"
	"github.com/coco-sandbox/coco/pkg/pool"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/visor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := config.Load()
	if cfg.NodeID == "" {
		hostname, _ := os.Hostname()
		cfg.NodeID = hostname
	}

	log.Printf("coco-node starting (node_id=%s, visor_socket=%s)", cfg.NodeID, cfg.VisorSocket)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx, cfg); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	st, err := store.NewBadgerStore(cfg.StoreDir)
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}
	defer st.Close()

	visorPool := visor.NewPool(cfg.VisorSocket, 10)
	defer visorPool.Close()

	poolCfg := pool.PoolConfig{
		TargetSize:   cfg.PoolSize,
		MaxWaitTime:  5 * time.Second,
		RefillTicker: 30 * time.Second,
		DefaultMem:   512,
		DefaultVCPU:  2,
	}
	vmPool := pool.NewPool(visorPool, poolCfg)
	defer vmPool.Stop()

	log.Printf("VM pool initialized (target_size=%d)", cfg.PoolSize)

	mux := http.NewServeMux()
	nodeServer := NewNodeServer(cfg.NodeID, cfg.GRPCAddr, st, vmPool, visorPool)
	path, handler := v1connect.NewNodeServiceHandler(nodeServer)
	mux.Handle(path, handler)
	mux.HandleFunc("/health", healthHandler)

	httpServer := &http.Server{
		Addr:    cfg.GRPCAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.Printf("ConnectRPC server listening on %s", cfg.GRPCAddr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	<-sigChan
	log.Printf("Shutting down")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	return httpServer.Shutdown(ctx2)
}
