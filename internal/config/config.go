// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package config provides configuration management for the Coco API server.
package config

import (
	"os"
	"time"
)

// Config holds all configuration for the Coco API server
type Config struct {
	// Server configuration
	ListenAddr       string
	GRPCAddr         string
	ShutdownTimeout  time.Duration

	// Data directories
	DataDir      string
	ImagesDir   string
	StoreDir    string
	Checkpoints string
	Hibernation string
	Replays    string

	// Runtime
	RuntimeDir string

	// Network
	VisorSocket string
	NetSocket   string

	// Cluster
	ClusterEnabled     bool
	ClusterPort       int
	ClusterPeers      []string
	ElectionTimeout   time.Duration
	HeartbeatInterval time.Duration

	// Rate limiting
	RateLimitEnabled bool
	RateLimitRPS     float64
	RateLimitBurst   int

	// Security
	AuthEnabled  bool
	MTLSEnabled bool
	TLSCertFile string
	TLSKeyFile  string

	// Metrics
	MetricsEnabled bool
	MetricsPort   int
}

// Default returns a Config with default values
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
		RuntimeDir:         "/run/coco",
		VisorSocket:       "/run/coco/visor.sock",
		NetSocket:         "/run/coco/net.sock",
		ClusterEnabled:    false,
		ClusterPort:       4748,
		ElectionTimeout:   5 * time.Second,
		HeartbeatInterval: 1 * time.Second,
		RateLimitEnabled:  true,
		RateLimitRPS:     100,
		RateLimitBurst:   200,
		AuthEnabled:      false,
		MTLSEnabled:      false,
		MetricsEnabled:   true,
		MetricsPort:      9090,
	}
}

// Load loads configuration from environment variables
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
	}
	if socket := os.Getenv("COCO_VISOR_SOCKET"); socket != "" {
		cfg.VisorSocket = socket
	}

	return cfg
}
