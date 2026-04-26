package template

import (
	"context"
	"fmt"

	"coco/pkg/types"
)

type DockerClient struct {
	dockerHost string
}

func NewDockerClient(dockerHost string) *DockerClient {
	return &DockerClient{
		dockerHost: dockerHost,
	}
}

func (c *DockerClient) PullImage(ctx context.Context, image string) (*types.Template, error) {
	template := &types.Template{
		ID:          image,
		Name:        image,
		Description: fmt.Sprintf("Pulled from Docker: %s", image),
	}

	return template, nil
}

func (c *DockerClient) BuildImage(ctx context.Context, dockerfile string, opts BuildOptions) (*types.Template, error) {
	template := &types.Template{
		ID:          opts.Name,
		Name:        opts.Name,
		Description: opts.Description,
	}

	return template, nil
}

type BuildOptions struct {
	Name        string
	Description string
	Context     string
	BuildArgs   map[string]string
}

func (c *DockerClient) ListImages(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (c *DockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	return false, nil
}

type DockerImage struct {
	id     string
	config DockerConfig
}

type DockerConfig struct {
	Entrypoint []string
	Cmd        []string
	Env        []string
	WorkingDir string
	User       string
}

func ParseDockerConfig(config []byte) (*DockerConfig, error) {
	return &DockerConfig{}, nil
}

func (c *DockerConfig) ToTemplate() *types.Template {
	return &types.Template{
		Name: "docker-image",
	}
}
