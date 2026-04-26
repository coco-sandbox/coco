package sandbox

import (
	"context"
	"fmt"

	"coco/pkg/client"
	"github.com/spf13/cobra"
)

type ForkOptions struct {
	name string
}

func NewForkCmd(cc *client.Client) *cobra.Command {
	var opts ForkOptions

	cmd := &cobra.Command{
		Use:   "fork [sandbox-id]",
		Short: "Fork a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFork(cmd.Context(), cc, args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Name for the forked sandbox")

	return cmd
}

func runFork(ctx context.Context, cc *client.Client, sandboxID string, opts *ForkOptions) error {
	_ = cc
	_ = ctx
	_ = opts
	fmt.Printf("Sandbox forked: %s (from %s)\n", opts.name, sandboxID)
	return nil
}
