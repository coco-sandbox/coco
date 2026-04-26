package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/scheduler"
	"go.etcd.io/etcd/client/v3"
)

const (
	nodeRegistryPrefix = "/coco/nodes/"
	heartbeatInterval  = 10 * time.Second
	nodeTTL            = 30
)

type NodeDiscovery struct {
	mu            sync.RWMutex
	nodeID        string
	nodeAddr      string
	etcdClient    *clientv3.Client
	etcdEndpoints []string
	capacity      NodeCapacity
	registered    bool
	stopChan      chan struct{}
}

type NodeCapacity struct {
	MemoryMB     uint64
	CPUCount     int
	MaxSandboxes int
}

func NewNodeDiscovery(nodeID, nodeAddr string, etcdEndpoints []string, capacity NodeCapacity) (*NodeDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &NodeDiscovery{
		nodeID:        nodeID,
		nodeAddr:      nodeAddr,
		etcdClient:    cli,
		etcdEndpoints: etcdEndpoints,
		capacity:      capacity,
		stopChan:      make(chan struct{}),
	}, nil
}

func (nd *NodeDiscovery) Start(ctx context.Context) error {
	if err := nd.register(ctx); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	go nd.heartbeat(ctx)
	go nd.watchNodes(ctx)

	log.Printf("Node discovery started (node_id=%s, addr=%s)", nd.nodeID, nd.nodeAddr)
	return nil
}

func (nd *NodeDiscovery) Stop() error {
	close(nd.stopChan)

	if nd.registered {
		return nd.deregister()
	}

	if nd.etcdClient != nil {
		return nd.etcdClient.Close()
	}

	return nil
}

func (nd *NodeDiscovery) register(ctx context.Context) error {
	key := nodeRegistryPrefix + nd.nodeID
	value := fmt.Sprintf(`{"id":"%s","addr":"%s","memory_mb":%d,"cpu_count":%d,"max_sandboxes":%d,"registered_at":"%s"}`,
		nd.nodeID,
		nd.nodeAddr,
		nd.capacity.MemoryMB,
		nd.capacity.CPUCount,
		nd.capacity.MaxSandboxes,
		time.Now().Format(time.RFC3339),
	)

	leaseID, err := nd.grantLease(ctx)
	if err != nil {
		return err
	}

	_, err = nd.etcdClient.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	nd.mu.Lock()
	nd.registered = true
	nd.mu.Unlock()

	log.Printf("Node registered in etcd (node_id=%s)", nd.nodeID)
	return nil
}

func (nd *NodeDiscovery) deregister() error {
	key := nodeRegistryPrefix + nd.nodeID
	_, err := nd.etcdClient.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to deregister node: %w", err)
	}

	nd.mu.Lock()
	nd.registered = false
	nd.mu.Unlock()

	log.Printf("Node deregistered from etcd (node_id=%s)", nd.nodeID)
	return nil
}

func (nd *NodeDiscovery) grantLease(ctx context.Context) (clientv3.LeaseID, error) {
	resp, err := nd.etcdClient.Grant(ctx, nodeTTL)
	if err != nil {
		return 0, fmt.Errorf("failed to grant lease: %w", err)
	}
	return resp.ID, nil
}

func (nd *NodeDiscovery) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nd.stopChan:
			return
		case <-ticker.C:
			nd.refreshLease(ctx)
		}
	}
}

func (nd *NodeDiscovery) refreshLease(ctx context.Context) {
	key := nodeRegistryPrefix + nd.nodeID

	leaseID, err := nd.grantLease(ctx)
	if err != nil {
		log.Printf("Failed to refresh lease: %v", err)
		return
	}

	value := fmt.Sprintf(`{"id":"%s","addr":"%s","memory_mb":%d,"cpu_count":%d,"max_sandboxes":%d,"last_heartbeat":"%s"}`,
		nd.nodeID,
		nd.nodeAddr,
		nd.capacity.MemoryMB,
		nd.capacity.CPUCount,
		nd.capacity.MaxSandboxes,
		time.Now().Format(time.RFC3339),
	)

	_, err = nd.etcdClient.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		log.Printf("Failed to refresh node registration: %v", err)
	}
}

func (nd *NodeDiscovery) watchNodes(ctx context.Context) {
	rch := nd.etcdClient.Watch(ctx, nodeRegistryPrefix, clientv3.WithPrefix())

	for {
		select {
		case <-nd.stopChan:
			return
		case wresp := <-rch:
			for _, ev := range wresp.Events {
				log.Printf("Node registry change: %s %s %s", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

func (nd *NodeDiscovery) GetNodes() ([]*scheduler.NodeEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := nd.etcdClient.Get(ctx, nodeRegistryPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	nodes := make([]*scheduler.NodeEntry, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		entry := &scheduler.NodeEntry{
			ID:        string(kv.Key),
			Addr:      string(kv.Value),
			Available: true,
			UpdatedAt: time.Now(),
		}
		nodes = append(nodes, entry)
	}

	return nodes, nil
}
