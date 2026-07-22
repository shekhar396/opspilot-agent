package cli

import (
	"bytes"
	"testing"

	buildversion "github.com/shekhar396/opspilot-agent/internal/version"
)

func TestVersionCommandDefaultOutput(t *testing.T) {
	got := executeCommand(t, "version")
	want := "opspilot-agent version dev\ncommit: unknown\nbuilt: unknown\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestVersionCommandInjectedOutput(t *testing.T) {
	originalVersion := buildversion.Version
	originalCommit := buildversion.Commit
	originalDate := buildversion.Date
	t.Cleanup(func() {
		buildversion.Version = originalVersion
		buildversion.Commit = originalCommit
		buildversion.Date = originalDate
	})

	buildversion.Version = "v0.1.0"
	buildversion.Commit = "abc1234"
	buildversion.Date = "2026-07-22T12:00:00Z"

	got := executeCommand(t, "version")
	want := "opspilot-agent version v0.1.0\ncommit: abc1234\nbuilt: 2026-07-22T12:00:00Z\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestVersionCommandDoesNotModifyMetadata(t *testing.T) {
	wantVersion := buildversion.Version
	wantCommit := buildversion.Commit
	wantDate := buildversion.Date

	executeCommand(t, "version")

	if buildversion.Version != wantVersion || buildversion.Commit != wantCommit || buildversion.Date != wantDate {
		t.Fatalf(
			"metadata changed to version=%q commit=%q date=%q",
			buildversion.Version,
			buildversion.Commit,
			buildversion.Date,
		)
	}
}

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	cmd := NewRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	return output.String()
}
