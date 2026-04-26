package sandbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coco-sandbox/coco/pkg/client"
	"github.com/spf13/cobra"
)

func NewGetCmd(cc *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [sandbox-id]",
		Short: "Get sandbox details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), cc, args[0])
		},
	}

	return cmd
}

func runGet(ctx context.Context, cc *client.Client, id string) error {
	_ = cc
	_ = ctx
	data, _ := json.MarshalIndent(map[string]string{"id": id, "status": "running"}, "", "  ")
	fmt.Println(string(data))
	return nil
}
