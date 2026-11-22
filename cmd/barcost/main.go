// Command barcost runs one feature by name: barcost <feature> [args...].
//
// Features live in internal/features, one package each, and are listed in
// Features below. Each takes its arguments and returns the text to print.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/iamTheo20/bar-cost-demo/internal/core/config"
	"github.com/iamTheo20/bar-cost-demo/internal/core/log"
	"github.com/iamTheo20/bar-cost-demo/internal/features/hello"
	"github.com/iamTheo20/bar-cost-demo/internal/features/invoice"
	"github.com/iamTheo20/bar-cost-demo/internal/features/margin"
	"github.com/iamTheo20/bar-cost-demo/internal/features/prices"
)

// Prog is the command's name in messages.
const Prog = "barcost"

// Feature runs with its arguments and returns its output.
type Feature func(args []string) (string, error)

// Features maps each command name to its feature.
var Features = map[string]Feature{
	hello.Name:   hello.Run,
	invoice.Name: invoice.Run,
	margin.Name:  margin.Run,
	prices.Name:  prices.Run,
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run is main without the process: 0 on success, 1 when the feature
// fails, 2 for a usage error.
func run(args []string, stdout, stderr io.Writer) int {
	settings, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", Prog, err)
		return 1
	}
	log.Configure(settings, stderr)
	var feature Feature
	if len(args) > 0 {
		feature = Features[args[0]]
	}
	if feature == nil {
		names := make([]string, 0, len(Features))
		for name := range Features {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(stderr, "usage: %s <%s> [args...]\n", Prog, strings.Join(names, "|"))
		return 2
	}
	log.Get("main").Debug("running feature", "feature", args[0])
	out, err := feature(args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", Prog, err)
		return 1
	}
	fmt.Fprintln(stdout, out)
	return 0
}
