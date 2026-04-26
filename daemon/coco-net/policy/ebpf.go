// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package policy

import (
	"fmt"
	"net"
	"sync"

	"github.com/cilium/ebpf"
)

type EBPFUpdater struct {
	mu      sync.RWMutex
	maps    map[string]*ebpf.Map
	vmMap   *ebpf.Map
	ruleMap *ebpf.Map
}

type VMInfo struct {
	IP         uint32
	Active     bool
	SandboxID  [32]byte
	RateLimit  uint64
	BurstLimit uint64
}

type RuleKey struct {
	SandboxID [32]byte
	Priority  uint32
	RuleID    [32]byte
}

type RuleValue struct {
	Action     uint8
	Direction  uint8
	Protocol   uint8
	SrcIP      uint32
	DstIP      uint32
	SrcPort    uint16
	DstPort    uint16
	RateLimit  uint64
	BurstLimit uint64
}

func NewEBPFUpdater(vmMap, ruleMap *ebpf.Map) *EBPFUpdater {
	return &EBPFUpdater{
		vmMap:   vmMap,
		ruleMap: ruleMap,
		maps:    make(map[string]*ebpf.Map),
	}
}

func (e *EBPFUpdater) RegisterMap(name string, m *ebpf.Map) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maps[name] = m
}

func (e *EBPFUpdater) AddVM(sandboxID string, ip net.IP, rateLimit, burstLimit uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.vmMap == nil {
		return fmt.Errorf("vm map not initialized")
	}

	var sandboxBytes [32]byte
	copy(sandboxBytes[:], sandboxID)

	vm := VMInfo{
		IP:         ipToUint32(ip),
		Active:     true,
		SandboxID:  sandboxBytes,
		RateLimit:  rateLimit,
		BurstLimit: burstLimit,
	}

	if err := e.vmMap.Put(ipToUint32(ip), vm); err != nil {
		return fmt.Errorf("failed to add VM to eBPF map: %w", err)
	}

	return nil
}

func (e *EBPFUpdater) RemoveVM(ip net.IP) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.vmMap == nil {
		return fmt.Errorf("vm map not initialized")
	}

	if err := e.vmMap.Delete(ipToUint32(ip)); err != nil {
		return fmt.Errorf("failed to remove VM from eBPF map: %w", err)
	}

	return nil
}

func (e *EBPFUpdater) AddRule(sandboxID string, rule *Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.ruleMap == nil {
		return fmt.Errorf("rule map not initialized")
	}

	var sandboxBytes [32]byte
	var ruleBytes [32]byte
	copy(sandboxBytes[:], sandboxID)
	copy(ruleBytes[:], rule.ID)

	key := RuleKey{
		SandboxID: sandboxBytes,
		Priority:  0,
		RuleID:    ruleBytes,
	}

	value := RuleValue{
		Action:     uint8(rule.Action),
		Direction:  uint8(rule.Direction),
		Protocol:   uint8(rule.Protocol),
		SrcIP:      ipToUint32(net.ParseIP(rule.SrcIP)),
		DstIP:      ipToUint32(net.ParseIP(rule.DstIP)),
		SrcPort:    rule.SrcPort,
		DstPort:    rule.DstPort,
		RateLimit:  rule.RateLimit,
		BurstLimit: rule.Burst,
	}

	if err := e.ruleMap.Put(key, value); err != nil {
		return fmt.Errorf("failed to add rule to eBPF map: %w", err)
	}

	return nil
}

func (e *EBPFUpdater) RemoveRules(sandboxID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.ruleMap == nil {
		return fmt.Errorf("rule map not initialized")
	}

	var sandboxBytes [32]byte
	copy(sandboxBytes[:], sandboxID)

	iter := e.ruleMap.Iterate()
	var key RuleKey
	var value RuleValue
	for iter.Next(&key, &value) {
		if key.SandboxID == sandboxBytes {
			if err := e.ruleMap.Delete(key); err != nil {
				return fmt.Errorf("failed to delete rule: %w", err)
			}
		}
	}

	return nil
}

func (e *EBPFUpdater) UpdateRateLimit(ip net.IP, rateLimit, burstLimit uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.vmMap == nil {
		return fmt.Errorf("vm map not initialized")
	}

	var vm VMInfo
	if err := e.vmMap.Lookup(ipToUint32(ip), &vm); err != nil {
		return fmt.Errorf("failed to lookup VM: %w", err)
	}

	vm.RateLimit = rateLimit
	vm.BurstLimit = burstLimit

	if err := e.vmMap.Put(ipToUint32(ip), vm); err != nil {
		return fmt.Errorf("failed to update rate limit: %w", err)
	}

	return nil
}

func ipToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
}

func uint32ToIP(v uint32) net.IP {
	return net.IP([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}
