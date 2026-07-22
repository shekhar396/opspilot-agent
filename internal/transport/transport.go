package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pathpkg "path"
	"strings"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/heartbeat"
)

// HeartbeatPath is the server endpoint for heartbeat payloads.
const HeartbeatPath = "/api/v1/agent/heartbeat"

// MaxResponseBodyBytes bounds all response-body reads.
const MaxResponseBodyBytes = 8 * 1024

// Transport sends prebuilt heartbeat payloads using an injected HTTP client.
type Transport struct {
	client  *http.Client
	baseURL *url.URL
	timeout time.Duration
}

// New constructs an HTTPS heartbeat transport without modifying the client.
func New(client *http.Client, serverURL string, timeout time.Duration) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("HTTP client is required")
	}
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	if parsed.Opaque != "" {
		return nil, fmt.Errorf("server URL must not be opaque")
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("server URL must be absolute")
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("server URL must use https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("server URL must include a host")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("server URL must not include user information")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return nil, fmt.Errorf("server URL must not include a query")
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("server URL must not include a fragment")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("request timeout must be greater than zero")
	}

	baseURL := *parsed
	return &Transport{client: client, baseURL: &baseURL, timeout: timeout}, nil
}

// SendHeartbeat posts one validated heartbeat payload using the caller context.
func (t *Transport) SendHeartbeat(ctx context.Context, payload heartbeat.Payload) (Response, error) {
	if t == nil || t.client == nil || t.baseURL == nil {
		return Response{}, fmt.Errorf("heartbeat transport is not initialized")
	}
	if ctx == nil {
		return Response{}, fmt.Errorf("heartbeat context is required")
	}
	if err := heartbeat.Validate(payload); err != nil {
		return Response{}, fmt.Errorf("validate heartbeat payload: %w", err)
	}
	encoded, err := heartbeat.Encode(payload)
	if err != nil {
		return Response{}, fmt.Errorf("encode heartbeat payload: %w", err)
	}

	requestContext, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, t.heartbeatURL(), bytes.NewReader(encoded))
	if err != nil {
		return Response{}, fmt.Errorf("create heartbeat request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "opspilot-agent/"+payload.AgentVersion)
	request.Header.Set("X-OpsPilot-Agent-ID", payload.AgentID)
	request.Header.Set("X-OpsPilot-Schema-Version", payload.SchemaVersion)

	httpResponse, err := t.client.Do(request)
	if err != nil {
		return Response{}, fmt.Errorf("send heartbeat request: %w", err)
	}
	response := Response{
		StatusCode: httpResponse.StatusCode,
		RequestID:  strings.TrimSpace(httpResponse.Header.Get("X-Request-ID")),
	}
	if httpResponse.Body == nil {
		return response, fmt.Errorf("read heartbeat response body: body is nil")
	}

	if response.Accepted() {
		_, readErr := io.Copy(io.Discard, io.LimitReader(httpResponse.Body, MaxResponseBodyBytes))
		closeErr := httpResponse.Body.Close()
		if readErr != nil {
			return response, fmt.Errorf("read heartbeat response body: %w", readErr)
		}
		if closeErr != nil {
			return response, fmt.Errorf("close heartbeat response body: %w", closeErr)
		}
		return response, nil
	}

	message, truncated := readErrorMessage(httpResponse.Body)
	_ = httpResponse.Body.Close()
	return response, &HTTPStatusError{
		StatusCode: response.StatusCode,
		RequestID:  response.RequestID,
		Message:    message,
		Truncated:  truncated,
	}
}

func (t *Transport) heartbeatURL() string {
	endpoint := *t.baseURL
	endpoint.Path = pathpkg.Join("/", endpoint.Path, HeartbeatPath)
	endpoint.RawPath = ""
	endpoint.RawQuery = ""
	endpoint.ForceQuery = false
	endpoint.Fragment = ""
	return endpoint.String()
}
