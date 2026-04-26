package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"coco/daemon/coco-net/conntrack"
	"coco/daemon/coco-net/ipam"
	"coco/daemon/coco-net/netns"
	"coco/daemon/coco-net/policy"
	"coco/daemon/coco-net/rate"
)

type Agent struct {
	ipam        *ipam.IPAM
	conntrack   *conntrack.Conntrack
	netnsMgr    *netns.Manager
	policyEng   *policy.Engine
	rateLimiter *rate.Limiter
	config      *Config
}

type Config struct {
	ListenAddr  string
	Subnet      string
	MetricsAddr string
}

func NewAgent(cfg *Config) (*Agent, error) {
	ipam, err := ipam.New(cfg.Subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPAM: %w", err)
	}

	return &Agent{
		ipam:        ipam,
		conntrack:   conntrack.New(),
		netnsMgr:    netns.NewManager(),
		policyEng:   policy.NewEngine(),
		rateLimiter: rate.NewLimiter(1000, 2000),
		config:      cfg,
	}, nil
}

func (a *Agent) Start() error {
	log.Printf("coco-net starting on %s", a.config.ListenAddr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down coco-net")
		os.Exit(0)
	}()

	return nil
}

func main() {
	cfg := &Config{
		ListenAddr:  ":8081",
		Subnet:      "10.0.0.0/24",
		MetricsAddr: ":9091",
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}
}
