package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// goldenCase is one recorded run of the Python CLI; see testdata/golden/capture.py.
type goldenCase struct {
	Name           string   `json:"name"`
	Args           []string `json:"args"`
	ClaudeReply    string   `json:"claude_reply"`
	Exit           int      `json:"exit"`
	StderrLastLine string   `json:"stderr_last_line"`
}

var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "barcost-golden-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "barcost")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		panic("go build: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// expectedStderr is what the Go CLI should say on stderr for a Python
// error line: the message without Python's exception class, and the
// program's new name in usage lines.
func expectedStderr(pythonLine string) string {
	if _, msg, ok := strings.Cut(pythonLine, "Error: "); ok {
		return msg
	}
	return strings.ReplaceAll(pythonLine, "bar-cost-demo", Prog)
}

func cleanEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "BAR_COST_DEMO_") && !strings.HasPrefix(kv, "BARCOST_FAKE_") {
			env = append(env, kv)
		}
	}
	return env
}

// TestGoldenParity runs the Go CLI on every input the Python CLI was
// recorded on and requires the same stdout, exit code and error message.
// LLM cases replay the recorded real Claude reply through testdata/fakeclaude,
// and also require the same claude command line and folder contents.
func TestGoldenParity(t *testing.T) {
	root := repoRoot(t)
	golden := filepath.Join(root, "testdata", "golden")
	raw, err := os.ReadFile(filepath.Join(golden, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []goldenCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no golden cases")
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.Args...)
			cmd.Dir = root
			cmd.Env = cleanEnv()
			argsFile := filepath.Join(t.TempDir(), "claude-args")
			if tc.ClaudeReply != "" {
				cmd.Env = append(cmd.Env,
					"PATH="+filepath.Join(root, "testdata", "fakeclaude")+string(os.PathListSeparator)+os.Getenv("PATH"),
					"BARCOST_FAKE_CLAUDE_REPLY="+filepath.Join(golden, tc.ClaudeReply),
					"BARCOST_FAKE_CLAUDE_ARGS="+argsFile,
				)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			exit := 0
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exit = ee.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join(golden, "out", tc.Name+".stdout"))
			if err != nil {
				t.Fatal(err)
			}
			if got := stdout.String(); got != string(want) {
				t.Errorf("stdout differs\n--- got\n%s--- want\n%s", got, want)
			}
			if exit != tc.Exit {
				t.Errorf("exit %d, want %d (stderr %q)", exit, tc.Exit, stderr.String())
			}
			if tc.StderrLastLine != "" && !strings.Contains(stderr.String(), expectedStderr(tc.StderrLastLine)) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), expectedStderr(tc.StderrLastLine))
			}
			if tc.ClaudeReply != "" {
				compareFile(t, argsFile, filepath.Join(golden, "out", tc.Name+".claude-args"))
				compareFile(t, argsFile+".cwd", filepath.Join(golden, "out", tc.Name+".claude-args.cwd"))
			}
		})
	}
}

func compareFile(t *testing.T, gotPath, wantPath string) {
	t.Helper()
	want, wantErr := os.ReadFile(wantPath)
	got, gotErr := os.ReadFile(gotPath)
	if os.IsNotExist(wantErr) && os.IsNotExist(gotErr) {
		return
	}
	if wantErr != nil || gotErr != nil {
		t.Fatalf("reading %s / %s: %v / %v", gotPath, wantPath, gotErr, wantErr)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from %s\n--- got\n%s--- want\n%s", gotPath, filepath.Base(wantPath), got, want)
	}
}
