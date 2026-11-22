// Package cli is the arguments every invoice feature takes:
// <invoice file> [--summary] [--llm].
package cli

import (
	"errors"
	"slices"
	"strings"

	"github.com/iamTheo20/bar-cost-demo/internal/core/config"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
	"github.com/iamTheo20/bar-cost-demo/internal/core/llmextract"
)

// Flags are the switches every invoice feature accepts, anywhere in its arguments.
var Flags = []string{"--summary", "--llm"}

// ErrUsage is returned when the arguments do not name exactly one invoice.
var ErrUsage = errors.New("usage: <feature> <invoice file> [--summary] [--llm]")

// InvoiceArgs is which invoice to read and how, where the venue's data
// lives, and how much to say.
type InvoiceArgs struct {
	Invoice     string
	DataDir     string
	SummaryOnly bool
	UseLLM      bool
	ClaudeModel string
}

// Extractor is Claude for PDFs, photos or --llm, and the rule-based reader
// for the demo's text layout.
func (a InvoiceArgs) Extractor() invoices.Extractor {
	if a.UseLLM || strings.ToLower(llmextract.Suffix(a.Invoice)) != ".txt" {
		return llmextract.New(a.ClaudeModel)
	}
	return invoices.RuleBasedExtractor{}
}

// Load is the invoice, read by the right extractor.
func (a InvoiceArgs) Load() (invoices.Invoice, error) {
	return invoices.Load(a.Invoice, a.Extractor())
}

// ParseInvoiceArgs parses <invoice file> [--summary] [--llm] with the given settings.
func ParseInvoiceArgs(args []string, s config.Settings) (InvoiceArgs, error) {
	var rest []string
	for _, a := range args {
		if !slices.Contains(Flags, a) {
			rest = append(rest, a)
		}
	}
	if len(rest) != 1 {
		return InvoiceArgs{}, ErrUsage
	}
	return InvoiceArgs{
		Invoice:     rest[0],
		DataDir:     s.DataDir,
		SummaryOnly: slices.Contains(args, "--summary"),
		UseLLM:      slices.Contains(args, "--llm"),
		ClaudeModel: s.ClaudeModel,
	}, nil
}

// InvoiceArgsFromEnv parses the arguments with settings loaded from the environment.
func InvoiceArgsFromEnv(args []string) (InvoiceArgs, error) {
	s, err := config.Load()
	if err != nil {
		return InvoiceArgs{}, err
	}
	return ParseInvoiceArgs(args, s)
}
