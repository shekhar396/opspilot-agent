package heartbeat

import (
	"strings"
	"testing"
	"time"
)

const testAgentID = "9fb42f1c-8a12-4db5-a42c-7a4be50efaf1"

var testSentAt = time.Date(2026, 7, 22, 12, 0, 0, 123456789, time.FixedZone("test", 2*60*60))

func TestNew(t *testing.T) {
	payload, err := New(" \t"+testAgentID+"\n", " app-server-01 ", " dev ", testSentAt, 7)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if payload.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", payload.SchemaVersion, SchemaVersion)
	}
	if payload.AgentID != testAgentID {
		t.Errorf("AgentID = %q, want %q", payload.AgentID, testAgentID)
	}
	if payload.AgentName != "app-server-01" {
		t.Errorf("AgentName = %q", payload.AgentName)
	}
	if payload.AgentVersion != "dev" {
		t.Errorf("AgentVersion = %q", payload.AgentVersion)
	}
	if payload.SentAt.Location() != time.UTC {
		t.Errorf("SentAt location = %v, want UTC", payload.SentAt.Location())
	}
	if !payload.SentAt.Equal(testSentAt) {
		t.Errorf("SentAt = %s, want same instant as %s", payload.SentAt, testSentAt)
	}
	if payload.Sequence != 7 {
		t.Errorf("Sequence = %d, want 7", payload.Sequence)
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name         string
		agentID      string
		agentName    string
		agentVersion string
		sentAt       time.Time
		sequence     uint64
	}{
		{name: "zero timestamp", agentID: testAgentID, agentName: "app-server-01", agentVersion: "dev", sequence: 1},
		{name: "old timestamp", agentID: testAgentID, agentName: "app-server-01", agentVersion: "dev", sentAt: minimumSentAt.Add(-time.Nanosecond), sequence: 1},
		{name: "zero sequence", agentID: testAgentID, agentName: "app-server-01", agentVersion: "dev", sentAt: testSentAt},
		{name: "invalid agent ID", agentID: "invalid", agentName: "app-server-01", agentVersion: "dev", sentAt: testSentAt, sequence: 1},
		{name: "empty agent name", agentID: testAgentID, agentVersion: "dev", sentAt: testSentAt, sequence: 1},
		{name: "invalid agent name", agentID: testAgentID, agentName: "app server", agentVersion: "dev", sentAt: testSentAt, sequence: 1},
		{name: "long agent name", agentID: testAgentID, agentName: strings.Repeat("a", 129), agentVersion: "dev", sentAt: testSentAt, sequence: 1},
		{name: "empty agent version", agentID: testAgentID, agentName: "app-server-01", sentAt: testSentAt, sequence: 1},
		{name: "version with spaces", agentID: testAgentID, agentName: "app-server-01", agentVersion: "version 1", sentAt: testSentAt, sequence: 1},
		{name: "long agent version", agentID: testAgentID, agentName: "app-server-01", agentVersion: strings.Repeat("v", 65), sentAt: testSentAt, sequence: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.agentID, test.agentName, test.agentVersion, test.sentAt, test.sequence); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestNewAcceptsVersionFormats(t *testing.T) {
	for _, version := range []string{"dev", "v0.1.0", "0.1.0", "v0.1.0-rc.1", "v0.1.0+build.42"} {
		t.Run(version, func(t *testing.T) {
			if _, err := New(testAgentID, "app-server-01", version, testSentAt, 1); err != nil {
				t.Fatalf("New() error = %v", err)
			}
		})
	}
}

func validPayload() Payload {
	payload, err := New(testAgentID, "app-server-01", "dev", time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC), 1)
	if err != nil {
		panic(err)
	}
	return payload
}
