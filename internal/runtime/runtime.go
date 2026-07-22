package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/shekhar396/opspilot-agent/internal/config"
)

type Runtime struct {
	cfg config.Config
}

func New(cfg config.Config) (*Runtime, error) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if strings.TrimSpace(cfg.Agent.ServerURL) == "" {
		return nil, fmt.Errorf("server URL is required")
	}

	return &Runtime{cfg: cfg}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
