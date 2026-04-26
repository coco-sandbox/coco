// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	State       string    `json:"state"`
	RootfsPath  string    `json:"rootfs_path"`
	KernelPath  string    `json:"kernel_path"`
	InitrdPath  string    `json:"initrd_path"`
	MemoryMB    int       `json:"memory_mb"`
	VCPUs       int       `json:"vcpus"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name"`
	RootfsPath  string `json:"rootfs_path"`
	KernelPath  string `json:"kernel_path"`
	InitrdPath  string `json:"initrd_path"`
	Description string `json:"description,omitempty"`
}
