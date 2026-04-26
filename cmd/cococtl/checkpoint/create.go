package checkpoint

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	sandboxID string
	output    string
	compress  bool
}

func NewCreateCmd(cc *client.CocoClient) *cobra.Command {
	var opts CreateOptions

	cmd := &cobra.Command{
		Use:   "create [sandbox-id]",
		Short: "Create a checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.sandboxID = args[0]
			return runCreate(cmd.Context(), cc, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output file for checkpoint")
	cmd.Flags().BoolVarP(&opts.compress, "compress", "c", false, "Compress checkpoint")

	return cmd
}

func runCreate(ctx context.Context, cc *client.CocoClient, opts *CreateOptions) error {
	req := &client.CreateCheckpointRequest{
		SandboxID: opts.sandboxID,
		Output:    opts.output,
		Compress:  opts.compress,
	}

	resp, err := cc.Checkpoint().Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}

	fmt.Printf("Checkpoint created: %s\n", resp.CheckpointID)
	return nil
}
