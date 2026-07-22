package heartbeat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/identity"
)

var (
	minimumSentAt       = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	agentNamePattern    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	agentVersionPattern = regexp.MustCompile(`^[A-Za-z0-9._+\-]+$`)
)

// Validate verifies that a heartbeat payload satisfies schema version 1.
func Validate(payload Payload) error {
	if payload.SchemaVersion != SchemaVersion {
		return fmt.Errorf("validate schema version: must be %q", SchemaVersion)
	}
	if _, err := identity.Parse(payload.AgentID); err != nil {
		return fmt.Errorf("validate agent ID: %w", err)
	}
	if err := validateAgentName(payload.AgentName); err != nil {
		return fmt.Errorf("validate agent name: %w", err)
	}
	if err := validateAgentVersion(payload.AgentVersion); err != nil {
		return fmt.Errorf("validate agent version: %w", err)
	}
	if payload.SentAt.IsZero() {
		return fmt.Errorf("validate sent timestamp: must not be zero")
	}
	if payload.SentAt.Before(minimumSentAt) {
		return fmt.Errorf("validate sent timestamp: must not be before %s", minimumSentAt.Format(time.RFC3339))
	}
	if payload.Sequence == 0 {
		return fmt.Errorf("validate sequence: must be at least 1")
	}
	return nil
}

func validateAgentName(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("is required")
	}
	if len(value) > 128 {
		return fmt.Errorf("must not exceed 128 characters")
	}
	if !agentNamePattern.MatchString(value) {
		return fmt.Errorf("contains unsupported characters")
	}
	return nil
}

func validateAgentVersion(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("is required")
	}
	if len(value) > 64 {
		return fmt.Errorf("must not exceed 64 characters")
	}
	if !agentVersionPattern.MatchString(value) {
		return fmt.Errorf("contains unsupported characters")
	}
	return nil
}

func trim(value string) string {
	return strings.TrimSpace(value)
}
