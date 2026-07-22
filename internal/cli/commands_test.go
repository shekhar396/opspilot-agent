package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandOutput(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "print-capabilities", want: "cli\nversion\nconfig-validation\nstructured-logging\nruntime\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := executeCommand(t, test.name); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRunCommandLoadsConfigurationAndExitsOnCancellation(t *testing.T) {
	path := writeCLIConfig(t, validCLIConfig)
	var output bytes.Buffer
	executeCancelledRun(t, &output, path)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("log line count = %d, want 2; output = %q", len(lines), output.String())
	}
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("JSON log is invalid: %v; line = %q", err, line)
		}
	}
	if !strings.Contains(lines[0], "agent runtime started") || !strings.Contains(lines[1], "agent runtime stopped") {
		t.Fatalf("output does not contain lifecycle logs in order: %q", output.String())
	}
}

func TestRunCommandTextLogging(t *testing.T) {
	path := writeCLIConfig(t, strings.Replace(validCLIConfig, "format: json", "format: text", 1))
	var output bytes.Buffer
	executeCancelledRun(t, &output, path)

	for _, want := range []string{
		"level=INFO",
		`msg="agent runtime started"`,
		"agent_name=app-server-01",
		"server_url=https://opspilot.example.com",
		`msg="agent runtime stopped"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output %q does not contain %q", output.String(), want)
		}
	}
}

func TestRunCommandErrorLevelSuppressesLifecycleLogs(t *testing.T) {
	path := writeCLIConfig(t, strings.Replace(validCLIConfig, "level: info", "level: error", 1))
	var output bytes.Buffer
	executeCancelledRun(t, &output, path)
	if output.Len() != 0 {
		t.Fatalf("output = %q, want empty", output.String())
	}
}

func TestRunCommandConfigurationErrors(t *testing.T) {
	tests := []struct {
		name string
		path func(*testing.T) string
	}{
		{
			name: "missing configuration",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.yaml") },
		},
		{
			name: "invalid configuration",
			path: func(t *testing.T) string {
				return writeCLIConfig(t, strings.Replace(validCLIConfig, "https://", "http://", 1))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := newRootCommand(&output)
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"run", "--config", test.path(t)})
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil")
			}
			if output.Len() != 0 {
				t.Fatalf("output = %q, want empty", output.String())
			}
		})
	}
}

func TestRunConfigDefaultPath(t *testing.T) {
	cmd := newRunCommand(io.Discard)
	flag := cmd.Flags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag is missing")
	}
	if flag.DefValue != "configs/opspilot-agent.yaml" {
		t.Fatalf("config default = %q, want %q", flag.DefValue, "configs/opspilot-agent.yaml")
	}
	if flag.Shorthand != "c" {
		t.Fatalf("config shorthand = %q, want c", flag.Shorthand)
	}
}

func TestValidateConfigCommand(t *testing.T) {
	path := writeCLIConfig(t, validCLIConfig)
	if got := executeCommand(t, "validate-config", "--config", path); got != "Configuration is valid\n" {
		t.Fatalf("output = %q, want %q", got, "Configuration is valid\n")
	}
}

func TestValidateConfigCommandShortFlag(t *testing.T) {
	path := writeCLIConfig(t, validCLIConfig)
	if got := executeCommand(t, "validate-config", "-c", path); got != "Configuration is valid\n" {
		t.Fatalf("output = %q, want %q", got, "Configuration is valid\n")
	}
}

func TestValidateConfigCommandErrors(t *testing.T) {
	tests := []struct {
		name string
		path func(*testing.T) string
	}{
		{
			name: "missing file",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.yaml") },
		},
		{
			name: "invalid configuration",
			path: func(t *testing.T) string {
				return writeCLIConfig(t, strings.Replace(validCLIConfig, "https://", "http://", 1))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := NewRootCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"validate-config", "--config", test.path(t)})
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil")
			}
			if output.Len() != 0 {
				t.Fatalf("output = %q, want empty", output.String())
			}
		})
	}
}

func TestValidateConfigDefaultPath(t *testing.T) {
	cmd := newValidateConfigCommand()
	flag := cmd.Flags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag is missing")
	}
	if flag.DefValue != "configs/opspilot-agent.yaml" {
		t.Fatalf("config default = %q, want %q", flag.DefValue, "configs/opspilot-agent.yaml")
	}
	if flag.Shorthand != "c" {
		t.Fatalf("config shorthand = %q, want c", flag.Shorthand)
	}
}

func TestCommandsRejectPositionalArguments(t *testing.T) {
	for _, name := range []string{"run", "version", "validate-config", "print-capabilities"} {
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{name, "extra"})
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, want positional argument error")
			}
		})
	}
}

func writeCLIConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func executeCancelledRun(t *testing.T, output *bytes.Buffer, path string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := newRootCommand(output)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"run", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

const validCLIConfig = `agent:
  name: app-server-01
  server_url: https://opspilot.example.com
logging:
  level: info
  format: json
`
