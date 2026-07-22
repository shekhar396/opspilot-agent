package cli

import (
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	return newRootCommandWithDependencies(os.Stdout, productionDependencies())
}

func newRootCommand(runtimeOutput io.Writer) *cobra.Command {
	return newRootCommandWithDependencies(runtimeOutput, productionDependencies())
}

type dependencies struct {
	newHTTPClient func() *http.Client
}

func productionDependencies() dependencies {
	return dependencies{
		newHTTPClient: func() *http.Client {
			return &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
		},
	}
}

func newRootCommandWithDependencies(runtimeOutput io.Writer, deps dependencies) *cobra.Command {
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
		newRunCommand(runtimeOutput, deps),
		newVersionCommand(),
		newValidateConfigCommand(),
		newPrintCapabilitiesCommand(),
	)

	return rootCmd
}
