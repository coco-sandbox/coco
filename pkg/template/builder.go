package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coco-sandbox/coco/pkg/types"
)

type Builder struct {
	outputDir string
}

func NewBuilder(outputDir string) *Builder {
	return &Builder{
		outputDir: outputDir,
	}
}

func (b *Builder) BuildFromOCI(image string) (*types.Template, error) {
	template := &types.Template{
		ID:          image,
		Name:        image,
		Description: "Built from OCI image",
	}

	rootfs := filepath.Join(b.outputDir, image, "rootfs")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	template.Rootfs = rootfs

	return template, nil
}

func (b *Builder) BuildFromDockerfile(dockerfile string) (*types.Template, error) {
	template := &types.Template{
		ID:          "custom",
		Name:        "Custom template",
		Description: "Built from Dockerfile",
	}

	rootfs := filepath.Join(b.outputDir, "custom", "rootfs")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	template.Rootfs = rootfs

	return template, nil
}

func (b *Builder) BuildFromDirectory(dir string) (*types.Template, error) {
	template := &types.Template{
		ID:          filepath.Base(dir),
		Name:        filepath.Base(dir),
		Description: "Built from directory",
	}

	rootfs := filepath.Join(b.outputDir, filepath.Base(dir), "rootfs")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	template.Rootfs = rootfs

	return template, nil
}

func (b *Builder) ExtractKernel(template *types.Template) error {
	if template.Rootfs == "" {
		return fmt.Errorf("template has no rootfs")
	}

	kernelPaths := []string{
		filepath.Join(template.Rootfs, "boot", "vmlinuz"),
		filepath.Join(template.Rootfs, "usr", "lib", "modules"),
	}

	for _, path := range kernelPaths {
		if _, err := os.Stat(path); err == nil {
			template.Kernel = path
			return nil
		}
	}

	return fmt.Errorf("no kernel found in template")
}
