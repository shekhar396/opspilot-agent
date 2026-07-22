package config

import "time"

func Default() Config {
	return Config{
		Agent: AgentConfig{
			HeartbeatInterval: Duration{Duration: 30 * time.Second},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
