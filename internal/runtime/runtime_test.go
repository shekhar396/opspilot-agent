package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/shekhar396/opspilot-agent/internal/identity"
)

func TestNew(t *testing.T) {
	runtime, err := New(validConfig(), testLogger(io.Discard), testIdentity(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("New() returned a nil runtime")
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*config.Config)
		logger     *slog.Logger
		identityFn func(*testing.T) identity.Identity
	}{
		{
			name: "empty agent name",
			modify: func(cfg *config.Config) {
				cfg.Agent.Name = ""
			},
			logger:     testLogger(io.Discard),
			identityFn: testIdentity,
		},
		{
			name: "empty server URL",
			modify: func(cfg *config.Config) {
				cfg.Agent.ServerURL = ""
			},
			logger:     testLogger(io.Discard),
			identityFn: testIdentity,
		},
		{
			name:       "nil logger",
			modify:     func(*config.Config) {},
			logger:     nil,
			identityFn: testIdentity,
		},
		{
			name:       "empty identity",
			modify:     func(*config.Config) {},
			logger:     testLogger(io.Discard),
			identityFn: func(*testing.T) identity.Identity { return identity.Identity{} },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.modify(&cfg)
			if _, err := New(cfg, test.logger, test.identityFn(t)); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestRunWithCancelledContextEmitsLifecycleLogs(t *testing.T) {
	var output bytes.Buffer
	agentIdentity := testIdentity(t)
	runtime, err := New(validConfig(), testLogger(&output), agentIdentity)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entries := decodeEntries(t, output.String())
	if len(entries) != 2 {
		t.Fatalf("log entry count = %d, want 2", len(entries))
	}
	if entries[0]["msg"] != "agent runtime started" {
		t.Errorf("startup msg = %v", entries[0]["msg"])
	}
	if entries[0]["agent_name"] != "app-server-01" {
		t.Errorf("startup agent_name = %v", entries[0]["agent_name"])
	}
	if entries[0]["agent_id"] != agentIdentity.ID() {
		t.Errorf("startup agent_id = %v, want %s", entries[0]["agent_id"], agentIdentity.ID())
	}
	if entries[0]["server_url"] != "https://opspilot.example.com" {
		t.Errorf("startup server_url = %v", entries[0]["server_url"])
	}
	if entries[1]["msg"] != "agent runtime stopped" {
		t.Errorf("shutdown msg = %v", entries[1]["msg"])
	}
	if entries[1]["agent_name"] != "app-server-01" {
		t.Errorf("shutdown agent_name = %v", entries[1]["agent_name"])
	}
	if entries[1]["agent_id"] != agentIdentity.ID() {
		t.Errorf("shutdown agent_id = %v, want %s", entries[1]["agent_id"], agentIdentity.ID())
	}
}

func TestRunExitsAfterContextCancellation(t *testing.T) {
	runtime, err := New(validConfig(), testLogger(io.Discard), testIdentity(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runtime.Run(ctx)
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestRunDoesNotLeakGoroutines(t *testing.T) {
	runtime, err := New(validConfig(), testLogger(io.Discard), testIdentity(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	before := goruntime.NumGoroutine()
	for range 100 {
		if err := runtime.Run(ctx); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	after := goruntime.NumGoroutine()
	if after > before {
		t.Fatalf("goroutine count increased from %d to %d", before, after)
	}
}

func testLogger(output io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, nil))
}

func testIdentity(t *testing.T) identity.Identity {
	t.Helper()
	agentIdentity, err := identity.LoadOrCreate(filepath.Join(t.TempDir(), "agent-id"))
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}
	return agentIdentity
}

func decodeEntries(t *testing.T, output string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	entries := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Unmarshal() error = %v; line = %q", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}

func validConfig() config.Config {
	cfg := config.Default()
	cfg.Agent.Name = "app-server-01"
	cfg.Agent.ServerURL = "https://opspilot.example.com"
	return cfg
}
