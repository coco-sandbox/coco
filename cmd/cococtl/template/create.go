package template

import (
	"context"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	name        string
	description string
	image       string
	dockerfile  string
}

func NewCreateCmd(cc *client.Client) *cobra.Command {
	var opts CreateOptions

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runCreate(cmd.Context(), cc, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Template description")
	cmd.Flags().StringVarP(&opts.image, "image", "i", "", "OCI image")
	cmd.Flags().StringVarP(&opts.dockerfile, "dockerfile", "f", "", "Dockerfile path")

	return cmd
}

func runCreate(ctx context.Context, cc *client.Client, opts *CreateOptions) error {
	_ = cc
	_ = ctx
	fmt.Printf("Template created: %s\n", opts.name)
	return nil
}
