package invoice

import (
	"strings"
	"testing"
)

const invoiceDir = "../../../data/invoices/"

func TestRun(t *testing.T) {
	tests := []struct {
		name, file string
		summary    bool
		want       string
	}{
		{"summary counts lines, total and issues", "SF-1093.txt", true, "invoice SF-1093: 4 lines, 2308.00 SEK, 1 arithmetic issue(s)"},
		{"a clean invoice", "NB-48211.txt", true, "invoice NB-48211: 4 lines, 5823.20 SEK, 0 arithmetic issue(s)"},
		{"the report shows the issue", "SF-1093.txt", false, "  ISSUE Cheddar: 2 x 145.00 = 290.00, invoiced 300.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{invoiceDir + tt.file}
			if tt.summary {
				args = append(args, "--summary")
			}
			got, err := Run(args)
			if err != nil {
				t.Fatal(err)
			}
			if (tt.summary && got != tt.want) || (!tt.summary && !strings.Contains(got, tt.want)) {
				t.Errorf("got %q", got)
			}
		})
	}
}

func TestAMissingFileIsAnError(t *testing.T) {
	if _, err := Run([]string{invoiceDir + "nope.txt"}); err == nil {
		t.Error("no error")
	}
}
