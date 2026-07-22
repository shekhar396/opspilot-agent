package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-config",
		Short: "Validate agent configuration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Configuration validation is not implemented yet")
		},
	}
}
