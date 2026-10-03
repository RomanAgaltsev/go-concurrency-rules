// Command gate is the admission gate for go-concurrency-rules.
//
//	go run ./internal/cmd/gate check [-root dir] [-j n] [id ...]
//
// Exit status: 0 every checked rule was admitted; 1 at least one violation;
// 2 the gate could not check (bad usage, no rules, no -race on this host).
// A gate that checked nothing must never exit 0.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"

	"github.com/RomanAgaltsev/go-concurrency-rules/internal/gate"
)

const (
	exitOK        = 0
	exitViolation = 1
	exitCannot    = 2
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitCannot
	}
	switch args[0] {
	case "check":
		return runCheck(ctx, args[1:], stdout, stderr)
	case "-h", "-help", "--help", "help":
		usage(stdout)
		return exitOK
	default:
		fmt.Fprintf(stderr, "gate: unknown command %q\n", args[0])
		usage(stderr)
		return exitCannot
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: gate check [-root dir] [-j n] [id ...]

check  run G1 (proof), G2 (embeds) and G6 (front matter) over every rule,
       or over the rules named by id
`)
}

func runCheck(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	jobs := fs.Int("j", runtime.GOMAXPROCS(0), "broken-variant processes to run at once")
	if err := fs.Parse(args); err != nil {
		return exitCannot
	}

	rep, err := gate.Check(ctx, gate.Options{
		Root:   *root,
		IDs:    fs.Args(),
		Prover: gate.Prover{Jobs: *jobs},
	})
	if err != nil {
		fmt.Fprintln(stderr, "gate:", err)
		return exitCannot
	}
	for _, a := range rep.Admitted {
		fmt.Fprintf(stdout, "%s ok  %s\n", a.ID, a.Summary)
	}
	for _, v := range rep.Violations {
		fmt.Fprintln(stdout, v)
	}
	if n := len(rep.Violations); n > 0 {
		fmt.Fprintf(stdout, "gate: %d violation(s); %d rule(s) admitted\n", n, len(rep.Admitted))
		return exitViolation
	}
	fmt.Fprintf(stdout, "gate: %d rule(s) admitted\n", len(rep.Admitted))
	return exitOK
}
