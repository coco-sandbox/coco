package sandbox

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewDeleteCmd(cc *client.CocoClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [sandbox-id]",
		Short: "Delete a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), cc, args[0])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, cc *client.CocoClient, id string) error {
	if err := cc.Sandbox().Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	fmt.Printf("Sandbox deleted: %s\n", id)
	return nil
}
