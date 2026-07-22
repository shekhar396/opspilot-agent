package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:           "opspilot-agent",
		Short:         "a lightweight Linux operations agent for the OpsPilot ecosystem",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		newRunCommand(),
		newVersionCommand(),
		newValidateConfigCommand(),
		newPrintCapabilitiesCommand(),
	)

	return rootCmd
}
