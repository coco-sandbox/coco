package checkpoint

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewListCmd(cc *client.CocoClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List checkpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cc)
		},
	}

	return cmd
}

func runList(ctx context.Context, cc *client.CocoClient) error {
	checkpoints, err := cc.Checkpoint().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		fmt.Println("No checkpoints found")
		return nil
	}

	fmt.Println("CHECKPOINT ID\tSANDBOX ID\tCREATED AT")
	for _, cp := range checkpoints {
		fmt.Printf("%s\t%s\t%s\n", cp.ID, cp.SandboxID, cp.CreatedAt)
	}

	return nil
}
