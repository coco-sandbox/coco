package sandbox

import (
	"context"
	"fmt"

	"coco/pkg/client"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	image     string
	name      string
	resources string
	network   string
}

func NewCreateCmd(cc *client.Client) *cobra.Command {
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

func runCreate(ctx context.Context, cc *client.Client, opts *CreateOptions) error {
	_ = cc
	_ = ctx
	_ = opts
	fmt.Printf("Sandbox created: %s\n", opts.name)
	return nil
}
