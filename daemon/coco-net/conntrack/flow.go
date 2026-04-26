package conntrack

import (
	"fmt"
	"sync"
	"time"
)

type Flow struct {
	mu        sync.RWMutex
	SrcIP     string
	DstIP     string
	SrcPort   uint16
	DstPort   uint16
	Proto     uint8
	State     FlowState
	Packets   uint64
	Bytes     uint64
	StartTime time.Time
	LastSeen  int64
	SandboxID string
}

type FlowState uint8

const (
	FlowNew FlowState = iota
	FlowEstablished
	FlowClosing
	FlowClosed
)

func NewFlow(srcIP, dstIP string, srcPort, dstPort uint16, proto uint8, sandboxID string) *Flow {
	now := time.Now()
	return &Flow{
		SrcIP:     srcIP,
		DstIP:     dstIP,
		SrcPort:   srcPort,
		DstPort:   dstPort,
		Proto:     proto,
		State:     FlowNew,
		StartTime: now,
		LastSeen:  now.Unix(),
		SandboxID: sandboxID,
	}
}

func (f *Flow) KeyString() string {
	return fmt.Sprintf("%s:%d->%s:%d:%d", f.SrcIP, f.SrcPort, f.DstIP, f.DstPort, f.Proto)
}

func (f *Flow) Update(packets, bytes uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Packets = packets
	f.Bytes = bytes
	f.LastSeen = time.Now().Unix()
}

func (f *Flow) SetState(state FlowState) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.State = state
}

func (f *Flow) StateString() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	switch f.State {
	case FlowNew:
		return "NEW"
	case FlowEstablished:
		return "ESTABLISHED"
	case FlowClosing:
		return "CLOSING"
	case FlowClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

func (f *Flow) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return fmt.Sprintf("%s:%d -> %s:%d [%s] pkts=%d bytes=%d state=%s",
		f.SrcIP, f.SrcPort, f.DstIP, f.DstPort, f.Proto,
		f.Packets, f.Bytes, f.StateString())
}

func (f *Flow) IsOutgoing(sandboxID string) bool {
	return f.SandboxID == sandboxID
}

func (f *Flow) Age() time.Duration {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return time.Since(f.StartTime)
}

func (f *Flow) MarkEstablished() {
	f.SetState(FlowEstablished)
}

func (f *Flow) MarkClosing() {
	f.SetState(FlowClosing)
}

func (f *Flow) MarkClosed() {
	f.SetState(FlowClosed)
}
