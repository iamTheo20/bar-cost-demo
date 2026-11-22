// Package config is the one way to read settings: settings, err := config.Load().
//
// Nothing else in the module reads the environment; the architecture test
// enforces that.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Levels are the accepted values of BAR_COST_DEMO_LOG_LEVEL.
var Levels = []string{"DEBUG", "INFO", "WARNING", "ERROR"}

// Settings are validated settings, read from the environment once.
type Settings struct {
	LogLevel    string
	DataDir     string
	ClaudeModel string
}

// Lookup returns an environment variable and whether it is set.
type Lookup func(key string) (string, bool)

// Load reads settings from BAR_COST_DEMO_* environment variables.
func Load() (Settings, error) { return LoadFrom(os.LookupEnv) }

// LoadFrom reads settings through lookup, so tests need not touch the
// process environment.
func LoadFrom(lookup Lookup) (Settings, error) {
	get := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return v
		}
		return fallback
	}
	const name = "BAR_COST_DEMO_LOG_LEVEL"
	level := strings.ToUpper(get(name, "WARNING"))
	valid := false
	for _, l := range Levels {
		valid = valid || l == level
	}
	if !valid {
		return Settings{}, fmt.Errorf("%s must be one of %s, not %q", name, strings.Join(Levels, ", "), level)
	}
	return Settings{
		LogLevel:    level,
		DataDir:     get("BAR_COST_DEMO_DATA_DIR", "data"),
		ClaudeModel: get("BAR_COST_DEMO_CLAUDE_MODEL", ""),
	}, nil
}
