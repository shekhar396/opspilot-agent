package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shekhar396/opspilot-agent/internal/config"
)

type Runtime struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) (*Runtime, error) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if strings.TrimSpace(cfg.Agent.ServerURL) == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Runtime{cfg: cfg, logger: logger}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	r.logger.Info(
		"agent runtime started",
		"agent_name", r.cfg.Agent.Name,
		"server_url", r.cfg.Agent.ServerURL,
	)
	<-ctx.Done()
	r.logger.Info(
		"agent runtime stopped",
		"agent_name", r.cfg.Agent.Name,
	)
	return nil
}
