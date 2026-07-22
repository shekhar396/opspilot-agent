package heartbeat

import (
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	if err := Validate(validPayload()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidPayload(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Payload)
		want   string
	}{
		{name: "wrong schema", modify: func(p *Payload) { p.SchemaVersion = "2" }, want: "schema version"},
		{name: "empty schema", modify: func(p *Payload) { p.SchemaVersion = "" }, want: "schema version"},
		{name: "invalid agent ID", modify: func(p *Payload) { p.AgentID = "invalid" }, want: "agent ID"},
		{name: "invalid agent name", modify: func(p *Payload) { p.AgentName = "app server" }, want: "agent name"},
		{name: "invalid agent version", modify: func(p *Payload) { p.AgentVersion = "v1/dirty" }, want: "agent version"},
		{name: "zero timestamp", modify: func(p *Payload) { p.SentAt = time.Time{} }, want: "sent timestamp"},
		{name: "old timestamp", modify: func(p *Payload) { p.SentAt = minimumSentAt.Add(-time.Second) }, want: "sent timestamp"},
		{name: "zero sequence", modify: func(p *Payload) { p.Sequence = 0 }, want: "sequence"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := validPayload()
			test.modify(&payload)
			err := Validate(payload)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want context %q", err, test.want)
			}
		})
	}
}
