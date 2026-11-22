// Package internal_test holds the architecture rules the Python version
// enforced with import-linter and .harness/rules.py.
package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/iamTheo20/bar-cost-demo/internal/"

type goFile struct {
	path    string
	pkgDir  string
	imports []string
	source  string
}

func goFiles(t *testing.T) []goFile {
	t.Helper()
	var files []goFile
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		gf := goFile{path: filepath.ToSlash(path), pkgDir: filepath.ToSlash(filepath.Dir(path))}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			gf.imports = append(gf.imports, p)
		}
		src, err := os.ReadFile(path)
		gf.source = string(src)
		files = append(files, gf)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestCoreNeverImportsFeatures(t *testing.T) {
	for _, f := range goFiles(t) {
		if !strings.HasPrefix(f.pkgDir, "core") {
			continue
		}
		for _, imp := range f.imports {
			if strings.HasPrefix(imp, module+"features") {
				t.Errorf("%s imports %s: core is shared by every feature and may not depend on one", f.path, imp)
			}
		}
	}
}

func TestFeaturesDoNotImportEachOther(t *testing.T) {
	for _, f := range goFiles(t) {
		if !strings.HasPrefix(f.pkgDir, "features/") {
			continue
		}
		for _, imp := range f.imports {
			if strings.HasPrefix(imp, module+"features/") && imp != module+f.pkgDir {
				t.Errorf("%s imports %s: features use core, never each other", f.path, imp)
			}
		}
	}
}

func TestOnePavedRoad(t *testing.T) {
	rules := []struct {
		call, onlyIn, fix string
	}{
		{"os.Getenv", "core/config", "settings, err := config.Load()"},
		{"os.LookupEnv", "core/config", "settings, err := config.Load()"},
		{"os.Environ", "core/config", "settings, err := config.Load()"},
		{"slog.New(", "core/log", "logger := log.Get(name)"},
		{"slog.SetDefault", "core/log", "logger := log.Get(name)"},
		{"fmt.Print", "", "return the text from the feature; main prints it"},
	}
	for _, f := range goFiles(t) {
		for _, r := range rules {
			if strings.Contains(f.source, r.call) && f.pkgDir != r.onlyIn {
				t.Errorf("%s calls %s; use %s", f.path, r.call, r.fix)
			}
		}
	}
}
