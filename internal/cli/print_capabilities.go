package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPrintCapabilitiesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "print-capabilities",
		Short: "Print implemented capabilities",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "cli")
			fmt.Fprintln(cmd.OutOrStdout(), "version")
			fmt.Fprintln(cmd.OutOrStdout(), "config-validation")
			fmt.Fprintln(cmd.OutOrStdout(), "structured-logging")
			fmt.Fprintln(cmd.OutOrStdout(), "runtime")
		},
	}
}
