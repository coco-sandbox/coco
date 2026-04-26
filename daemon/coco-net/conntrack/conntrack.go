package conntrack

import (
	"fmt"
	"sync"
	"time"
)

type Conntrack struct {
	mu       sync.RWMutex
	flows    map[string]*Flow
	interval time.Duration
}

func New() *Conntrack {
	ct := &Conntrack{
		flows:    make(map[string]*Flow),
		interval: 5 * time.Minute,
	}

	go ct.cleanupLoop()

	return ct
}

func (ct *Conntrack) Add(flow *Flow) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	key := flow.KeyString()

	if _, ok := ct.flows[key]; ok {
		return fmt.Errorf("flow already exists")
	}

	ct.flows[key] = flow

	return nil
}

func (ct *Conntrack) Get(key string) (*Flow, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	flow, ok := ct.flows[key]
	return flow, ok
}

func (ct *Conntrack) Delete(key string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	delete(ct.flows, key)
}

func (ct *Conntrack) Update(key string, packets, bytes uint64) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	flow, ok := ct.flows[key]
	if !ok {
		return fmt.Errorf("flow not found")
	}

	flow.Packets = packets
	flow.Bytes = bytes
	flow.LastSeen = time.Now().Unix()

	return nil
}

func (ct *Conntrack) List() []*Flow {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	flows := make([]*Flow, 0, len(ct.flows))
	for _, flow := range ct.flows {
		flows = append(flows, flow)
	}

	return flows
}

func (ct *Conntrack) Count() int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return len(ct.flows)
}

func (ct *Conntrack) cleanupLoop() {
	ticker := time.NewTicker(ct.interval)
	defer ticker.Stop()

	for range ticker.C {
		ct.cleanup()
	}
}

func (ct *Conntrack) cleanup() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now().Unix()
	cutoff := now - int64(ct.interval.Seconds())

	for key, flow := range ct.flows {
		if flow.LastSeen < cutoff {
			delete(ct.flows, key)
		}
	}
}

type FlowKey struct {
	SrcIP   string
	DstIP   string
	SrcPort uint16
	DstPort uint16
	Proto   uint8
}

func (k FlowKey) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d (%d)",
		k.SrcIP, k.SrcPort, k.DstIP, k.DstPort, k.Proto)
}

type Protocol uint8

const (
	ProtocolTCP  Protocol = 6
	ProtocolUDP  Protocol = 17
	ProtocolICMP Protocol = 1
)

func (p Protocol) String() string {
	switch p {
	case ProtocolTCP:
		return "tcp"
	case ProtocolUDP:
		return "udp"
	case ProtocolICMP:
		return "icmp"
	default:
		return "unknown"
	}
}
