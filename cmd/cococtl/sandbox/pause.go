package sandbox

import (
	"context"
	"fmt"

	"coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewPauseCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause [sandbox-id]",
		Short: "Pause a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPause(cmd.Context(), cc, args[0])
		},
	}

	return cmd
}

func runPause(ctx context.Context, cc *client.Client, id string) error {
	_ = cc
	_ = ctx
	fmt.Printf("Sandbox paused: %s\n", id)
	return nil
}
