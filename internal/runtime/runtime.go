package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/shekhar396/opspilot-agent/internal/identity"
)

type Runtime struct {
	cfg      config.Config
	logger   *slog.Logger
	identity identity.Identity
}

func New(cfg config.Config, logger *slog.Logger, agentIdentity identity.Identity) (*Runtime, error) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if strings.TrimSpace(cfg.Agent.ServerURL) == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if agentIdentity.ID() == "" {
		return nil, fmt.Errorf("agent identity is required")
	}

	return &Runtime{cfg: cfg, logger: logger, identity: agentIdentity}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	r.logger.Info(
		"agent runtime started",
		"agent_id", r.identity.ID(),
		"agent_name", r.cfg.Agent.Name,
		"server_url", r.cfg.Agent.ServerURL,
	)
	<-ctx.Done()
	r.logger.Info(
		"agent runtime stopped",
		"agent_id", r.identity.ID(),
		"agent_name", r.cfg.Agent.Name,
	)
	return nil
}
