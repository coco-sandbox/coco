// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -cflags "-Wno-compare-distinct-pointer-types -I../../../ebpf/headers -I/usr/include/x86_64-linux-gnu" xdp_filter ../../../ebpf/xdp/xdp_filter.bpf.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -cflags "-Wno-compare-distinct-pointer-types -I../../../ebpf/headers -I/usr/include/x86_64-linux-gnu" xdp_fwd ../../../ebpf/xdp/xdp_fwd.bpf.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -cflags "-Wno-compare-distinct-pointer-types -I../../../ebpf/headers -I/usr/include/x86_64-linux-gnu" xdp_nat ../../../ebpf/xdp/xdp_nat.bpf.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -cflags "-Wno-compare-distinct-pointer-types -I../../../ebpf/headers -I/usr/include/x86_64-linux-gnu" xdp_conntrack ../../../ebpf/xdp/xdp_conntrack.bpf.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -cflags "-Wno-compare-distinct-pointer-types -I../../../ebpf/headers -I/usr/include/x86_64-linux-gnu" xdp_stats ../../../ebpf/xdp/xdp_stats.bpf.c

package ebpf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
)

type Loader struct {
	mu       sync.RWMutex
	programs map[string]*ebpf.Program
	maps     map[string]*ebpf.Map
	links    []link.Link
}

func NewLoader() *Loader {
	return &Loader{
		programs: make(map[string]*ebpf.Program),
		maps:     make(map[string]*ebpf.Map),
		links:    make([]link.Link, 0),
	}
}

func (l *Loader) LoadBTF() error {
	spec, err := btf.LoadKernelSpec()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load BTF spec: %w", err)
	}
	_ = spec
	return nil
}

func (l *Loader) LoadProgramsFromDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read %s: %w", dir, err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".o") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		spec, err := ebpf.LoadCollectionSpec(path)
		if err != nil {
			return count, fmt.Errorf("load %s: %w", path, err)
		}

		coll, err := ebpf.NewCollectionWithOptions(spec, ebpf.CollectionOptions{})
		if err != nil {
			return count, fmt.Errorf("install %s: %w", path, err)
		}

		l.mu.Lock()
		for name, prog := range coll.Programs {
			l.programs[name] = prog
		}
		for name, mp := range coll.Maps {
			l.maps[name] = mp
		}
		l.mu.Unlock()
		count++
	}

	return count, nil
}

func (l *Loader) LoadFromSpec(spec *ebpf.CollectionSpec) error {
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	defer coll.Close()

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

func (l *Loader) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".o" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		spec, err := ebpf.LoadCollectionSpec(path)
		if err != nil {
			return fmt.Errorf("load spec %s: %w", path, err)
		}

		if err := l.LoadFromSpec(spec); err != nil {
			return fmt.Errorf("load from spec: %w", err)
		}
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

	lk, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: ifindex,
	})
	if err != nil {
		return fmt.Errorf("attach XDP: %w", err)
	}

	l.mu.Lock()
	l.links = append(l.links, lk)
	l.mu.Unlock()

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

	for _, lk := range l.links {
		lk.Close()
	}
	l.links = nil

	for _, prog := range l.programs {
		prog.Close()
	}

	for _, mp := range l.maps {
		mp.Close()
	}

	l.programs = make(map[string]*ebpf.Program)
	l.maps = make(map[string]*ebpf.Map)

	return nil
}
