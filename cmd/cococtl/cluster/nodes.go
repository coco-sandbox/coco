package cluster

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewNodesCmd(cc *client.CocoClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List cluster nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNodes(cmd.Context(), cc)
		},
	}

	return cmd
}

func runNodes(ctx context.Context, cc *client.CocoClient) error {
	nodes, err := cc.Cluster().Nodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	fmt.Println("Cluster Nodes:")
	fmt.Println("ID\t\tADDRESS\t\tSTATUS\tSANDBOXES")
	for _, node := range nodes {
		fmt.Printf("%s\t%s\t%s\t%d\n", node.ID, node.Address, node.Status, node.SandboxCount)
	}

	return nil
}
