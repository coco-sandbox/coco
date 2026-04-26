package checkpoint

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

type RestoreOptions struct {
	checkpointID string
	sandboxID    string
	network      bool
}

func NewRestoreCmd(cc *client.CocoClient) *cobra.Command {
	var opts RestoreOptions

	cmd := &cobra.Command{
		Use:   "restore [checkpoint-id]",
		Short: "Restore from a checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.checkpointID = args[0]
			return runRestore(cmd.Context(), cc, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.sandboxID, "sandbox-id", "s", "", "New sandbox ID (optional)")
	cmd.Flags().BoolVarP(&opts.network, "network", "n", false, "Restore network namespace")

	return cmd
}

func runRestore(ctx context.Context, cc *client.CocoClient, opts *RestoreOptions) error {
	req := &client.RestoreCheckpointRequest{
		CheckpointID: opts.checkpointID,
		SandboxID:    opts.sandboxID,
		RestoreNetwork: opts.network,
	}

	resp, err := cc.Checkpoint().Restore(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to restore checkpoint: %w", err)
	}

	fmt.Printf("Restored to sandbox: %s\n", resp.SandboxID)
	return nil
}
