package checkpoint

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewListCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List checkpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cc)
		},
	}

	return cmd
}

func runList(ctx context.Context, cc *client.Client) error {
	_ = cc
	_ = ctx
	fmt.Println("No checkpoints found")
	return nil
}
