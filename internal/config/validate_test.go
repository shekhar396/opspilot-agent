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
		{url: "", valid: false},
		{url: "http://opspilot.example.com", valid: false},
		{url: "opspilot.example.com", valid: false},
		{url: "https://user:pass@opspilot.example.com", valid: false},
		{url: "https://opspilot.example.com/api", valid: false},
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
