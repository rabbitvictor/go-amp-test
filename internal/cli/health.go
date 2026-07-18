package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newHealthCmd(flags *rootFlags, out, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check service health",
		Long:  "Calls GET /health and prints the service health response.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := flags.newAPIClient().Health(cmd.Context())
			if err != nil {
				return err
			}
			if h.Status != "" {
				fmt.Fprintf(errOut, "status: %s\n", h.Status)
			}
			return writeOut(out, flags.outFormat(), h)
		},
	}
}
