package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
	"github.com/shekhar396/opspilot-agent/internal/transport"
	"github.com/shekhar396/opspilot-agent/internal/version"
)

func TestRunEndToEndHeartbeat(t *testing.T) {
	type receivedRequest struct {
		method  string
		path    string
		headers http.Header
		payload heartbeat.Payload
		err     error
	}
	received := make(chan receivedRequest, 1)
	var requestCount atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		body, err := io.ReadAll(request.Body)
		var payload heartbeat.Payload
		if err == nil {
			payload, err = heartbeat.Decode(body)
		}
		received <- receivedRequest{
			method:  request.Method,
			path:    request.URL.Path,
			headers: request.Header.Clone(),
			payload: payload,
			err:     err,
		}
		writer.Header().Set("X-Request-ID", "e2e-request")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	directory := t.TempDir()
	identityPath := filepath.Join(directory, "state", "agent-id")
	configPath := writeRunIntegrationConfig(t, server.URL+"/control-plane", identityPath, "info")
	output := newNotifyingBuffer("heartbeat delivered")
	ctx, cancel := context.WithCancel(context.Background())
	command := newRootCommandWithDependencies(output, tlsTestDependencies(server.Client()))
	command.SetContext(ctx)
	command.SetArgs([]string{"run", "--config", configPath})
	done := make(chan error, 1)
	startedAt := time.Now().Add(-time.Second)
	go func() { done <- command.Execute() }()

	var request receivedRequest
	select {
	case request = <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("TLS server did not receive heartbeat")
	}
	output.wait(t)
	cancel()
	if err := waitCommand(t, done); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if request.err != nil {
		t.Fatalf("request decode error = %v", request.err)
	}
	if request.method != http.MethodPost || request.path != "/control-plane"+transport.HeartbeatPath {
		t.Errorf("method/path = %s %s", request.method, request.path)
	}
	if request.payload.Sequence != 1 || request.payload.SchemaVersion != heartbeat.SchemaVersion {
		t.Errorf("sequence/schema = %d/%q", request.payload.Sequence, request.payload.SchemaVersion)
	}
	if request.payload.AgentName != "app-server-01" || request.payload.AgentVersion != version.Version {
		t.Errorf("agent name/version = %q/%q", request.payload.AgentName, request.payload.AgentVersion)
	}
	if request.payload.SentAt.Location() != time.UTC || request.payload.SentAt.Before(startedAt) || request.payload.SentAt.After(time.Now().Add(time.Second)) {
		t.Errorf("sent_at = %s", request.payload.SentAt)
	}
	wantHeaders := map[string]string{
		"Content-Type":              "application/json",
		"Accept":                    "application/json",
		"User-Agent":                "opspilot-agent/" + version.Version,
		"X-OpsPilot-Agent-ID":       request.payload.AgentID,
		"X-OpsPilot-Schema-Version": heartbeat.SchemaVersion,
	}
	for name, want := range wantHeaders {
		if got := request.headers.Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
	identityContent, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(identityContent)) != request.payload.AgentID {
		t.Errorf("persisted identity does not match payload agent ID")
	}
	logs := output.String()
	for _, message := range []string{"agent runtime started", "heartbeat delivered", "agent runtime stopped", "e2e-request"} {
		if !strings.Contains(logs, message) {
			t.Errorf("logs do not contain %q: %s", message, logs)
		}
	}
	if requestCount.Load() != 1 {
		t.Errorf("request count = %d, want 1", requestCount.Load())
	}
}

func TestRunHeartbeatRejectionContinues(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		writer.Header().Set("X-Request-ID", "rejected-request")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(writer, "unavailable")
	}))
	defer server.Close()

	output := newNotifyingBuffer("heartbeat rejected")
	ctx, cancel := context.WithCancel(context.Background())
	command := newRootCommandWithDependencies(output, tlsTestDependencies(server.Client()))
	command.SetContext(ctx)
	command.SetArgs([]string{"run", "--config", writeRunIntegrationConfig(t, server.URL, filepath.Join(t.TempDir(), "state", "agent-id"), "info")})
	done := make(chan error, 1)
	go func() { done <- command.Execute() }()
	output.wait(t)
	select {
	case err := <-done:
		t.Fatalf("runtime exited after rejection: %v", err)
	default:
	}
	cancel()
	if err := waitCommand(t, done); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	logs := output.String()
	for _, value := range []string{"heartbeat rejected", "rejected-request", `"status_code":503`} {
		if !strings.Contains(logs, value) {
			t.Errorf("logs do not contain %q: %s", value, logs)
		}
	}
	if requestCount.Load() != 1 {
		t.Errorf("request count = %d, want 1", requestCount.Load())
	}
}

func TestRunRedirectIsNotFollowed(t *testing.T) {
	var targetCount atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCount.Add(1)
	}))
	defer target.Close()
	var sourceCount atomic.Int32
	source := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		sourceCount.Add(1)
		writer.Header().Set("Location", target.URL)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	output := newNotifyingBuffer("heartbeat rejected")
	ctx, cancel := context.WithCancel(context.Background())
	command := newRootCommandWithDependencies(output, tlsTestDependencies(source.Client()))
	command.SetContext(ctx)
	command.SetArgs([]string{"run", "--config", writeRunIntegrationConfig(t, source.URL, filepath.Join(t.TempDir(), "state", "agent-id"), "info")})
	done := make(chan error, 1)
	go func() { done <- command.Execute() }()
	output.wait(t)
	cancel()
	if err := waitCommand(t, done); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sourceCount.Load() != 1 || targetCount.Load() != 0 {
		t.Errorf("source/target request counts = %d/%d", sourceCount.Load(), targetCount.Load())
	}
}

func TestProductionRedirectPolicy(t *testing.T) {
	client := productionDependencies().newHTTPClient()
	if client == nil || client.CheckRedirect == nil {
		t.Fatal("production redirect policy is missing")
	}
	if err := client.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v", err)
	}
}

func tlsTestDependencies(client *http.Client) dependencies {
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return dependencies{newHTTPClient: func() *http.Client { return client }}
}

func writeRunIntegrationConfig(t *testing.T, serverURL, identityPath, level string) string {
	t.Helper()
	content := fmt.Sprintf(`agent:
  name: app-server-01
  server_url: %s
  heartbeat_interval: 30s
  request_timeout: 1s
  identity_file: %s
logging:
  level: %s
  format: json
`, serverURL, identityPath, level)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

type notifyingBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
	match  string
	done   chan struct{}
	once   sync.Once
}

func newNotifyingBuffer(match string) *notifyingBuffer {
	return &notifyingBuffer{match: match, done: make(chan struct{})}
}

func (buffer *notifyingBuffer) Write(data []byte) (int, error) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	written, err := buffer.buffer.Write(data)
	if strings.Contains(buffer.buffer.String(), buffer.match) {
		buffer.once.Do(func() { close(buffer.done) })
	}
	return written, err
}

func (buffer *notifyingBuffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.String()
}

func (buffer *notifyingBuffer) wait(t *testing.T) {
	t.Helper()
	select {
	case <-buffer.done:
	case <-time.After(3 * time.Second):
		t.Fatalf("logs did not contain %q: %s", buffer.match, buffer.String())
	}
}

func waitCommand(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("command did not exit")
		return nil
	}
}

func decodeLogLines(t *testing.T, output string) []map[string]any {
	t.Helper()
	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON log: %v", err)
		}
		entries = append(entries, entry)
	}
	return entries
}
