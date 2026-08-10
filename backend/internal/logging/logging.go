package logging

import (
	"log/slog"
	"os"
)

// New returns a *slog.Logger configured for the given environment. Production
// emits JSON for machine parsing; development emits human-readable text.
func New(environment string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if environment == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
