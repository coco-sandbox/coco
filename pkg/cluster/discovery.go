package cluster

import (
	"context"
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
	}, nil
}

func (d *Discovery) RegisterNode(ctx context.Context, nodeID, addr string, metadata map[string]string) error {
	key := fmt.Sprintf("%s/%s", d.nodePrefix, nodeID)

	value := fmt.Sprintf(`{"id":"%s","addr":"%s","metadata":%v,"registered_at":"%s"}`,
		nodeID, addr, metadata, time.Now().Format(time.RFC3339))

	lease, err := d.client.Grant(ctx, int64(d.ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to grant lease: %w", err)
	}

	_, err = d.client.Put(ctx, key, value, clientv3.WithLease(lease.ID))
	if err != nil {
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
	key := fmt.Sprintf("%s/%s", d.nodePrefix, nodeID)

	lease, err := d.client.Grant(ctx, int64(d.ttl.Seconds()))
	if err != nil {
		return err
	}

	_, err = d.client.Put(ctx, key, "", clientv3.WithLease(lease.ID))
	return err
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
