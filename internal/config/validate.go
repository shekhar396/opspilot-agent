package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var agentNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return fmt.Errorf("agent.name is required")
	}
	if len(cfg.Agent.Name) > 128 {
		return fmt.Errorf("agent.name must not exceed 128 characters")
	}
	if !agentNamePattern.MatchString(cfg.Agent.Name) {
		return fmt.Errorf("agent.name may contain only A-Z, a-z, 0-9, period, underscore, and hyphen")
	}

	if strings.TrimSpace(cfg.Agent.ServerURL) == "" {
		return fmt.Errorf("agent.server_url is required")
	}
	serverURL, err := url.Parse(cfg.Agent.ServerURL)
	if err != nil {
		return fmt.Errorf("agent.server_url must be a valid URL: %w", err)
	}
	if serverURL.Scheme != "https" {
		return fmt.Errorf("agent.server_url must use https")
	}
	if serverURL.Host == "" {
		return fmt.Errorf("agent.server_url must include a host")
	}
	if serverURL.User != nil {
		return fmt.Errorf("agent.server_url must not include user information")
	}
	if serverURL.RawQuery != "" {
		return fmt.Errorf("agent.server_url must not include a query string")
	}
	if serverURL.Fragment != "" {
		return fmt.Errorf("agent.server_url must not include a fragment")
	}
	if serverURL.Path != "" && serverURL.Path != "/" {
		return fmt.Errorf("agent.server_url path must be empty or /")
	}

	if cfg.Agent.HeartbeatInterval.Duration < 5*time.Second {
		return fmt.Errorf("agent.heartbeat_interval must be at least 5s")
	}
	if cfg.Agent.HeartbeatInterval.Duration > time.Hour {
		return fmt.Errorf("agent.heartbeat_interval must not exceed 1h")
	}

	switch cfg.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	switch cfg.Logging.Format {
	case "json", "text":
	default:
		return fmt.Errorf("logging.format must be one of: json, text")
	}

	return nil
}
