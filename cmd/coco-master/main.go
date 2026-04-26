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

	"github.com/coco-sandbox/coco/pkg/config"
	"github.com/coco-sandbox/coco/pkg/scheduler"
	"google.golang.org/grpc"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := config.Load()

	log.Printf("coco-master starting (listen_addr=%s)", cfg.GRPCAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx, cfg); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	sched := scheduler.NewScheduler()

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.GRPCAddr, err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	masterServer := NewMasterServer(sched)

	// Register gRPC service once generated code is available
	// pb.RegisterGatewayServiceServer(grpcServer, masterServer)

	log.Printf("gRPC server listening on %s", cfg.GRPCAddr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	_ = masterServer

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down", sig)
	grpcServer.GracefulStop()

	return nil
}
