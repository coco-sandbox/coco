package ipam

import (
	"fmt"
	"net"
	"sync"
)

type IPAM struct {
	mu      sync.RWMutex
	pool    *Pool
	alloc   *Allocator
	subnets []net.IPNet
}

func New(subnet string) (*IPAM, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	pool := NewPool(ipnet)
	alloc := NewAllocator(pool)

	return &IPAM{
		pool:    pool,
		alloc:   alloc,
		subnets: []net.IPNet{*ipnet},
	}, nil
}

func (i *IPAM) Allocate(sandboxID string) (net.IP, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	ip, err := i.alloc.Allocate(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	return ip, nil
}

func (i *IPAM) Release(sandboxID string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.alloc.Release(sandboxID)
}

func (i *IPAM) Get(sandboxID string) (net.IP, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.alloc.Get(sandboxID)
}

func (i *IPAM) List() map[string]net.IP {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.alloc.List()
}

func (i *IPAM) Reserve(ip net.IP, sandboxID string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.alloc.Reserve(ip, sandboxID)
}
