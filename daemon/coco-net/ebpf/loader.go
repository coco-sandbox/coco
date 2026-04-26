// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package ebpf

import (
	"fmt"
	"os"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
)

type Loader struct {
	mu       sync.RWMutex
	programs map[string]*ebpf.Program
	maps     map[string]*ebpf.Map
	specs    map[string]*ebpf.ProgramSpec
	btfSpec  *btf.Spec
}

func NewLoader() *Loader {
	return &Loader{
		programs: make(map[string]*ebpf.Program),
		maps:     make(map[string]*ebpf.Map),
		specs:    make(map[string]*ebpf.ProgramSpec),
	}
}

func (l *Loader) Load(coll *ebpf.Collection) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for name, prog := range coll.Programs {
		l.programs[name] = prog
	}

	for name, mp := range coll.Maps {
		l.maps[name] = mp
	}

	return nil
}

func (l *Loader) LoadFromSpec(spec *ebpf.CollectionSpec, opts ebpf.CollectionOptions) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	coll, err := ebpf.NewCollectionWithOptions(spec, opts)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer coll.Close()

	for name, prog := range coll.Programs {
		l.programs[name] = prog
	}

	for name, mp := range coll.Maps {
		l.maps[name] = mp
	}

	return nil
}

func (l *Loader) GetProgram(name string) (*ebpf.Program, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	prog, ok := l.programs[name]
	if !ok {
		return nil, fmt.Errorf("program %s not found", name)
	}

	return prog, nil
}

func (l *Loader) GetMap(name string) (*ebpf.Map, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mp, ok := l.maps[name]
	if !ok {
		return nil, fmt.Errorf("map %s not found", name)
	}

	return mp, nil
}

func (l *Loader) ListPrograms() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.programs))
	for name := range l.programs {
		names = append(names, name)
	}

	return names
}

func (l *Loader) ListMaps() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.maps))
	for name := range l.maps {
		names = append(names, name)
	}

	return names
}

func (l *Loader) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error

	for _, prog := range l.programs {
		if err := prog.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	for _, mp := range l.maps {
		if err := mp.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	l.programs = make(map[string]*ebpf.Program)
	l.maps = make(map[string]*ebpf.Map)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing resources: %v", errs)
	}

	return nil
}

func (l *Loader) AttachXDP(ifindex int, progName string) error {
	l.mu.RLock()
	prog, ok := l.programs[progName]
	l.mu.RUnlock()

	if !ok {
		return fmt.Errorf("program %s not found", progName)
	}

	link, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: ifindex,
	})
	if err != nil {
		return fmt.Errorf("failed to attach XDP: %w", err)
	}

	fmt.Printf("Attached XDP program %s to interface %d\n", progName, ifindex)
	_ = link

	return nil
}

func (l *Loader) LoadBTF() error {
	spec, err := btf.LoadKernelSpec()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load BTF: %w", err)
	}

	l.btfSpec = spec
	return nil
}
