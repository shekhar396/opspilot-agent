package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
)

func TestNew(t *testing.T) {
	client := &http.Client{}
	for _, serverURL := range []string{
		"https://opspilot.example.com",
		" https://opspilot.example.com/control-plane/ ",
	} {
		transport, err := New(client, serverURL, 10*time.Second)
		if err != nil {
			t.Fatalf("New(%q) error = %v", serverURL, err)
		}
		if transport.client != client {
			t.Fatal("New() did not preserve the supplied client")
		}
		if transport.timeout != 10*time.Second {
			t.Errorf("timeout = %s", transport.timeout)
		}
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	client := &http.Client{}
	tests := []struct {
		name      string
		client    *http.Client
		serverURL string
		timeout   time.Duration
	}{
		{name: "nil client", serverURL: "https://example.com", timeout: time.Second},
		{name: "empty URL", client: client, timeout: time.Second},
		{name: "relative URL", client: client, serverURL: "/relative", timeout: time.Second},
		{name: "HTTP URL", client: client, serverURL: "http://example.com", timeout: time.Second},
		{name: "missing host", client: client, serverURL: "https:///path", timeout: time.Second},
		{name: "user info", client: client, serverURL: "https://user:pass@example.com", timeout: time.Second},
		{name: "query", client: client, serverURL: "https://example.com?x=1", timeout: time.Second},
		{name: "empty query", client: client, serverURL: "https://example.com?", timeout: time.Second},
		{name: "fragment", client: client, serverURL: "https://example.com#fragment", timeout: time.Second},
		{name: "opaque URL", client: client, serverURL: "https:opaque", timeout: time.Second},
		{name: "zero timeout", client: client, serverURL: "https://example.com"},
		{name: "negative timeout", client: client, serverURL: "https://example.com", timeout: -time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.client, test.serverURL, test.timeout); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestHeartbeatURL(t *testing.T) {
	for _, test := range []struct {
		base string
		want string
	}{
		{base: "https://opspilot.example.com", want: "https://opspilot.example.com/api/v1/agent/heartbeat"},
		{base: "https://opspilot.example.com/control-plane", want: "https://opspilot.example.com/control-plane/api/v1/agent/heartbeat"},
		{base: "https://opspilot.example.com/control-plane/", want: "https://opspilot.example.com/control-plane/api/v1/agent/heartbeat"},
	} {
		transport, err := New(&http.Client{}, test.base, time.Second)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if got := transport.heartbeatURL(); got != test.want {
			t.Errorf("heartbeatURL() = %q, want %q", got, test.want)
		}
	}
}

func TestSendHeartbeatRequest(t *testing.T) {
	payload := testPayload(t, 7)
	original := payload
	var received heartbeat.Payload
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if request.URL.Path != "/control-plane"+HeartbeatPath {
			t.Errorf("path = %q", request.URL.Path)
		}
		if request.URL.RawQuery != "" || request.URL.Fragment != "" || request.TLS == nil {
			t.Errorf("unexpected URL/TLS state: %#v", request.URL)
		}
		wantHeaders := map[string]string{
			"Content-Type":              "application/json",
			"Accept":                    "application/json",
			"User-Agent":                "opspilot-agent/dev",
			"X-OpsPilot-Agent-ID":       payload.AgentID,
			"X-OpsPilot-Schema-Version": heartbeat.SchemaVersion,
		}
		for name, want := range wantHeaders {
			if got := request.Header.Get(name); got != want {
				t.Errorf("header %s = %q, want %q", name, got, want)
			}
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("ReadAll() error = %v", err)
		}
		if bytes.HasSuffix(body, []byte("\n")) {
			t.Error("request body ends with newline")
		}
		received, err = heartbeat.Decode(body)
		if err != nil {
			t.Errorf("heartbeat.Decode() error = %v", err)
		}
		writer.Header().Set("X-Request-ID", " request-7 ")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	transport, err := New(server.Client(), server.URL+"/control-plane/", time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	response, err := transport.SendHeartbeat(context.Background(), payload)
	if err != nil {
		t.Fatalf("SendHeartbeat() error = %v", err)
	}
	if response.StatusCode != http.StatusAccepted || response.RequestID != "request-7" {
		t.Errorf("response = %#v", response)
	}
	if !reflect.DeepEqual(received, payload) {
		t.Errorf("received payload = %#v, want %#v", received, payload)
	}
	if !reflect.DeepEqual(payload, original) {
		t.Fatal("SendHeartbeat() mutated payload")
	}
}

func TestSendHeartbeatSuccessStatuses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("X-Request-ID", " success-id ")
				writer.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = io.WriteString(writer, "unexpected but accepted body")
				}
			}))
			defer server.Close()
			transport := mustTransport(t, server.Client(), server.URL, time.Second)
			response, err := transport.SendHeartbeat(context.Background(), testPayload(t, 1))
			if err != nil {
				t.Fatalf("SendHeartbeat() error = %v", err)
			}
			if response.StatusCode != status || response.RequestID != "success-id" {
				t.Errorf("response = %#v", response)
			}
		})
	}
}

func TestSendHeartbeatFailureStatuses(t *testing.T) {
	for _, status := range []int{400, 401, 404, 409, 429, 500, 503, http.StatusCreated} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("X-Request-ID", " failure-id ")
				writer.WriteHeader(status)
				if status != 404 {
					_, _ = io.WriteString(writer, " service\n unavailable ")
				}
			}))
			defer server.Close()
			transport := mustTransport(t, server.Client(), server.URL, time.Second)
			response, err := transport.SendHeartbeat(context.Background(), testPayload(t, 1))
			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("error = %v, want HTTPStatusError", err)
			}
			if response.StatusCode != status || statusErr.StatusCode != status {
				t.Errorf("response/error status = %d/%d", response.StatusCode, statusErr.StatusCode)
			}
			if response.RequestID != "failure-id" || statusErr.RequestID != "failure-id" {
				t.Errorf("request IDs = %q/%q", response.RequestID, statusErr.RequestID)
			}
			if status == 404 && statusErr.Message != "" {
				t.Errorf("empty body message = %q", statusErr.Message)
			}
			if status != 404 && statusErr.Message != "service unavailable" {
				t.Errorf("message = %q", statusErr.Message)
			}
		})
	}
}

func TestSendHeartbeatBoundsResponseBodies(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(writer, strings.Repeat("x", MaxResponseBodyBytes+5000))
		}))
		defer server.Close()
		_, err := mustTransport(t, server.Client(), server.URL, time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) || !statusErr.Truncated {
			t.Fatalf("error = %#v, want truncated HTTPStatusError", err)
		}
		if len([]rune(statusErr.Message)) > MaxErrorMessageLength {
			t.Errorf("message length = %d", len([]rune(statusErr.Message)))
		}
	})

	t.Run("success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Length", "16384")
			writer.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(writer, strings.Repeat("x", MaxResponseBodyBytes*2))
		}))
		defer server.Close()
		if _, err := mustTransport(t, server.Client(), server.URL, time.Second).SendHeartbeat(context.Background(), testPayload(t, 1)); err != nil {
			t.Fatalf("SendHeartbeat() error = %v", err)
		}
	})
}

func TestSendHeartbeatContextCancellation(t *testing.T) {
	t.Run("already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		transport := mustTransport(t, &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return nil, request.Context().Err()
		})}, "https://example.com", time.Second)
		_, err := transport.SendHeartbeat(ctx, testPayload(t, 1))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})

	t.Run("while blocked", func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(started)
			select {
			case <-request.Context().Done():
			case <-release:
			}
		}))
		defer server.Close()
		ctx, cancel := context.WithCancel(context.Background())
		transport := mustTransport(t, server.Client(), server.URL, time.Second)
		payload := testPayload(t, 1)
		done := make(chan error, 1)
		go func() {
			_, err := transport.SendHeartbeat(ctx, payload)
			done <- err
		}()
		<-started
		cancel()
		err := <-done
		close(release)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})
}

func TestSendHeartbeatTimeoutAndCallerDeadline(t *testing.T) {
	for _, test := range []struct {
		name             string
		transportTimeout time.Duration
		callerTimeout    time.Duration
	}{
		{name: "transport timeout", transportTimeout: 50 * time.Millisecond},
		{name: "caller deadline", transportTimeout: time.Second, callerTimeout: 30 * time.Millisecond},
	} {
		t.Run(test.name, func(t *testing.T) {
			release := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				select {
				case <-request.Context().Done():
				case <-release:
				}
			}))
			defer server.Close()
			ctx := context.Background()
			var cancel context.CancelFunc
			if test.callerTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, test.callerTimeout)
				defer cancel()
			}
			started := time.Now()
			_, err := mustTransport(t, server.Client(), server.URL, test.transportTimeout).SendHeartbeat(ctx, testPayload(t, 1))
			close(release)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want context.DeadlineExceeded", err)
			}
			if elapsed := time.Since(started); elapsed > 750*time.Millisecond {
				t.Fatalf("elapsed time = %s", elapsed)
			}
		})
	}
}

func TestSendHeartbeatInjectedNetworkAndRedirectErrors(t *testing.T) {
	t.Run("network", func(t *testing.T) {
		sentinel := errors.New("network failed")
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, sentinel })}
		_, err := mustTransport(t, client, "https://example.com", time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want sentinel", err)
		}
		var statusErr *HTTPStatusError
		if errors.As(err, &statusErr) {
			t.Fatalf("network error classified as HTTPStatusError: %v", err)
		}
	})

	t.Run("redirect", func(t *testing.T) {
		sentinel := errors.New("redirect rejected")
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Location", "/target")
			writer.WriteHeader(http.StatusFound)
		}))
		defer server.Close()
		client := server.Client()
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return sentinel }
		_, err := mustTransport(t, client, server.URL, time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want redirect sentinel", err)
		}
	})
}

func TestSendHeartbeatBodyErrors(t *testing.T) {
	closeErr := errors.New("close failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       &errorReadCloser{Reader: strings.NewReader("body"), closeErr: closeErr},
		}, nil
	})}
	response, err := mustTransport(t, client, "https://example.com", time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
	if response.StatusCode != http.StatusOK || !errors.Is(err, closeErr) {
		t.Fatalf("response/error = %#v/%v", response, err)
	}

	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       &errorReadCloser{Reader: strings.NewReader("rejected"), closeErr: closeErr},
		}, nil
	})
	_, err = mustTransport(t, client, "https://example.com", time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error close replaced HTTPStatusError: %v", err)
	}

	readErr := errors.New("read failed")
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       &errorReadCloser{Reader: errorReader{err: readErr}},
		}, nil
	})
	response, err = mustTransport(t, client, "https://example.com", time.Second).SendHeartbeat(context.Background(), testPayload(t, 1))
	if response.StatusCode != http.StatusOK || !errors.Is(err, readErr) {
		t.Fatalf("read response/error = %#v/%v", response, err)
	}
}

func TestSendHeartbeatRejectsInvalidInputs(t *testing.T) {
	transport := mustTransport(t, &http.Client{}, "https://example.com", time.Second)
	if _, err := (*Transport)(nil).SendHeartbeat(context.Background(), testPayload(t, 1)); err == nil {
		t.Fatal("nil receiver error = nil")
	}
	if _, err := transport.SendHeartbeat(nil, testPayload(t, 1)); err == nil {
		t.Fatal("nil context error = nil")
	}
	invalid := testPayload(t, 1)
	invalid.Sequence = 0
	if _, err := transport.SendHeartbeat(context.Background(), invalid); err == nil {
		t.Fatal("invalid payload error = nil")
	}
}

func TestSendHeartbeatConcurrent(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("ReadAll() error = %v", err)
		}
		if _, err := heartbeat.Decode(body); err != nil {
			t.Errorf("Decode() error = %v", err)
		}
		requests.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	transport := mustTransport(t, server.Client(), server.URL, time.Second)
	const count = 16
	payloads := make([]heartbeat.Payload, 0, count)
	for sequence := 1; sequence <= count; sequence++ {
		payloads = append(payloads, testPayload(t, uint64(sequence)))
	}

	var group sync.WaitGroup
	errorsChannel := make(chan error, count)
	for _, payload := range payloads {
		group.Add(1)
		go func(payload heartbeat.Payload) {
			defer group.Done()
			_, err := transport.SendHeartbeat(context.Background(), payload)
			errorsChannel <- err
		}(payload)
	}
	group.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Errorf("SendHeartbeat() error = %v", err)
		}
	}
	if requests.Load() != count {
		t.Errorf("requests = %d, want %d", requests.Load(), count)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type errorReadCloser struct {
	io.Reader
	closeErr error
}

type errorReader struct {
	err error
}

func (reader errorReader) Read([]byte) (int, error) {
	return 0, reader.err
}

func (body *errorReadCloser) Close() error {
	return body.closeErr
}

func mustTransport(t *testing.T, client *http.Client, serverURL string, timeout time.Duration) *Transport {
	t.Helper()
	transport, err := New(client, serverURL, timeout)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return transport
}

func testPayload(t *testing.T, sequence uint64) heartbeat.Payload {
	t.Helper()
	payload, err := heartbeat.New(
		"9fb42f1c-8a12-4db5-a42c-7a4be50efaf1",
		"app-server-01",
		"dev",
		time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC),
		sequence,
	)
	if err != nil {
		t.Fatalf("heartbeat.New() error = %v", err)
	}
	return payload
}
