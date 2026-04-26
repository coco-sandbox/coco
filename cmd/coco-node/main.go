// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/pkg/api/v1/v1connect"
	"github.com/coco-sandbox/coco/pkg/cluster"
	"github.com/coco-sandbox/coco/pkg/config"
	"github.com/coco-sandbox/coco/pkg/metrics"
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

	if len(cfg.EtcdEndpoints) > 0 {
		discovery, err := cluster.NewDiscovery(cluster.DiscoveryConfig{
			Endpoints:  cfg.EtcdEndpoints,
			NodePrefix: "/coco/nodes",
			TTL:        30 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to create discovery: %w", err)
		}
		defer discovery.Close()
		nodeServer.SetDiscovery(discovery)

		if err := discovery.RegisterNode(ctx, cfg.NodeID, cfg.GRPCAddr, nil); err != nil {
			return fmt.Errorf("failed to register node: %w", err)
		}
		log.Printf("registered node in etcd (node_id=%s, prefix=/coco/nodes)", cfg.NodeID)

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := discovery.RefreshLease(ctx, cfg.NodeID); err != nil {
						log.Printf("heartbeat error: %v", err)
					}
				}
			}
		}()
	} else {
		log.Printf("warning: COCO_ETCD_ENDPOINTS empty; node not registering in cluster")
	}

	path, handler := v1connect.NewNodeServiceHandler(nodeServer)
	mux.Handle(path, handler)

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

		checks := map[string]string{
			"node": "ok",
		}

		// Check visor connectivity by acquiring a client
		done := make(chan error, 1)
		go func() {
			_, err := visorPool.Acquire()
			done <- err
		}()
		select {
		case err := <-done:
			if err != nil {
				checks["visor"] = "unreachable"
			} else {
				checks["visor"] = "ok"
				visorPool.Release(nil) // release immediately
			}
		case <-time.After(2 * time.Second):
			checks["visor"] = "unreachable"
		}

		// Check storage by attempting a read operation
		if _, err := st.GetSandbox("__health_check__"); err != nil {
			// Not found is fine - storage is reachable
			checks["storage"] = "ok"
		} else {
			checks["storage"] = "ok"
		}

		status := "ready"
		for _, v := range checks {
			if v != "ok" {
				status = "degraded"
				break
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"checks": checks,
		})
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