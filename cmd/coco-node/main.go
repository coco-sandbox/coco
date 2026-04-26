// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/pkg/config"
	"github.com/coco-sandbox/coco/pkg/pool"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/visor"
	"google.golang.org/grpc"
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

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.GRPCAddr, err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	nodeServer := NewNodeServer(cfg.NodeID, cfg.GRPCAddr, st, vmPool, visorPool)

	_ = nodeServer
	// Register gRPC service once generated code is available
	// pb.RegisterNodeServiceServer(grpcServer, nodeServer)

	log.Printf("gRPC server listening on %s", cfg.GRPCAddr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down", sig)
	grpcServer.GracefulStop()

	return nil
}
