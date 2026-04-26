package cluster

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewInfoCmd(cc *client.CocoClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show cluster information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(cmd.Context(), cc)
		},
	}

	return cmd
}

func runInfo(ctx context.Context, cc *client.CocoClient) error {
	info, err := cc.Cluster().Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	fmt.Println("Cluster Information:")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Nodes: %d\n", info.NodeCount)
	fmt.Printf("  Sandboxes: %d\n", info.SandboxCount)
	fmt.Printf("  Leader: %s\n", info.Leader)

	return nil
}
