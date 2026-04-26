package template

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/coco-sandbox/coco/pkg/types"
)

type Manager struct {
	mu      sync.RWMutex
	templates map[string]*types.Template
	store    *Store
}

func NewManager(store *Store) *Manager {
	return &Manager{
		templates: make(map[string]*types.Template),
		store:    store,
	}
}

func (m *Manager) List() ([]*types.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]*types.Template, 0, len(m.templates))
	for _, t := range m.templates {
		templates = append(templates, t)
	}

	return templates, nil
}

func (m *Manager) Get(id string) (*types.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.templates[id]
	if !ok {
		return nil, fmt.Errorf("template %s not found", id)
	}

	return t, nil
}

func (m *Manager) Create(template *types.Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.templates[template.ID]; exists {
		return fmt.Errorf("template %s already exists", template.ID)
	}

	m.templates[template.ID] = template

	if m.store != nil {
		return m.store.Save(template)
	}

	return nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.templates[id]; !ok {
		return fmt.Errorf("template %s not found", id)
	}

	delete(m.templates, id)

	if m.store != nil {
		return m.store.Delete(id)
	}

	return nil
}

func (m *Manager) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templateDir := filepath.Join(dir, entry.Name())
		template, err := m.loadTemplate(templateDir)
		if err != nil {
			continue
		}

		m.templates[template.ID] = template
	}

	return nil
}

func (m *Manager) loadTemplate(dir string) (*types.Template, error) {
	metadataFile := filepath.Join(dir, "metadata.json")

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	template := &types.Template{
		ID: filepath.Base(dir),
	}

	_ = data

	return template, nil
}
