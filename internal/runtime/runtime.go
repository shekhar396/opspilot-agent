package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
	"github.com/shekhar396/opspilot-agent/internal/identity"
	"github.com/shekhar396/opspilot-agent/internal/transport"
)

type Runtime struct {
	cfg          config.Config
	logger       *slog.Logger
	identity     identity.Identity
	sender       HeartbeatSender
	agentVersion string
	now          func() time.Time
	newTicker    tickerFactory
}

type ticker interface {
	Chan() <-chan time.Time
	Stop()
}

type tickerFactory func(time.Duration) ticker

type realTicker struct {
	ticker *time.Ticker
}

func (t realTicker) Chan() <-chan time.Time {
	return t.ticker.C
}

func (t realTicker) Stop() {
	t.ticker.Stop()
}

func New(
	cfg config.Config,
	logger *slog.Logger,
	agentIdentity identity.Identity,
	sender HeartbeatSender,
	agentVersion string,
) (*Runtime, error) {
	return newWithClockAndTicker(cfg, logger, agentIdentity, sender, agentVersion, time.Now, func(interval time.Duration) ticker {
		return realTicker{ticker: time.NewTicker(interval)}
	})
}

func newWithClockAndTicker(
	cfg config.Config,
	logger *slog.Logger,
	agentIdentity identity.Identity,
	sender HeartbeatSender,
	agentVersion string,
	now func() time.Time,
	newTicker tickerFactory,
) (*Runtime, error) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if strings.TrimSpace(cfg.Agent.ServerURL) == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	if cfg.Agent.HeartbeatInterval.Duration <= 0 {
		return nil, fmt.Errorf("heartbeat interval must be greater than zero")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if agentIdentity.ID() == "" {
		return nil, fmt.Errorf("agent identity is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("heartbeat sender is required")
	}
	agentVersion = strings.TrimSpace(agentVersion)
	if agentVersion == "" {
		return nil, fmt.Errorf("agent version is required")
	}
	if now == nil {
		return nil, fmt.Errorf("clock function is required")
	}
	if newTicker == nil {
		return nil, fmt.Errorf("ticker factory is required")
	}

	return &Runtime{
		cfg:          cfg,
		logger:       logger,
		identity:     agentIdentity,
		sender:       sender,
		agentVersion: agentVersion,
		now:          now,
		newTicker:    newTicker,
	}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	r.logger.Info(
		"agent runtime started",
		"agent_id", r.identity.ID(),
		"agent_name", r.cfg.Agent.Name,
		"server_url", r.cfg.Agent.ServerURL,
		"heartbeat_interval", r.cfg.Agent.HeartbeatInterval.Duration.String(),
	)
	defer r.logger.Info(
		"agent runtime stopped",
		"agent_id", r.identity.ID(),
		"agent_name", r.cfg.Agent.Name,
	)

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	ticker := r.newTicker(r.cfg.Agent.HeartbeatInterval.Duration)
	defer ticker.Stop()

	sequence := uint64(1)
	for {
		if err := r.sendHeartbeat(ctx, sequence); err != nil {
			return err
		}
		sequence++

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.Chan():
		}
	}
}

func (r *Runtime) sendHeartbeat(ctx context.Context, sequence uint64) error {
	payload, err := heartbeat.New(
		r.identity.ID(),
		r.cfg.Agent.Name,
		r.agentVersion,
		r.now().UTC(),
		sequence,
	)
	if err != nil {
		return fmt.Errorf("build heartbeat payload: %w", err)
	}

	response, err := r.sender.SendHeartbeat(ctx, payload)
	if err == nil {
		attributes := []any{
			"agent_id", r.identity.ID(),
			"sequence", sequence,
			"status_code", response.StatusCode,
		}
		if response.RequestID != "" {
			attributes = append(attributes, "request_id", response.RequestID)
		}
		r.logger.Info("heartbeat delivered", attributes...)
		return nil
	}

	if ctx.Err() != nil {
		return nil
	}

	var statusError *transport.HTTPStatusError
	if errors.As(err, &statusError) {
		attributes := []any{
			"agent_id", r.identity.ID(),
			"sequence", sequence,
			"status_code", statusError.StatusCode,
			"error", err.Error(),
		}
		requestID := statusError.RequestID
		if requestID == "" {
			requestID = response.RequestID
		}
		if requestID != "" {
			attributes = append(attributes, "request_id", requestID)
		}
		r.logger.Warn("heartbeat rejected", attributes...)
		return nil
	}

	failureType := "network"
	if errors.Is(err, context.DeadlineExceeded) {
		failureType = "timeout"
	} else if response.StatusCode != 0 {
		failureType = "unknown"
	}
	r.logger.Error(
		"heartbeat delivery failed",
		"agent_id", r.identity.ID(),
		"sequence", sequence,
		"failure_type", failureType,
		"error", err.Error(),
	)
	return nil
}
