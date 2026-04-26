package template

import (
	"context"
	"fmt"

	"coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewListCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cc)
		},
	}

	return cmd
}

func runList(ctx context.Context, cc *client.Client) error {
	_ = cc
	_ = ctx
	fmt.Println("No templates found")
	return nil
}
