package prices

import (
	"strings"
	"testing"
)

const data = "../../../data/"

func TestRun(t *testing.T) {
	t.Setenv("BAR_COST_DEMO_DATA_DIR", data)
	tests := []struct {
		name, file  string
		summary     bool
		want        string
		wantFlagged int
	}{
		{"the summary names the largest move", "SF-1093.txt", true, "prices SF-1093: 2 of 4 items moved 5% or more (largest: Avocado +23.4%)", 0},
		{"moves of five percent or more are flagged", "NB-48211.txt", false, "Fever-Tree Tonic 20cl", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{data + "invoices/" + tt.file}
			if tt.summary {
				args = append(args, "--summary")
			}
			got, err := Run(args)
			if err != nil {
				t.Fatal(err)
			}
			if tt.summary {
				if got != tt.want {
					t.Errorf("got %q", got)
				}
				return
			}
			flagged := 0
			for _, line := range strings.Split(got, "\n") {
				if strings.HasSuffix(line, "FLAG") {
					flagged++
				}
			}
			if !strings.Contains(got, tt.want) || flagged != tt.wantFlagged {
				t.Errorf("flagged %d in %q", flagged, got)
			}
		})
	}
}
