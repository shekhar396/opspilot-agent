package cli

import (
	"bytes"
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
		{name: "run", want: "OpsPilot Agent runtime is not implemented yet\n"},
		{name: "print-capabilities", want: "cli\nversion\nconfig-validation\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := executeCommand(t, test.name); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
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

const validCLIConfig = `agent:
  name: app-server-01
  server_url: https://opspilot.example.com
`
