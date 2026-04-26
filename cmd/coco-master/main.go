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
	"github.com/coco-sandbox/coco/pkg/scheduler"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	masterServer := NewMasterServer(sched)

	if len(cfg.EtcdEndpoints) > 0 {
		election, err := NewElection(cfg.EtcdEndpoints, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to create election: %w", err)
		}
		defer election.Close()
		masterServer.SetElection(election)
		go func() {
			if err := election.Start(ctx); err != nil {
				log.Printf("election error: %v", err)
			}
		}()
		log.Printf("etcd election started (endpoints=%v)", cfg.EtcdEndpoints)
	} else {
		log.Printf("warning: COCO_ETCD_ENDPOINTS empty; running without leader election")
	}

	mux := http.NewServeMux()
	path, handler := v1connect.NewMasterServiceHandler(masterServer)
	mux.Handle(path, handler)

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
