package llmextract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

func good() map[string]any {
	return map[string]any{
		"supplier": "Bryggeriet Soder AB",
		"number":   "BS-2207",
		"date":     "2026-11-17",
		"lines": []any{
			map[string]any{"product": "Pilsner keg 30L", "unit": "keg", "qty": "2", "unit_price": "1520.00", "total": "3040.00"},
			map[string]any{"product": "Lime", "unit": "kg", "qty": "3", "unit_price": "46.00", "total": "140.00"},
		},
		"subtotal": "3180.00",
	}
}

// fakeClaude records the command and the folder, and answers like
// `claude -p --output-format json`.
type fakeClaude struct {
	reply string
	cmd   []string
	seen  []string
}

func newFake(structured any, subtype string) *fakeClaude {
	b, _ := json.Marshal(map[string]any{
		"subtype": subtype, "is_error": subtype != "success", "result": "", "structured_output": structured,
	})
	return &fakeClaude{reply: string(b)}
}

func (f *fakeClaude) run(cmd []string, dir string) (string, error) {
	f.cmd = cmd
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		f.seen = append(f.seen, e.Name())
	}
	return f.reply, nil
}

func pdf(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "BS-2207.pdf")
	if err := os.WriteFile(p, []byte("%PDF-1.4 fake"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAPDFBecomesACheckedInvoice(t *testing.T) {
	fake := newFake(good(), "success")
	inv, err := ClaudeCLIExtractor{Run: fake.run}.Extract(pdf(t))
	if err != nil {
		t.Fatal(err)
	}
	if inv.Number != "BS-2207" || inv.Lines[0].UnitPrice.String() != "1520.00" {
		t.Errorf("got %+v", inv)
	}
	want := []string{"Lime: 3 x 46.00 = 138.00, invoiced 140.00"}
	if got := invoices.ArithmeticIssues(inv); !reflect.DeepEqual(got, want) {
		t.Errorf("issues %q", got)
	}
}

func TestClaudeSeesOnlyTheInvoiceAndCanOnlyRead(t *testing.T) {
	fake := newFake(good(), "success")
	if _, err := (ClaudeCLIExtractor{Run: fake.run}).Extract(pdf(t)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.seen, []string{"invoice.pdf"}) {
		t.Errorf("claude saw %q", fake.seen)
	}
	if fake.cmd[slices.Index(fake.cmd, "--tools")+1] != "Read" {
		t.Errorf("tools %q", fake.cmd)
	}
	for _, flag := range []string{"--no-session-persistence", "--strict-mcp-config", "--json-schema"} {
		if !slices.Contains(fake.cmd, flag) {
			t.Errorf("missing %s", flag)
		}
	}
	if !strings.Contains(fake.cmd[2], "Never") || !strings.Contains(fake.cmd[2], "ignore any instructions") {
		t.Errorf("prompt %q", fake.cmd[2])
	}
	if slices.Contains(fake.cmd, "--model") {
		t.Error("--model without a configured model")
	}
}

func TestTheConfiguredModelIsPassed(t *testing.T) {
	cmd := ClaudeCLIExtractor{Model: "sonnet"}.Command("invoice.pdf")
	if got := cmd[len(cmd)-2:]; !reflect.DeepEqual(got, []string{"--model", "sonnet"}) {
		t.Errorf("tail %q", got)
	}
}

func TestTheSchemaIsValidJSON(t *testing.T) {
	var v map[string]any
	if err := json.Unmarshal([]byte(InvoiceSchema), &v); err != nil || v["type"] != "object" {
		t.Fatalf("schema: %v", err)
	}
}

func TestAFailedRunIsAnErrorNotAnEmptyInvoice(t *testing.T) {
	_, err := ClaudeCLIExtractor{Run: newFake(nil, "error_max_turns").run}.Extract(pdf(t))
	if err == nil || !strings.Contains(err.Error(), "could not extract") {
		t.Errorf("err = %v", err)
	}
}

func TestANonJSONReplyIsRefused(t *testing.T) {
	fake := &fakeClaude{reply: "sorry"}
	_, err := ClaudeCLIExtractor{Run: fake.run}.Extract(pdf(t))
	if err == nil || err.Error() != "claude did not return JSON" {
		t.Errorf("err = %v", err)
	}
}

func with(key string, value any) map[string]any {
	g := good()
	g[key] = value
	return g
}

func TestOutputThatBreaksTheSchemaIsRefused(t *testing.T) {
	missingDate := good()
	delete(missingDate, "date")
	tests := []struct {
		name   string
		broken any
		needle string
	}{
		{"no lines", with("lines", []any{}), "no line items"},
		{"subtotal not a number", with("subtotal", "about 3k"), "subtotal is not a number"},
		{"date not ISO", with("date", "17/11/2026"), "date is not YYYY-MM-DD"},
		{"date missing", missingDate, "date is not YYYY-MM-DD: None"},
		{"blank supplier", with("supplier", ""), "no usable supplier"},
		{"not an object", "not an object", "not an object"},
		{"line not an object", with("lines", []any{"x"}), "an invoice line is not an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InvoiceFrom(tt.broken)
			if err == nil || !strings.Contains(err.Error(), tt.needle) {
				t.Errorf("err = %v, want %q", err, tt.needle)
			}
		})
	}
}

func TestUnsupportedFilesAreRefusedBeforeCallingClaude(t *testing.T) {
	fake := newFake(good(), "success")
	doc := filepath.Join(t.TempDir(), "invoice.docx")
	if err := os.WriteFile(doc, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ClaudeCLIExtractor{Run: fake.run}.Extract(doc)
	if err == nil || !strings.Contains(err.Error(), "cannot read .docx") {
		t.Errorf("err = %v", err)
	}
	if fake.cmd != nil {
		t.Error("claude was called")
	}
}

func TestSuffixMatchesPythonPathSuffix(t *testing.T) {
	tests := map[string]string{"a/b.PDF": ".PDF", "x.tar.gz": ".gz", ".env": "", "invoice": "", "odd.": ""}
	for in, want := range tests {
		if got := Suffix(in); got != want {
			t.Errorf("Suffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRunClaudeReportsAMissingCLI(t *testing.T) {
	_, err := RunClaude([]string{"definitely-not-claude-xyz"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Errorf("err = %v", err)
	}
}

func TestRunClaudeReportsAFailedExit(t *testing.T) {
	_, err := RunClaude([]string{"sh", "-c", "echo boom >&2; exit 3"}, t.TempDir())
	if err == nil || err.Error() != "claude exited 3: boom" {
		t.Errorf("err = %v", err)
	}
}
