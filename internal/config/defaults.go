package config

import "time"

func Default() Config {
	return Config{
		Agent: AgentConfig{
			HeartbeatInterval: Duration{Duration: 30 * time.Second},
			RequestTimeout:    Duration{Duration: 10 * time.Second},
			IdentityFile:      "/var/lib/opspilot-agent/agent-id",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
