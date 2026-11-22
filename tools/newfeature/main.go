// Command newfeature scaffolds internal/features/<name> with a passing test:
//
//	make new-feature NAME=<name>      (runs go run ./tools/newfeature <name>)
//
// It writes <name>.go and <name>_test.go, refuses to overwrite, and prints
// the two steps it cannot do for you: registering the feature in
// cmd/barcost/main.go and adding its row to FEATURES.md.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

const featureTemplate = `// Package {{name}} is the {{name}} feature: barcost {{name}} [args...].
package {{name}}

// Name is the feature's command name.
const Name = "{{name}}"

// Run returns the feature's output for its arguments.
func Run(args []string) (string, error) {
	return "{{name}}: not implemented yet", nil
}
`

const testTemplate = `package {{name}}

import "testing"

func TestRun(t *testing.T) {
	got, err := Run(nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{{name}}: not implemented yet"; got != want {
		t.Errorf("Run(nil) = %q, want %q", got, want)
	}
}
`

func main() { os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr)) }

func run(args []string, root string, stdout, stderr io.Writer) int {
	if len(args) != 1 || !validName.MatchString(args[0]) {
		fmt.Fprintln(stderr, "usage: make new-feature NAME=<name>   (lowercase letters and digits, starting with a letter)")
		return 2
	}
	name := args[0]
	if err := scaffold(root, name); err != nil {
		fmt.Fprintln(stderr, "newfeature:", err)
		return 1
	}
	fmt.Fprintf(stdout, `created internal/features/%[1]s/%[1]s.go and %[1]s_test.go
next:
  1. register it in cmd/barcost/main.go: import the package and add  %[1]s.Name: %[1]s.Run,  to Features
  2. add a row to FEATURES.md:  | %[1]s | `+"`go run ./cmd/barcost %[1]s`"+` | `+"`<expected output>`"+` |
  3. make check && make verify
`, name)
	return 0
}

func scaffold(root, name string) error {
	dir := filepath.Join(root, "internal", "features", name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s already exists", dir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string]string{name + ".go": featureTemplate, name + "_test.go": testTemplate}
	for file, tmpl := range files {
		body := strings.ReplaceAll(tmpl, "{{name}}", name)
		if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}
