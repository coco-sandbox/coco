// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr      string
	GRPCAddr        string
	ShutdownTimeout time.Duration

	DataDir     string
	ImagesDir   string
	StoreDir    string
	Checkpoints string
	Hibernation string
	Replays     string
	Templates   string
	SnapshotDir string

	RuntimeDir string

	VisorSocket string
	NetSocket   string

	ClusterEnabled    bool
	ClusterPort       int
	ClusterPeers      []string
	ElectionTimeout   time.Duration
	HeartbeatInterval time.Duration

	RateLimitEnabled bool
	RateLimitRPS     float64
	RateLimitBurst   int

	AuthEnabled bool
	APIKeys     map[string]string
	MTLSEnabled bool
	TLSCertFile string
	TLSKeyFile  string

	MetricsEnabled bool
	MetricsPort    int

	NodeID     string
	PoolSize   int
	MasterAddr string

	SchedulerStrategy string

	EtcdEndpoints []string
}

func Default() *Config {
	return &Config{
		ListenAddr:        ":4747",
		GRPCAddr:          ":4748",
		ShutdownTimeout:   30 * time.Second,
		DataDir:           "/var/lib/coco",
		ImagesDir:         "/var/lib/coco/images",
		StoreDir:          "/var/lib/coco/store",
		Checkpoints:       "/var/lib/coco/checkpoints",
		Hibernation:       "/var/lib/coco/hibernation",
		Replays:           "/var/lib/coco/replays",
		Templates:         "/var/lib/coco/templates",
		SnapshotDir:       "/var/lib/coco/snapshots",
		RuntimeDir:        "/run/coco",
		VisorSocket:       "/run/coco/visor.sock",
		NetSocket:         "/run/coco/net.sock",
		ClusterEnabled:    false,
		ClusterPort:       4748,
		ElectionTimeout:   5 * time.Second,
		HeartbeatInterval: 1 * time.Second,
		RateLimitEnabled:  true,
		RateLimitRPS:      100,
		RateLimitBurst:    200,
		AuthEnabled:       false,
		APIKeys:           make(map[string]string),
		MTLSEnabled:       false,
		MetricsEnabled:    true,
		MetricsPort:       9090,
		NodeID:            func() string { h, _ := os.Hostname(); return h }(),
		PoolSize:          5,
		MasterAddr:        ":4746",
		SchedulerStrategy: "least-loaded",
	}
}

func Load() *Config {
	cfg := Default()

	if addr := os.Getenv("COCO_LISTEN_ADDR"); addr != "" {
		cfg.ListenAddr = addr
	}
	if dir := os.Getenv("COCO_DATA_DIR"); dir != "" {
		cfg.DataDir = dir
		cfg.ImagesDir = dir + "/images"
		cfg.StoreDir = dir + "/store"
		cfg.Checkpoints = dir + "/checkpoints"
		cfg.Hibernation = dir + "/hibernation"
		cfg.Replays = dir + "/replays"
		cfg.Templates = dir + "/templates"
		cfg.SnapshotDir = dir + "/snapshots"
	}
	if socket := os.Getenv("COCO_VISOR_SOCKET"); socket != "" {
		cfg.VisorSocket = socket
	}
	if master := os.Getenv("COCO_MASTER_ADDR"); master != "" {
		cfg.MasterAddr = master
	}
	if nodeID := os.Getenv("COCO_NODE_ID"); nodeID != "" {
		cfg.NodeID = nodeID
	}
	if poolSize := os.Getenv("COCO_POOL_SIZE"); poolSize != "" {
		if n, err := strconv.Atoi(poolSize); err == nil {
			cfg.PoolSize = n
		}
	}
	if strategy := os.Getenv("COCO_SCHEDULER_STRATEGY"); strategy != "" {
		cfg.SchedulerStrategy = strategy
	}
	if endpoints := os.Getenv("COCO_ETCD_ENDPOINTS"); endpoints != "" {
		cfg.EtcdEndpoints = splitCSV(endpoints)
	}
	if keys := os.Getenv("COCO_API_KEYS"); keys != "" {
		for _, pair := range splitCSV(keys) {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				cfg.APIKeys[parts[0]] = parts[1]
			}
		}
	}

	return cfg
}

func splitCSV(s string) []string {
	out := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}
