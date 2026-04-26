package cluster

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewInfoCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show cluster information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(cmd.Context(), cc)
		},
	}

	return cmd
}

func runInfo(ctx context.Context, cc *client.Client) error {
	_ = cc
	_ = ctx
	fmt.Println("Cluster Information:")
	fmt.Printf("  Name: %s\n", "coco-cluster")
	fmt.Printf("  Nodes: %d\n", 0)
	fmt.Printf("  Sandboxes: %d\n", 0)
	fmt.Printf("  Leader: %s\n", "unknown")
	return nil
}
