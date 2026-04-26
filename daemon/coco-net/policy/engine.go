package policy

import (
	"fmt"
	"sync"
)

type Engine struct {
	mu          sync.RWMutex
	policies    map[string][]*Rule
	defaultRule *Rule
	evaluator   *Evaluator
	cache       *Cache
}

func NewEngine() *Engine {
	return &Engine{
		policies:    make(map[string][]*Rule),
		defaultRule: &Rule{Action: Deny},
		evaluator:   NewEvaluator(),
		cache:       NewCache(),
	}
}

func (e *Engine) AddPolicy(sandboxID string, rules []*Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(rules) == 0 {
		return fmt.Errorf("no rules provided")
	}

	e.policies[sandboxID] = rules
	e.cache.Invalidate(sandboxID)

	return nil
}

func (e *Engine) RemovePolicy(sandboxID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.policies[sandboxID]; !ok {
		return fmt.Errorf("policy not found for sandbox %s", sandboxID)
	}

	delete(e.policies, sandboxID)
	e.cache.Invalidate(sandboxID)

	return nil
}

func (e *Engine) Evaluate(sandboxID string, direction Direction, conn *Connection) (Action, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if cached, ok := e.cache.Get(sandboxID, direction, conn); ok {
		return cached, nil
	}

	rules, ok := e.policies[sandboxID]
	if !ok {
		return e.defaultRule.Action, nil
	}

	action := e.evaluator.Evaluate(rules, direction, conn)

	e.cache.Set(sandboxID, direction, conn, action)

	return action, nil
}

func (e *Engine) ListPolicies() map[string][]*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string][]*Rule)
	for id, rules := range e.policies {
		result[id] = rules
	}

	return result
}

func (e *Engine) GetPolicy(sandboxID string) ([]*Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules, ok := e.policies[sandboxID]
	return rules, ok
}

func (e *Engine) SetDefaultAction(action Action) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.defaultRule = &Rule{Action: action}
	e.cache.Clear()
}

func (e *Engine) DefaultAction() Action {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.defaultRule.Action
}

type Direction uint8

const (
	DirectionIngress Direction = iota
	DirectionEgress
)

func (d Direction) String() string {
	switch d {
	case DirectionIngress:
		return "ingress"
	case DirectionEgress:
		return "egress"
	default:
		return "unknown"
	}
}

type Action uint8

const (
	Allow Action = iota
	Deny
	Log
)

func (a Action) String() string {
	switch a {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	case Log:
		return "log"
	default:
		return "unknown"
	}
}

type Protocol uint8

const (
	ProtocolTCP Protocol = 6
	ProtocolUDP Protocol = 17
	ProtocolICMP Protocol = 1
	ProtocolAny Protocol = 0
)

type Rule struct {
	ID          string
	Action      Action
	Direction   Direction
	Protocol    Protocol
	SrcIP       string
	DstIP       string
	SrcPort     uint16
	DstPort     uint16
	RateLimit   uint64
	Burst       uint64
	Description string
}

func (r *Rule) String() string {
	return fmt.Sprintf("Rule{id=%s action=%s dir=%s proto=%d %s:%d -> %s:%d}",
		r.ID, r.Action, r.Direction, r.Protocol,
		r.SrcIP, r.SrcPort, r.DstIP, r.DstPort)
}

type Connection struct {
	SrcIP   string
	DstIP   string
	SrcPort uint16
	DstPort uint16
	Proto   Protocol
}
