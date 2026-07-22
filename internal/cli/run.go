package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/shekhar396/opspilot-agent/internal/identity"
	"github.com/shekhar396/opspilot-agent/internal/logging"
	agentruntime "github.com/shekhar396/opspilot-agent/internal/runtime"
	"github.com/spf13/cobra"
)

func newRunCommand(output io.Writer) *cobra.Command {
	configPath := "configs/opspilot-agent.yaml"

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the OpsPilot Agent",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load runtime configuration: %w", err)
			}

			logger, err := logging.New(cfg.Logging, output)
			if err != nil {
				return fmt.Errorf("create logger: %w", err)
			}

			agentIdentity, err := identity.LoadOrCreate(cfg.Agent.IdentityFile)
			if err != nil {
				return fmt.Errorf("load agent identity: %w", err)
			}

			runtime, err := agentruntime.New(cfg, logger, agentIdentity)
			if err != nil {
				return fmt.Errorf("create agent runtime: %w", err)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if err := runtime.Run(ctx); err != nil {
				return fmt.Errorf("run agent runtime: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", configPath, "path to the configuration file")

	return cmd
}
