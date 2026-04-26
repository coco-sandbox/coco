package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type ClusterManager struct {
	mu         sync.RWMutex
	nodes      map[string]*Node
	membership *Membership
	discovery  *Discovery
	config     *ClusterConfig
	stopCh     chan struct{}
}

type ClusterConfig struct {
	NodeID            string
	NodeAddr          string
	NodePort          int
	EtcdEndpoints     []string
	HeartbeatInterval time.Duration
	NodeTimeout       time.Duration
}

func NewClusterManager(cfg ClusterConfig) (*ClusterManager, error) {
	discovery, err := NewDiscovery(DiscoveryConfig{
		Endpoints:  cfg.EtcdEndpoints,
		NodePrefix: "/coco/nodes",
		TTL:        cfg.NodeTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery: %w", err)
	}

	return &ClusterManager{
		nodes:      make(map[string]*Node),
		membership: NewMembership(),
		discovery:  discovery,
		config:     &cfg,
		stopCh:     make(chan struct{}),
	}, nil
}

func (cm *ClusterManager) Start(ctx context.Context) error {
	log.Println("Starting cluster manager...")

	if err := cm.discovery.RegisterNode(ctx, cm.config.NodeID, cm.config.NodeAddr, nil); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	go cm.heartbeatLoop(ctx)
	go cm.healthCheckLoop(ctx)

	log.Printf("Cluster manager started (node_id=%s)", cm.config.NodeID)
	return nil
}

func (cm *ClusterManager) Stop() error {
	log.Println("Stopping cluster manager...")
	close(cm.stopCh)

	ctx := context.Background()
	if err := cm.discovery.DeregisterNode(ctx, cm.config.NodeID); err != nil {
		log.Printf("Failed to deregister node: %v", err)
	}

	return cm.discovery.Close()
}

func (cm *ClusterManager) RegisterNode(node *Node) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	cm.nodes[node.ID] = node
	cm.membership.AddMember(&Member{
		ID:        node.ID,
		Name:      node.Name,
		Addr:      node.Addr,
		Port:      node.Port,
		IsHealthy: true,
		JoinTime:  time.Now(),
		Status:    StatusJoined,
	})

	log.Printf("Registered node: %s", node.ID)
	return nil
}

func (cm *ClusterManager) DeregisterNode(nodeID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(cm.nodes, nodeID)
	cm.membership.RemoveMember(nodeID)

	log.Printf("Deregistered node: %s", nodeID)
	return nil
}

func (cm *ClusterManager) GetNode(nodeID string) (*Node, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	node, ok := cm.nodes[nodeID]
	return node, ok
}

func (cm *ClusterManager) ListNodes() []*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodes := make([]*Node, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (cm *ClusterManager) GetHealthyNodes() []*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range cm.nodes {
		if node.IsHealthy(cm.config.NodeTimeout) {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (cm *ClusterManager) FindNodeForSandbox(memMB uint64, vCPUs int) (*Node, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var best *Node
	var minLoad float64 = 1.0

	for _, node := range cm.nodes {
		if !node.IsHealthy(cm.config.NodeTimeout) {
			continue
		}

		if !node.CanFit(memMB, vCPUs) {
			continue
		}

		load := float64(node.Usage.Sandboxes) / float64(node.Capacity.MaxSandboxes)
		if best == nil || load < minLoad {
			best = node
			minLoad = load
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	return best, nil
}

func (cm *ClusterManager) UpdateNodeUsage(nodeID string, usage NodeUsage) error {
	cm.mu.RLock()
	node, ok := cm.nodes[nodeID]
	cm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.UpdateUsage(usage)
	return nil
}

func (cm *ClusterManager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cm.discovery.RefreshLease(ctx, cm.config.NodeID); err != nil {
				log.Printf("Failed to refresh lease: %v", err)
			}
		}
	}
}

func (cm *ClusterManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.checkNodeHealth()
		}
	}
}

func (cm *ClusterManager) checkNodeHealth() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	timeout := cm.config.NodeTimeout

	for _, node := range cm.nodes {
		if !node.IsHealthy(timeout) {
			log.Printf("Node %s is unhealthy, marking as offline", node.ID)
			node.SetStatus(NodeStatusOffline)
			cm.membership.UpdateMemberHealth(node.ID, false)
		}
	}
}
