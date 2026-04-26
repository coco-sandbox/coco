package sandbox

import (
	"context"
	"fmt"
	"os"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

type ExecOptions struct {
	command string
	tty     bool
}

func NewExecCmd(cc *client.Client) *cobra.Command {
	var opts ExecOptions

	cmd := &cobra.Command{
		Use:   "exec [sandbox-id] -- [command...]",
		Short: "Execute a command in a sandbox",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]
			opts.command = args[1]
			return runExec(cmd.Context(), cc, sandboxID, &opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.tty, "tty", "t", false, "Allocate a pseudo-TTY")

	return cmd
}

func runExec(ctx context.Context, cc *client.Client, sandboxID string, opts *ExecOptions) error {
	_ = cc
	_ = ctx
	fmt.Printf("Executing in sandbox: %s\n", sandboxID)
	if opts.tty {
		os.Exit(0)
	}
	return nil
}
