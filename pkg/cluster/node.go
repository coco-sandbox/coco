package cluster

import (
	"fmt"
	"sync"
	"time"
)

type Node struct {
	mu         sync.RWMutex
	ID         string
	Name       string
	Addr       string
	Port       int
	Capacity   NodeCapacity
	Usage      NodeUsage
	Status     NodeStatus
	Labels     map[string]string
	Metadata   map[string]string
	LastSeen   time.Time
	Registered time.Time
}

type NodeCapacity struct {
	MemoryMB    uint64
	CPUCount    int
	DiskGB      uint64
	MaxSandboxes int
}

type NodeUsage struct {
	MemoryUsedMB uint64
	CPUUsed     int
	DiskUsedGB  uint64
	Sandboxes   int
}

type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusDraining NodeStatus = "draining"
	NodeStatusUnknown NodeStatus = "unknown"
)

func NewNode(id, name, addr string, port int) *Node {
	return &Node{
		ID:         id,
		Name:       name,
		Addr:       addr,
		Port:       port,
		Status:     NodeStatusUnknown,
		Labels:     make(map[string]string),
		Metadata:   make(map[string]string),
		LastSeen:   time.Now(),
		Registered: time.Now(),
	}
}

func (n *Node) UpdateCapacity(capacity NodeCapacity) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Capacity = capacity
}

func (n *Node) UpdateUsage(usage NodeUsage) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Usage = usage
}

func (n *Node) SetStatus(status NodeStatus) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Status = status
}

func (n *Node) GetStatus() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Status
}

func (n *Node) SetLabel(key, value string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Labels[key] = value
}

func (n *Node) GetLabel(key string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	value, ok := n.Labels[key]
	return value, ok
}

func (n *Node) SetMetadata(key, value string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Metadata[key] = value
}

func (n *Node) GetMetadata(key string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	value, ok := n.Metadata[key]
	return value, ok
}

func (n *Node) Refresh() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.LastSeen = time.Now()
}

func (n *Node) IsHealthy(timeout time.Duration) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return time.Since(n.LastSeen) < timeout
}

func (n *Node) AvailableMemoryMB() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Capacity.MemoryMB - n.Usage.MemoryUsedMB
}

func (n *Node) AvailableCPU() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Capacity.CPUCount - n.Usage.CPUUsed
}

func (n *Node) CanFit(memMB uint64, vCPUs int) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Usage.MemoryUsedMB+memMB <= n.Capacity.MemoryMB &&
		n.Usage.CPUUsed+vCPUs <= n.Capacity.CPUCount &&
		n.Usage.Sandboxes < n.Capacity.MaxSandboxes
}

func (n *Node) String() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return fmt.Sprintf("Node{id=%s, addr=%s, status=%s, sandboxes=%d}",
		n.ID, n.Addr, n.Status, n.Usage.Sandboxes)
}

type NodePool struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

func NewNodePool() *NodePool {
	return &NodePool{
		nodes: make(map[string]*Node),
	}
}

func (p *NodePool) Add(node *Node) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	p.nodes[node.ID] = node
	return nil
}

func (p *NodePool) Remove(nodeID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(p.nodes, nodeID)
	return nil
}

func (p *NodePool) Get(nodeID string) (*Node, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	node, ok := p.nodes[nodeID]
	return node, ok
}

func (p *NodePool) List() []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	nodes := make([]*Node, 0, len(p.nodes))
	for _, node := range p.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (p *NodePool) Filter(predicate func(*Node) bool) []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range p.nodes {
		if predicate(node) {
			nodes = append(nodes, node)
		}
	}

	return nodes
}
