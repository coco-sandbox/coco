// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package template

import (
	"testing"
)

func TestTemplateCreateAndList(t *testing.T) {
	m := NewManager("/tmp/coco-test-templates")

	id, err := m.Create("python-3.11", CreateOpts{
		RootfsPath: "/var/lib/coco/images/python.rootfs",
		KernelPath: "/var/lib/coco/vmlinux",
		MemoryMB:   512,
		VCPUs:      2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id == "" {
		t.Fatal("Template ID is empty")
	}

	list, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(list))
	}
}

func TestTemplateNotFound(t *testing.T) {
	m := NewManager("/tmp/coco-test-templates")

	_, err := m.Get("nonexistent")
	if err != ErrTemplateNotFound {
		t.Fatalf("Expected ErrTemplateNotFound, got %v", err)
	}
}