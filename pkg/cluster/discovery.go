package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

type Discovery struct {
	mu            sync.RWMutex
	client        *clientv3.Client
	etcdEndpoints []string
	nodePrefix    string
	ttl           time.Duration
	stopCh        chan struct{}
	records       map[string]NodeRecord
}

type NodeRecord struct {
	ID            string            `json:"id"`
	Addr          string            `json:"addr"`
	MemoryMB      uint64            `json:"memory_mb"`
	CPUCount      int               `json:"cpu_count"`
	MaxSandboxes  int               `json:"max_sandboxes"`
	LastHeartbeat string            `json:"last_heartbeat"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	RegisteredAt  time.Time         `json:"registered_at"`
}

type DiscoveryConfig struct {
	Endpoints  []string
	NodePrefix string
	TTL        time.Duration
}

func NewDiscovery(cfg DiscoveryConfig) (*Discovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 30 * time.Second
	}

	return &Discovery{
		client:        cli,
		etcdEndpoints: cfg.Endpoints,
		nodePrefix:    cfg.NodePrefix,
		ttl:           ttl,
		stopCh:        make(chan struct{}),
		records:       make(map[string]NodeRecord),
	}, nil
}

func (d *Discovery) RegisterNode(ctx context.Context, nodeID, addr string, metadata map[string]string) error {
	record := NodeRecord{
		ID:           nodeID,
		Addr:         addr,
		Metadata:     metadata,
		RegisteredAt: time.Now(),
	}
	value, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	d.mu.Lock()
	d.records[nodeID] = record
	d.mu.Unlock()

	key := fmt.Sprintf("%s/%s", d.nodePrefix, nodeID)
	lease, err := d.client.Grant(ctx, int64(d.ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to grant lease: %w", err)
	}

	if _, err := d.client.Put(ctx, key, string(value), clientv3.WithLease(lease.ID)); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	return nil
}

func (d *Discovery) DeregisterNode(ctx context.Context, nodeID string) error {
	key := fmt.Sprintf("%s/%s", d.nodePrefix, nodeID)

	_, err := d.client.Delete(ctx, key)
	return err
}

func (d *Discovery) ListNodes(ctx context.Context) ([]*NodeInfo, error) {
	resp, err := d.client.Get(ctx, d.nodePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodes := make([]*NodeInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		nodes = append(nodes, &NodeInfo{
			Key:   string(kv.Key),
			Value: string(kv.Value),
		})
	}

	return nodes, nil
}

func (d *Discovery) WatchNodes(ctx context.Context, callback func(Event)) {
	ch := d.client.Watch(ctx, d.nodePrefix, clientv3.WithPrefix())

	for {
		select {
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		case resp := <-ch:
			for _, ev := range resp.Events {
				eventType := "unknown"
				if ev.IsCreate() {
					eventType = "put"
				} else if ev.IsModify() {
					eventType = "put"
				} else if ev.Type == clientv3.EventTypeDelete {
					eventType = "delete"
				}
				callback(Event{
					Type:  eventType,
					Key:   string(ev.Kv.Key),
					Value: string(ev.Kv.Value),
				})
			}
		}
	}
}

func (d *Discovery) RefreshLease(ctx context.Context, nodeID string) error {
	d.mu.RLock()
	record, ok := d.records[nodeID]
	d.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no cached record for node %s; call RegisterNode first", nodeID)
	}
	return d.RegisterNode(ctx, record.ID, record.Addr, record.Metadata)
}

func ParseNodeRecord(value []byte) (NodeRecord, error) {
	var r NodeRecord
	err := json.Unmarshal(value, &r)
	return r, err
}

func (d *Discovery) Close() error {
	close(d.stopCh)
	return d.client.Close()
}

type NodeInfo struct {
	Key   string
	Value string
}

type Event struct {
	Type  string
	Key   string
	Value string
}
