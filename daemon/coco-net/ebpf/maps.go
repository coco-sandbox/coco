package ebpf

import (
	"fmt"

	"github.com/cilium/ebpf"
)

type MapConfig struct {
	Name       string
	Type       ebpf.MapType
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
	Flags      uint32
}

func CreateMap(cfg MapConfig) (*ebpf.Map, error) {
	spec := &ebpf.MapSpec{
		Name:       cfg.Name,
		Type:       cfg.Type,
		KeySize:    cfg.KeySize,
		ValueSize:  cfg.ValueSize,
		MaxEntries: cfg.MaxEntries,
		Flags:      cfg.Flags,
	}

	mp, err := ebpf.NewMap(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create map %s: %w", cfg.Name, err)
	}

	return mp, nil
}

type FlowKey struct {
	SrcIP    [4]byte
	DstIP    [4]byte
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
	_        [3]byte
}

type FlowValue struct {
	Packets  uint64
	Bytes    uint64
	LastSeen uint64
	State    uint8
	_        [7]byte
}

func CreateFlowMap(name string, maxEntries uint32) (*ebpf.Map, error) {
	return CreateMap(MapConfig{
		Name:       name,
		Type:       ebpf.Hash,
		KeySize:    uint32(20),
		ValueSize:  uint32(32),
		MaxEntries: maxEntries,
	})
}

type PolicyKey struct {
	SandboxID uint32
	Direction uint8
	_         [3]byte
}

type PolicyValue struct {
	Allow     uint8
	_         [7]byte
	RateLimit uint64
	Burst     uint64
}

func CreatePolicyMap(name string, maxEntries uint32) (*ebpf.Map, error) {
	return CreateMap(MapConfig{
		Name:       name,
		Type:       ebpf.Hash,
		KeySize:    uint32(8),
		ValueSize:  uint32(24),
		MaxEntries: maxEntries,
	})
}

type StatsKey struct {
	Direction uint8
	_         [7]byte
}

type StatsValue struct {
	Packets uint64
	Bytes   uint64
	Drops   uint64
	Errors  uint64
}

func CreateStatsMap(name string) (*ebpf.Map, error) {
	return CreateMap(MapConfig{
		Name:       name,
		Type:       ebpf.Array,
		KeySize:    4,
		ValueSize:  uint32(40),
		MaxEntries: 2,
	})
}

func (l *Loader) CreateFlowTable(maxEntries uint32) (*ebpf.Map, error) {
	mp, err := CreateFlowMap("flow_table", maxEntries)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.maps["flow_table"] = mp
	l.mu.Unlock()

	return mp, nil
}

func (l *Loader) CreatePolicyTable(maxEntries uint32) (*ebpf.Map, error) {
	mp, err := CreatePolicyMap("policy_table", maxEntries)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.maps["policy_table"] = mp
	l.mu.Unlock()

	return mp, nil
}

func (l *Loader) GetFlowTable() (*ebpf.Map, error) {
	return l.GetMap("flow_table")
}

func (l *Loader) GetPolicyTable() (*ebpf.Map, error) {
	return l.GetMap("policy_table")
}
