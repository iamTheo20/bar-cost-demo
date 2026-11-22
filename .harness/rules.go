//go:build ignore

// Command rules is the project's rule engine (G5): checks whose findings name
// the fix. It is a single stdlib-only file, run with go run:
//
//	go run .harness/rules.go                     check the whole project
//	go run .harness/rules.go -- PATH ...         check these files (the -- stops go run reading them as source)
//	go run .harness/rules.go --init-baseline     record today's violations (once)
//	go run .harness/rules.go --update-baseline   shrink the baseline to what is left
//
// Every finding is printed as
//
//	<RULE> <path>:<line> <what is wrong>
//	  FIX: <the replacement>
//	  WHY: <one line>
//	  SEE: <a correct example or RULES.md>
//
//	HR001  suppression comments (//nolint, //lint:ignore, //lint:file-ignore)
//	HR002  swallowed errors: an empty `if err != nil {}`, or `_ = call()`
//	HR003  banned calls: print, println, the std log package, plus "banned" in rules.json
//	HR004  imports across a "forbidden" boundary in rules.json
//	HR006  files over max_file_lines, functions over max_function_lines
//	HR007  Test functions that never report through their *testing.T
//
// The baseline (.harness/baseline.json) maps "RULE path" to the number of old
// violations that are tolerated. A key at or under its count is quiet; one
// over it is reported in full. --update-baseline can only lower counts and
// drop keys, never raise or add them.
//
// Exit 0 clean, 1 findings, 2 a usage or config error.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const defaultSee = ".harness/RULES.md"

type ban struct {
	Call   string   `json:"call"`
	Fix    string   `json:"fix"`
	Why    string   `json:"why"`
	See    string   `json:"see"`
	Exempt []string `json:"exempt"`
}

type boundary struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Fix    string `json:"fix"`
	See    string `json:"see"`
}

type config struct {
	MaxFileLines     int        `json:"max_file_lines"`
	MaxFunctionLines int        `json:"max_function_lines"`
	Allow            []string   `json:"allow"`
	Banned           []ban      `json:"banned"`
	Forbidden        []boundary `json:"forbidden"`
}

type finding struct {
	rule, path          string
	line                int
	what, fix, why, see string
}

const logFix = "logger := log.Get(name) from internal/core/log"

// defaultBans apply everywhere unless rules.json lists them under "allow".
// A call is "importpath.Name", or a bare builtin name.
var defaultBans = []ban{
	{Call: "print", Fix: "return the text from the feature; cmd/barcost prints it", Why: "output goes through one place"},
	{Call: "println", Fix: "return the text from the feature; cmd/barcost prints it", Why: "output goes through one place"},
	{Call: "log.Print", Fix: logFix, Why: "logging is configured once, in core/log", See: "internal/core/log/log.go"},
	{Call: "log.Printf", Fix: logFix, Why: "logging is configured once, in core/log", See: "internal/core/log/log.go"},
	{Call: "log.Println", Fix: logFix, Why: "logging is configured once, in core/log", See: "internal/core/log/log.go"},
	{Call: "log.Fatal", Fix: "return the error; cmd/barcost turns it into exit 1", Why: "only main decides how the process exits"},
	{Call: "log.Fatalf", Fix: "return the error; cmd/barcost turns it into exit 1", Why: "only main decides how the process exits"},
	{Call: "log.Panic", Fix: "return the error", Why: "errors are values here"},
	{Call: "log.Panicf", Fix: "return the error", Why: "errors are values here"},
}

var (
	suppression = regexp.MustCompile(`^//\s*(nolint\b|lint:(file-)?ignore\b)`)
	skipDirs    = map[string]bool{".git": true, ".harness": true, ".refactor": true, "testdata": true, "vendor": true, "node_modules": true}
	reporters   = map[string]bool{"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Fail": true, "FailNow": true, "Run": true, "Skip": true, "Skipf": true, "SkipNow": true}
)

func main() { os.Exit(run(os.Args[1:])) }

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "rules:", err)
	return 2
}

func run(args []string) int {
	mode := ""
	var paths []string
	for _, a := range args {
		switch {
		case a == "--":
			continue
		case a == "--init-baseline" || a == "--update-baseline":
			mode = a
		case strings.HasPrefix(a, "-"):
			fmt.Fprintln(os.Stderr, "usage: go run .harness/rules.go [--init-baseline|--update-baseline] [-- PATH ...]")
			return 2
		default:
			paths = append(paths, a)
		}
	}
	root, err := projectRoot()
	if err != nil {
		return fail(err)
	}
	cfg, err := loadConfig(filepath.Join(root, ".harness", "rules.json"))
	if err != nil {
		return fail(err)
	}
	module, err := modulePath(root)
	if err != nil {
		return fail(err)
	}
	files, err := collect(root, paths)
	if err != nil {
		return fail(err)
	}
	var found []finding
	for _, f := range files {
		got, err := checkFile(root, f, module, cfg)
		if err != nil {
			return fail(err)
		}
		found = append(found, got...)
	}
	return report(root, mode, found)
}

func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".harness", "rules.json")); err == nil {
			return dir, nil
		}
		if filepath.Dir(dir) == dir {
			return "", errors.New("no .harness/rules.json above the current directory")
		}
		dir = filepath.Dir(dir)
	}
}

func loadConfig(p string) (config, error) {
	cfg := config{MaxFileLines: 400, MaxFunctionLines: 60}
	raw, err := os.ReadFile(p)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", p, err)
	}
	return cfg, nil
}

func modulePath(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", errors.New("go.mod has no module line")
}

func collect(root string, paths []string) ([]string, error) {
	var out []string
	if len(paths) > 0 {
		for _, p := range paths {
			abs, err := filepath.Abs(p)
			if err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				return nil, err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return out, nil
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel))
		return err
	})
	sort.Strings(out)
	return out, err
}

type fileCheck struct {
	rel    string
	fset   *token.FileSet
	file   *ast.File
	cfg    config
	module string
	out    []finding
}

func (c *fileCheck) add(rule string, pos token.Pos, what, fix, why, see string) {
	if see == "" {
		see = defaultSee
	}
	c.out = append(c.out, finding{rule, c.rel, c.fset.Position(pos).Line, what, fix, why, see})
}

func checkFile(root, rel, module string, cfg config) ([]finding, error) {
	src, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	c := &fileCheck{rel: rel, fset: fset, file: file, cfg: cfg, module: module}
	c.suppressions()
	imports, err := c.imports()
	if err != nil {
		return nil, err
	}
	c.calls(imports)
	c.lengths(strings.Count(string(src), "\n"))
	c.tests()
	return c.out, nil
}

func (c *fileCheck) suppressions() {
	for _, cg := range c.file.Comments {
		for _, cm := range cg.List {
			if suppression.MatchString(cm.Text) {
				c.add("HR001", cm.Pos(), "suppression comment "+strings.TrimSpace(cm.Text),
					"fix what the check reports", "a silenced check protects nothing and gets copied", "")
			}
		}
	}
}

// imports checks HR004 and returns local name -> import path.
func (c *fileCheck) imports() (map[string]string, error) {
	names := map[string]string{}
	pkg := c.module + "/" + path.Dir(c.rel)
	for _, imp := range c.file.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		name := path.Base(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = p
		for _, b := range c.cfg.Forbidden {
			if within(pkg, c.module+"/"+b.Source) && within(p, c.module+"/"+b.Target) {
				c.add("HR004", imp.Pos(), fmt.Sprintf("%s imports %s across %s -> %s", path.Dir(c.rel), p, b.Source, b.Target),
					b.Fix, "layers stay independently changeable", b.See)
			}
		}
	}
	return names, nil
}

// calls checks HR002 and HR003.
func (c *fileCheck) calls(imports map[string]string) {
	bans := activeBans(c.cfg, c.rel)
	ast.Inspect(c.file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok {
				if p, ok := imports[x.Name]; ok {
					if b, ok := bans[p+"."+n.Sel.Name]; ok {
						c.add("HR003", n.Pos(), "uses "+p+"."+n.Sel.Name, b.Fix, b.Why, b.See)
					}
				}
			}
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && id.Obj == nil {
				if b, ok := bans[id.Name]; ok {
					c.add("HR003", n.Pos(), "calls builtin "+id.Name, b.Fix, b.Why, b.See)
				}
			}
		case *ast.IfStmt:
			if isErrCheck(n.Cond) && len(n.Body.List) == 0 && n.Else == nil {
				c.add("HR002", n.Pos(), "empty `if err != nil {}` swallows the error",
					"return the error, or handle it", "swallowed errors become wrong output", "")
			}
		case *ast.AssignStmt:
			if _, ok := n.Rhs[0].(*ast.CallExpr); ok && len(n.Rhs) == 1 && allBlank(n.Lhs) {
				c.add("HR002", n.Pos(), "`_ = call()` discards the call's result",
					"check the error the call returns", "swallowed errors become wrong output", "")
			}
		}
		return true
	})
}

func (c *fileCheck) lengths(lines int) {
	if lines > c.cfg.MaxFileLines {
		c.add("HR006", c.file.Package, fmt.Sprintf("file has %d lines (max %d)", lines, c.cfg.MaxFileLines),
			"split by feature or concern", "long files attract more of the same", "")
	}
	for _, d := range c.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		n := c.fset.Position(fn.End()).Line - c.fset.Position(fn.Pos()).Line + 1
		if n > c.cfg.MaxFunctionLines {
			c.add("HR006", fn.Pos(), fmt.Sprintf("func %s has %d lines (max %d)", fn.Name.Name, n, c.cfg.MaxFunctionLines),
				"extract steps into named functions", "long functions hide their steps", "")
		}
	}
}

func (c *fileCheck) tests() {
	if !strings.HasSuffix(c.rel, "_test.go") {
		return
	}
	for _, d := range c.file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && isTest(fn) && !reports(fn) {
			c.add("HR007", fn.Pos(), "test "+fn.Name.Name+" never reports through its *testing.T",
				"assert the result with t.Error/t.Fatal (or t.Run subtests)", "such tests pass whatever the code does", "")
		}
	}
}

func within(p, prefix string) bool { return p == prefix || strings.HasPrefix(p, prefix+"/") }

func activeBans(cfg config, rel string) map[string]ban {
	allowed := map[string]bool{}
	for _, a := range cfg.Allow {
		allowed[a] = true
	}
	out := map[string]ban{}
	for _, b := range append(append([]ban{}, defaultBans...), cfg.Banned...) {
		if allowed[b.Call] || exempt(b.Exempt, rel) {
			continue
		}
		if _, ok := out[b.Call]; !ok {
			out[b.Call] = b
		}
	}
	return out
}

func exempt(globs []string, rel string) bool {
	for _, g := range globs {
		if ok, err := path.Match(g, rel); ok && err == nil {
			return true
		}
	}
	return false
}

func isErrCheck(e ast.Expr) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	x, xok := b.X.(*ast.Ident)
	y, yok := b.Y.(*ast.Ident)
	return xok && yok && y.Name == "nil" && strings.HasSuffix(strings.ToLower(x.Name), "err")
}

func allBlank(es []ast.Expr) bool {
	for _, e := range es {
		if id, ok := e.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	return true
}

func isTest(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	if fn.Recv != nil || name == "TestMain" || !strings.HasPrefix(name, "Test") {
		return false
	}
	return testingParam(fn) != ""
}

func testingParam(fn *ast.FuncDecl) string {
	ps := fn.Type.Params.List
	if len(ps) != 1 || len(ps[0].Names) != 1 {
		return ""
	}
	star, ok := ps[0].Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	if sel, ok := star.X.(*ast.SelectorExpr); !ok || sel.Sel.Name != "T" {
		return ""
	}
	return ps[0].Names[0].Name
}

// reports is true when the test calls a reporting method on its *testing.T
// or hands the T to a helper.
func reports(fn *ast.FuncDecl) bool {
	t := testingParam(fn)
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if found || !ok {
			return !found
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == t && reporters[sel.Sel.Name] {
				found = true
			}
		}
		for _, a := range call.Args {
			if id, ok := a.(*ast.Ident); ok && id.Name == t {
				found = true
			}
		}
		return !found
	})
	return found
}

func report(root, mode string, found []finding) int {
	basePath := filepath.Join(root, ".harness", "baseline.json")
	baseline := map[string]int{}
	if raw, err := os.ReadFile(basePath); err == nil {
		if err := json.Unmarshal(raw, &baseline); err != nil {
			return fail(fmt.Errorf("baseline.json: %w", err))
		}
	}
	counts := map[string]int{}
	for _, f := range found {
		counts[f.rule+" "+f.path]++
	}
	switch mode {
	case "--init-baseline":
		if len(baseline) > 0 {
			return fail(errors.New("the baseline is already recorded; use --update-baseline"))
		}
		return writeBaseline(basePath, counts)
	case "--update-baseline":
		next := map[string]int{}
		for k, old := range baseline {
			if c := counts[k]; c > 0 {
				next[k] = min(c, old)
			}
		}
		return writeBaseline(basePath, next)
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].path != found[j].path {
			return found[i].path < found[j].path
		}
		return found[i].line < found[j].line
	})
	shown := 0
	for _, f := range found {
		if counts[f.rule+" "+f.path] <= baseline[f.rule+" "+f.path] {
			continue
		}
		shown++
		fmt.Printf("%s %s:%d %s\n  FIX: %s\n  WHY: %s\n  SEE: %s\n", f.rule, f.path, f.line, f.what, f.fix, f.why, f.see)
	}
	if shown > 0 {
		fmt.Printf("rules: %d finding(s)\n", shown)
		return 1
	}
	fmt.Println("rules: clean")
	return 0
}

func writeBaseline(p string, counts map[string]int) int {
	raw, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(p, append(raw, '\n'), 0o644); err != nil {
		return fail(err)
	}
	fmt.Printf("rules: baseline has %d key(s)\n", len(counts))
	return 0
}
