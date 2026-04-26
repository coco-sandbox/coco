package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const (
	electionKey   = "/coco/master/leader"
	masterPrefix  = "/coco/master/"
	sessionTTL    = 10
)

type Election struct {
	client     *clientv3.Client
	session    *concurrency.Session
	election   *concurrency.Election
	mu         sync.RWMutex
	isLeader   bool
	leaderCh   chan bool
	onCampaign func(ctx context.Context) error
	onResign   func() error
}

func NewElection(endpoints []string, onCampaign func(ctx context.Context) error, onResign func() error) (*Election, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	session, err := concurrency.NewSession(client, concurrency.WithTTL(sessionTTL))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	election := concurrency.NewElection(session, electionKey)

	return &Election{
		client:    client,
		session:   session,
		election:  election,
		leaderCh:  make(chan bool, 1),
		onCampaign: onCampaign,
		onResign:  onResign,
	}, nil
}

func (e *Election) Start(ctx context.Context) error {
	go e.observe(ctx)
	return e.campaign(ctx)
}

func (e *Election) campaign(ctx context.Context) error {
	if e.onCampaign != nil {
		if err := e.onCampaign(ctx); err != nil {
			return err
		}
	}

	err := e.election.Campaign(ctx, "master")
	if err != nil {
		return fmt.Errorf("failed to campaign: %w", err)
	}

	e.mu.Lock()
	e.isLeader = true
	e.mu.Unlock()

	select {
	case e.leaderCh <- true:
	default:
	}

	log.Println("Won election, now leader")
	return nil
}

func (e *Election) observe(ctx context.Context) {
	ch := e.election.Observe(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-ch:
			if !ok {
				return
			}
			if len(resp.Kvs) > 0 {
				leader := string(resp.Kvs[0].Value)
				log.Printf("Current leader: %s", leader)
			}
		case <-e.session.Done():
			e.mu.Lock()
			wasLeader := e.isLeader
			e.isLeader = false
			e.mu.Unlock()

			if wasLeader && e.onResign != nil {
				e.onResign()
			}

			select {
			case e.leaderCh <- false:
			default:
			}

			log.Println("Lost leadership, re-campaigning")
			if err := e.campaign(ctx); err != nil {
				log.Printf("Failed to re-campaign: %v", err)
			}
		}
	}
}

func (e *Election) IsLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isLeader
}

func (e *Election) LeaderCh() <-chan bool {
	return e.leaderCh
}

func (e *Election) Resign(ctx context.Context) error {
	e.mu.Lock()
	if !e.isLeader {
		e.mu.Unlock()
		return nil
	}
	e.isLeader = false
	e.mu.Unlock()

	if e.onResign != nil {
		e.onResign()
	}

	return e.election.Resign(ctx)
}

func (e *Election) Close() error {
	if e.session != nil {
		e.session.Close()
	}
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}
