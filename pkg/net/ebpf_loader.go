// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package net

import (
    "fmt"
    "os/exec"
)

// EBpfLoader loads and attaches eBPF programs for network isolation
type EBpfLoader struct {
    tapManager *TAPManager
    ipam       *IPAM
}

// NewEBpfLoader creates a new eBPF loader
func NewEBpfLoader(tap *TAPManager, ip *IPAM) *EBpfLoader {
    return &EBpfLoader{tapManager: tap, ipam: ip}
}

// LoadAndAttach loads eBPF programs and attaches to interfaces
func (l *EBpfLoader) LoadAndAttach() error {
    // Load from_sandbox program on TAP devices
    taps := l.tapManager.ListDevices()
    for _, tap := range taps {
        if err := l.attachToTap(tap.Name); err != nil {
            return fmt.Errorf("attach to %s: %w", tap.Name, err)
        }
    }

    // Load from_world on host NIC (eth0)
    if err := l.attachToHostNIC("eth0"); err != nil {
        return fmt.Errorf("attach to eth0: %w", err)
    }

    return nil
}

func (l *EBpfLoader) attachToTap(tapName string) error {
    // Add qdisc clsact (classifier action) for.attach
    cmd := exec.Command("tc", "qdisc", "add", "dev", tapName, "clsact")
    if err := cmd.Run(); err != nil {
        // May already exist
    }

    // Attach from_sandbox program for SNAT on outbound
    cmd = exec.Command("tc", "filter", "add", "dev", tapName, "ingress",
        "bpf", "obj", "ebpf/from_sandbox.o", "section", "xdp", "from_sandbox")
    return cmd.Run()
}

func (l *EBpfLoader) attachToHostNIC(nic string) error {
    cmd := exec.Command("tc", "qdisc", "add", "dev", nic, "clsact")
    cmd.Run()

    // Attach from_world program for DNAT on inbound
    cmd = exec.Command("tc", "filter", "add", "dev", nic, "ingress",
        "bpf", "obj", "ebpf/from_world.o", "section", "xdp", "from_world")
    return cmd.Run()
}

// UpdateNAT updates the NAT mapping in eBPF maps
func (l *EBpfLoader) UpdateNAT(sandboxIP string, natIP string, port uint16) error {
    // Write to BPF map via bpftool
    cmd := exec.Command("bpftool", "map", "update",
        "id", "42", "key", sandboxIP, "value", natIP)
    return cmd.Run()
}