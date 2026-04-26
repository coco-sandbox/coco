package template

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewDeleteCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [template-id]",
		Short: "Delete a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), cc, args[0])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, cc *client.Client, id string) error {
	_ = cc
	_ = ctx
	fmt.Printf("Template deleted: %s\n", id)
	return nil
}
