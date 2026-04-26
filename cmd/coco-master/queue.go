package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type RequestQueue struct {
	mu          sync.RWMutex
	pending     []*QueuedRequest
	processing  map[string]*QueuedRequest
	completed   map[string]*QueuedRequest
	maxSize     int
	workerCount int
	resultTTL   time.Duration
}

type QueuedRequest struct {
	ID          string
	Type        RequestType
	Payload     interface{}
	Result      interface{}
	Error       error
	Priority    int
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Cancel      context.CancelFunc
}

type RequestType int

const (
	RequestCreateSandbox RequestType = iota
	RequestDeleteSandbox
	RequestPauseSandbox
	RequestResumeSandbox
	RequestForkSandbox
	RequestExecSandbox
	RequestCheckpoint
	RequestRestore
)

func NewRequestQueue(maxSize, workerCount int) *RequestQueue {
	return &RequestQueue{
		pending:     make([]*QueuedRequest, 0),
		processing:  make(map[string]*QueuedRequest),
		completed:   make(map[string]*QueuedRequest),
		maxSize:     maxSize,
		workerCount: workerCount,
		resultTTL:   5 * time.Minute,
	}
}

func (q *RequestQueue) Enqueue(req *QueuedRequest) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.pending) >= q.maxSize {
		return fmt.Errorf("queue is full")
	}

	req.CreatedAt = time.Now()
	q.pending = append(q.pending, req)
	q.sortPending()

	log.Printf("Enqueued request %s (type: %d, pending: %d)", req.ID, req.Type, len(q.pending))

	return nil
}

func (q *RequestQueue) Dequeue() *QueuedRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.pending) == 0 {
		return nil
	}

	req := q.pending[0]
	q.pending = q.pending[1:]

	now := time.Now()
	req.StartedAt = &now
	q.processing[req.ID] = req

	log.Printf("Dequeued request %s (processing: %d)", req.ID, len(q.processing))

	return req
}

func (q *RequestQueue) Complete(reqID string, result interface{}, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	req, ok := q.processing[reqID]
	if !ok {
		log.Printf("Request %s not found in processing", reqID)
		return
	}

	delete(q.processing, reqID)

	now := time.Now()
	req.CompletedAt = &now
	req.Result = result
	req.Error = err
	q.completed[req.ID] = req

	go q.cleanupCompleted()

	log.Printf("Completed request %s (err: %v)", reqID, err)
}

func (q *RequestQueue) Cancel(reqID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, req := range q.pending {
		if req.ID == reqID {
			q.pending = append(q.pending[:i], q.pending[i+1:]...)
			if req.Cancel != nil {
				req.Cancel()
			}
			log.Printf("Cancelled request %s", reqID)
			return
		}
	}

	if req, ok := q.processing[reqID]; ok {
		if req.Cancel != nil {
			req.Cancel()
		}
		log.Printf("Cancelled processing request %s", reqID)
	}
}

func (q *RequestQueue) GetStatus(reqID string) (*QueuedRequest, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if req, ok := q.completed[reqID]; ok {
		return req, nil
	}

	for _, req := range q.pending {
		if req.ID == reqID {
			return req, nil
		}
	}

	if req, ok := q.processing[reqID]; ok {
		return req, nil
	}

	return nil, fmt.Errorf("request not found")
}

func (q *RequestQueue) GetPendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending)
}

func (q *RequestQueue) GetProcessingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.processing)
}

func (q *RequestQueue) GetCompletedCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.completed)
}

func (q *RequestQueue) sortPending() {
	for i := 0; i < len(q.pending)-1; i++ {
		for j := i + 1; j < len(q.pending); j++ {
			if q.pending[i].Priority > q.pending[j].Priority {
				q.pending[i], q.pending[j] = q.pending[j], q.pending[i]
			}
		}
	}
}

func (q *RequestQueue) cleanupCompleted() {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-q.resultTTL)
	for id, req := range q.completed {
		if req.CompletedAt != nil && req.CompletedAt.Before(cutoff) {
			delete(q.completed, id)
		}
	}
}

func (q *RequestQueue) StartWorkers(ctx context.Context, handler func(context.Context, *QueuedRequest) error) {
	for i := 0; i < q.workerCount; i++ {
		workerID := i
		go func() {
			log.Printf("Worker %d started", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d stopping", workerID)
					return
				default:
					req := q.Dequeue()
					if req == nil {
						time.Sleep(10 * time.Millisecond)
						continue
					}

					result := handler(ctx, req)
					q.Complete(req.ID, result, nil)
				}
			}
		}()
	}
}

type QueueStats struct {
	Pending    int
	Processing int
	Completed  int
}
