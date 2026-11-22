package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"runs a feature", []string{"hello", "Bo"}, 0, "Hello, Bo!\n", ""},
		{"unknown feature is a usage error", []string{"nope"}, 2, "", "usage: barcost <hello|invoice|margin|prices> [args...]\n"},
		{"no feature is a usage error", nil, 2, "", "usage: barcost <hello|invoice|margin|prices> [args...]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(tt.args, &out, &errOut)
			if code != tt.wantCode || out.String() != tt.wantStdout || errOut.String() != tt.wantStderr {
				t.Errorf("run(%q) = %d, %q, %q", tt.args, code, out.String(), errOut.String())
			}
		})
	}
}

func TestEveryFeatureIsRegistered(t *testing.T) {
	for _, name := range []string{"hello", "invoice", "margin", "prices"} {
		if Features[name] == nil {
			t.Errorf("feature %s is not registered", name)
		}
	}
}

func TestBadLogLevelFails(t *testing.T) {
	t.Setenv("BAR_COST_DEMO_LOG_LEVEL", "loud")
	var out, errOut bytes.Buffer
	if code := run([]string{"hello"}, &out, &errOut); code != 1 || !bytes.Contains(errOut.Bytes(), []byte("must be one of")) {
		t.Errorf("got %d, %q", code, errOut.String())
	}
}
