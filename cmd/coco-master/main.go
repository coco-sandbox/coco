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
	"strings"
	"syscall"
	"time"

	"coco/pkg/api/v1/v1connect"
	"coco/pkg/checkpoint"
	"coco/pkg/cluster"
	"coco/pkg/config"
	"coco/pkg/metrics"
	"coco/pkg/scheduler"
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

	// Initialize checkpoint store and manager for failover
	cpStore, err := checkpoint.NewStore("/var/lib/coco/checkpoints")
	if err != nil {
		return fmt.Errorf("failed to create checkpoint store: %w", err)
	}
	cpManager := checkpoint.NewCheckpointManager("/var/lib/coco/checkpoints", cpStore)

	fm := NewFailoverManager(sched, cpManager)
	go fm.Start(ctx)

	masterServer := NewMasterServer(sched, fm)

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

		discovery, err := cluster.NewDiscovery(cluster.DiscoveryConfig{
			Endpoints:  cfg.EtcdEndpoints,
			NodePrefix: "/coco/nodes",
			TTL:        30 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to create discovery: %w", err)
		}
		defer discovery.Close()

		if existing, err := discovery.ListNodes(ctx); err == nil {
			for _, ni := range existing {
				record, perr := cluster.ParseNodeRecord([]byte(ni.Value))
				if perr != nil {
					log.Printf("seed: parse error for %s: %v", ni.Key, perr)
					continue
				}
				_ = sched.RegisterNode(&scheduler.NodeEntry{
					ID:        record.ID,
					Addr:      record.Addr,
					Available: true,
				})
			}
			log.Printf("seeded scheduler with %d existing nodes", len(existing))
		}

		go discovery.WatchNodes(ctx, func(ev cluster.Event) {
			nodeID := strings.TrimPrefix(ev.Key, "/coco/nodes/")
			switch ev.Type {
			case "put":
				record, perr := cluster.ParseNodeRecord([]byte(ev.Value))
				if perr != nil {
					log.Printf("watch: parse error: %v", perr)
					return
				}
				// Check if node already exists, if so update its load
				if _, err := sched.GetNode(record.ID); err == nil {
					// Node exists, update load via heartbeat report
					sched.UpdateLoadFromReport(&scheduler.LoadReport{
						NodeID:    record.ID,
						Sandboxes: 0, // Heartbeats don't include sandbox count; preserve existing
						MemUsedMB: record.MemoryMB,
						CPUs:      record.CPUCount,
						Timestamp: time.Now(),
					})
				} else {
					// New node, register it
					_ = sched.RegisterNode(&scheduler.NodeEntry{
						ID:        record.ID,
						Addr:      record.Addr,
						Available: true,
					})
				}
			case "delete":
				sched.DeregisterNode(nodeID)
			}
		})
	} else {
		log.Printf("warning: COCO_ETCD_ENDPOINTS empty; running without leader election")
	}

	mux := http.NewServeMux()
	path, handler := v1connect.NewMasterServiceHandler(masterServer)
	mux.Handle(path, handler)

	metrics.Register()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health/live", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.Handle("/metrics", metrics.Handler())

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
