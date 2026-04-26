package ipam

import (
	"fmt"
	"net"
	"sync"
)

type Allocator struct {
	mu        sync.RWMutex
	pool      *Pool
	allocated map[string]net.IP
}

func NewAllocator(pool *Pool) *Allocator {
	return &Allocator{
		pool:      pool,
		allocated: make(map[string]net.IP),
	}
}

func (a *Allocator) Allocate(id string) (net.IP, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if ip, ok := a.allocated[id]; ok {
		return ip, nil
	}

	ip, err := a.pool.Acquire()
	if err != nil {
		return nil, err
	}

	a.allocated[id] = ip

	return ip, nil
}

func (a *Allocator) Release(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip, ok := a.allocated[id]
	if !ok {
		return fmt.Errorf("ID %s not found", id)
	}

	delete(a.allocated, id)

	return a.pool.Release(ip)
}

func (a *Allocator) Get(id string) (net.IP, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	ip, ok := a.allocated[id]
	return ip, ok
}

func (a *Allocator) Reserve(ip net.IP, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.allocated[id]; ok {
		return fmt.Errorf("ID %s already has an IP", id)
	}

	if err := a.pool.Reserve(ip); err != nil {
		return err
	}

	a.allocated[id] = ip

	return nil
}

func (a *Allocator) List() map[string]net.IP {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]net.IP)
	for k, v := range a.allocated {
		result[k] = v
	}

	return result
}

func (a *Allocator) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.allocated)
}
