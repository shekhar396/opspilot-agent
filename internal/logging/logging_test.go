package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shekhar396/opspilot-agent/internal/config"
)

func TestJSONFormat(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(config.LoggingConfig{Level: "info", Format: "json"}, &output)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("test message", "component", "test")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("Unmarshal() error = %v; output = %q", err, output.String())
	}
	if entry["msg"] != "test message" {
		t.Errorf("msg = %v, want test message", entry["msg"])
	}
	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", entry["level"])
	}
	if entry["component"] != "test" {
		t.Errorf("component = %v, want test", entry["component"])
	}
	if _, ok := entry["time"]; !ok {
		t.Error("time field is missing")
	}
}

func TestTextFormat(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(config.LoggingConfig{Level: "info", Format: "text"}, &output)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("test message", "component", "test")
	for _, want := range []string{"level=INFO", `msg="test message"`, "component=test"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output %q does not contain %q", output.String(), want)
		}
	}
}

func TestLevelFiltering(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(config.LoggingConfig{Level: "warn", Format: "json"}, &output)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	got := output.String()
	if strings.Contains(got, "debug message") || strings.Contains(got, "info message") {
		t.Errorf("output contains filtered messages: %q", got)
	}
	if !strings.Contains(got, "warn message") || !strings.Contains(got, "error message") {
		t.Errorf("output is missing enabled messages: %q", got)
	}
}

func TestDebugLevel(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(config.LoggingConfig{Level: "debug", Format: "json"}, &output)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Debug("debug message")
	if !strings.Contains(output.String(), "debug message") {
		t.Fatalf("output = %q, want debug message", output.String())
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  config.LoggingConfig
		want string
	}{
		{name: "level", cfg: config.LoggingConfig{Level: "trace", Format: "json"}, want: `unsupported logging level "trace"`},
		{name: "format", cfg: config.LoggingConfig{Level: "info", Format: "xml"}, want: `unsupported logging format "xml"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.cfg, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("New() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestWritesToSuppliedWriter(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(config.LoggingConfig{Level: "info", Format: "json"}, &output)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Info("writer test")
	if output.Len() == 0 {
		t.Fatal("supplied writer received no output")
	}
}
