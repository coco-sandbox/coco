// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/visor"
)

type PooledVM struct {
	ID        string
	VsockCID  uint32
	PID       uint32
	Template  string
	State     string
	AllocAt   time.Time
	Available bool
}

type Pool struct {
	mu     sync.Mutex
	free   chan *PooledVM
	active map[string]*PooledVM
	visor  *visor.Pool
	target int
	cfg    PoolConfig
	stop   chan struct{}
	done   sync.WaitGroup
}

type PoolConfig struct {
	TargetSize   int
	MaxWaitTime  time.Duration
	RefillTicker time.Duration
	DefaultMem   int
	DefaultVCPU  int
}

func NewPool(visorPool *visor.Pool, cfg PoolConfig) *Pool {
	if cfg.TargetSize == 0 {
		cfg.TargetSize = 5
	}
	if cfg.MaxWaitTime == 0 {
		cfg.MaxWaitTime = 5 * time.Second
	}
	if cfg.RefillTicker == 0 {
		cfg.RefillTicker = 30 * time.Second
	}
	if cfg.DefaultMem == 0 {
		cfg.DefaultMem = 512
	}
	if cfg.DefaultVCPU == 0 {
		cfg.DefaultVCPU = 2
	}

	p := &Pool{
		free:   make(chan *PooledVM, cfg.TargetSize),
		active: make(map[string]*PooledVM),
		visor:  visorPool,
		target: cfg.TargetSize,
		cfg:    cfg,
		stop:   make(chan struct{}),
	}

	p.done.Add(1)
	go p.refillLoop()

	return p
}

func (p *Pool) Acquire(ctx context.Context, sandboxID string, template string) (*PooledVM, error) {
	timeout := time.NewTimer(p.cfg.MaxWaitTime)
	defer timeout.Stop()

	select {
	case vm := <-p.free:
		p.mu.Lock()
		p.active[sandboxID] = vm
		p.mu.Unlock()
		return vm, nil

	case <-timeout.C:
		return p.bootNew(sandboxID, template)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) Release(sandboxID string) error {
	p.mu.Lock()
	vm, ok := p.active[sandboxID]
	delete(p.active, sandboxID)
	p.mu.Unlock()

	if !ok {
		return fmt.Errorf("sandbox %s not in active pool", sandboxID)
	}

	select {
	case p.free <- vm:
		return nil
	default:
		return p.destroy(vm)
	}
}

func resolveTemplate(template string) (rootfs, kernel, initrd string) {
	switch template {
	case "alpine", "default", "":
		return "/var/lib/coco/images/alpine/rootfs.ext4",
			"/var/lib/coco/images/vmlinux",
			""
	default:
		return "/var/lib/coco/images/" + template + "/rootfs.ext4",
			"/var/lib/coco/images/vmlinux",
			""
	}
}

func (p *Pool) bootNew(sandboxID string, template string) (*PooledVM, error) {
	visorConn, err := p.visor.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire visor: %w", err)
	}
	defer p.visor.Release(visorConn)

	rootfs, kernel, initrd := resolveTemplate(template)
	bootReq := visor.BootRequest{
		ID:       sandboxID,
		Rootfs:   rootfs,
		Kernel:   kernel,
		Initrd:   initrd,
		MemoryMB: uint32(p.cfg.DefaultMem),
		VCPUs:    uint32(p.cfg.DefaultVCPU),
	}

	bootResp, err := visorConn.Boot(bootReq)
	if err != nil {
		return nil, fmt.Errorf("visor boot failed: %w", err)
	}

	vm := &PooledVM{
		ID:        sandboxID,
		VsockCID:  bootResp.VsockCID,
		PID:       bootResp.PID,
		Template:  template,
		State:     "running",
		AllocAt:   time.Now(),
		Available: true,
	}

	p.mu.Lock()
	p.active[sandboxID] = vm
	p.mu.Unlock()

	return vm, nil
}

func (p *Pool) destroy(vm *PooledVM) error {
	visorConn, err := p.visor.Acquire()
	if err != nil {
		return err
	}
	defer p.visor.Release(visorConn)

	return visorConn.Destroy(vm.ID)
}

func (p *Pool) refillLoop() {
	defer p.done.Done()

	ticker := time.NewTicker(p.cfg.RefillTicker)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.refill()
		}
	}
}

func (p *Pool) refill() {
	p.mu.Lock()
	freeCount := len(p.free)
	p.mu.Unlock()

	deficit := p.target - freeCount
	if deficit <= 0 {
		return
	}

	for i := 0; i < deficit; i++ {
		vm, err := p.bootNew(fmt.Sprintf("pool-%d", time.Now().UnixNano()), "default")
		if err != nil {
			continue
		}
		select {
		case p.free <- vm:
		default:
			p.destroy(vm)
		}
	}
}

func (p *Pool) Stop() error {
	close(p.stop)
	p.done.Wait()

	p.mu.Lock()
	for _, vm := range p.active {
		p.destroy(vm)
	}
	p.mu.Unlock()

	for {
		select {
		case vm := <-p.free:
			p.destroy(vm)
		default:
			return nil
		}
	}
}

func (p *Pool) Stats() (active, free int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.active), len(p.free)
}
