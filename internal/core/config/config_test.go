package config

import (
	"strings"
	"testing"
)

func env(vars map[string]string) Lookup {
	return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }
}

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		want    Settings
		wantErr string
	}{
		{"defaults", nil, Settings{LogLevel: "WARNING", DataDir: "data"}, ""},
		{"level is case-insensitive", map[string]string{"BAR_COST_DEMO_LOG_LEVEL": "debug"}, Settings{LogLevel: "DEBUG", DataDir: "data"}, ""},
		{"all set", map[string]string{"BAR_COST_DEMO_DATA_DIR": "/srv/d", "BAR_COST_DEMO_CLAUDE_MODEL": "sonnet"}, Settings{LogLevel: "WARNING", DataDir: "/srv/d", ClaudeModel: "sonnet"}, ""},
		{"bad level", map[string]string{"BAR_COST_DEMO_LOG_LEVEL": "loud"}, Settings{}, `must be one of DEBUG, INFO, WARNING, ERROR, not "LOUD"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadFrom(env(tt.vars))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}
