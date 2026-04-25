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
	baseDir   string
	mu        sync.RWMutex
	templates map[string]*Template
}

func NewManager(baseDir string) *Manager {
	os.MkdirAll(baseDir, 0755)
	return &Manager{
		baseDir:   baseDir,
		templates: make(map[string]*Template),
	}
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
	data, _ := json.Marshal(tpl)
	return os.WriteFile(metaPath, data, 0644)
}