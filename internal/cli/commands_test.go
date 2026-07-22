package cli

import (
	"bytes"
	"testing"
)

func TestCommandOutput(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "run", want: "OpsPilot Agent runtime is not implemented yet\n"},
		{name: "validate-config", want: "Configuration validation is not implemented yet\n"},
		{name: "print-capabilities", want: "cli\nversion\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := executeCommand(t, test.name); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
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
