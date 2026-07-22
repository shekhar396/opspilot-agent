package logging

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/shekhar396/opspilot-agent/internal/config"
)

func New(cfg config.LoggingConfig, output io.Writer) (*slog.Logger, error) {
	if output == nil {
		return nil, fmt.Errorf("logging output writer is required")
	}

	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("unsupported logging level %q", cfg.Level)
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(output, options)
	case "text":
		handler = slog.NewTextHandler(output, options)
	default:
		return nil, fmt.Errorf("unsupported logging format %q", cfg.Format)
	}

	return slog.New(handler), nil
}
