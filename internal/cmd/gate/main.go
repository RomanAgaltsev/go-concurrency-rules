// Command gate is the admission gate for go-concurrency-rules.
//
//	gate check [-root dir] [-j n] [id ...]   prove every rule (or those named)
//	gate gen   [-root dir] [-check]          regenerate the site's index pages
//	gate run   [-root dir] id                run one rule both ways and explain
//
// Build it and run the binary (task gate, task gen and task rule do): `go run`
// reports every non-zero exit as 1, which would erase the contract below.
//
// Exit status: 0 success; 1 a violation, a stale page, or a rule that did not
// behave as declared; 2 the gate could not do its job (bad usage, no rules,
// invalid front matter, no -race where a verdict needs it, interrupted).
// A gate that checked nothing must never exit 0.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"slices"

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
	case "gen":
		return runGen(args[1:], stdout, stderr)
	case "run":
		return runRun(ctx, args[1:], stdout, stderr)
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
       gate gen   [-root dir] [-check]
       gate run   [-root dir] id

check  run G1 (proof), G2 (embeds) and G6 (front matter) over every rule and
       page, or over the rules named by id
gen    regenerate docs/groups, docs/activity and docs/retired/index.md from the
       rules' front matter; -check reports stale pages instead of writing
run    run one rule's fixed and broken variants once and explain what happened
`)
}

// parse parses a subcommand's flags. It reports whether to go on, and the exit
// code when not: asking for help is not an error.
func parse(fs *flag.FlagSet, args []string) (bool, int) {
	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		return false, exitOK
	case err != nil:
		return false, exitCannot
	}
	return true, exitOK
}

func runCheck(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	jobs := fs.Int("j", runtime.GOMAXPROCS(0), "broken-variant processes to run at once")
	if ok, code := parse(fs, args); !ok {
		return code
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
		fmt.Fprintf(stdout, "gate: %d violation(s); %s\n", n, admitted(rep.Admitted))
		return exitViolation
	}
	fmt.Fprintf(stdout, "gate: %s\n", admitted(rep.Admitted))
	return exitOK
}

// admitted totals the admitted entries, the proven apart from the retired ones
// admitted without a proof: those ran nothing.
func admitted(as []gate.Admitted) string {
	proven := 0
	for _, a := range as {
		if a.Proven {
			proven++
		}
	}
	return fmt.Sprintf("%d admitted: %d proven, %d retired without a proof", len(as), proven, len(as)-proven)
}

func runGen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	check := fs.Bool("check", false, "report stale pages instead of writing them")
	if ok, code := parse(fs, args); !ok {
		return code
	}

	rules, unparsed, err := gate.Load(*root)
	if err != nil {
		fmt.Fprintln(stderr, "gate:", err)
		return exitCannot
	}
	if len(rules)+len(unparsed) == 0 {
		fmt.Fprintf(stderr, "gate: no rule pages under %s\n", *root)
		return exitCannot
	}
	// Indexes built from invalid front matter would publish it.
	if vs := slices.Concat(unparsed, gate.Validate(*root, rules)); len(vs) > 0 {
		for _, v := range vs {
			fmt.Fprintln(stderr, v)
		}
		fmt.Fprintf(stderr, "gate: fix the front matter first (%d G6 violation(s))\n", len(vs))
		return exitCannot
	}

	files := gate.Generate(rules)
	if *check {
		stale := gate.StaleGenerated(*root, files)
		for _, p := range stale {
			fmt.Fprintf(stdout, "stale: %s\n", p)
		}
		if len(stale) > 0 {
			fmt.Fprintf(stdout, "gate: %d generated page(s) out of date — run task gen\n", len(stale))
			return exitViolation
		}
		fmt.Fprintf(stdout, "gate: %d generated page(s) up to date\n", len(files))
		return exitOK
	}
	if err := gate.WriteGenerated(*root, files); err != nil {
		fmt.Fprintln(stderr, "gate:", err)
		return exitCannot
	}
	fmt.Fprintf(stdout, "gate: wrote %d page(s)\n", len(files))
	return exitOK
}

func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	if ok, code := parse(fs, args); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: gate run [-root dir] id")
		return exitCannot
	}

	race := gate.RaceAvailable(ctx, "") == nil
	ok, err := gate.Explain(ctx, gate.Options{Root: *root}, fs.Arg(0), race, stdout)
	switch {
	case err != nil:
		fmt.Fprintln(stderr, "gate:", err)
		return exitCannot
	case !ok:
		return exitViolation
	}
	return exitOK
}
