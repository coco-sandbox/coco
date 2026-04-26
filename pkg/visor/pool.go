package visor

import (
	"context"
	"sync"
)

type Pool struct {
	mu       sync.RWMutex
	visors   map[string]*Visor
	capacity int
}

func NewPool(capacity int) *Pool {
	return &Pool{
		visors:   make(map[string]*Visor),
		capacity: capacity,
	}
}

func (p *Pool) Get(id string) (*Visor, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	v, ok := p.visors[id]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (p *Pool) Add(v *Visor) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.visors) >= p.capacity {
		return ErrPoolFull
	}

	if _, exists := p.visors[v.ID]; exists {
		return ErrAlreadyExists
	}

	p.visors[v.ID] = v
	return nil
}

func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.visors[id]; !ok {
		return ErrNotFound
	}

	delete(p.visors, id)
	return nil
}

func (p *Pool) List() []*Visor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	visors := make([]*Visor, 0, len(p.visors))
	for _, v := range p.visors {
		visors = append(visors, v)
	}
	return visors
}

func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.visors)
}

func (p *Pool) StartAll(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, v := range p.visors {
		if err := v.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pool) StopAll() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, v := range p.visors {
		if err := v.Stop(); err != nil {
			return err
		}
	}
	return nil
}
