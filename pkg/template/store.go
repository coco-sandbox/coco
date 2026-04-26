package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coco-sandbox/coco/pkg/types"
)

type Store struct {
	baseDir string
}

func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
	}
}

func (s *Store) Save(template *types.Template) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	metadataFile := filepath.Join(s.baseDir, template.ID, "metadata.json")

	if err := os.MkdirAll(filepath.Dir(metadataFile), 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (s *Store) Load(id string) (*types.Template, error) {
	metadataFile := filepath.Join(s.baseDir, id, "metadata.json")

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	template := &types.Template{}
	if err := json.Unmarshal(data, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	return template, nil
}

func (s *Store) Delete(id string) error {
	templateDir := filepath.Join(s.baseDir, id)

	if err := os.RemoveAll(templateDir); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read store: %w", err)
	}

	ids := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}

	return ids, nil
}

func (s *Store) Exists(id string) bool {
	_, err := os.Stat(filepath.Join(s.baseDir, id))
	return err == nil
}
