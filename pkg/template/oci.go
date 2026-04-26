package template

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/types"
)

type OCIClient struct {
	registry string
}

func NewOCIClient(registry string) *OCIClient {
	return &OCIClient{
		registry: registry,
	}
}

func (c *OCIClient) PullImage(ctx context.Context, image string) (*types.Template, error) {
	template := &types.Template{
		ID:          image,
		Name:        image,
		Description: fmt.Sprintf("Pulled from OCI registry: %s", c.registry),
	}

	return template, nil
}

func (c *OCIClient) ListImages(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (c *OCIClient) ImageExists(ctx context.Context, image string) (bool, error) {
	return false, nil
}

type OCIImage struct {
	ref    string
	digest string
	layers []string
}

func ParseImageRef(ref string) (*OCIImage, error) {
	return &OCIImage{
		ref: ref,
	}, nil
}

func (i *OCIImage) Manifest() ([]byte, error) {
	return nil, nil
}

func (i *OCIImage) Layers() []string {
	return i.layers
}

func (i *OCIImage) Config() ([]byte, error) {
	return nil, nil
}
