package cli

import (
	"errors"
	"testing"

	"github.com/iamTheo20/bar-cost-demo/internal/core/config"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
	"github.com/iamTheo20/bar-cost-demo/internal/core/llmextract"
)

func TestParseInvoiceArgs(t *testing.T) {
	s := config.Settings{DataDir: "d", ClaudeModel: "m"}
	tests := []struct {
		name    string
		args    []string
		want    InvoiceArgs
		wantErr error
	}{
		{"file only", []string{"a.txt"}, InvoiceArgs{Invoice: "a.txt", DataDir: "d", ClaudeModel: "m"}, nil},
		{"flags anywhere", []string{"--llm", "a.txt", "--summary"}, InvoiceArgs{Invoice: "a.txt", DataDir: "d", SummaryOnly: true, UseLLM: true, ClaudeModel: "m"}, nil},
		{"no file", []string{"--summary"}, InvoiceArgs{}, ErrUsage},
		{"two files", []string{"a", "b"}, InvoiceArgs{}, ErrUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInvoiceArgs(tt.args, s)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf("got %+v, %v", got, err)
			}
		})
	}
}

func TestExtractorChoice(t *testing.T) {
	tests := []struct {
		args InvoiceArgs
		llm  bool
	}{
		{InvoiceArgs{Invoice: "a.txt"}, false},
		{InvoiceArgs{Invoice: "a.TXT"}, false},
		{InvoiceArgs{Invoice: "a.txt", UseLLM: true}, true},
		{InvoiceArgs{Invoice: "a.pdf"}, true},
		{InvoiceArgs{Invoice: "photo.jpg"}, true},
	}
	for _, tt := range tests {
		_, isLLM := tt.args.Extractor().(llmextract.ClaudeCLIExtractor)
		_, isRule := tt.args.Extractor().(invoices.RuleBasedExtractor)
		if isLLM != tt.llm || isRule == tt.llm {
			t.Errorf("%+v: llm=%v", tt.args, isLLM)
		}
	}
}
