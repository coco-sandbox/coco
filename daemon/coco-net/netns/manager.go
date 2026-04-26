package netns

import (
	"fmt"
	"sync"
)

type Manager struct {
	mu          sync.RWMutex
	namespaces  map[string]*Namespace
}

func NewManager() *Manager {
	return &Manager{
		namespaces: make(map[string]*Namespace),
	}
}

func (m *Manager) Create(id string) (*Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.namespaces[id]; ok {
		return nil, fmt.Errorf("namespace %s already exists", id)
	}

	ns, err := New(id)
	if err != nil {
		return nil, err
	}

	m.namespaces[id] = ns

	return ns, nil
}

func (m *Manager) Get(id string) (*Namespace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, ok := m.namespaces[id]
	if !ok {
		return nil, fmt.Errorf("namespace %s not found", id)
	}

	return ns, nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ns, ok := m.namespaces[id]
	if !ok {
		return fmt.Errorf("namespace %s not found", id)
	}

	if err := ns.Close(); err != nil {
		return fmt.Errorf("failed to close namespace: %w", err)
	}

	delete(m.namespaces, id)

	return nil
}

func (m *Manager) List() []*Namespace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaces := make([]*Namespace, 0, len(m.namespaces))
	for _, ns := range m.namespaces {
		namespaces = append(namespaces, ns)
	}

	return namespaces
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.namespaces)
}
