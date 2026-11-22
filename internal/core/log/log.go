// Package log is the one way to log: logger := log.Get("component").
//
// It configures log/slog once, from config.Settings, writing text records to
// stderr so stdout stays the feature's output.
package log

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/iamTheo20/bar-cost-demo/internal/core/config"
)

var (
	mu   sync.Mutex
	base *slog.Logger
)

// Level maps a config log level to its slog level.
func Level(name string) slog.Level {
	switch name {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

// Configure sets the level and destination every logger uses. main calls
// it once after loading settings; tests may point it elsewhere.
func Configure(s config.Settings, w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	base = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: Level(s.LogLevel)}))
}

// Get is a logger tagged with the component name. Before Configure it logs
// warnings and above to stderr.
func Get(name string) *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if base == nil {
		base = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	return base.With("logger", name)
}
