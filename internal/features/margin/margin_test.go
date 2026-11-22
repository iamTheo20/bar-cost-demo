package margin

import (
	"strings"
	"testing"
)

const data = "../../../data/"

func TestRun(t *testing.T) {
	t.Setenv("BAR_COST_DEMO_DATA_DIR", data)
	tests := []struct {
		name, file      string
		summary         bool
		want, mustNotBe string
	}{
		{"items pushed below target are named", "SF-1093.txt", true,
			"margin SF-1093: 1 of 2 affected menu items below target (Guacamole & chips 77.8% < 80%)", ""},
		{"only items using an invoiced product are shown", "NB-48211.txt", false, "Gin & Tonic", "Cheeseburger"},
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
			if tt.summary && got != tt.want {
				t.Errorf("got %q", got)
			}
			if !tt.summary && (!strings.Contains(got, tt.want) || strings.Contains(got, tt.mustNotBe)) {
				t.Errorf("got %q", got)
			}
		})
	}
}
