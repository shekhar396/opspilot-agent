package heartbeat

import "time"

// SchemaVersion is the current heartbeat payload schema version.
const SchemaVersion = "1"

// Payload is the versioned heartbeat message exchanged with a future server.
type Payload struct {
	SchemaVersion string    `json:"schema_version"`
	AgentID       string    `json:"agent_id"`
	AgentName     string    `json:"agent_name"`
	AgentVersion  string    `json:"agent_version"`
	SentAt        time.Time `json:"sent_at"`
	Sequence      uint64    `json:"sequence"`
}

// New constructs and validates a heartbeat payload from explicit input values.
func New(agentID, agentName, agentVersion string, sentAt time.Time, sequence uint64) (Payload, error) {
	payload := Payload{
		SchemaVersion: SchemaVersion,
		AgentID:       trim(agentID),
		AgentName:     trim(agentName),
		AgentVersion:  trim(agentVersion),
		SentAt:        sentAt.UTC().Round(0),
		Sequence:      sequence,
	}
	if err := Validate(payload); err != nil {
		return Payload{}, err
	}
	return payload, nil
}
