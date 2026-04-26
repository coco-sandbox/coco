package sandbox

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewResumeCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [sandbox-id]",
		Short: "Resume a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(cmd.Context(), cc, args[0])
		},
	}

	return cmd
}

func runResume(ctx context.Context, cc *client.Client, id string) error {
	_ = cc
	_ = ctx
	fmt.Printf("Sandbox resumed: %s\n", id)
	return nil
}
