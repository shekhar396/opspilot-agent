package config

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Agent.Name != "" {
		t.Errorf("Agent.Name = %q, want empty", cfg.Agent.Name)
	}
	if cfg.Agent.ServerURL != "" {
		t.Errorf("Agent.ServerURL = %q, want empty", cfg.Agent.ServerURL)
	}
	if cfg.Agent.HeartbeatInterval.Duration != 30*time.Second {
		t.Errorf("Agent.HeartbeatInterval = %s, want 30s", cfg.Agent.HeartbeatInterval.Duration)
	}
	if cfg.Agent.IdentityFile != "/var/lib/opspilot-agent/agent-id" {
		t.Errorf("Agent.IdentityFile = %q, want default path", cfg.Agent.IdentityFile)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want info", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want json", cfg.Logging.Format)
	}
}
