package cluster

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewNodesCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List cluster nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNodes(cmd.Context(), cc)
		},
	}

	return cmd
}

func runNodes(ctx context.Context, cc *client.Client) error {
	_ = cc
	_ = ctx
	fmt.Println("No nodes found")
	return nil
}
