package logging

import (
	"log/slog"
	"os"
	"slate-backend/pkg/config"
	"strings"
)

// New returns a *slog.Logger honoring the configured format and level.
// LOG_FORMAT is "json" (machine-parseable, default in production) or "text"
// (human-readable). LOG_LEVEL is debug/info/warn/error (default info).
func New(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}

	switch strings.ToLower(cfg.LogFormat) {
	case "text":
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	default:
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
