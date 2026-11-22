package hello

import "testing"

func TestRun(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"Ada"}, "Hello, Ada!"},
		{nil, "Hello, World!"},
	}
	for _, tt := range tests {
		if got, err := Run(tt.args); err != nil || got != tt.want {
			t.Errorf("Run(%q) = %q, %v", tt.args, got, err)
		}
	}
}
