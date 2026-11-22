//go:build ignore

// Command config_guard is the config ratchet (G9): it fails when this branch
// weakens the checks against a base ref.
//
//	go run .harness/config_guard.go --base origin/main [--root DIR]
//
// CI runs the base branch's copy of this file, so a PR cannot weaken the
// guard that judges it. Each finding names what got weaker:
//
//	CONFIG003  .harness/rules.json: a rule removed or changed, a higher length limit, a default ban allowed
//	CONFIG004  .harness/baseline.json: a count raised, or a key added
//	CONFIG005  Makefile: a `check` prerequisite dropped
//	CONFIG006  .claude/settings.json: a deny rule dropped
//	CONFIG007  internal/arch_test.go: an architecture test removed
//
// (CONFIG001/002 judged ruff.toml and pyproject.toml, retired with Python.)
//
// Stricter and unchanged config passes. A deliberate weakening is merged by
// Top Baldy over a red CI, never by an agent. Exit 0 pass, 1 weaker, 2 unreadable.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type guard struct{ root, base string }

// errUnreadable marks a config the guard cannot parse; that is never a pass.
var errUnreadable = errors.New("unreadable")

func (g guard) atBase(rel string) (string, bool) {
	out, err := exec.Command("git", "-C", g.root, "show", g.base+":"+rel).Output()
	return string(out), err == nil
}

func (g guard) now(rel string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(g.root, rel))
	return string(raw), err == nil
}

func decode(text string, present bool, rel string, v any) error {
	if !present {
		return nil
	}
	if err := json.Unmarshal([]byte(text), v); err != nil {
		return fmt.Errorf("%w: %s: %v", errUnreadable, rel, err)
	}
	return nil
}

type rulesFile struct {
	MaxFileLines     *int              `json:"max_file_lines"`
	MaxFunctionLines *int              `json:"max_function_lines"`
	Allow            []string          `json:"allow"`
	Banned           []json.RawMessage `json:"banned"`
	Forbidden        []json.RawMessage `json:"forbidden"`
}

func canonical(raw json.RawMessage) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

func limit(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func rulesFindings(old, cur rulesFile) []string {
	var out []string
	for kind, pair := range map[string][2][]json.RawMessage{"banned": {old.Banned, cur.Banned}, "forbidden": {old.Forbidden, cur.Forbidden}} {
		kept := map[string]bool{}
		for _, e := range pair[1] {
			kept[canonical(e)] = true
		}
		for _, e := range pair[0] {
			if !kept[canonical(e)] {
				out = append(out, fmt.Sprintf("%s rule removed or changed: %s", kind, canonical(e)))
			}
		}
	}
	if o, n := limit(old.MaxFileLines, 400), limit(cur.MaxFileLines, 400); n > o {
		out = append(out, fmt.Sprintf("max_file_lines %d -> %d", o, n))
	}
	if o, n := limit(old.MaxFunctionLines, 60), limit(cur.MaxFunctionLines, 60); n > o {
		out = append(out, fmt.Sprintf("max_function_lines %d -> %d", o, n))
	}
	had := map[string]bool{}
	for _, a := range old.Allow {
		had[a] = true
	}
	for _, a := range cur.Allow {
		if !had[a] {
			out = append(out, "default ban now allowed: "+a)
		}
	}
	return out
}

func baselineFindings(old, cur map[string]int) []string {
	var out []string
	for k, n := range cur {
		o, ok := old[k]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("baseline adds %s (%d)", k, n))
		case n > o:
			out = append(out, fmt.Sprintf("baseline %s %d -> %d", k, o, n))
		}
	}
	return out
}

var checkLine = regexp.MustCompile(`(?m)^check:([^#\n]*)`)

func checkPrereqs(text string) map[string]bool {
	out := map[string]bool{}
	if m := checkLine.FindStringSubmatch(text); m != nil {
		for _, t := range strings.Fields(m[1]) {
			out[t] = true
		}
	}
	return out
}

func denies(text string, present bool, rel string) (map[string]bool, error) {
	var s struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	out := map[string]bool{}
	if err := decode(text, present, rel, &s); err != nil {
		return nil, err
	}
	for _, d := range s.Permissions.Deny {
		out[d] = true
	}
	return out, nil
}

func testNames(text string, present bool, rel string) (map[string]bool, error) {
	out := map[string]bool{}
	if !present {
		return out, nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), rel, text, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", errUnreadable, rel, err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
			out[fn.Name.Name] = true
		}
	}
	return out, nil
}

func missing(old, cur map[string]bool) []string {
	var out []string
	for k := range old {
		if !cur[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func (g guard) jsonPair(rel string, old, cur any) error {
	bt, bok := g.atBase(rel)
	nt, nok := g.now(rel)
	if err := decode(bt, bok, rel, old); err != nil {
		return err
	}
	return decode(nt, nok, rel, cur)
}

func (g guard) findings() ([]string, error) {
	var out []string

	var oldRules, curRules rulesFile
	if err := g.jsonPair(".harness/rules.json", &oldRules, &curRules); err != nil {
		return nil, err
	}
	for _, f := range rulesFindings(oldRules, curRules) {
		out = append(out, "CONFIG003 .harness/rules.json: "+f)
	}

	oldBase, curBase := map[string]int{}, map[string]int{}
	if err := g.jsonPair(".harness/baseline.json", &oldBase, &curBase); err != nil {
		return nil, err
	}
	for _, f := range baselineFindings(oldBase, curBase) {
		out = append(out, "CONFIG004 .harness/baseline.json: "+f)
	}

	bt, _ := g.atBase("Makefile")
	nt, _ := g.now("Makefile")
	for _, t := range missing(checkPrereqs(bt), checkPrereqs(nt)) {
		out = append(out, "CONFIG005 Makefile: check no longer runs "+t)
	}

	rel := ".claude/settings.json"
	bt, bok := g.atBase(rel)
	nt, nok := g.now(rel)
	oldDeny, err := denies(bt, bok, rel)
	if err != nil {
		return nil, err
	}
	curDeny, err := denies(nt, nok, rel)
	if err != nil {
		return nil, err
	}
	for _, d := range missing(oldDeny, curDeny) {
		out = append(out, "CONFIG006 "+rel+": deny rule dropped: "+d)
	}

	rel = "internal/arch_test.go"
	bt, bok = g.atBase(rel)
	nt, nok = g.now(rel)
	oldTests, err := testNames(bt, bok, rel)
	if err != nil {
		return nil, err
	}
	curTests, err := testNames(nt, nok, rel)
	if err != nil {
		return nil, err
	}
	for _, t := range missing(oldTests, curTests) {
		out = append(out, "CONFIG007 "+rel+": architecture test removed: "+t)
	}

	sort.Strings(out)
	return out, nil
}

func main() {
	base := flag.String("base", "", "the ref to compare against, e.g. origin/main")
	root := flag.String("root", ".", "the project to judge")
	flag.Parse()
	if *base == "" {
		fmt.Fprintln(os.Stderr, "usage: go run .harness/config_guard.go --base <ref> [--root DIR]")
		os.Exit(2)
	}
	abs, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config guard:", err)
		os.Exit(2)
	}
	found, err := guard{root: abs, base: *base}.findings()
	if err != nil {
		fmt.Printf("config guard: cannot read %v; a config the guard cannot read is not a pass\n", err)
		os.Exit(2)
	}
	for _, f := range found {
		fmt.Println(f)
	}
	if len(found) > 0 {
		fmt.Printf("config guard: %d weakening(s) against %s. FIX: restore the check, or ask Top "+
			"Baldy, who alone merges a deliberate weakening over a red CI.\n", len(found), *base)
		os.Exit(1)
	}
	fmt.Printf("config guard: no check config is weaker than %s\n", *base)
}
