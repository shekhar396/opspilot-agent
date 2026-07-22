package identity

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const validID = "9fb42f1c-8a12-4db5-a42c-7a4be50efaf1"

func TestLoadOrCreateCreatesSecureIdentity(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "identity")
	path := filepath.Join(parent, "agent-id")

	identity, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}
	if err := validate(identity.ID()); err != nil {
		t.Fatalf("generated identity is invalid: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != identity.ID()+"\n" {
		t.Fatalf("file content = %q, want returned identity plus newline", content)
	}
	if runtime.GOOS != "windows" {
		assertMode(t, parent, 0o700)
		assertMode(t, path, 0o600)
	}
}

func TestLoadOrCreateReusesIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "agent-id")
	first, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("first LoadOrCreate() error = %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	second, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("second LoadOrCreate() error = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if first.ID() != second.ID() {
		t.Fatalf("identities differ: %q and %q", first.ID(), second.ID())
	}
	if string(before) != string(after) {
		t.Fatal("identity file content changed")
	}
}

func TestLoadOrCreateLoadsExistingIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-id")
	if err := os.WriteFile(path, []byte(" \t"+validID+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	identity, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}
	if identity.ID() != validID {
		t.Fatalf("ID() = %q, want %q", identity.ID(), validID)
	}
}

func TestLoadOrCreateRejectsMalformedIdentityWithoutReplacement(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "empty", content: ""},
		{name: "invalid length", content: "invalid-agent-id\n"},
		{name: "uppercase", content: "9FB42F1C-8A12-4DB5-A42C-7A4BE50EFAF1\n"},
		{name: "wrong version", content: "9fb42f1c-8a12-3db5-a42c-7a4be50efaf1\n"},
		{name: "wrong variant", content: "9fb42f1c-8a12-4db5-742c-7a4be50efaf1\n"},
		{name: "invalid hexadecimal", content: "9fb42f1c-8a12-4db5-a42c-7a4be50efag1\n"},
		{name: "incorrect separator", content: "9fb42f1c_8a12-4db5-a42c-7a4be50efaf1\n"},
		{name: "two identities", content: validID + "\n" + validID + "\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agent-id")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if _, err := LoadOrCreate(path); err == nil {
				t.Fatal("LoadOrCreate() error = nil")
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(content) != test.content {
				t.Fatalf("malformed content was replaced: got %q, want %q", content, test.content)
			}
		})
	}
}

func TestLoadOrCreatePathFailures(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(parentFile, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	identityDirectory := filepath.Join(t.TempDir(), "agent-id")
	if err := os.Mkdir(identityDirectory, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	for _, path := range []string{
		"",
		"relative/agent-id",
		filepath.Join(parentFile, "agent-id"),
		identityDirectory,
	} {
		if _, err := LoadOrCreate(path); err == nil {
			t.Errorf("LoadOrCreate(%q) error = nil", path)
		}
	}
}

func TestLoadOrCreateConcurrentCalls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "agent-id")
	const calls = 20
	results := make(chan Identity, calls)
	errors := make(chan error, calls)
	var group sync.WaitGroup
	for range calls {
		group.Add(1)
		go func() {
			defer group.Done()
			identity, err := LoadOrCreate(path)
			if err != nil {
				errors <- err
				return
			}
			results <- identity
		}()
	}
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent LoadOrCreate calls did not finish")
	}
	close(errors)
	for err := range errors {
		t.Errorf("LoadOrCreate() error = %v", err)
	}
	close(results)
	var expected string
	count := 0
	for identity := range results {
		count++
		if expected == "" {
			expected = identity.ID()
		}
		if identity.ID() != expected {
			t.Errorf("ID() = %q, want %q", identity.ID(), expected)
		}
	}
	if count != calls {
		t.Fatalf("successful calls = %d, want %d", count, calls)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(content)) != expected {
		t.Fatalf("final identity = %q, want %q", strings.TrimSpace(string(content)), expected)
	}
	if err := validate(expected); err != nil {
		t.Fatalf("final identity is invalid: %v", err)
	}
}

func TestGeneratedIdentityFormat(t *testing.T) {
	for range 10 {
		identity, err := generate()
		if err != nil {
			t.Fatalf("generate() error = %v", err)
		}
		if err := validate(identity.ID()); err != nil {
			t.Fatalf("generated identity %q is invalid: %v", identity.ID(), err)
		}
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("mode for %q = %04o, want %04o", path, got, want)
	}
}
