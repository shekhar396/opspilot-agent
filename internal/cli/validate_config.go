package cli

import (
	"fmt"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/spf13/cobra"
)

func newValidateConfigCommand() *cobra.Command {
	configPath := "configs/opspilot-agent.yaml"

	cmd := &cobra.Command{
		Use:   "validate-config",
		Short: "Validate agent configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := config.Load(configPath); err != nil {
				return fmt.Errorf("validate configuration file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Configuration is valid")
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", configPath, "path to the configuration file")

	return cmd
}
