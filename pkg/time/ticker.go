// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package time

import (
	"sync"
	"time"
)

// Ticker ticks at regular intervals
type Ticker struct {
	C     <-chan time.Time
	timer *time.Ticker
	stop  chan struct{}
	wg    sync.WaitGroup
}

// NewTicker creates a new Ticker that ticks at the given interval
func NewTicker(interval time.Duration) *Ticker {
	t := &Ticker{
		stop: make(chan struct{}),
	}
	t.timer = time.NewTicker(interval)
	t.C = t.timer.C
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-t.stop
		t.timer.Stop()
	}()
	return t
}

// Stop stops the ticker
func (t *Ticker) Stop() {
	close(t.stop)
	t.wg.Wait()
}

// Tick waits for the next tick or returns immediately if already tick time
func (t *Ticker) Tick() bool {
	select {
	case <-t.C:
		return true
	default:
		return false
	}
}

// Reset restarts the ticker with a new interval
func (t *Ticker) Reset(interval time.Duration) {
	t.timer.Reset(interval)
}

// Periodic performs an action periodically
func Periodic(interval time.Duration, fn func()) {
	ticker := NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		fn()
	}
}

// Debounce waits for silence before executing
func Debounce(fn func(), delay time.Duration) func() {
	timer := time.NewTimer(delay)
	stopCh := make(chan struct{})
	ticker := time.NewTicker(delay)
	done := make(chan struct{})

	go func() {
		defer timer.Stop()
		defer ticker.Stop()
		for {
			select {
			case <-timer.C:
				fn()
				return
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
		<-done
	}
}

// Throttle ensures fn is called at most once per interval
func Throttle(interval time.Duration, fn func()) func() {
	last := time.Time{}
	mu := sync.Mutex{}

	return func() {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(last) >= interval {
			fn()
			last = time.Now()
		}
	}
}

// Delay executes fn after delay
func Delay(delay time.Duration, fn func()) {
	time.AfterFunc(delay, fn)
}

// Retry retries fn until it succeeds or maxAttempts is reached
func Retry(interval time.Duration, maxAttempts int, fn func() error) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < maxAttempts-1 {
			time.Sleep(interval)
		}
	}
	return err
}