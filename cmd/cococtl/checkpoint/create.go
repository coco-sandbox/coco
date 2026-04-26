package checkpoint

import (
	"context"
	"fmt"

	"coco/pkg/client"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	sandboxID string
	output    string
	compress  bool
}

func NewCreateCmd(cc *client.Client) *cobra.Command {
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

func runCreate(ctx context.Context, cc *client.Client, opts *CreateOptions) error {
	_ = cc
	_ = opts
	fmt.Printf("Checkpoint create for sandbox: %s\n", opts.sandboxID)
	fmt.Printf("Output: %s, Compress: %v\n", opts.output, opts.compress)
	return nil
}
