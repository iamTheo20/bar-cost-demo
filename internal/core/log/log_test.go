package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/iamTheo20/bar-cost-demo/internal/core/config"
)

func TestLevel(t *testing.T) {
	tests := map[string]slog.Level{"DEBUG": slog.LevelDebug, "INFO": slog.LevelInfo, "WARNING": slog.LevelWarn, "ERROR": slog.LevelError}
	for name, want := range tests {
		if got := Level(name); got != want {
			t.Errorf("Level(%s) = %v, want %v", name, got, want)
		}
	}
}

func TestConfigureFiltersByLevel(t *testing.T) {
	var buf bytes.Buffer
	Configure(config.Settings{LogLevel: "INFO"}, &buf)
	Get("test").Debug("hidden")
	Get("test").Info("shown", "n", 1)
	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, "msg=shown") || !strings.Contains(out, "logger=test") {
		t.Fatalf("unexpected log output: %q", out)
	}
}
