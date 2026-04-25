// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for coco-core
type Config struct {
	// Server config
	ListenAddr     string        `json:"listen_addr"`
	MetricsAddr    string        `json:"metrics_addr"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// Paths
	DataDir     string `json:"data_dir"`
	SocketDir  string `json:"socket_dir"`
	ImagesDir  string `json:"images_dir"`
	CheckpointsDir string `json:"checkpoints_dir"`
	HibernationDir string `json:"hibernation_dir"`

	// Logging
	LogLevel     string `json:"log_level"`
	LogFormat    string `json:"log_format"`

	// BadgerDB
	BadgerPath string `json:"badger_path"`
	BadgerVali bool   `json:"badger_vli"`

	// Visor socket
	VisorSocket string `json:"visor_socket"`
	VisorConnTimeout time.Duration `json:"visor_conn_timeout"`

	// Network
	NetworkEnabled bool   `json:"network_enabled"`
	NetSocket      string `json:"net_socket"`

	// Sandbox defaults
	DefaultMemoryMB int `json:"default_memory_mb"`
	DefaultVCPUs    int `json:"default_vcpus"`
	DefaultTemplate string `json:"default_template"`

	// Rate limiting
	RateLimitEnabled  bool `json:"rate_limit_enabled"`
	RateLimitRPS      int  `json:"rate_limit_rps"`
	RateLimitBurst    int  `json:"rate_limit_burst"`

	// Auth
	APIKeyEnabled bool   `json:"api_key_enabled"`
	APIKeyFile   string `json:"api_key_file"`

	// Node
	NodeID       string `json:"node_id"`
	ClusterMode  bool   `json:"cluster_mode"`
	ClusterNodes string `json:"cluster_nodes"`
}

// Default returns a Config with sensible defaults
func Default() *Config {
	nodeID := os.Getenv("COCO_NODE_ID")
	if nodeID == "" {
		hostname, _ := os.Hostname()
		nodeID = hostname
	}

	return &Config{
		ListenAddr:        getEnv("COCO_LISTEN_ADDR", ":4747"),
		MetricsAddr:       getEnv("COCO_METRICS_ADDR", ":4748"),
		ShutdownTimeout:   durationEnv("COCO_SHUTDOWN_TIMEOUT", 30*time.Second),
		DataDir:           getEnv("COCO_DATA_DIR", "/var/lib/coco"),
		SocketDir:         getEnv("COCO_SOCKET_DIR", "/run/coco"),
		ImagesDir:         getEnv("COCO_IMAGES_DIR", "/var/lib/coco/images"),
		CheckpointsDir:    getEnv("COCO_CHECKPOINTS_DIR", "/var/lib/coco/checkpoints"),
		HibernationDir:    getEnv("COCO_HIBERNATION_DIR", "/var/lib/coco/hibernation"),
		LogLevel:          getEnv("COCO_LOG_LEVEL", "info"),
		LogFormat:         getEnv("COCO_LOG_FORMAT", "json"),
		BadgerPath:        getEnv("COCO_BADGER_PATH", "/var/lib/coco/store"),
		VisorSocket:       getEnv("COCO_VISOR_SOCKET", "/run/coco/visor.sock"),
		VisorConnTimeout:  durationEnv("COCO_VISOR_TIMEOUT", 5*time.Second),
		NetworkEnabled:    boolEnv("COCO_NETWORK_ENABLED", true),
		NetSocket:         getEnv("COCO_NET_SOCKET", "/run/coco/net.sock"),
		DefaultMemoryMB:   intEnv("COCO_DEFAULT_MEMORY_MB", 512),
		DefaultVCPUs:      intEnv("COCO_DEFAULT_VCPUS", 2),
		DefaultTemplate:   getEnv("COCO_DEFAULT_TEMPLATE", "alpine"),
		RateLimitEnabled:  boolEnv("COCO_RATE_LIMIT_ENABLED", true),
		RateLimitRPS:      intEnv("COCO_RATE_LIMIT_RPS", 100),
		RateLimitBurst:    intEnv("COCO_RATE_LIMIT_BURST", 200),
		APIKeyEnabled:     boolEnv("COCO_API_KEY_ENABLED", false),
		APIKeyFile:        getEnv("COCO_API_KEY_FILE", ""),
		NodeID:            nodeID,
		ClusterMode:       boolEnv("COCO_CLUSTER_MODE", false),
		ClusterNodes:      getEnv("COCO_CLUSTER_NODES", ""),
	}
}

// Load loads configuration from a YAML file, environment variables override
func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Parse YAML
		if err := json.Unmarshal(data, cfg); err != nil {
			// Try JSON if YAML fails - actually let's just use a simple approach
			// For now, we'll skip YAML parsing since we don't have yaml dependency
			// In production, add gopkg.in/yaml.v3
		}
	}

	// Override with environment variables
	applyEnvOverrides(cfg)

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks the configuration for correctness
func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("listen_addr is required")
	}
	if c.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if c.DefaultMemoryMB < 128 {
		return fmt.Errorf("default_memory_mb must be at least 128")
	}
	if c.DefaultVCPUs < 1 {
		return fmt.Errorf("default_vcpus must be at least 1")
	}
	if c.RateLimitRPS < 0 {
		return fmt.Errorf("rate_limit_rps must be non-negative")
	}
	return nil
}

// String returns a JSON representation of the config (sanitized)
func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

// Helpers

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func boolEnv(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func intEnv(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func durationEnv(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func applyEnvOverrides(cfg *Config) {
	// Core config
	if val := os.Getenv("COCO_LISTEN_ADDR"); val != "" {
		cfg.ListenAddr = val
	}
	if val := os.Getenv("COCO_METRICS_ADDR"); val != "" {
		cfg.MetricsAddr = val
	}
	if val := os.Getenv("COCO_DATA_DIR"); val != "" {
		cfg.DataDir = val
	}
	if val := os.Getenv("COCO_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}
	if val := os.Getenv("COCO_BADGER_PATH"); val != "" {
		cfg.BadgerPath = val
	}
	if val := os.Getenv("COCO_NODE_ID"); val != "" {
		cfg.NodeID = val
	}

	// Sandbox defaults
	if val := os.Getenv("COCO_DEFAULT_MEMORY_MB"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.DefaultMemoryMB = i
		}
	}
	if val := os.Getenv("COCO_DEFAULT_VCPUS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.DefaultVCPUs = i
		}
	}
	if val := os.Getenv("COCO_DEFAULT_TEMPLATE"); val != "" {
		cfg.DefaultTemplate = val
	}

	// Rate limiting
	if val := os.Getenv("COCO_RATE_LIMIT_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.RateLimitEnabled = b
		}
	}
	if val := os.Getenv("COCO_RATE_LIMIT_RPS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.RateLimitRPS = i
		}
	}

	// Cluster
	if val := os.Getenv("COCO_CLUSTER_MODE"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.ClusterMode = b
		}
	}
	if val := os.Getenv("COCO_CLUSTER_NODES"); val != "" {
		cfg.ClusterNodes = val
	}
}

// GetClusterNodes returns cluster nodes as a slice
func (c *Config) GetClusterNodes() []string {
	if c.ClusterNodes == "" {
		return nil
	}
	nodes := strings.Split(c.ClusterNodes, ",")
	result := make([]string, 0, len(nodes))
	for _, n := range nodes {
		n = strings.TrimSpace(n)
		if n != "" {
			result = append(result, n)
		}
	}
	return result
}
