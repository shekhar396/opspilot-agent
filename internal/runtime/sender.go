package runtime

import (
	"context"

	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
	"github.com/shekhar396/opspilot-agent/internal/transport"
)

// HeartbeatSender delivers one prebuilt heartbeat payload.
type HeartbeatSender interface {
	SendHeartbeat(context.Context, heartbeat.Payload) (transport.Response, error)
}
