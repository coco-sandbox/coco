// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrTemplateNotFound = fmt.Errorf("template not found")

type CreateOpts struct {
	RootfsPath string
	KernelPath string
	InitrdPath string
	MemoryMB   uint32
	VCPUs      uint32
}

type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Rootfs    string `json:"rootfs"`
	Kernel    string `json:"kernel"`
	Initrd    string `json:"initrd"`
	MemoryMB  uint32 `json:"memory_mb"`
	VCPUs     uint32 `json:"vcpus"`
	SnapshotPath string `json:"snapshot_path"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt int64  `json:"created_at"`
}

type Manager struct {
	baseDir string
	mu      sync.RWMutex
	templates map[string]*Template
	pool     *TemplatePool
}

type TemplatePool struct {
	templates []*Template
	mu        sync.Mutex
}

type WarmVM struct {
	ID           string
	TemplateID   string
	SnapshotPath string
	State        string
}

func NewManager(baseDir string) *Manager {
	os.MkdirAll(baseDir, 0755)
	return &Manager{
		baseDir:   baseDir,
		templates: make(map[string]*Template),
		pool:      &TemplatePool{},
	}
}

func (m *Manager) PreWarmPool(ctx context.Context, count int) (int, error) {
	m.pool.mu.Lock()
	defer m.pool.mu.Unlock()

	var warmed int
	for i := 0; i < count; i++ {
		base, err := m.Get("default")
		if err != nil {
			continue
		}

		vm := &WarmVM{
			ID:           fmt.Sprintf("warm_%d", time.Now().UnixNano()),
			TemplateID:   base.ID,
			SnapshotPath: base.SnapshotPath,
			State:        "warm",
		}

		m.pool.templates = append(m.pool.templates, &Template{
			ID:           vm.ID,
			Name:         "warm",
			SnapshotPath: vm.SnapshotPath,
			State:        "warm",
		})
		warmed++
	}

	return warmed, nil
}

func (m *Manager) GetFromPool() *Template {
	m.pool.mu.Lock()
	defer m.pool.mu.Unlock()

	if len(m.pool.templates) == 0 {
		return nil
	}

	tpl := m.pool.templates[0]
	m.pool.templates = m.pool.templates[1:]
	return tpl
}

func (m *Manager) Create(name string, opts CreateOpts) (string, error) {
	id := fmt.Sprintf("tpl_%s_%d", name, time.Now().UnixNano())
	tpl := &Template{
		ID:        id,
		Name:      name,
		Rootfs:    opts.RootfsPath,
		Kernel:    opts.KernelPath,
		Initrd:    opts.InitrdPath,
		MemoryMB:  opts.MemoryMB,
		VCPUs:     opts.VCPUs,
		SnapshotPath: filepath.Join(m.baseDir, id, "snapshot.mem"),
		CreatedAt: time.Now().Unix(),
	}

	if err := os.MkdirAll(filepath.Dir(tpl.SnapshotPath), 0755); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.templates[id] = tpl
	m.mu.Unlock()

	return id, m.saveMeta(tpl)
}

func (m *Manager) Get(id string) (*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if tpl, ok := m.templates[id]; ok {
		return tpl, nil
	}
	return nil, ErrTemplateNotFound
}

func (m *Manager) List() ([]*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Template, 0, len(m.templates))
	for _, t := range m.templates {
		out = append(out, t)
	}
	return out, nil
}

func (m *Manager) saveMeta(tpl *Template) error {
	metaPath := filepath.Join(m.baseDir, tpl.ID, "meta.json")
	data, err := json.Marshal(tpl)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	return os.WriteFile(metaPath, data, 0644)
}