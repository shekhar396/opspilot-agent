package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(*testing.T, Config)
	}{
		{
			name: "full configuration",
			content: `agent:
  name: app-server-01
  server_url: https://opspilot.example.com
  heartbeat_interval: 1m
  request_timeout: 20s
  identity_file: /tmp/opspilot-agent-test/agent-id
logging:
  level: debug
  format: text
`,
			check: func(t *testing.T, cfg Config) {
				if cfg.Agent.HeartbeatInterval.Duration != time.Minute {
					t.Errorf("HeartbeatInterval = %s, want 1m", cfg.Agent.HeartbeatInterval.Duration)
				}
				if cfg.Agent.RequestTimeout.Duration != 20*time.Second {
					t.Errorf("RequestTimeout = %s, want 20s", cfg.Agent.RequestTimeout.Duration)
				}
				if cfg.Logging.Level != "debug" || cfg.Logging.Format != "text" {
					t.Errorf("Logging = %#v, want debug/text", cfg.Logging)
				}
				if cfg.Agent.IdentityFile != "/tmp/opspilot-agent-test/agent-id" {
					t.Errorf("IdentityFile = %q, want explicit path", cfg.Agent.IdentityFile)
				}
			},
		},
		{
			name: "minimal configuration uses defaults",
			content: `agent:
  name: app-server-01
  server_url: https://opspilot.example.com
`,
			check: func(t *testing.T, cfg Config) {
				if cfg.Agent.HeartbeatInterval.Duration != 30*time.Second {
					t.Errorf("HeartbeatInterval = %s, want 30s", cfg.Agent.HeartbeatInterval.Duration)
				}
				if cfg.Agent.RequestTimeout.Duration != 10*time.Second {
					t.Errorf("RequestTimeout = %s, want 10s", cfg.Agent.RequestTimeout.Duration)
				}
				if cfg.Logging.Level != "info" || cfg.Logging.Format != "json" {
					t.Errorf("Logging = %#v, want info/json", cfg.Logging)
				}
				if cfg.Agent.IdentityFile != "/var/lib/opspilot-agent/agent-id" {
					t.Errorf("IdentityFile = %q, want default path", cfg.Agent.IdentityFile)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfig(t, test.content)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			test.check(t, cfg)
		})
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError string
	}{
		{name: "invalid YAML", content: "agent: [\n", wantError: "decode configuration file"},
		{name: "unknown root field", content: validYAML + "unknown: value\n", wantError: "field unknown not found"},
		{name: "unknown nested field", content: strings.Replace(validYAML, "  name:", "  unknown_field: value\n  name:", 1), wantError: "field unknown_field not found"},
		{name: "invalid duration", content: strings.Replace(validYAML, "30s", "soon", 1), wantError: "invalid duration"},
		{name: "integer duration", content: strings.Replace(validYAML, "30s", "30", 1), wantError: "duration must be a string"},
		{name: "multiple documents", content: validYAML + "---\nagent:\n  name: another-server\n", wantError: "multiple YAML documents"},
		{name: "validation failure", content: strings.Replace(validYAML, "https://", "http://", 1), wantError: "validate configuration"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, test.content))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Load() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil || !strings.Contains(err.Error(), "open configuration file") {
		t.Fatalf("Load() error = %v, want open configuration file error", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

const validYAML = `agent:
  name: app-server-01
  server_url: https://opspilot.example.com
  heartbeat_interval: 30s
  request_timeout: 10s
  identity_file: /tmp/opspilot-agent-test/agent-id
logging:
  level: info
  format: json
`
