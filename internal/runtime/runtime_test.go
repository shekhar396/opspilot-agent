package runtime

import (
	"context"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/shekhar396/opspilot-agent/internal/config"
)

func TestNew(t *testing.T) {
	runtime, err := New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("New() returned a nil runtime")
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
	}{
		{
			name: "empty agent name",
			modify: func(cfg *config.Config) {
				cfg.Agent.Name = ""
			},
		},
		{
			name: "empty server URL",
			modify: func(cfg *config.Config) {
				cfg.Agent.ServerURL = ""
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.modify(&cfg)
			if _, err := New(cfg); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestRunWithCancelledContextReturnsImmediately(t *testing.T) {
	runtime, err := New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunExitsAfterContextCancellation(t *testing.T) {
	runtime, err := New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runtime.Run(ctx)
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestRunDoesNotLeakGoroutines(t *testing.T) {
	runtime, err := New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	before := goruntime.NumGoroutine()
	for range 100 {
		if err := runtime.Run(ctx); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	after := goruntime.NumGoroutine()
	if after > before {
		t.Fatalf("goroutine count increased from %d to %d", before, after)
	}
}

func validConfig() config.Config {
	cfg := config.Default()
	cfg.Agent.Name = "app-server-01"
	cfg.Agent.ServerURL = "https://opspilot.example.com"
	return cfg
}
