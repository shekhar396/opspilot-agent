package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "opspilot-agent" {
		t.Fatalf("Use = %q, want %q", cmd.Use, "opspilot-agent")
	}

	wantCommands := []string{"run", "version", "validate-config", "print-capabilities"}
	commands := cmd.Commands()
	if len(commands) != len(wantCommands) {
		t.Fatalf("command count = %d, want %d", len(commands), len(wantCommands))
	}
	for i, want := range wantCommands {
		if commands[i].Name() != want {
			t.Errorf("command %d = %q, want %q", i, commands[i].Name(), want)
		}
	}
}

func TestRootCommandDisplaysOrderedHelp(t *testing.T) {
	cmd := NewRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := output.String()
	previous := -1
	for _, name := range []string{"run", "version", "validate-config", "print-capabilities"} {
		position := strings.Index(help, "  "+name)
		if position < 0 {
			t.Errorf("help does not contain command %q", name)
		}
		if position <= previous {
			t.Errorf("command %q is out of order in help", name)
		}
		previous = position
	}
}

func TestRootCommandDisablesCompletionCommand(t *testing.T) {
	cmd := NewRootCommand()
	for _, subcommand := range cmd.Commands() {
		if subcommand.Name() == "completion" {
			t.Fatal("completion command is present")
		}
	}
}
