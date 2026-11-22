package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// featureRow matches a FEATURES.md row: | feature | `command` | `expected` |
var featureRow = regexp.MustCompile("^\\|\\s*([^|`]+?)\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`([^`]*)`\\s*\\|$")

const runPrefix = "go run ./cmd/barcost "

// TestFeaturesMD runs every row of FEATURES.md and compares the output
// exactly. `make verify` runs it.
func TestFeaturesMD(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "FEATURES.md"))
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, line := range strings.Split(string(raw), "\n") {
		m := featureRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		rows++
		name, command, want := m[1], m[2], m[3]
		t.Run(name, func(t *testing.T) {
			if !strings.HasPrefix(command, runPrefix) {
				t.Fatalf("command %q does not start with %q", command, runPrefix)
			}
			cmd := exec.Command(binary, strings.Fields(strings.TrimPrefix(command, runPrefix))...)
			cmd.Dir = root
			cmd.Env = cleanEnv()
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("%s: %v", command, err)
			}
			if got := strings.TrimSpace(string(out)); got != want {
				t.Errorf("gave %q, expected %q", got, want)
			}
		})
	}
	if rows == 0 {
		t.Fatal("FEATURES.md lists no features")
	}
}
