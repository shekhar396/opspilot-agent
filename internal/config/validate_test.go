package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAgentName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "app-server-01", valid: true},
		{name: "api_server_02", valid: true},
		{name: "worker.example", valid: true},
		{name: strings.Repeat("a", 128), valid: true},
		{name: strings.Repeat("a", 129), valid: false},
		{name: "", valid: false},
		{name: "   ", valid: false},
		{name: "app server", valid: false},
		{name: "server@01", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Agent.Name = test.name
			if err := Validate(cfg); (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidateServerURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{url: "https://opspilot.example.com", valid: true},
		{url: "https://opspilot.example.com/", valid: true},
		{url: "https://opspilot.example.com:8443", valid: true},
		{url: "https://opspilot.example.com/control-plane", valid: true},
		{url: "", valid: false},
		{url: "http://opspilot.example.com", valid: false},
		{url: "opspilot.example.com", valid: false},
		{url: "https://user:pass@opspilot.example.com", valid: false},
		{url: "https://opspilot.example.com?x=1", valid: false},
		{url: "https://opspilot.example.com#section", valid: false},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			cfg := validConfig()
			cfg.Agent.ServerURL = test.url
			if err := Validate(cfg); (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidateHeartbeatInterval(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration time.Duration
		valid    bool
	}{
		{name: "minimum", duration: 5 * time.Second, valid: true},
		{name: "maximum", duration: time.Hour, valid: true},
		{name: "below minimum", duration: 5*time.Second - time.Nanosecond, valid: false},
		{name: "above maximum", duration: time.Hour + time.Nanosecond, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Agent.HeartbeatInterval.Duration = test.duration
			cfg.Agent.RequestTimeout.Duration = 100 * time.Millisecond
			if err := Validate(cfg); (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidateRequestTimeout(t *testing.T) {
	for _, test := range []struct {
		name              string
		timeout           time.Duration
		heartbeatInterval time.Duration
		valid             bool
	}{
		{name: "minimum", timeout: 100 * time.Millisecond, heartbeatInterval: 30 * time.Second, valid: true},
		{name: "one second", timeout: time.Second, heartbeatInterval: 30 * time.Second, valid: true},
		{name: "default", timeout: 10 * time.Second, heartbeatInterval: 30 * time.Second, valid: true},
		{name: "below interval", timeout: 29 * time.Second, heartbeatInterval: 30 * time.Second, valid: true},
		{name: "maximum", timeout: 2 * time.Minute, heartbeatInterval: 3 * time.Minute, valid: true},
		{name: "zero", timeout: 0, heartbeatInterval: 30 * time.Second, valid: false},
		{name: "negative", timeout: -time.Second, heartbeatInterval: 30 * time.Second, valid: false},
		{name: "below minimum", timeout: 50 * time.Millisecond, heartbeatInterval: 30 * time.Second, valid: false},
		{name: "above maximum", timeout: 2*time.Minute + time.Second, heartbeatInterval: 3 * time.Minute, valid: false},
		{name: "equal to interval", timeout: 30 * time.Second, heartbeatInterval: 30 * time.Second, valid: false},
		{name: "greater than interval", timeout: 45 * time.Second, heartbeatInterval: 30 * time.Second, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Agent.RequestTimeout.Duration = test.timeout
			cfg.Agent.HeartbeatInterval.Duration = test.heartbeatInterval
			if err := Validate(cfg); (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidateIdentityFile(t *testing.T) {
	for _, test := range []struct {
		name  string
		path  string
		valid bool
	}{
		{name: "default absolute path", path: "/var/lib/opspilot-agent/agent-id", valid: true},
		{name: "temporary absolute path", path: "/tmp/opspilot-agent-test/agent-id", valid: true},
		{name: "empty", path: "", valid: false},
		{name: "whitespace", path: "   ", valid: false},
		{name: "relative", path: "agent-id", valid: false},
		{name: "dot relative", path: "./agent-id", valid: false},
		{name: "parent relative", path: "../agent-id", valid: false},
		{name: "root", path: "/", valid: false},
		{name: "trailing slash", path: "/var/lib/opspilot-agent/", valid: false},
		{name: "null byte", path: "/tmp/agent\x00-id", valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Agent.IdentityFile = test.path
			if err := Validate(cfg); (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidateLogging(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run("level "+level, func(t *testing.T) {
			cfg := validConfig()
			cfg.Logging.Level = level
			if err := Validate(cfg); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
	for _, level := range []string{"INFO", "trace", ""} {
		t.Run("invalid level "+level, func(t *testing.T) {
			cfg := validConfig()
			cfg.Logging.Level = level
			if err := Validate(cfg); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
	for _, format := range []string{"json", "text"} {
		t.Run("format "+format, func(t *testing.T) {
			cfg := validConfig()
			cfg.Logging.Format = format
			if err := Validate(cfg); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
	for _, format := range []string{"JSON", "console", ""} {
		t.Run("invalid format "+format, func(t *testing.T) {
			cfg := validConfig()
			cfg.Logging.Format = format
			if err := Validate(cfg); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func validConfig() Config {
	cfg := Default()
	cfg.Agent.Name = "app-server-01"
	cfg.Agent.ServerURL = "https://opspilot.example.com"
	return cfg
}
