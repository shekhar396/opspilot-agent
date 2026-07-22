package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/config"
	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
	"github.com/shekhar396/opspilot-agent/internal/identity"
	"github.com/shekhar396/opspilot-agent/internal/transport"
)

func TestNew(t *testing.T) {
	runtime, err := New(validConfig(), testLogger(io.Discard, slog.LevelInfo), testIdentity(t), newFakeSender(), "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("New() returned a nil runtime")
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	validLogger := testLogger(io.Discard, slog.LevelInfo)
	validIdentity := testIdentity(t)
	validSender := newFakeSender()
	validClock := func() time.Time { return time.Now() }
	validTicker := func(time.Duration) ticker { return newManualTicker() }

	tests := []struct {
		name          string
		modify        func(*config.Config)
		logger        *slog.Logger
		agentIdentity identity.Identity
		sender        HeartbeatSender
		version       string
		now           func() time.Time
		newTicker     tickerFactory
	}{
		{name: "empty agent name", modify: func(cfg *config.Config) { cfg.Agent.Name = "" }, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "dev", now: validClock, newTicker: validTicker},
		{name: "empty server URL", modify: func(cfg *config.Config) { cfg.Agent.ServerURL = "" }, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "dev", now: validClock, newTicker: validTicker},
		{name: "invalid heartbeat interval", modify: func(cfg *config.Config) { cfg.Agent.HeartbeatInterval.Duration = 0 }, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "dev", now: validClock, newTicker: validTicker},
		{name: "nil logger", modify: func(*config.Config) {}, agentIdentity: validIdentity, sender: validSender, version: "dev", now: validClock, newTicker: validTicker},
		{name: "empty identity", modify: func(*config.Config) {}, logger: validLogger, sender: validSender, version: "dev", now: validClock, newTicker: validTicker},
		{name: "nil sender", modify: func(*config.Config) {}, logger: validLogger, agentIdentity: validIdentity, version: "dev", now: validClock, newTicker: validTicker},
		{name: "empty version", modify: func(*config.Config) {}, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "   ", now: validClock, newTicker: validTicker},
		{name: "nil clock", modify: func(*config.Config) {}, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "dev", newTicker: validTicker},
		{name: "nil ticker factory", modify: func(*config.Config) {}, logger: validLogger, agentIdentity: validIdentity, sender: validSender, version: "dev", now: validClock},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.modify(&cfg)
			if _, err := newWithClockAndTicker(cfg, test.logger, test.agentIdentity, test.sender, test.version, test.now, test.newTicker); err == nil {
				t.Fatal("newWithClockAndTicker() error = nil")
			}
		})
	}
}

func TestRunAlreadyCancelled(t *testing.T) {
	var output bytes.Buffer
	sender := newFakeSender()
	var tickerCreated atomic.Bool
	runtime := mustRuntime(t, testLogger(&output, slog.LevelInfo), sender, func() time.Time { return fixedTime(1) }, func(time.Duration) ticker {
		tickerCreated.Store(true)
		return newManualTicker()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if sender.count() != 0 || tickerCreated.Load() {
		t.Fatalf("sends/ticker = %d/%v, want 0/false", sender.count(), tickerCreated.Load())
	}
	assertMessages(t, output.String(), "agent runtime started", "agent runtime stopped")
}

func TestRunImmediateAndPeriodicHeartbeats(t *testing.T) {
	sender := newFakeSender()
	manualTicker := newManualTicker()
	times := []time.Time{fixedTime(1), fixedTime(2), fixedTime(3)}
	clock := queuedClock(times)
	runtime := mustRuntime(t, testLogger(io.Discard, slog.LevelInfo), sender, clock, func(time.Duration) ticker { return manualTicker })
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(runtime, ctx)

	waitForSends(t, sender, 1)
	select {
	case <-sender.sent:
		t.Fatal("heartbeat sent without ticker event")
	case <-time.After(20 * time.Millisecond):
	}
	manualTicker.tick()
	waitForSends(t, sender, 2)
	manualTicker.tick()
	waitForSends(t, sender, 3)
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	payloads := sender.recordedPayloads()
	for index, payload := range payloads {
		wantSequence := uint64(index + 1)
		if payload.Sequence != wantSequence {
			t.Errorf("sequence %d = %d", index, payload.Sequence)
		}
		if payload.AgentID != runtime.identity.ID() || payload.AgentName != "app-server-01" || payload.AgentVersion != "dev" {
			t.Errorf("payload identity/name/version = %#v", payload)
		}
		if !payload.SentAt.Equal(times[index]) || payload.SentAt.Location() != time.UTC {
			t.Errorf("timestamp %d = %s", index, payload.SentAt)
		}
		if err := heartbeat.Validate(payload); err != nil {
			t.Errorf("payload %d invalid: %v", index, err)
		}
	}
	if manualTicker.stopCount.Load() != 1 {
		t.Errorf("ticker stop count = %d, want 1", manualTicker.stopCount.Load())
	}
}

func TestRunDeliveryResultsContinueAndLog(t *testing.T) {
	var output bytes.Buffer
	sentinel := errors.New("network unavailable")
	sender := newFakeSender()
	sender.results = []sendResult{
		{response: transport.Response{StatusCode: 503, RequestID: "reject-id"}, err: &transport.HTTPStatusError{StatusCode: 503, RequestID: "reject-id", Message: "unavailable"}},
		{err: fmt.Errorf("request timeout: %w", context.DeadlineExceeded)},
		{err: sentinel},
		{response: transport.Response{StatusCode: 202, RequestID: "success-id"}},
	}
	manualTicker := newManualTicker()
	runtime := mustRuntime(t, testLogger(&output, slog.LevelDebug), sender, func() time.Time { return fixedTime(1) }, func(time.Duration) ticker { return manualTicker })
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(runtime, ctx)
	for count := 1; count <= 4; count++ {
		if count > 1 {
			manualTicker.tick()
		}
		waitForSends(t, sender, count)
	}
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entries := decodeEntries(t, output.String())
	assertLog(t, entries, "heartbeat rejected", "WARN", map[string]any{"sequence": float64(1), "status_code": float64(503), "request_id": "reject-id"})
	assertLog(t, entries, "heartbeat delivery failed", "ERROR", map[string]any{"sequence": float64(2), "failure_type": "timeout"})
	assertLog(t, entries, "heartbeat delivery failed", "ERROR", map[string]any{"sequence": float64(3), "failure_type": "network"})
	assertLog(t, entries, "heartbeat delivered", "INFO", map[string]any{"sequence": float64(4), "status_code": float64(202), "request_id": "success-id"})
}

func TestRunCancellationDuringSend(t *testing.T) {
	var output bytes.Buffer
	sender := newFakeSender()
	sender.block = true
	manualTicker := newManualTicker()
	runtime := mustRuntime(t, testLogger(&output, slog.LevelInfo), sender, func() time.Time { return fixedTime(1) }, func(time.Duration) ticker { return manualTicker })
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(runtime, ctx)
	waitForSends(t, sender, 1)
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(output.String(), "heartbeat delivery failed") {
		t.Fatalf("normal cancellation logged as delivery failure: %s", output.String())
	}
	assertMessages(t, output.String(), "agent runtime started", "agent runtime stopped")
	if manualTicker.stopCount.Load() != 1 {
		t.Errorf("ticker stop count = %d", manualTicker.stopCount.Load())
	}
}

func TestRunDoesNotOverlapSends(t *testing.T) {
	sender := newFakeSender()
	sender.block = true
	sender.release = make(chan struct{})
	manualTicker := newManualTicker()
	runtime := mustRuntime(t, testLogger(io.Discard, slog.LevelInfo), sender, func() time.Time { return fixedTime(1) }, func(time.Duration) ticker { return manualTicker })
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(runtime, ctx)
	waitForSends(t, sender, 1)
	manualTicker.tick()
	manualTicker.tick()
	time.Sleep(20 * time.Millisecond)
	if sender.count() != 1 || sender.maxActive.Load() != 1 {
		t.Fatalf("send count/max active = %d/%d", sender.count(), sender.maxActive.Load())
	}
	close(sender.release)
	waitForSends(t, sender, 2)
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if sender.maxActive.Load() != 1 {
		t.Errorf("max active sends = %d", sender.maxActive.Load())
	}
}

func TestRunErrorLevelFiltering(t *testing.T) {
	var output bytes.Buffer
	sender := newFakeSender()
	sender.results = []sendResult{
		{response: transport.Response{StatusCode: 202}},
		{err: &transport.HTTPStatusError{StatusCode: 503}},
		{err: errors.New("network failed")},
	}
	manualTicker := newManualTicker()
	runtime := mustRuntime(t, testLogger(&output, slog.LevelError), sender, func() time.Time { return fixedTime(1) }, func(time.Duration) ticker { return manualTicker })
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(runtime, ctx)
	for count := 1; count <= 3; count++ {
		if count > 1 {
			manualTicker.tick()
		}
		waitForSends(t, sender, count)
	}
	cancel()
	_ = waitRun(t, done)
	entries := decodeEntries(t, output.String())
	if len(entries) != 1 || entries[0]["msg"] != "heartbeat delivery failed" {
		t.Fatalf("error-level entries = %#v", entries)
	}
}

func TestRunHeartbeatConstructionFailure(t *testing.T) {
	runtime := mustRuntime(t, testLogger(io.Discard, slog.LevelInfo), newFakeSender(), func() time.Time { return fixedTime(1) }, func(time.Duration) ticker { return newManualTicker() })
	runtime.agentVersion = "invalid version"
	err := runtime.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "build heartbeat payload") {
		t.Fatalf("Run() error = %v", err)
	}
}

type sendResult struct {
	response transport.Response
	err      error
}

type fakeSender struct {
	mu        sync.Mutex
	payloads  []heartbeat.Payload
	results   []sendResult
	sent      chan struct{}
	block     bool
	release   chan struct{}
	active    atomic.Int32
	maxActive atomic.Int32
}

func newFakeSender() *fakeSender {
	return &fakeSender{sent: make(chan struct{}, 100)}
}

func (sender *fakeSender) SendHeartbeat(ctx context.Context, payload heartbeat.Payload) (transport.Response, error) {
	active := sender.active.Add(1)
	defer sender.active.Add(-1)
	for {
		maximum := sender.maxActive.Load()
		if active <= maximum || sender.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}

	sender.mu.Lock()
	index := len(sender.payloads)
	sender.payloads = append(sender.payloads, payload)
	var result sendResult
	if index < len(sender.results) {
		result = sender.results[index]
	} else {
		result.response = transport.Response{StatusCode: 202}
	}
	block := sender.block
	release := sender.release
	sender.mu.Unlock()
	sender.sent <- struct{}{}

	if block {
		if release == nil {
			<-ctx.Done()
			return transport.Response{}, ctx.Err()
		}
		select {
		case <-ctx.Done():
			return transport.Response{}, ctx.Err()
		case <-release:
		}
	}
	return result.response, result.err
}

func (sender *fakeSender) count() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return len(sender.payloads)
}

func (sender *fakeSender) recordedPayloads() []heartbeat.Payload {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]heartbeat.Payload(nil), sender.payloads...)
}

type manualTicker struct {
	channel   chan time.Time
	stopCount atomic.Int32
}

func newManualTicker() *manualTicker {
	return &manualTicker{channel: make(chan time.Time, 10)}
}

func (ticker *manualTicker) Chan() <-chan time.Time {
	return ticker.channel
}

func (ticker *manualTicker) Stop() {
	ticker.stopCount.Add(1)
}

func (ticker *manualTicker) tick() {
	ticker.channel <- time.Time{}
}

func mustRuntime(t *testing.T, logger *slog.Logger, sender HeartbeatSender, now func() time.Time, newTicker tickerFactory) *Runtime {
	t.Helper()
	runtime, err := newWithClockAndTicker(validConfig(), logger, testIdentity(t), sender, "dev", now, newTicker)
	if err != nil {
		t.Fatalf("newWithClockAndTicker() error = %v", err)
	}
	return runtime
}

func runAsync(runtime *Runtime, ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx) }()
	return done
}

func waitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("Run() did not return")
		return nil
	}
}

func waitForSends(t *testing.T, sender *fakeSender, count int) {
	t.Helper()
	select {
	case <-sender.sent:
		if sender.count() < count {
			t.Fatalf("send count = %d, want at least %d", sender.count(), count)
		}
	case <-time.After(time.Second):
		t.Fatalf("send count = %d, want %d", sender.count(), count)
	}
}

func queuedClock(values []time.Time) func() time.Time {
	var mutex sync.Mutex
	index := 0
	return func() time.Time {
		mutex.Lock()
		defer mutex.Unlock()
		value := values[index]
		index++
		return value
	}
}

func fixedTime(second int) time.Time {
	return time.Date(2026, 7, 22, 14, 0, second, 0, time.FixedZone("test", 2*60*60))
}

func testLogger(output io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level}))
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
	if strings.TrimSpace(output) == "" {
		return nil
	}
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

func assertMessages(t *testing.T, output string, messages ...string) {
	t.Helper()
	entries := decodeEntries(t, output)
	if len(entries) != len(messages) {
		t.Fatalf("entry count = %d, want %d: %s", len(entries), len(messages), output)
	}
	for index, message := range messages {
		if entries[index]["msg"] != message {
			t.Errorf("message %d = %v, want %q", index, entries[index]["msg"], message)
		}
	}
}

func assertLog(t *testing.T, entries []map[string]any, message, level string, fields map[string]any) {
	t.Helper()
	for _, entry := range entries {
		if entry["msg"] != message {
			continue
		}
		matches := true
		for name, want := range fields {
			if entry[name] != want {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if entry["level"] != level {
			t.Errorf("%s level = %v, want %s", message, entry["level"], level)
		}
		return
	}
	t.Errorf("log message %q not found", message)
}

func validConfig() config.Config {
	cfg := config.Default()
	cfg.Agent.Name = "app-server-01"
	cfg.Agent.ServerURL = "https://opspilot.example.com"
	return cfg
}
