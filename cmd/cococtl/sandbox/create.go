package sandbox

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	image     string
	name      string
	resources string
	network   string
}

func NewCreateCmd(cc *client.CocoClient) *cobra.Command {
	var opts CreateOptions

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runCreate(cmd.Context(), cc, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.image, "image", "i", "", "Container image")
	cmd.Flags().StringVarP(&opts.resources, "resources", "r", "", "Resources (JSON)")
	cmd.Flags().StringVarP(&opts.network, "network", "n", "", "Network configuration")

	return cmd
}

func runCreate(ctx context.Context, cc *client.CocoClient, opts *CreateOptions) error {
	req := &client.CreateSandboxRequest{
		Name:    opts.name,
		Image:   opts.image,
		Resources: opts.resources,
		Network: opts.network,
	}

	resp, err := cc.Sandbox().Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}

	fmt.Printf("Sandbox created: %s\n", resp.ID)
	return nil
}
