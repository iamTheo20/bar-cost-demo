package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// featureRow matches a FEATURES.md row: | feature | `command` | `expected` |
var featureRow = regexp.MustCompile("^\\|\\s*([^|`]+?)\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`([^`]*)`\\s*\\|$")

const runPrefix = "go run ./cmd/barcost "

// featureResult is one row of the evidence summary the finish gate reads.
type featureResult struct {
	Feature string  `json:"feature"`
	Command string  `json:"command"`
	Exit    int     `json:"exit"`
	OK      bool    `json:"ok"`
	Problem *string `json:"problem"`
}

// TestFeaturesMD runs every row of FEATURES.md and compares the output
// exactly. `make verify` runs it. Each run keeps its evidence in
// .harness/evidence/<timestamp>/: one <feature>.out per feature (stdout, then
// stderr) and a summary.json with {"timestamp", "ok", "features"}.
func TestFeaturesMD(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "FEATURES.md"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	stamp := fmt.Sprintf("%s-%06d", now.Format("20060102-150405"), now.Nanosecond()/1000)
	run := filepath.Join(root, ".harness", "evidence", stamp)
	if err := os.MkdirAll(run, 0o755); err != nil {
		t.Fatal(err)
	}
	var results []featureResult
	for _, line := range strings.Split(string(raw), "\n") {
		m := featureRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		name, command, want := m[1], m[2], m[3]
		t.Run(name, func(t *testing.T) {
			res := runFeature(t, root, run, name, command, want)
			results = append(results, res)
			if !res.OK {
				t.Error(*res.Problem)
			}
		})
	}
	if len(results) == 0 {
		t.Fatal("FEATURES.md lists no features")
	}
	writeSummary(t, run, stamp, results)
	rel, err := filepath.Rel(root, run)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("evidence: %s", rel)
}

func runFeature(t *testing.T, root, run, name, command, want string) featureResult {
	t.Helper()
	res := featureResult{Feature: name, Command: command}
	fail := func(format string, args ...any) featureResult {
		p := fmt.Sprintf(format, args...)
		res.Problem = &p
		return res
	}
	if !strings.HasPrefix(command, runPrefix) {
		return fail("command %q does not start with %q", command, runPrefix)
	}
	cmd := exec.Command(binary, strings.Fields(strings.TrimPrefix(command, runPrefix))...)
	cmd.Dir = root
	cmd.Env = cleanEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		res.Exit = exitErr.ExitCode()
	case err != nil:
		return fail("could not run: %v", err)
	}
	record := stdout.String()
	if stderr.Len() > 0 {
		record += "\n--- stderr ---\n" + stderr.String()
	}
	if err := os.WriteFile(filepath.Join(run, name+".out"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	if res.Exit != 0 {
		return fail("exited %d", res.Exit)
	}
	if got := strings.TrimSpace(stdout.String()); got != want {
		return fail("gave %q, expected %q", got, want)
	}
	res.OK = true
	return res
}

func writeSummary(t *testing.T, run, stamp string, results []featureResult) {
	t.Helper()
	ok := true
	for _, r := range results {
		ok = ok && r.OK
	}
	summary := struct {
		Timestamp string          `json:"timestamp"`
		OK        bool            `json:"ok"`
		Features  []featureResult `json:"features"`
	}{stamp, ok, results}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "summary.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
